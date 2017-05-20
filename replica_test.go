package raft

import (
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

type clientMock struct {
	requestVote    []*pb.RequestVoteResponse
	appendEntries  []*pb.AppendEntriesResponse
	executeCommand []*pb.ExecuteCommandResponse

	requestVoteIndex    int
	appendEntriesIndex  int
	executeCommandIndex int
}

func newRequestVoteClient(votes ...bool) *clientMock {
	var requestVote []*pb.RequestVoteResponse
	for _, vote := range votes {
		requestVote = append(requestVote, &pb.RequestVoteResponse{VoteGranted: vote})
	}
	return &clientMock{requestVote: requestVote}
}

func (c *clientMock) RequestVote(ctx context.Context,
	in *pb.RequestVoteRequest,
	opts ...grpc.CallOption) (*pb.RequestVoteResponse, error) {
	if len(c.requestVote) > c.requestVoteIndex {
		response := c.requestVote[c.requestVoteIndex]
		c.requestVoteIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}

func (c *clientMock) AppendEntries(ctx context.Context,
	in *pb.AppendEntriesRequest,
	opts ...grpc.CallOption) (*pb.AppendEntriesResponse, error) {
	if len(c.appendEntries) > c.appendEntriesIndex {
		response := c.appendEntries[c.appendEntriesIndex]
		c.appendEntriesIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}

func (c *clientMock) ExecuteCommand(ctx context.Context,
	in *pb.ExecuteCommandRequest,
	opts ...grpc.CallOption) (*pb.ExecuteCommandResponse, error) {
	if len(c.executeCommand) > c.executeCommandIndex {
		response := c.executeCommand[c.executeCommandIndex]
		c.executeCommandIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}
