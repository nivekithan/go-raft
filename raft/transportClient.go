package raft

import (
	"fmt"
	"net/rpc"
	"sync"
)

type RequestVoteArgs struct {
	Term        int
	CandidateId int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type TransportClient interface {
	SendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) error
	Connect() error
}

type RpcRaftTransportClient struct {
	mu          sync.Mutex
	peers       map[int]string
	connections map[int]*rpc.Client
}

type RpcRaftTransportConfig struct {
	Peers map[int]string
}

func NewRpcRaftTransportClient(config RpcRaftTransportConfig) *RpcRaftTransportClient {
	return &RpcRaftTransportClient{
		peers: config.Peers,
	}
}

func (r *RpcRaftTransportClient) Connect() error {

	r.mu.Lock()
	defer r.mu.Unlock()

	for peerId, peerAdr := range r.peers {
		client, err := rpc.Dial("tcp", peerAdr)

		if err != nil {
			return err
		}

		r.connections[peerId] = client
	}

	return nil
}

func (r *RpcRaftTransportClient) SendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := r.connections[server]

	if client == nil {
		return fmt.Errorf("No connection to server %d", server)
	}

	err := client.Call("Raft.RequestVote", args, reply)

	return err
}
