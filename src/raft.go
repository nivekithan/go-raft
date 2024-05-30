package raft

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"
)

type RaftConfig struct {
	Id               int
	Peers            []int
	ElectionTimeout  int
	HeartbeatTimeout int

	// Clients sends commands through this channel
	ClientCommandsChan <-chan []byte
}

type logEntry struct {
	command []byte
	term    int
}

type Raft struct {

	// Raft config variables
	id               int
	peers            []int
	electionTimeout  int
	heartbeatTimeout int
	state            RaftState

	// Raft Perist variables
	currentTerm int
	votedFor    int
	log         []logEntry

	// Volatile state on all server
	commitIndex int
	lastApplied int

	// Volatile state on leaders
	nextIndex  map[int]int
	matchIndex map[int]int

	// Internal Variables
	clientCommandsChan <-chan []byte

	rpcServerListern net.Listener
	stopServerWg     sync.WaitGroup

	electionTimer       *timer
	electionTimeoutChan chan int

	heartbeatTimer       *timer
	heartbeatTimeoutChan chan int

	stateChangeChan chan stateChangeReq

	requestVoteRpcArgsChan  chan RequestVoteArgs
	requestVoteRpcReplyChan chan RequestVoteReply

	appendEntriesRpcArgsChan  chan AppendEntiesArgs
	appendEntriesRpcReplyChan chan AppendEntriesReply

	done chan interface{}

	// Logging
	l *slog.Logger

	// Variables used for testing
	connected bool
}

func NewRaft(config RaftConfig) *Raft {

	raft := &Raft{
		id:                        config.Id,
		peers:                     config.Peers,
		electionTimeout:           config.ElectionTimeout,
		heartbeatTimeout:          config.HeartbeatTimeout,
		votedFor:                  NullId,
		state:                     Follower,
		currentTerm:               0,
		log:                       []logEntry{},
		commitIndex:               0,
		lastApplied:               0,
		nextIndex:                 make(map[int]int),
		matchIndex:                make(map[int]int),
		clientCommandsChan:        config.ClientCommandsChan,
		electionTimeoutChan:       make(chan int),
		heartbeatTimeoutChan:      make(chan int),
		stateChangeChan:           make(chan stateChangeReq),
		requestVoteRpcArgsChan:    make(chan RequestVoteArgs),
		requestVoteRpcReplyChan:   make(chan RequestVoteReply),
		appendEntriesRpcArgsChan:  make(chan AppendEntiesArgs),
		appendEntriesRpcReplyChan: make(chan AppendEntriesReply),
		done:                      make(chan interface{}),
		l: slog.New(
			slog.NewTextHandler(
				os.Stdout,
				&slog.HandlerOptions{Level: slog.LevelInfo},
			),
		).With("id", config.Id),

		connected: true,
	}

	return raft
}

func (r *Raft) Start() {
	r.setNewElectionTimer()
	go r.startRpcServer()
	r.mainLoop()
}

func (r *Raft) State() RaftState {
	return r.state
}

func (r *Raft) Id() int {
	return r.id
}

func (r *Raft) Term() int {
	return r.currentTerm
}

func (r *Raft) mainLoop() {
	for {
		switch r.state {
		case Follower:
			r.ilog("Running Follower Loop")
			r.startFollowerLoop()
		case Candidate:
			r.ilog("Running Canidate Loop")
			r.startCandidateLoop()
		case Leader:
			r.ilog("Running Leader Loop")
			r.startLeaderLoop()
		case Dead:
			r.ilog("Stopping Main Loop")
			return
		}
	}
}

func (r *Raft) startFollowerLoop() {
	for r.state == Follower {
		select {
		case term := <-r.electionTimeoutChan:

			if term != r.currentTerm {
				// Ignore this timeout
				continue
			}

			r.startElection()

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		case <-r.done:
			r.stopNode()
		}
	}
}

func (r *Raft) startCandidateLoop() {
	for r.state == Candidate {
		select {
		case term := <-r.electionTimeoutChan:
			if term != r.currentTerm {
				// Ignore this command
				continue
			}
			r.startElection()

		case changeState := <-r.stateChangeChan:
			if changeState.term() != r.currentTerm {
				continue
			}

			changeState.updateState(r)

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		case <-r.done:
			r.stopNode()
		}

	}

}

func (r *Raft) startLeaderLoop() {

	for r.state == Leader {
		select {
		case term := <-r.heartbeatTimeoutChan:
			if term != r.currentTerm {
				// Ignore this message
				continue
			}

			r.sendHearbeats()

		case changeState := <-r.stateChangeChan:
			r.ilog(fmt.Sprintf("changeState: %v", changeState))

			if changeState.term() != r.currentTerm {
				// Ignore this request
				continue
			}

			changeState.updateState(r)

		case args := <-r.requestVoteRpcArgsChan:
			r.respondToRequestVoteRpc(args)

		case args := <-r.appendEntriesRpcArgsChan:
			r.respondToAppendEntriesRpc(args)
		case command := <-r.clientCommandsChan:
			r.processLogEntry(logEntry{command: command, term: r.currentTerm})

		case <-r.done:
			r.stopNode()
		}

	}

}

func (r *Raft) respondToRequestVoteRpc(args RequestVoteArgs) {

	if r.currentTerm > args.Term {
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: false}
	}

	if args.Term > r.currentTerm {
		r.convertToFollower(args.Term)
	}

	if r.votedFor == NullId || r.votedFor == args.CandidateId {
		r.votedFor = args.CandidateId
		r.setNewElectionTimer()
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: true}
	} else {
		r.requestVoteRpcReplyChan <- RequestVoteReply{Term: r.currentTerm, VoteGranted: false}
	}
}

func (r *Raft) respondToAppendEntriesRpc(args AppendEntiesArgs) {
	if r.currentTerm > args.Term {
		r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: false}
		return
	}

	if args.Term > r.currentTerm {
		r.convertToFollower(args.Term)
	}

	isEntriesEmpty := len(args.Entries) == 0

	if isEntriesEmpty {
		r.setNewElectionTimer()
		r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: true}
		return
	}

	if args.PrevLogIndex != -1 {
		// There are entries to be checked

		if args.PrevLogIndex >= len(r.log) {
			r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: false}
			return
		}

		if args.PrevLogTerm != r.log[args.PrevLogIndex].term {
			r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: false}
			return
		}

	}

	for index, entry := range args.Entries {
		r.log[index] = entry
	}

	if args.LeaderCommit > r.commitIndex {

		r.commitIndex = func() int {
			indexOfLastEntry := len(args.Entries) - 1

			if args.LeaderCommit > indexOfLastEntry {
				return indexOfLastEntry
			} else {
				return args.LeaderCommit
			}
		}()
	}

	r.setNewElectionTimer()
	r.appendEntriesRpcReplyChan <- AppendEntriesReply{Term: r.currentTerm, Success: true}

}

func (r *Raft) processLogEntry(logEntry logEntry) {
	r.log = append(r.log, logEntry)
}

func (r *Raft) startElection() {
	if r.state != Candidate {
		r.ilog("Converting to candidate")
	}
	r.currentTerm++
	r.votedFor = r.id
	r.state = Candidate

	r.dlog("Sending requestVoteFromAll")
	requestVoteFromAll(r.peers, r.currentTerm, r.id, r.stateChangeChan, r.connected)
	r.setNewElectionTimer()
}

func (r *Raft) convertToFollower(newTerm int) {
	if r.state != Follower {
		r.ilog("Converting to Follower")
	}
	r.currentTerm = newTerm
	r.votedFor = -1
	r.state = Follower
	r.setNewElectionTimer()

	if r.heartbeatTimer != nil {
		CancelTimer(r.heartbeatTimer)
	}

}

func (r *Raft) convertToLeader() {
	r.state = Leader

	// Reset nextIndex and matchIndex whenever we become leader
	for _, id := range r.peers {
		r.nextIndex[id] = len(r.log)
		r.matchIndex[id] = 0
	}

	r.ilog("Converting to Leader")
	r.sendHearbeats()
}

func (r *Raft) sendHearbeats() {
	appendEntiresConfig := r.getAppendEntiresConfig()

	sendAppendEntiresToAll(appendEntiresConfig, r.currentTerm, r.stateChangeChan, r.connected)
	r.setNewHeartbeatTimer()

}

func (r *Raft) getAppendEntiresConfig() *[]appendEntriesArgsWithId {
	appendEntiresConfig := []appendEntriesArgsWithId{}

	for _, peerId := range r.peers {
		logIndexToSend := r.nextIndex[peerId]
		prevLogIndex := logIndexToSend - 1

		prevLogTerm := func() int {
			if prevLogIndex < 0 {
				return 0
			}
			return r.log[prevLogIndex].term
		}()

		entries := []logEntry{}

		if len(r.log) > logIndexToSend {
			// We can also send multiple entires at once as a optimization
			// but first lets make sure sending only one entry at a time works
			entries = append(entries, r.log[logIndexToSend])
		}

		appendEntiresConfig = append(
			appendEntiresConfig,
			appendEntriesArgsWithId{
				id: peerId,
				AppendEntiesArgs: &AppendEntiesArgs{
					Term:         r.currentTerm,
					LeaderId:     r.id,
					PrevLogIndex: prevLogIndex,
					PrevLogTerm:  prevLogTerm,
					Entries:      entries,
					LeaderCommit: r.commitIndex,
				},
			},
		)
	}

	return &appendEntiresConfig
}

func (r *Raft) setNewElectionTimer() {
	r.dlog("Reseting election timer")
	newElectionTimeout := randomTimeout(r.electionTimeout)

	if r.electionTimer != nil {
		CancelTimer(r.electionTimer)
	}

	r.electionTimer = NewAndStartTimer(
		time.Duration(newElectionTimeout)*time.Millisecond,
		r.currentTerm,
		r.electionTimeoutChan,
	)
}

func (r *Raft) setNewHeartbeatTimer() {
	if r.heartbeatTimer != nil {
		CancelTimer(r.heartbeatTimer)
	}

	r.heartbeatTimer = NewAndStartTimer(
		time.Duration(r.heartbeatTimeout)*time.Millisecond,
		r.currentTerm,
		r.heartbeatTimeoutChan,
	)
}

func (r *Raft) stopNode() {
	if r.heartbeatTimer != nil {
		CancelTimer(r.heartbeatTimer)
	}

	if r.electionTimer != nil {
		CancelTimer(r.electionTimer)
	}

	r.ilog("Stopped all timers")
	r.stopRpcServer()
	r.ilog("Stopped RPC Server")

	close(r.requestVoteRpcReplyChan)
	close(r.appendEntriesRpcReplyChan)

	r.ilog("Closed all rpc reply channels")
	r.state = Dead
	r.ilog("Updated state to Dead")
}

func (r *Raft) Stop() {
	r.done <- true
	r.stopServerWg.Wait()
}
