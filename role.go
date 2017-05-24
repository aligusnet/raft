package raft

import "context"

type RoleHandle int

const (
	LeaderRoleHandle    RoleHandle = iota
	FollowerRoleHandle  RoleHandle = iota
	CandidateRoleHandle RoleHandle = iota
	ExitRoleHandle      RoleHandle = iota
)

type Role interface {
	RunRole(ctx context.Context, state *State) (RoleHandle, *State)
}

type ExitRole struct{}

func (r *ExitRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	return ExitRoleHandle, state
}

var exitRoleInstance *ExitRole = new(ExitRole)
