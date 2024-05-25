package raft

import (
	"errors"
	"fmt"
	"net"
	"net/rpc"
)

func (r *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	if !r.connected {
		return errors.New("Unable to process this request")
	}

	r.dlog("Responding to RequestVote RPC")
	r.requestVoteRpcArgsChan <- args
	*reply = <-r.requestVoteRpcReplyChan
	r.dlog("Responded to RequestVote")
	return nil
}

func (r *Raft) AppendEntries(args AppendEntiesArgs, reply *AppendEntriesReply) error {
	if !r.connected {
		return errors.New("Unable to process this request")
	}

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

	r.ilog(fmt.Sprintf("Started listening on port: :%v", 9000+r.id))

	if err != nil {
		panic(err)
	}

	r.rpcServerListern = lis
	for {
		conn, err := lis.Accept()

		if err != nil {
			return
		}

		r.stopServerWg.Add(1)
		go func() {
			rpcServer.ServeConn(conn)
			r.stopServerWg.Done()
		}()
	}
}

func (r *Raft) stopRpcServer() {
	r.rpcServerListern.Close()
	r.stopServerWg.Wait()
	r.ilog(fmt.Sprintf("Stopped listening on port :%v", 9000+r.id))
}
