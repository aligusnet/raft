package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
)

type channelSet struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage
}

func newChannelSet() *channelSet {
	return &channelSet{requestVoteCh: make(chan *requestVoteMessage),
		appendEntriesCh:  make(chan *appendEntriesMessage),
		executeCommandCh: make(chan *executeCommandMessage)}
}

type requestVoteMessage struct {
	ctx context.Context
	in  *pb.RequestVoteRequest
	out chan struct {
		*pb.RequestVoteResponse
		error
	}
}

type appendEntriesMessage struct {
	ctx context.Context
	in  *pb.AppendEntriesRequest
	out chan struct {
		*pb.AppendEntriesResponse
		error
	}
}

type executeCommandMessage struct {
	ctx context.Context
	in  *pb.ExecuteCommandRequest
	out chan struct {
		*pb.ExecuteCommandResponse
		error
	}
}
