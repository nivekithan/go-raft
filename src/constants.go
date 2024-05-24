package raft

type RaftState int

const (
	Follower RaftState = iota
	Leader
	Candidate
)

func (r RaftState) String() string {
	switch r {
	case Follower:
		return "Follower"
	case Leader:
		return "Leader"
	case Candidate:
		return "Candidate"
	default:
		panic("Unknown raftState")
	}
}

const (
	NullId = -1
)