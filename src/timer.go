package raft

import "time"

type timer struct {
	timeoutOn time.Time
	term      int
	control   chan controlTimerCommand
	respond   chan<- int
	finished  bool
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
		finished:  false,
	}

	go timer.runAfterStartCommand()

	return timer
}

func StartTimer(timer *timer) {
	if !timer.finished {
		timer.control <- startTimerCommand
	}
}

func CancelTimer(timer *timer) {
	if !timer.finished {
		timer.control <- cancelTimerCommand
	}
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
	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:
			isTimeoutPassed := time.Now().After(t.timeoutOn)
			if isTimeoutPassed {
				t.finished = true

				timer := time.NewTimer(2 * time.Second)
				select {
				case t.respond <- t.term:
					return
				case <-timer.C:
					return
				}
			}

		case command := <-t.control:

			if command != cancelTimerCommand {
				continue
			}
			t.finished = true
			return
		}
	}
}
