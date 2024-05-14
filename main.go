package main

import (
	"github.com/nivekithan/go-raft/raft"
)

func main() {
	raft := raft.NewRaft(raft.RaftConfig{
		Id:                   1,
		PeerIds:              []int{},
		CurrentTerm:          0,
		VotedFor:             -1,
		MinElectionTimeoutMs: 150,
	})

	raft.Start()

	for {

	}

}
