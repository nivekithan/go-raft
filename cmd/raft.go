package main

import (
	"raft"
	"time"
)

func main() {
	raftServer := raft.NewRaft(raft.Config{
		Id:            1,
		ElectionTick:  10,
		HeartbeatTick: 1,
	})

	raftServer.Start()
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-ticker.C:
			raftServer.Tick()
		}
	}
}
