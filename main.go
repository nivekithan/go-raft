package main

import "github.com/nivekithan/go-raft/raft"

func main() {
	raftClusterIds := []int{1, 2, 3}

	for _, id := range raftClusterIds {
		peerIds := []int{}

		for _, peerId := range raftClusterIds {
			if peerId == id {
				continue
			}

			peerIds = append(peerIds, peerId)
		}

		raftServer := raft.NewServer(id, peerIds)

		raftServer.Start()
	}

	for {

	}
}
