package raft

import (
	"log/slog"
	"os"
	"time"
)

type RaftConfig struct {
	Id               int
	Peers            []int
	ElectionTimeout  int
	HeartbeatTimeout int
}

type Raft struct {

	// Raft config variables
	id               int
	peers            []int
	electionTimeout  int
	heartbeatTimeout int
	state            RaftState

	// Raft Perist variables
	currentTerm int
	votedFor    int

	// Internal Variables
	electionTimer       *timer
	electionTimeoutChan chan int

	heartbeatTimer       *timer
	heartbeatTimeoutChan chan int

	stateChangeChan chan stateChangeReq

	requestVoteRpcArgsChan  chan RequestVoteArgs
	requestVoteRpcReplyChan chan RequestVoteReply

	appendEntriesRpcArgsChan  chan AppendEntiesArgs
	appendEntriesRpcReplyChan chan AppendEntriesReply

	// Logging
	l *slog.Logger
}

func NewRaft(config RaftConfig) *Raft {

	raft := &Raft{
		id:                        config.Id,
		peers:                     config.Peers,
		electionTimeout:           config.ElectionTimeout,
		heartbeatTimeout:          config.HeartbeatTimeout,
		votedFor:                  NullId,
		state:                     Follower,
		currentTerm:               0,
		electionTimeoutChan:       make(chan int),
		heartbeatTimeoutChan:      make(chan int),
		stateChangeChan:           make(chan stateChangeReq),
		requestVoteRpcArgsChan:    make(chan RequestVoteArgs),
		requestVoteRpcReplyChan:   make(chan RequestVoteReply),
		appendEntriesRpcArgsChan:  make(chan AppendEntiesArgs),
		appendEntriesRpcReplyChan: make(chan AppendEntriesReply),
		l:                         slog.New(slog.NewTextHandler(os.Stdout, nil)).With("id", config.Id),
	}

	return raft
}

func (r *Raft) Start() {
	r.setNewElectionTimer()
	go r.startRpcServer()
	r.mainLoop()
}

func (r *Raft) State() RaftState {
	return r.state
}

func (r *Raft) Id() int {
	return r.id
}

func (r *Raft) mainLoop() {
	for {
		switch r.state {
		case Follower:
			r.startFollowerLoop()
		case Candidate:
			r.startCandidateLoop()
		case Leader:
			r.startLeaderLoop()
		}
	}
}

func (r *Raft) startFollowerLoop() {
	for r.state == Follower {
		select {
		case term := <-r.electionTimeoutChan:

			if term != r.currentTerm {
				// Ignore this timeout
				continue
			}

			r.startElection()

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		}
	}
}

func (r *Raft) startCandidateLoop() {
	for r.state == Candidate {
		select {
		case term := <-r.electionTimeoutChan:
			if term != r.currentTerm {
				// Ignore this command
				continue
			}
			r.startElection()

		case resFromReqeustVote := <-r.stateChangeChan:
			if resFromReqeustVote.term != r.currentTerm {
				continue
			}

			if resFromReqeustVote.command == convertToFollower {
				r.convertToFollower(resFromReqeustVote.newTerm)
			} else if resFromReqeustVote.command == convertToLeader {
				r.convertToLeader()
			}

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		}

	}

}

func (r *Raft) startLeaderLoop() {

	for r.state == Leader {
		select {
		case term := <-r.heartbeatTimeoutChan:
			if term != r.currentTerm {
				// Ignore this message
				continue
			}

			r.sendHearbeats()

		case stateChangeReq := <-r.stateChangeChan:
			if stateChangeReq.term != r.currentTerm {
				// Ignore this request
				continue
			}

			if stateChangeReq.command == convertToFollower {
				r.convertToFollower(stateChangeReq.newTerm)
			}

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		}

	}

}

func (r *Raft) respondToRequestVoteRpc(args RequestVoteArgs) {

	if r.currentTerm > args.Term {
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: false}
	}

	if args.Term > r.currentTerm {
		r.convertToFollower(args.Term)
	}

	if r.votedFor == NullId || r.votedFor == args.CandidateId {
		r.votedFor = args.CandidateId
		r.setNewElectionTimer()
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: true}
	} else {
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: false}
	}
}

func (r *Raft) respondToAppendEntriesRpc(args AppendEntiesArgs) {
	if r.currentTerm > args.Term {
		r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: false}
	}

	if args.Term > r.currentTerm {
		r.convertToFollower(args.Term)
	}

	r.setNewElectionTimer()
	r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: true}
}

func (r *Raft) startElection() {
	r.currentTerm++
	r.votedFor = r.id
	r.state = Candidate

	requestVoteFromAll(r.peers, r.currentTerm, r.id, r.stateChangeChan)
	r.setNewElectionTimer()
}

func (r *Raft) convertToFollower(newTerm int) {
	r.currentTerm = newTerm
	r.votedFor = -1
	r.state = Follower
	r.setNewElectionTimer()

	if r.heartbeatTimer != nil {
		CancelTimer(r.heartbeatTimer)
	}
}

func (r *Raft) convertToLeader() {
	r.state = Leader
	r.sendHearbeats()
}

func (r *Raft) sendHearbeats() {
	sendAppendEntiresToAll(r.peers, r.currentTerm, r.id, r.stateChangeChan)
	r.setNewHeartbeatTimer()

}

func (r *Raft) setNewElectionTimer() {
	newElectionTimeout := randomTimeout(r.electionTimeout)

	if r.electionTimer != nil {
		CancelTimer(r.electionTimer)
	}

	r.electionTimer = NewAndStartTimer(
		time.Duration(newElectionTimeout)*time.Millisecond,
		r.currentTerm,
		r.electionTimeoutChan,
	)
}

func (r *Raft) setNewHeartbeatTimer() {
	if r.heartbeatTimer != nil {
		CancelTimer(r.heartbeatTimer)
	}

	r.heartbeatTimer = NewAndStartTimer(
		time.Duration(r.heartbeatTimeout)*time.Millisecond,
		r.currentTerm,
		r.heartbeatTimeoutChan,
	)
}
