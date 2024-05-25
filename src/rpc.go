package raft

import (
	"fmt"
	"net"
	"net/rpc"
)

func (r *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	r.dlog("Responding to RequestVote RPC")
	r.requestVoteRpcArgsChan <- args
	*reply = <-r.requestVoteRpcReplyChan
	r.dlog("Responded to RequestVote")
	return nil
}

func (r *Raft) AppendEntries(args AppendEntiesArgs, reply *AppendEntriesReply) error {
	r.dlog("Responding to AppendEntries")
	r.appendEntriesRpcArgsChan <- args

	*reply = <-r.appendEntriesRpcReplyChan
	r.dlog("Responded to AppendEntries")
	return nil
}

func (r *Raft) startRpcServer() {

	rpcServer := rpc.NewServer()

	rpcServer.RegisterName("Raft", r)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000+r.id))

	if err != nil {
		panic(err)
	}

	r.ilog("Starting rpc server")
	rpcServer.Accept(lis)
}
