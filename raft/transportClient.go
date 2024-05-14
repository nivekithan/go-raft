package raft

import (
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

type AppendEntriesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	success bool
}

type TransportClient interface {
	SendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) error
	SendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) error
	Connect() error
	Peers() []int
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
		peers:       config.Peers,
		connections: make(map[int]*rpc.Client),
	}
}

func (r *RpcRaftTransportClient) Connect() error {

	r.mu.Lock()
	defer r.mu.Unlock()

	for peerId, peerAdr := range r.peers {
		r.connectToServer(peerId, peerAdr)
	}

	return nil
}

func (r *RpcRaftTransportClient) connectToServer(server int, addr string) error {
	client, err := rpc.Dial("tcp", addr)

	if err != nil {
		return err
	}

	r.connections[server] = client

	return nil
}

func (r *RpcRaftTransportClient) SendRequestVote(server int, args RequestVoteArgs, reply *RequestVoteReply) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := r.connections[server]

	if client == nil {
		if err := r.connectToServer(server, r.peers[server]); err != nil {

			return err
		}

		client = r.connections[server]
	}

	err := client.Call("Raft.RequestVote", args, reply)

	return err
}

func (r *RpcRaftTransportClient) SendAppendEntries(server int, args AppendEntriesArgs, reply *AppendEntriesReply) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	client := r.connections[server]

	if client == nil {
		if err := r.connectToServer(server, r.peers[server]); err != nil {
			return err
		}
		client = r.connections[server]
	}

	err := client.Call("Raft.AppendEntries", args, reply)

	return err
}

func (r *RpcRaftTransportClient) Peers() []int {
	r.mu.Lock()
	defer r.mu.Unlock()

	peers := make([]int, 0, len(r.peers))

	for peerId := range r.peers {
		peers = append(peers, peerId)
	}

	return peers
}
