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
}

func NewRaft(config RaftConfig) *Raft {

	return &Raft{
		id:                  config.Id,
		peers:               config.Peers,
		electionTimeout:     config.ElectionTimeout,
		heartbeatTimeout:    config.HeartbeatTimeout,
		votedFor:            NullId,
		state:               Follower,
		currentTerm:         0,
		electionTimeoutChan: make(chan int),
	}
}

func (r *Raft) Start() {

	for {
		switch r.state {
		case Follower:
			r.setNewElectionTimer()
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

}

func (r *Raft) startLeaderLoop() {

}

func (r *Raft) startElection() {
	r.currentTerm++
	r.votedFor = NullId
	r.state = Candidate
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
