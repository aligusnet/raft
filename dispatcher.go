package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
)

type Dispatcher struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage
}

func newDispatcher() *Dispatcher {
	return &Dispatcher{requestVoteCh: make(chan *requestVoteMessage),
		appendEntriesCh:  make(chan *appendEntriesMessage),
		executeCommandCh: make(chan *executeCommandMessage)}
}

func (d *Dispatcher) RequestVote(ctx context.Context, in *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resultCh := make(chan struct {
		*pb.RequestVoteResponse
		error
	}, 1)
	msg := &requestVoteMessage{ctx, in, resultCh}
	d.requestVoteCh <- msg
	result := <-resultCh
	return result.RequestVoteResponse, result.error
}

func (d *Dispatcher) AppendEntries(ctx context.Context, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resultCh := make(chan struct {
		*pb.AppendEntriesResponse
		error
	}, 1)
	msg := &appendEntriesMessage{ctx, in, resultCh}
	d.appendEntriesCh <- msg
	result := <-resultCh
	return result.AppendEntriesResponse, result.error
}

func (d *Dispatcher) ExecuteCommand(ctx context.Context, in *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	resultCh := make(chan struct {
		*pb.ExecuteCommandResponse
		error
	}, 1)
	msg := &executeCommandMessage{ctx, in, resultCh}
	d.executeCommandCh <- msg
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

func (msg *requestVoteMessage) send(response *pb.RequestVoteResponse) {
	msg.out <- struct {
		*pb.RequestVoteResponse
		error
	}{response, nil}
}

func (msg *requestVoteMessage) sendError(err error) {
	msg.out <- struct {
		*pb.RequestVoteResponse
		error
	}{nil, err}
}

type appendEntriesMessage struct {
	ctx context.Context
	in  *pb.AppendEntriesRequest
	out chan struct {
		*pb.AppendEntriesResponse
		error
	}
}

func (msg *appendEntriesMessage) send(response *pb.AppendEntriesResponse) {
	msg.out <- struct {
		*pb.AppendEntriesResponse
		error
	}{response, nil}
}

func (msg *appendEntriesMessage) sendError(err error) {
	msg.out <- struct {
		*pb.AppendEntriesResponse
		error
	}{nil, err}
}

type executeCommandMessage struct {
	ctx context.Context
	in  *pb.ExecuteCommandRequest
	out chan struct {
		*pb.ExecuteCommandResponse
		error
	}
}

func (msg *executeCommandMessage) send(response *pb.ExecuteCommandResponse) {
	msg.out <- struct {
		*pb.ExecuteCommandResponse
		error
	}{response, nil}
}

func (msg *executeCommandMessage) sendError(err error) {
	msg.out <- struct {
		*pb.ExecuteCommandResponse
		error
	}{nil, err}
}
