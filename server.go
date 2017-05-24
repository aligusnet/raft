package raft

import (
	"context"
)

type Server struct {
	*Balancer
	ctx    context.Context
	cancel context.CancelFunc
	roles  map[RoleHandle]Role
}

func newServer() *Server {
	s := &Server{Balancer: newBalancer(),
		roles: make(map[RoleHandle]Role),
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	return s
}

func (s *Server) getRole(handle RoleHandle) Role {
	if r, ok := s.roles[handle]; ok {
		return r
	} else {
		return exitRoleInstance
	}

}

func (s *Server) run(handle RoleHandle, state *State) {
	for ; handle != ExitRoleHandle; handle, state = s.getRole(handle).RunRole(s.ctx, state) {
	}
}

func (s *Server) Stop() {
	s.cancel()
}
