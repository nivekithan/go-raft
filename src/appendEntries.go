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

type sendAppendEntriesConfig struct {
	id int
	*AppendEntiesArgs
}

func sendAppendEntiresToAll(appendEntriesArgs *[]sendAppendEntriesConfig, term int, respond chan<- stateChangeReq, connected bool) {

	go func() {

		if !connected {
			return
		}

		// This allows us to not to read from all responses from `indivialResponses` in case
		// we know already know the result.
		indivialResponses := make(chan AppendEntriesReply, len(*appendEntriesArgs))

		var wg sync.WaitGroup

		go func() {
			wg.Wait()
			close(indivialResponses)
		}()

		for _, appendEntriesArg := range *appendEntriesArgs {
			wg.Add(1)
			go func(appendEntriesArg sendAppendEntriesConfig) {
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

				indivialResponses <- reply
				wg.Done()
			}(appendEntriesArg)
		}

		var response stateChangeReq

		for reply := range indivialResponses {

			if reply.Term > term {
				response = stateChangeReq{term: term, newTerm: reply.Term, command: convertToFollower}
				break
			}
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
