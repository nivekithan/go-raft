package raft

import (
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

	requestVoteResChan chan requestVoteRes
}

func NewRaft(config RaftConfig) *Raft {

	return &Raft{
		id:                   config.Id,
		peers:                config.Peers,
		electionTimeout:      config.ElectionTimeout,
		heartbeatTimeout:     config.HeartbeatTimeout,
		votedFor:             NullId,
		state:                Follower,
		currentTerm:          0,
		electionTimeoutChan:  make(chan int),
		heartbeatTimeoutChan: make(chan int),
		requestVoteResChan:   make(chan requestVoteRes),
	}
}

func (r *Raft) Start() {
	r.setNewElectionTimer()
	r.mainLoop()
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

		case resFromReqeustVote := <-r.requestVoteResChan:
			if resFromReqeustVote.term != r.currentTerm {
				// Ignore this command
				continue
			}

			if resFromReqeustVote.command == convertToFollower {
				r.convertToFollower(resFromReqeustVote.newTerm)
			} else if resFromReqeustVote.command == convertToLeader {
				r.convertToLeader()
			}

		}

	}

}

func (r *Raft) startLeaderLoop() {

}

func (r *Raft) startElection() {
	r.currentTerm++
	r.votedFor = r.id
	r.state = Candidate

	requestVoteFromAll(r.peers, r.currentTerm, r.id, r.requestVoteResChan)
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

	// TODO:
	// Send heartbeats
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
