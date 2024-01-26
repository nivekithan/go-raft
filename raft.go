package raft

import (
	"fmt"
	"math/rand"
	"sync"

	"go.uber.org/zap"
)

type Config struct {
	// Id of the local raft node. It cannot be 0
	Id int

	// ElectionTick is the number raft.Tick invocations that must pass between
	// elections
	ElectionTick int

	// HeartbeatTick is the number of Node.Tick invocations that must pass between
	// heartbeats
	HeartbeatTick int

	// Grpc server address
	Address string
}

type entry = string

type raft struct {
	currentTerm               int
	votedFor                  int
	log                       []entry
	Tick                      func()
	config                    Config
	tickElapsed               int
	randomizedElectionTimeout int

	mu sync.Mutex
	l  *zap.Logger
}

func (r *raft) Start() {
	r.l.Info("Starting raft node")
}

// Increase raft.ElectionTickElasped if the
func (r *raft) electionTick() {
	r.mu.Lock()
	defer r.mu.Unlock()

	logger := r.l.With(zap.String("group", "election"))
	r.tickElapsed++

	logger.Debug("Increasing electionTick")

	if r.tickElapsed > r.randomizedElectionTimeout {
		logger.Info("Election timemout passed. Starting Election")
		// TODO: Start election

		logger.Info("Reseting electionTick")
		r.tickElapsed = 0
	}

}

func getDevelopmentLogger(config Config) *zap.Logger {
	loggerDevelopmentConfig := zap.NewDevelopmentConfig()
	loggerDevelopmentConfig.Encoding = "json"
	loggerDevelopmentConfig.OutputPaths = append(loggerDevelopmentConfig.OutputPaths, fmt.Sprintf("./local_logs/%d.txt", config.Id))

	logger, err := loggerDevelopmentConfig.Build()
	if err != nil {
		panic(err)
	}

	logger = logger.With(zap.Int("id", config.Id))

	return logger
}

func NewRaft(config Config) *raft {
	randomizedElectionTimeout := config.ElectionTick + rand.Intn(config.ElectionTick)

	logger := getDevelopmentLogger(config)

	node := &raft{
		currentTerm:               0,
		votedFor:                  0,
		log:                       []entry{},
		config:                    config,
		tickElapsed:               0,
		randomizedElectionTimeout: randomizedElectionTimeout,
		l:                         logger,
	}

	node.Tick = node.electionTick

	return node
}
