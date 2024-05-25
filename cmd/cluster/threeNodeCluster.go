package main

import (
	"fmt"
	"time"

	raft "github.com/nivekithan/go-raft/src"
)

func main() {

	node1 := raft.NewRaft(raft.RaftConfig{Id: 1, Peers: []int{2, 3}, ElectionTimeout: 5 * 100, HeartbeatTimeout: 50})
	node2 := raft.NewRaft(raft.RaftConfig{Id: 2, Peers: []int{1, 3}, ElectionTimeout: 5 * 100, HeartbeatTimeout: 50})
	node3 := raft.NewRaft(raft.RaftConfig{Id: 3, Peers: []int{2, 1}, ElectionTimeout: 5 * 100, HeartbeatTimeout: 50})

	go node1.Start()
	go node2.Start()
	go node3.Start()

	cluster := []*raft.Raft{node1, node2, node3}

	ticker := time.NewTicker(time.Duration(3*100) * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C

		for _, node := range cluster {
			if node.State() == raft.Leader {
				fmt.Printf("Node: %d become leader\n", node.Id())
				return
			}
		}

	}

}
