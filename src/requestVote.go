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

func requestVoteFromAll(peers []int, term int, id int, respond chan<- stateChangeReq, connected bool) {

	if !connected {
		fmt.Println("Not sending requestVoteFromAll since we are not connected")
		return
	}

	go func() {
		// This allows us to not to read from all responses from `indivialResponses` in case
		// we know already know the result.
		indivialResponses := make(chan RequestVoteReply, len(peers))

		var wg sync.WaitGroup

		go func() {
			wg.Wait()
			close(indivialResponses)
		}()

		for _, peerId := range peers {
			wg.Add(1)
			go func(peerId int) {
				defer wg.Done()
				client, err := rpc.Dial("tcp", fmt.Sprintf(":%d", 9000+peerId))

				defer client.Close()

				if err != nil {

					return
				}

				var reply RequestVoteReply
				err = client.Call("Raft.RequestVote", RequestVoteArgs{Term: term, CandidateId: id}, &reply)

				if err != nil {
					// Ignore this request
					return
				}

				indivialResponses <- reply

			}(peerId)
		}

		totalVotes := 1 // We have voted for ourveles already
		responsesRecived := 0

		for reply := range indivialResponses {
			responsesRecived++

			if reply.VoteGranted {
				totalVotes++
			}

			if reply.Term > term {
				respond <- stateChangeReq{term: term, command: convertToFollower, newTerm: reply.Term}
				return
			}

			isWonELection := totalVotes > (len(peers)+1)/2

			if isWonELection {
				respond <- stateChangeReq{term: term, command: convertToLeader}
				return
			}
		}

	}()
}
