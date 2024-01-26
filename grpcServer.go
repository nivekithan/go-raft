package raft

import (
	"context"
	"fmt"
	"raft/rpc"

	"go.uber.org/zap"
)

func (r *Raft) RequestVote(_ context.Context, args *rpc.RequestVoteArgs) (*rpc.RequestVoteReply, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	logger := r.l.With(zap.String("group", "requestVote"))

	logger.Debug("Responding to RequestVoteRPC")

	isCurrentTermBigger := r.currentTerm > int(args.Term)
	if isCurrentTermBigger {
		logger.Info(
			fmt.Sprintf(
				"Current Term (%d) is bigger then incoming term (%d). Therefore rejecting the vote",
				r.currentTerm,
				args.Term,
			),
		)
		return &rpc.RequestVoteReply{Term: int64(r.currentTerm), VoteGranted: false}, nil
	}

	canVoteToCandidate := r.votedFor == -1 || r.votedFor == int(args.CandidateId)

	if !canVoteToCandidate {
		logger.Info(fmt.Sprintf("Already voted to another candidate (%d). Therefore rejecting the vote", r.votedFor))

		return &rpc.RequestVoteReply{Term: int64(r.currentTerm), VoteGranted: false}, nil
	}

	currentLogLength := len(r.log)
	candidateLogLength := args.LastLogIndex + 1

	isCandidateLogLengthGreaterOrEqual := candidateLogLength >= int64(currentLogLength)

	if !isCandidateLogLengthGreaterOrEqual {
		logger.Info(
			fmt.Sprintf(
				"Candidate logLength (%d) is not greater or equal to currentLogLength (%d). Therefore rejecting the vote",
				candidateLogLength,
				currentLogLength,
			),
		)

		return &rpc.RequestVoteReply{Term: int64(r.currentTerm), VoteGranted: false}, nil
	}

	if candidateLogLength == int64(currentLogLength) {
		isLogIndexMatch := r.log[args.LastLogIndex].term == int(args.LastLogTerm)

		if !isLogIndexMatch {
			logger.Info(fmt.Sprintf("Candidate lastLogTerm (%d) does match", args.LastLogTerm))
			return &rpc.RequestVoteReply{Term: int64(r.currentTerm), VoteGranted: false}, nil
		}
	}

	logger.Info(fmt.Sprintf("Providing vote to candidate %d", args.CandidateId))

	return &rpc.RequestVoteReply{Term: int64(r.currentTerm), VoteGranted: true}, nil
}
