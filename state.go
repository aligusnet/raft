package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"time"
)

type State struct {
	currentTerm int64
	votedFor    int64
	commitIndex int64
	lastApplied int64
	timeout     time.Duration
}

func requestVoteRequest(s *State) *pb.RequestVoteRequest {
	return &pb.RequestVoteRequest{
		CandidateId:  0,
		Term:         0,
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

}
