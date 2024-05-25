package raft

import (
	"fmt"
	"net"
	"net/rpc"
)

func (r *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	r.requestVoteRpcArgsChan <- args

	*reply = <-r.requestVoteRpcReplyChan

	return nil
}

func (r *Raft) AppendEntries() {

}

func (r *Raft) startRpcServer() {

	rpcServer := rpc.NewServer()

	rpcServer.RegisterName("Raft", r)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", 9000+r.id))

	if err != nil {
		panic(err)
	}

	rpcServer.Accept(lis)
}
