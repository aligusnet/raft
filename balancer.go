package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
)

type Balancer struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage
}

func newBalancer() *Balancer {
	return &Balancer{requestVoteCh: make(chan *requestVoteMessage),
		appendEntriesCh:  make(chan *appendEntriesMessage),
		executeCommandCh: make(chan *executeCommandMessage)}
}

func (c *Balancer) RequestVote(ctx context.Context, in *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resultCh := make(chan struct {
		*pb.RequestVoteResponse
		error
	}, 1)
	msg := &requestVoteMessage{ctx, in, resultCh}
	c.requestVoteCh <- msg
	result := <-resultCh
	return result.RequestVoteResponse, result.error
}

func (c *Balancer) AppendEntries(ctx context.Context, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resultCh := make(chan struct {
		*pb.AppendEntriesResponse
		error
	}, 1)
	msg := &appendEntriesMessage{ctx, in, resultCh}
	c.appendEntriesCh <- msg
	result := <-resultCh
	return result.AppendEntriesResponse, result.error
}

func (c *Balancer) ExecuteCommand(ctx context.Context, in *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	resultCh := make(chan struct {
		*pb.ExecuteCommandResponse
		error
	}, 1)
	msg := &executeCommandMessage{ctx, in, resultCh}
	c.executeCommandCh <- msg
	result := <-resultCh
	return result.ExecuteCommandResponse, result.error
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
