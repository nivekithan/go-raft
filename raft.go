package raft

import (
	"fmt"
	"math/rand"
	"net"
	"raft/rpc"
	"sync"

	"go.uber.org/zap"
	"google.golang.org/grpc"
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

type entry struct {
	term    int
	command string
}

type Raft struct {
	currentTerm               int
	votedFor                  int
	log                       []entry
	Tick                      func()
	config                    Config
	tickElapsed               int
	randomizedElectionTimeout int

	mu sync.Mutex
	l  *zap.Logger

	*rpc.UnimplementedRaftRpcServiceServer
}

func (r *Raft) Start() {
	r.l.Info("Starting raft node")
	r.startGrpcServer()

}

// Starts GrpcServer in address config.Address
func (r *Raft) startGrpcServer() {
	r.l.Info("Starting grpc Server")
	lis, err := net.Listen("tcp", r.config.Address)

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()

	rpc.RegisterRaftRpcServiceServer(grpcServer, r)
	go grpcServer.Serve(lis)
}

// Increase raft.ElectionTickElasped if the
func (r *Raft) electionTick() {
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

func NewRaft(config Config) *Raft {
	randomizedElectionTimeout := config.ElectionTick + rand.Intn(config.ElectionTick)

	logger := getDevelopmentLogger(config)

	node := &Raft{
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
