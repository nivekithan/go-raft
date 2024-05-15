package raft

import (
	"log/slog"
	"math/rand"
	"os"
	"sync"

	"github.com/nivekithan/go-raft/utils"
)

type RaftState int

const (
	Leader RaftState = iota
	Follower
	Candidate
)

func (s RaftState) String() string {
	switch s {
	case Leader:
		return "Leader"
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	default:
		return "Unknown"
	}
}

type Raft struct {

	// Persistant state
	currentTerm int
	votedFor    int

	// Other variables
	currentState       RaftState
	id                 int
	electionTimemoutMs int
	heartbeatTimeoutMs int

	// Fields not related to raft algorithm
	l                 *slog.Logger
	transporterClient TransportClient
	electionTimer     *timer
	heartbeatTimer    *timer
	mu                sync.Mutex
}

type RaftConfig struct {
	Id                   int
	CurrentTerm          int
	VotedFor             int
	MinElectionTimeoutMs int
	HeartbeatTimeoutMs   int
	TransporterClient    TransportClient
}

func NewRaft(config RaftConfig) *Raft {
	utils.Invariant(nil, config.MinElectionTimeoutMs > 0, "MinElectionTimeoutMs should be greater than 0")
	utils.Invariant(nil, config.HeartbeatTimeoutMs > 0, "HeartbeatTimeoutMs should be greater than 0")
	utils.Invariant(nil, config.MinElectionTimeoutMs > config.HeartbeatTimeoutMs, "MinElectionTimeoutMs should be greater than HeartbeatTimeoutMs")

	pseudoRandomElectionTimeout := rand.Intn(config.MinElectionTimeoutMs)*2 + config.MinElectionTimeoutMs

	return &Raft{
		currentTerm:        config.CurrentTerm,
		votedFor:           config.VotedFor,
		id:                 config.Id,
		currentState:       Follower,
		electionTimemoutMs: pseudoRandomElectionTimeout,
		heartbeatTimeoutMs: config.HeartbeatTimeoutMs,

		l: slog.New(slog.NewTextHandler(
			os.Stdout,
			&slog.HandlerOptions{
				Level: slog.LevelDebug,
			},
		)).With("id", config.Id),

		transporterClient: config.TransporterClient,
	}
}

func (r *Raft) Start() {
	utils.Invariant(r.l, r.currentState == Follower, "Call Start() only on a Follower")

	go r.startListenRpc()

	if err := r.transporterClient.Connect(); err != nil {
		r.l.Error(err.Error())
	}
	// Start the election timer
	go r.startElectionTimer()

}

// Blocks until the election timer is stopped or timeout occurs
func (r *Raft) startElectionTimer() {
	response := make(chan interface{})

	timer := newTimer(r.electionTimemoutMs, response, r.l.With("timer", "election"))

	go timer.start()

	r.mu.Lock()
	electionStartedAtTerm := r.currentTerm
	r.electionTimer = timer
	r.mu.Unlock()

	for range response {

		r.mu.Lock()

		if electionStartedAtTerm != r.currentTerm {
			// Term has changed so no need to start election
			r.mu.Unlock()
			return
		}

		timer.stop()
		// Term has not changed so we can start the election
		r.startElection()

		r.mu.Unlock()
		// Once the election has started we can stop the timer
	}

	// Response channel has been closed so election timer has been stopped
	return
}

// Assumes the mutex is not locked and blocks until the heartbeat timer is stopped or timeout occurs
func (r *Raft) startHeartbeatTimer() {
	r.l.Info("Starting heartbeat timer")
	response := make(chan interface{})

	timer := newTimer(r.heartbeatTimeoutMs, response, r.l.With("timer", "heartbeat"))

	go timer.start()

	r.mu.Lock()
	r.heartbeatTimer = timer
	r.mu.Unlock()

	for range response {
		r.mu.Lock()
		if r.currentState != Leader {
			// We are no longer a leader
			r.mu.Unlock()
			timer.stop()
			return
		}

		for _, peerId := range r.transporterClient.Peers() {

			go func(peerId int, currentTerm int, leaderId int) {
				r.l.Debug("Sending AppendEntriesArgs to peer", "peerId", peerId)

				reply := &AppendEntriesReply{}

				err := r.transporterClient.SendAppendEntries(peerId, AppendEntriesArgs{Term: currentTerm, LeaderId: leaderId}, reply)

				if err != nil {
					r.l.Error(err.Error())
					return
				}

				r.l.Debug("Received AppendEntriesReply from peer", "peerId", peerId, "reply", reply)

				r.mu.Lock()
				r.incomingTerm(reply.Term)
				r.mu.Unlock()

			}(peerId, r.currentTerm, r.id)
		}
		r.mu.Unlock()

		timer.reset()

	}

	return
}

// Function blocks and expects the mutex to be locked
func (r *Raft) startElection() {

	if r.currentState == Leader {
		// Ignore startElection request
		return
	}

	r.l.Info("Starting election")

	r.currentTerm++
	r.currentState = Candidate
	r.votedFor = r.id

	electionStartedAtTerm := r.currentTerm

	go r.startElectionTimer()
	go func() {

		resChan := make(chan bool)

		for _, peerId := range r.transporterClient.Peers() {
			r.l.Debug("Sending RequestVoteArgs to peer", "peerId", peerId)

			go func(peerId int, currentTerm int, candidateId int) {
				reply := &RequestVoteReply{}
				err := r.transporterClient.SendRequestVote(
					peerId,
					RequestVoteArgs{
						Term:        currentTerm,
						CandidateId: candidateId,
					},
					reply,
				)

				if err != nil {
					r.l.Error(err.Error())
				}

				r.l.Debug("Received RequestVoteReply from peer", "peerId", peerId, "reply", reply)

				r.mu.Lock()
				r.incomingTerm(reply.Term)
				r.mu.Unlock()

				resChan <- reply.VoteGranted
			}(peerId, electionStartedAtTerm, r.id)
		}

		// We already voted for ourselves
		totalPositiveVotes := 1
		isElectionFinished := false
		for voteGranted := range resChan {
			if voteGranted {
				totalPositiveVotes++
			}

			if totalPositiveVotes*2 > len(r.transporterClient.Peers())+1 && !isElectionFinished {
				// We have won the election let's verify the currentTerm and then change to leader
				r.mu.Lock()

				if r.currentTerm == electionStartedAtTerm {
					isElectionFinished = true
					r.l.Info("Becoming Leader")
					r.becomeLeader()
				}
				r.mu.Unlock()
			}

		}
	}()
	return
}

// Assumes the mutex is locked
func (r *Raft) becomeLeader() {
	r.currentState = Leader

	if r.electionTimer != nil {
		r.electionTimer.stop()
		r.electionTimer = nil
	}

	go r.startHeartbeatTimer()
}

// Assumes the mutex is locked
func (r *Raft) becomeFollower(term int, votedFor int) {
	r.currentState = Follower
	r.currentTerm = term
	r.votedFor = votedFor

	if r.heartbeatTimer != nil {
		r.heartbeatTimer.stop()
	}

	if r.electionTimer != nil {
		r.electionTimer.stop()
	}

	go r.startElectionTimer()
}
