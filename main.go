package main

import (
	"github.com/nivekithan/go-raft/raft"
)

func main() {
	defaultRaftTransport := raft.NewRpcRaftTransportClient(raft.RpcRaftTransportConfig{
		Peers: make(map[int]string),
	})

	raft := raft.NewRaft(raft.RaftConfig{
		Id:                   1,
		CurrentTerm:          0,
		VotedFor:             -1,
		MinElectionTimeoutMs: 150,
		TransporterClient:    defaultRaftTransport,
	})

	raft.Start()

	for {

	}

}
