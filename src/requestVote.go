package raft

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
)

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type requestVoteArgsWithId struct {
	id int
	*RequestVoteArgs
}

func requestVoteFromAll(allRequestVoteArgsWithId []requestVoteArgsWithId, term int, respond chan<- stateChangeReq, connected bool) {

	if !connected {
		fmt.Println("Not sending requestVoteFromAll since we are not connected")
		return
	}

	go func() {
		// This allows us to not to read from all responses from `indivialResponses` in case
		// we know already know the result or the server got shutdown
		indivialResponses := make(chan RequestVoteReply, len(allRequestVoteArgsWithId))

		var wg sync.WaitGroup

		go func() {
			wg.Wait()
			close(indivialResponses)
		}()

		for _, requestVoteArg := range allRequestVoteArgsWithId {
			wg.Add(1)
			go func(requestVoteArg requestVoteArgsWithId) {
				defer wg.Done()
				client, err := rpc.Dial("tcp", fmt.Sprintf(":%d", 9000+requestVoteArg.id))

				if err != nil {
					return
				}

				defer client.Close()

				var reply RequestVoteReply
				err = client.Call("Raft.RequestVote", requestVoteArg.RequestVoteArgs, &reply)

				if err != nil {
					fmt.Printf(err.Error())
					// Ignore this request
					return
				}

				indivialResponses <- reply

			}(requestVoteArg)
		}

		totalVotes := 1 // We have voted for ourveles already

		var response stateChangeReq

		for reply := range indivialResponses {

			if reply.VoteGranted {
				totalVotes++
			}

			if reply.Term > term {
				response = &convertToFollower{_term: term, newTerm: reply.Term}
				break
			}

			isWonELection := totalVotes > (len(allRequestVoteArgsWithId)+1)/2

			if isWonELection {
				response = &convertToLeader{_term: term}
				break
			}
		}

		if response == nil {
			return
		}

		timer := time.NewTimer(5 * time.Second)
		select {
		case respond <- response:
			return
		case <-timer.C:
			return
		}

	}()
}
