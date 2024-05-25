package raft

import (
	"fmt"
	"net/rpc"
	"sync"
)

type AppendEntiesArgs struct {
	Term     int
	LeaderId int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func sendAppendEntiresToAll(peers []int, term int, id int, respond chan<- stateChangeReq) {

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
			defer wg.Done()
			client, err := rpc.Dial("tcp", fmt.Sprintf(":%d", 9000+peerId))
			defer client.Close()

			if err != nil {
				return
			}

			var reply AppendEntriesReply
			err = client.Call("Raft.AppendEntries", AppendEntiesArgs{Term: term, LeaderId: id}, &reply)

			if err != nil {
				// Ignore this request
				return
			}

			indivialResponses <- reply
		}(peerId)
	}

	for reply := range indivialResponses {

		if reply.Term > term {
			respond <- stateChangeReq{term: term, newTerm: reply.Term, command: convertToFollower}
			return
		}

	}

}
