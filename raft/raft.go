package raft

import (
	"log/slog"
	"math/rand"
	"os"
	"time"

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
	peerIds            []int
	electionTimemoutMs int
	electionResetEvent time.Time

	// Fields not related to raft algorithm
	l *slog.Logger
}

type RaftConfig struct {
	Id                   int
	PeerIds              []int
	CurrentTerm          int
	VotedFor             int
	MinElectionTimeoutMs int
}

func NewRaft(config RaftConfig) *Raft {
	utils.Invariant(nil, config.MinElectionTimeoutMs > 0, "MinElectionTimeoutMs should be greater than 0")

	pseudoRandomElectionTimeout := rand.Intn(config.MinElectionTimeoutMs) + config.MinElectionTimeoutMs

	return &Raft{
		currentTerm:        config.CurrentTerm,
		votedFor:           config.VotedFor,
		id:                 config.Id,
		peerIds:            config.PeerIds,
		currentState:       Follower,
		electionTimemoutMs: pseudoRandomElectionTimeout,
		l:                  slog.New(slog.NewTextHandler(os.Stdout, nil)).With("id", config.Id),
	}
}

func (r *Raft) Start() {
	utils.Invariant(r.l, r.currentState == Follower, "Call Start() only on a Follower")

	// Start the election timer
	go r.startElectionTimer()

}

func (r *Raft) startElectionTimer() {
	control := make(chan electionTimerCommand)
	response := make(chan interface{})
	timer := newElectionTimer(r.electionTimemoutMs, control, response)

	go timer.start()

	for range response {
		// Timer has passed start the election

		r.l.Info("Starting election")

		return
	}

	// Response channel has been closed so election timer has been stopped somehow
	return
}
