package raft

import (
	"fmt"
	"testing"
	"time"
)

func TestElectionBasic(t *testing.T) {
	// t.Skip()
	h := newTestHarness(t, 3)

	t.Cleanup(func() { h.stop() })
	h.start()

	h.CheckSingleLeader()

}

func TestElectionLeaderDisconnect(t *testing.T) {
	// t.Skip()

	h := newTestHarness(t, 3)

	t.Cleanup(func() { h.stop() })

	h.start()

	origLeaderId, origTerm := h.CheckSingleLeader()

	h.disconnect(origLeaderId)
	time.Sleep(1 * time.Second)

	newLeaderId, newTerm := h.CheckSingleLeader()

	if origLeaderId == newLeaderId {
		t.Errorf("NewLeaderId %d must be different from orignalLeaderId %d", newLeaderId, origLeaderId)
	}

	if newTerm <= origTerm {
		t.Errorf("NewTerm %d must be greater than originalTerm %d", newTerm, origTerm)
	}

}

func TestElectionLeaderAndAnotherDisconnect(t *testing.T) {
	// t.Skip()

	h := newTestHarness(t, 3)
	h.start()

	t.Cleanup(func() { h.stop() })

	origLeaderId, _ := h.CheckSingleLeader()

	h.disconnect(origLeaderId)
	otherId := func() int {
		nextId := origLeaderId + 1

		if nextId > 3 {
			return 1
		}

		return nextId
	}()

	fmt.Printf("Disconnecting node %d \n", otherId)
	h.disconnect(otherId)

	// No quorum.
	time.Sleep(450 * time.Millisecond)
	h.checkNoLeader()

	// Reconnect one other server; now we'll have quorum.
	h.connect(otherId)
	h.CheckSingleLeader()
}

func TestDisconnectAllThenRestore(t *testing.T) {
	// t.Skip()

	h := newTestHarness(t, 3)

	h.start()

	t.Cleanup(func() { h.stop() })

	time.Sleep(100 * time.Millisecond)
	//	Disconnect all servers from the start. There will be no leader.
	for i := 0; i < 3; i++ {
		h.disconnect(i)
	}

	time.Sleep(450 * time.Millisecond)

	h.checkNoLeader()

	// Reconnect all servers. A leader will be found.
	for i := 0; i < 3; i++ {
		h.connect(i)
	}

	h.CheckSingleLeader()
}

func TestElectionLeaderDisconnectThenReconnect(t *testing.T) {
	// t.Skip()
	h := newTestHarness(t, 3)
	h.start()

	t.Cleanup(func() { h.stop() })

	origLeaderId, _ := h.CheckSingleLeader()

	h.disconnect(origLeaderId)

	time.Sleep(350 * time.Millisecond)

	newLeaderId, newTerm := h.CheckSingleLeader()

	fmt.Printf("Is newLeader and oldLeader same :%v\n", origLeaderId == newLeaderId)

	h.connect(origLeaderId)
	time.Sleep(150 * time.Millisecond)

	againLeaderId, againTerm := h.CheckSingleLeader()

	if newLeaderId != againLeaderId {
		t.Errorf("again leader id got %d; want %d", againLeaderId, newLeaderId)
	}
	if againTerm != newTerm {
		t.Errorf("again term got %d; want %d", againTerm, newTerm)
	}
}
