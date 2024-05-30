package raft

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
)

type AppendEntiesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []logEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

type appendEntriesArgsWithId struct {
	id int
	*AppendEntiesArgs
}

type appendEntriesReplyWithId struct {
	id int
	*AppendEntriesReply
}

func sendAppendEntiresToAll(appendEntriesArgs *[]appendEntriesArgsWithId, term int, respond chan<- stateChangeReq, connected bool) {

	go func() {

		if !connected {
			return
		}

		// This allows us to not to read from all responses from `indivialResponses` in case
		// we know already know the result.
		indivialResponses := make(chan appendEntriesReplyWithId, len(*appendEntriesArgs))

		var wg sync.WaitGroup

		go func() {
			wg.Wait()
			close(indivialResponses)
		}()

		for _, appendEntriesArg := range *appendEntriesArgs {
			wg.Add(1)
			go func(appendEntriesArg appendEntriesArgsWithId) {
				client, err := rpc.Dial("tcp", fmt.Sprintf(":%d", 9000+appendEntriesArg.id))

				if err != nil {
					wg.Done()
					return
				}

				defer client.Close()
				var reply AppendEntriesReply
				err = client.Call("Raft.AppendEntries", appendEntriesArg, &reply)

				if err != nil {
					wg.Done()
					// Ignore this request
					return
				}

				indivialResponses <- appendEntriesReplyWithId{id: appendEntriesArg.id, AppendEntriesReply: &reply}
				wg.Done()
			}(appendEntriesArg)
		}

		for reply := range indivialResponses {

			if reply.Term > term {
				timer := time.NewTimer(5 * time.Second)
				response := &convertToFollower{_term: term, newTerm: reply.Term}
				select {
				case respond <- response:
					return
				case <-timer.C:
					return
				}
			}

			if !reply.Success {
				// Decrease nextIndex for this node
				timer := time.NewTimer(5 * time.Second)
				response := &decreaseNextIndex{_term: term, id: reply.id}

				select {
				case respond <- response:
					continue
				case <-timer.C:
					continue
				}
			}
		}

	}()
}
