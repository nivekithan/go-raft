package raft

import (
	"log/slog"
	"math/rand"
	"sync"
	"time"
)

type timerCommand string

const (
	resetTimer timerCommand = "reset"
	stopTimer  timerCommand = "stop"
)

type timer struct {
	mu                sync.Mutex
	timeoutMs         int
	timeoutResetEvent time.Time
	control           chan timerCommand
	response          chan<- interface{}
	isStopped         bool
	l                 *slog.Logger
}

func newTimer(timeoutMs int, response chan<- interface{}, l *slog.Logger) *timer {
	pseudoRandomElectionTimeout := rand.Intn(timeoutMs) + timeoutMs

	return &timer{
		timeoutMs: pseudoRandomElectionTimeout,
		control:   make(chan timerCommand),
		response:  response,
		isStopped: false,
		l:         l,
	}
}

func (e *timer) reset() {
	e.control <- resetTimer
}

func (e *timer) stop() {
	e.control <- stopTimer

}

func (e *timer) start() {
	e.mu.Lock()
	e.timeoutResetEvent = time.Now().Add(time.Duration(e.timeoutMs) * time.Millisecond)

	go func() {
		ticker := time.NewTicker(10 * time.Millisecond)

		for {
			<-ticker.C

			e.mu.Lock()
			if e.isStopped {
				return
			}

			if time.Now().After(e.timeoutResetEvent) {
				e.response <- ""
			}

			e.mu.Unlock()
		}
	}()

	e.mu.Unlock()

	for {
		command := <-e.control

		e.l.Debug("Received command", "command", command)

		switch command {
		case resetTimer:
			e.mu.Lock()
			e.timeoutResetEvent = time.Now().Add(time.Duration(e.timeoutMs) * time.Millisecond)
			e.mu.Unlock()
		case stopTimer:
			e.mu.Lock()
			e.isStopped = true

			if e.response != nil {
				close(e.response)
				e.response = nil
			}
			e.mu.Unlock()
		default:
			panic("Unknown command")
		}
	}
}
