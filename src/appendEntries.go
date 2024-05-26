package raft

import (
	"fmt"
	"net/rpc"
	"sync"
	"time"
)

type AppendEntiesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func sendAppendEntiresToAll(peers []int, term int, id int, respond chan<- stateChangeReq, connected bool) {

	go func() {

		if !connected {
			return
		}

		// This allows us to not to read from all responses from `indivialResponses` in case
		// we know already know the result.
		indivialResponses := make(chan AppendEntriesReply, len(peers))

		var wg sync.WaitGroup

		go func() {
			wg.Wait()
			close(indivialResponses)
		}()

		for _, peerId := range peers {
			wg.Add(1)
			go func(peerId int) {
				client, err := rpc.Dial("tcp", fmt.Sprintf(":%d", 9000+peerId))

				if err != nil {
					wg.Done()
					return
				}

				defer client.Close()
				var reply AppendEntriesReply
				err = client.Call("Raft.AppendEntries", AppendEntiesArgs{Term: term, LeaderId: id}, &reply)

				if err != nil {
					wg.Done()
					// Ignore this request
					return
				}

				indivialResponses <- reply
				wg.Done()
			}(peerId)
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
