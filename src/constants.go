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

type stateChangeReqCommand int

const (
	convertToFollower stateChangeReqCommand = iota
	convertToLeader
)

func (s stateChangeReqCommand) String() string {

	switch s {
	case convertToFollower:
		return "ConvertToFollower"
	case convertToLeader:
		return "ConvertToLeader"
	default:
		panic("Unknown state")
	}

}

type stateChangeReq struct {
	term    int
	newTerm int
	command stateChangeReqCommand
}
