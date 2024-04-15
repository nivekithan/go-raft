package raft

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

type CmState int

const NullId = 0

const (
	Follower CmState = iota
	Candidate
	Leader
)

func (s CmState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	default:
		panic("Unknown state")
	}
}

type ConsensusModule struct {
	mu sync.Mutex

	id      int
	peerIds []int
	state   CmState

	// Persistant state
	currentTerm int
	votedFor    int
	log         []LogEntry

	// Timers
	electionTimeout    int
	electionResetEvent time.Time

	server *Server
}

func NewConsensusModule(id int, peerIds []int, electionTimeout int, server *Server) *ConsensusModule {
	randomElectionTimeout := rand.Intn(electionTimeout) + electionTimeout

	cm := &ConsensusModule{
		id:              id,
		state:           Follower,
		peerIds:         peerIds,
		currentTerm:     0,
		votedFor:        NullId,
		electionTimeout: randomElectionTimeout,
		server:          server,
	}

	return cm
}

// Starts the execution of ConsensusModule
func (cm *ConsensusModule) Start() {

	// Start election timer
	go func() {
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()

		cm.startElectionTimer()
	}()
}

func (cm *ConsensusModule) AppendEntriesRpc(args CallAppendEntriesArgs, reply *CallAppendEntriesReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Printf("Answering AppendEntriesRpc %v", args)

	currentTerm := cm.currentTerm

	if args.Term > currentTerm {
		cm.becomeFollower(args.Term)
	}

	reply.Term = currentTerm

	if currentTerm > args.Term {
		reply.Success = false
		return nil
	}

	cm.electionResetEvent = time.Now()

	reply.Success = true
	return nil
}

func (cm *ConsensusModule) RequestVoteRpc(args CallRequestVoteArgs, reply *CallRequestVoteReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	log.Printf("Answering RequestVoteRpc %v", args)
	currentTerm := cm.currentTerm

	if args.Term > currentTerm {
		cm.becomeFollower(args.Term)
	}

	reply.Term = currentTerm
	currentVotedFor := cm.votedFor

	if currentTerm > args.Term {
		log.Printf("Vote not granted since currentTerm is higher than candidate term")
		reply.VoteGranted = false
		return nil
	}

	if currentVotedFor != NullId && currentVotedFor != args.CandidateId {
		log.Printf("Vote not granted since voted for another candidate in this server")
		reply.VoteGranted = false
		return nil
	}

	// Check the candidateLog is atleast update to date with our log

	log.Printf("Granted vote to candidate")
	cm.electionResetEvent = time.Now()

	reply.VoteGranted = true
	cm.votedFor = args.CandidateId

	return nil
}

// Timer which runs when cmState is Follower and Candidate
func (cm *ConsensusModule) startElectionTimer() {
	log.Printf("Starting Election Timer")
	cm.mu.Lock()
	termElectionStartedAt := cm.currentTerm
	cm.mu.Unlock()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		cm.mu.Lock()
		if cm.state != Candidate && cm.state != Follower {
			// Candidate become leader so bail out of electionTimer
			cm.mu.Unlock()
			return
		}

		if cm.currentTerm != termElectionStartedAt {
			// If the candidate starts reElection then the currentTerm will get
			// incresed and starts new electionTimer. In that case old election timer
			// should bail out
			cm.mu.Unlock()
			return
		}

		if time.Since(cm.electionResetEvent) > time.Duration(cm.electionTimeout)*time.Millisecond {

			// ElectionTimeout has passed. We should start the election
			cm.startElection()
			cm.mu.Unlock()
			return
		}

		cm.mu.Unlock()

	}
}

// Expects the cm.mu to be locked
func (cm *ConsensusModule) startElection() {
	// Steps to follow while starting a election
	// 1. Increase the term by 1
	// 2. Transition to canidate state
	// 3. Vote for yourselves
	// 4. start new electionTimer with updated electionResetEvent
	// 4. Issue RequestVoteRpc's to all peers in parallel
	//
	// Now there are three possibilities
	// - It wins the election
	// - Another server estabilishes as leader
	// - ElectionTimer has passed again

	log.Printf("Starting Election")

	cm.currentTerm++
	savedCurrentTerm := cm.currentTerm
	savedId := cm.id
	cm.state = Candidate
	cm.votedFor = cm.id
	cm.electionResetEvent = time.Now()

	// This acts as timeout for election. We should majority of vote within electionTimer
	// otherwise new election will start
	go cm.startElectionTimer()

	votesReceived := 1
	var vrMu sync.Mutex

	for _, peerId := range cm.peerIds {
		go func(peerId int) {
			requestVoteReply := new(CallRequestVoteReply)

			requestVoteArgs := CallRequestVoteArgs{Term: savedCurrentTerm, CandidateId: savedId}

			if err := cm.server.callRequestVote(peerId, requestVoteArgs, requestVoteReply); err != nil {
				// if rpc error's do nothing
				log.Printf("Error on callRequestVote client: %v", err)
				return
			}

			cm.mu.Lock()
			defer cm.mu.Unlock()

			if cm.state != Candidate {
				// State got changed while we are making a rpc. Bail out of it
				log.Printf("State changed while waiting for requestVoteReply to %v", cm.state.String())
				return
			}

			if cm.currentTerm != savedCurrentTerm {
				// Term got updated while  we are making a rpc. Bail out of it
				log.Printf("CurrentTerm changed while waiting for requestVoteReply to %v", cm.currentTerm)
				return
			}

			if requestVoteReply.Term > cm.currentTerm {
				// Found another server with higher term. Become follower
				cm.becomeFollower(requestVoteReply.Term)
				return
			}

			if requestVoteReply.VoteGranted {
				vrMu.Lock()
				votesReceived += 1

				isMajorityVoted := votesReceived*2 > len(cm.peerIds)+1

				if isMajorityVoted {
					// Since majority voted. Let transition to leader
					cm.becomeLeader()
				}
				vrMu.Unlock()

			}

		}(peerId)
	}

}

func (cm *ConsensusModule) leaderSendHeartbeat() {
	cm.mu.Lock()
	peerIds := cm.peerIds
	savedCurrentTerm := cm.currentTerm
	savedId := cm.id
	cm.mu.Unlock()

	for _, peerId := range peerIds {
		go func(peerId int) {
			args := CallAppendEntriesArgs{Term: savedCurrentTerm, LeaderId: savedId}
			reply := new(CallAppendEntriesReply)

			if err := cm.server.callAppendEntries(peerId, args, reply); err != nil {
				log.Printf("CallAppendEntries client error %v", err)
				// Ignore if it errors
				return
			}

			cm.mu.Lock()
			defer cm.mu.Unlock()

			if cm.state != Leader {
				// State is not leader bail out
				return
			}

			if cm.currentTerm != savedCurrentTerm {
				// Term has changed while waiting for callAppendEntires reply. Bail out
				return
			}

			if reply.Term > cm.currentTerm {
				cm.becomeFollower(reply.Term)
			}

		}(peerId)
	}

}

// Expects cm.mu to be locked
func (cm *ConsensusModule) becomeLeader() {
	if cm.state == Leader {
		// Already leader ignoring this command
		return
	}

	cm.state = Leader

	go func() {
		// Loop which sends Heartbeat timer
		heartbeatTicker := time.NewTicker(500 * time.Millisecond)
		defer heartbeatTicker.Stop()

		for {
			cm.leaderSendHeartbeat()
			<-heartbeatTicker.C

			cm.mu.Lock()

			if cm.state != Leader {
				cm.mu.Unlock()
				// If the state is not leader anymore. Bail out of it
				return
			}

			cm.mu.Unlock()
		}

	}()

}

// Expects cm.mu to be Locked
func (cm *ConsensusModule) becomeFollower(newTerm int) {
	cm.currentTerm = newTerm
	cm.state = Follower
	cm.electionResetEvent = time.Now()
}

type LogEntry struct {
	Term    int
	Command interface{}
}
