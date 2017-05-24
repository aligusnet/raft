package raft

import (
	"context"
)

type Server struct {
	*Balancer

	roles map[RoleHandle]Role
}

func newServer() *Server {
	return &Server{Balancer: newBalancer(),
		roles: make(map[RoleHandle]Role),
	}
}

func (s *Server) getRole(handle RoleHandle) Role {
	if r, ok := s.roles[handle]; ok {
		return r
	} else {
		return exitRoleInstance
	}

}

func (s *Server) run(ctx context.Context, handle RoleHandle, state *State) {
	for ; handle != ExitRoleHandle; handle, state = s.getRole(handle).RunRole(ctx, state) {
	}
}
