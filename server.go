package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"golang.org/x/net/context"
)

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

type channelSet struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage
	quitCh           chan bool
}

func newChannelSet() *channelSet {
	return &channelSet{requestVoteCh: make(chan *requestVoteMessage),
		appendEntriesCh:  make(chan *appendEntriesMessage),
		executeCommandCh: make(chan *executeCommandMessage),
		quitCh:           make(chan bool)}
}

type Server struct {
	channels *channelSet

	roles map[RoleHandle]Role
}

func newServer() *Server {
	return &Server{channels: newChannelSet(),
		roles: make(map[RoleHandle]Role),
	}
}

func (s *Server) RequestVote(ctx context.Context, in *pb.RequestVoteRequest) (*pb.RequestVoteResponse, error) {
	resultCh := make(chan struct {
		*pb.RequestVoteResponse
		error
	}, 1)
	msg := &requestVoteMessage{ctx, in, resultCh}
	s.channels.requestVoteCh <- msg
	result := <-resultCh
	return result.RequestVoteResponse, result.error
}

func (s *Server) AppendEntries(ctx context.Context, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {
	resultCh := make(chan struct {
		*pb.AppendEntriesResponse
		error
	}, 1)
	msg := &appendEntriesMessage{ctx, in, resultCh}
	s.channels.appendEntriesCh <- msg
	result := <-resultCh
	return result.AppendEntriesResponse, result.error
}

func (s *Server) ExecuteCommand(ctx context.Context, in *pb.ExecuteCommandRequest) (*pb.ExecuteCommandResponse, error) {
	resultCh := make(chan struct {
		*pb.ExecuteCommandResponse
		error
	}, 1)
	msg := &executeCommandMessage{ctx, in, resultCh}
	s.channels.executeCommandCh <- msg
	result := <-resultCh
	return result.ExecuteCommandResponse, result.error
}

func (s *Server) getRole(handle RoleHandle) Role {
	if r, ok := s.roles[handle]; ok {
		return r
	} else {
		return exitRoleInstance
	}

}

func (s *Server) run(handle RoleHandle, state *State) {
	for ; handle != ExitRoleHandle; handle, state = s.getRole(handle).RunRole(state) {
	}
}

func (s *Server) Stop() error {
	s.channels.quitCh <- true
	return nil
}
