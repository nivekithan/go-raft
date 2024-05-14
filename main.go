package main

import (
	"github.com/nivekithan/go-raft/raft"
)

func main() {

	defaultRaftTransport := raft.NewRpcRaftTransportClient(raft.RpcRaftTransportConfig{
		Peers: map[int]string{2: ":9002"},
	})

	raftServer := raft.NewRaft(raft.RaftConfig{
		Id:                   1,
		CurrentTerm:          0,
		VotedFor:             -1,
		MinElectionTimeoutMs: 150,
		TransporterClient:    defaultRaftTransport,
	})

	raftServer.Start()

	defaultRaftTransport = raft.NewRpcRaftTransportClient(raft.RpcRaftTransportConfig{
		Peers: map[int]string{1: ":9001"},
	})

	raftServer = raft.NewRaft(raft.RaftConfig{
		Id:                   2,
		CurrentTerm:          0,
		VotedFor:             -1,
		MinElectionTimeoutMs: 150,
		TransporterClient:    defaultRaftTransport,
	})

	raftServer.Start()

	for {

	}

}
