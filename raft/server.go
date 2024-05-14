package raft

import (
	"fmt"
	"net"
	"net/rpc"
)

func (r *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	r.l.Debug("Received RequestVote RPC", "args", args)

	return nil
}

func (r *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	r.l.Debug("Received AppendEntries RPC", "args", args)

	return nil
}

func (r *Raft) startListenRpc() {
	server := rpc.NewServer()

	server.Register(r)
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", r.id+9000))

	if err != nil {
		panic(err)
	}

	r.l.Info("Listening for RPC")
	server.Accept(l)
}
