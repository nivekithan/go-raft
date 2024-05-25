package raft

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// These methods are used for testing raft implemenation
func (r *Raft) disconnect() {
	r.connected = false

}

func (r *Raft) connect() {
	r.connected = true

}

type raftHarness struct {
	clusters  []*Raft
	t         *testing.T
	connected map[int]bool
}

func newTestHarness(t *testing.T, noOfNodes int) *raftHarness {

	nodes := []*Raft{}
	connected := map[int]bool{}

	for i := 1; i <= noOfNodes; i++ {
		peers := []int{}

		for j := 1; j <= noOfNodes; j++ {
			if j == i {
				continue
			}

			peers = append(peers, j)
		}

		raftNode := NewRaft(RaftConfig{Id: i, Peers: peers, ElectionTimeout: 150, HeartbeatTimeout: 50})
		nodes = append(nodes, raftNode)
		connected[i] = true
	}

	harness := raftHarness{clusters: nodes, t: t, connected: connected}

	return &harness
}

func (h *raftHarness) start() {
	for _, node := range h.clusters {
		name := h.t.Name()

		fmt.Printf("Starting nodeId: %d on test: %v\n", node.Id(), name)
		go node.Start()
	}
}

func (h *raftHarness) stop() {
	var wg sync.WaitGroup
	for _, node := range h.clusters {
		wg.Add(1)

		go func(node *Raft) {
			node.Stop()
			wg.Done()
		}(node)
	}
	wg.Wait()
}

func (h *raftHarness) disconnect(id int) {

	for _, node := range h.clusters {
		if node.Id() == id {
			h.connected[id] = false
			node.disconnect()
		}
	}
}

func (h *raftHarness) connect(id int) {
	for _, node := range h.clusters {
		if node.Id() == id {
			h.connected[id] = true
			node.connect()
		}
	}
}

func (h *raftHarness) CheckSingleLeader() (leaderId, term int) {
	ticker := time.NewTicker(time.Duration(200) * time.Millisecond)
	timer := time.NewTimer(time.Duration(5) * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:
			noOfLeaders := 0
			leaderTerm := -1
			leaderId := NullId
			for _, node := range h.clusters {
				if node.State() == Leader && h.connected[node.Id()] {
					noOfLeaders++
					leaderTerm = node.Term()
					leaderId = node.Id()
				}
			}

			if noOfLeaders == 0 {
				// No leader has been choosen yet, so we can continue
				continue
			}

			if noOfLeaders > 1 {
				// There are mutliple leaders which should not happen. Since timers
				// and io works properly
				continue
			}

			return leaderId, leaderTerm

		case <-timer.C:

			h.t.Errorf("Cluster was unable to elect an leader within given time")

		}
	}

}

func (h *raftHarness) checkNoLeader() {

	for _, node := range h.clusters {
		if node.State() == Leader && h.connected[node.Id()] {
			h.t.Errorf("Found connected leader with id: %d", node.Id())
		}
	}

}
