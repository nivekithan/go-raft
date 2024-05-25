package raft

import "math/rand"

func randomTimeout(min int) int {
	return min*2 + rand.Intn(min)
}

func (r *Raft) dlog(msg string) {
	r.l.Debug(msg)
}

func (r *Raft) ilog(msg string) {
	r.l.Info(msg)
}
