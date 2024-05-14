package raft

import (
	"math/rand"
	"sync"
	"time"
)

type electionTimerCommand string

const (
	resetElectionTimer electionTimerCommand = "reset"
	stopElectionTimer  electionTimerCommand = "stop"
)

type electionTimer struct {
	mu                 sync.Mutex
	electionTimeoutMs  int
	electionResetEvent time.Time
	control            <-chan electionTimerCommand
	response           chan<- interface{}
	isStopped          bool
}

func newElectionTimer(electionTimeoutMs int, control <-chan electionTimerCommand, response chan<- interface{}) *electionTimer {
	pseudoRandomElectionTimeout := rand.Intn(electionTimeoutMs) + electionTimeoutMs

	return &electionTimer{
		electionTimeoutMs: pseudoRandomElectionTimeout,
		control:           control,
		response:          response,
		isStopped:         false,
	}
}

func (e *electionTimer) start() {
	e.mu.Lock()
	e.electionResetEvent = time.Now().Add(time.Duration(e.electionTimeoutMs) * time.Millisecond)

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)

		for {
			<-ticker.C

			e.mu.Lock()
			if e.isStopped {
				return
			}

			if time.Now().After(e.electionResetEvent) {
				e.response <- ""
			}

			e.mu.Unlock()
		}
	}()

	e.mu.Unlock()

	for {
		command := <-e.control

		switch command {
		case resetElectionTimer:
			e.mu.Lock()
			e.electionResetEvent = time.Now().Add(time.Duration(e.electionTimeoutMs) * time.Millisecond)
			e.mu.Unlock()
		case stopElectionTimer:
			e.mu.Lock()
			e.isStopped = true
			close(e.response)
			e.mu.Unlock()
			return
		}
	}
}
