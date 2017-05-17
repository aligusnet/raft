package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"golang.org/x/net/context"
)

type StateMachine interface {
	ExecuteCommand(command []byte) ([]byte, error)
	CommandToString(command []byte) (string, error)
}

type RoleHandle int

const (
	LeaderRole    RoleHandle = iota
	FollowerRole  RoleHandle = iota
	CanfifateRole RoleHandle = iota
)

type Role interface {
	RunRole() RoleHandle
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

type Server struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage

	roles       map[RoleHandle]Role
	currentRole Role
}

func newServer() *Server {
	return &Server{requestVoteCh: make(chan *requestVoteMessage),
		appendEntriesCh:  make(chan *appendEntriesMessage),
		executeCommandCh: make(chan *executeCommandMessage),
		roles:            make(map[RoleHandle]Role),
	}
}

func (s *Server) RequestVote(ctx context.Context, in *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resultCh := make(chan struct {
		*pb.RequestVoteResponse
		error
	}, 1)
	msg := &requestVoteMessage{ctx, in, resultCh}
	s.requestVoteCh <- msg
	result := <-resultCh
	return result.RequestVoteResponse, result.error
}

func (s *Server) AppendEntries(ctx context.Context, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resultCh := make(chan struct {
		*pb.AppendEntriesResponse
		error
	}, 1)
	msg := &appendEntriesMessage{ctx, in, resultCh}
	s.appendEntriesCh <- msg
	result := <-resultCh
	return result.AppendEntriesResponse, result.error
}

func (s *Server) ExecuteCommand(ctx context.Context, in *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	resultCh := make(chan struct {
		*pb.ExecuteCommandResponse
		error
	}, 1)
	msg := &executeCommandMessage{ctx, in, resultCh}
	s.executeCommandCh <- msg
	result := <-resultCh
	return result.ExecuteCommandResponse, result.error
}
