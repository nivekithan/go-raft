package raft

import "time"

type timer struct {
	timeoutOn time.Time
	term      int
	control   chan controlTimerCommand
	respond   chan<- int
}

type controlTimerCommand int

const (
	startTimerCommand controlTimerCommand = iota
	cancelTimerCommand
)

func NewTimer(duration time.Duration, term int, res chan<- int) *timer {
	timer := &timer{
		timeoutOn: time.Now().Add(duration),
		term:      term,
		control:   make(chan controlTimerCommand),
		respond:   res,
	}

	return timer
}

func StartTimer(timer *timer) {
	timer.control <- startTimerCommand
}

func CancelTimer(timer *timer) {
	timer.control <- cancelTimerCommand
}

func NewAndStartTimer(duration time.Duration, term int, res chan<- int) *timer {
	timer := NewTimer(duration, term, res)

	StartTimer(timer)

	return timer
}

func (t *timer) runAfterStartCommand() {
	for command := range t.control {
		if command != startTimerCommand {
			continue
		}

		break
	}

	ticker := time.NewTicker(10 * time.Millisecond)

	for {
		select {

		case <-ticker.C:
			isTimeoutPassed := time.Now().After(t.timeoutOn)
			if isTimeoutPassed {
				t.respond <- t.term
				return
			}

		case command := <-t.control:

			if command != cancelTimerCommand {
				continue
			}

			return
		}
	}
}
