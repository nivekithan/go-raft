package raft

import "math/rand"

func randomTimeout(min int) int {
	return min*2 + rand.Intn(min)
}
