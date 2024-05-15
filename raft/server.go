package raft

import (
	"fmt"
	"net"
	"net/rpc"
)

func (r *Raft) RequestVote(args RequestVoteArgs, reply *RequestVoteReply) error {
	r.l.Debug("Received RequestVote RPC", "args", args)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.incomingTerm(args.Term)

	reply.Term = r.currentTerm

	if args.Term < r.currentTerm {
		reply.VoteGranted = false
		return nil
	}

	if r.votedFor == -1 || r.votedFor == args.CandidateId {
		reply.VoteGranted = true
	} else {
		reply.VoteGranted = false
	}

	return nil
}

func (r *Raft) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	r.l.Debug("Received AppendEntries RPC", "args", args)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.incomingTerm(args.Term)

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

// Assumes the mutex is locked
func (r *Raft) incomingTerm(newTerm int) {

	if r.currentTerm > newTerm {
		// Do nothing
		return
	}

	if r.currentTerm == newTerm {
		r.becomeFollower(newTerm, r.votedFor)
	}

	if r.currentTerm < newTerm {
		r.becomeFollower(newTerm, -1)
	}
}
