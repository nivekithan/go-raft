package raft

import (
	"fmt"
	"log"
	"net"
	"net/rpc"
	"sync"
)

type Server struct {
	mu sync.Mutex

	id          int
	peerIds     []int
	peerClients map[int]*rpc.Client

	rpcServer *rpc.Server
	listener  net.Listener

	cm *ConsensusModule
}

func NewServer(id int, peerIds []int) *Server {
	server := new(Server)
	cm := NewConsensusModule(id, peerIds, 150, server)

	server.id = id
	server.peerIds = peerIds
	server.cm = cm
	server.peerClients = make(map[int]*rpc.Client)

	server.rpcServer = rpc.NewServer()
	server.rpcServer.RegisterName("cm", cm)

	addr := fmt.Sprintf(":%d", 5000+id)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatalf("Error on listening on addrr %v : %v", addr, err)
	}

	server.listener = listener
	log.Printf("Listening on addr %v", addr)

	return server
}

// Sets the rpc server and starts execution of all its modules
func (s *Server) Start() {
	// ConsensusModule
	s.cm.Start()

	// Start RPC Server
	go func() {
		for {
			conn, err := s.listener.Accept()

			if err != nil {
				log.Fatal("Listener Accpet err:", err)
			}

			go func() {
				s.rpcServer.ServeConn(conn)
			}()

		}
	}()
}

type CallRequestVoteArgs struct {
	Term        int
	CandidateId int
}

type CallRequestVoteReply struct {
	Term        int
	VoteGranted bool
}

// Expects the s.mu to be Unlocked
func (s *Server) callRequestVote(peerId int, args CallRequestVoteArgs, reply *CallRequestVoteReply) error {
	s.mu.Lock()
	client := s.getClient(peerId)
	s.mu.Unlock()

	if err := client.Call("cm.RequestVoteRpc", args, reply); err != nil {
		return err
	}

	return nil
}

type CallAppendEntriesArgs struct {
	Term     int
	LeaderId int
}

type CallAppendEntriesReply struct {
	Term    int
	Success bool
}

func (s *Server) callAppendEntries(peerId int, args CallAppendEntriesArgs, reply *CallAppendEntriesReply) error {
	s.mu.Lock()
	client := s.getClient(peerId)
	s.mu.Unlock()

	if err := client.Call("cm.AppendEntriesRpc", args, reply); err != nil {
		return err
	}

	return nil
}

// Expects s.mu to be locked
func (s *Server) getClient(peerId int) *rpc.Client {

	if s.peerClients[peerId] == nil {
		addr := fmt.Sprintf(":%v", 5000+peerId)
		client, err := rpc.Dial("tcp", addr)

		if err != nil {
			log.Fatalf("Error while getting client for peerId: %v, error: %v", peerId, err)
		}

		s.peerClients[peerId] = client

	}
	return s.peerClients[peerId]
}
