package raft

import (
	"fmt"
	"github.com/alexander-ignatyev/raft/state"
	"golang.org/x/net/context"
)

type RoleHandle int

const (
	LeaderRoleHandle RoleHandle = 1 + iota
	FollowerRoleHandle
	CandidateRoleHandle
	ExitRoleHandle
)

func (rh RoleHandle) String() string {
	switch rh {
	case LeaderRoleHandle:
		return "Leader"
	case FollowerRoleHandle:
		return "Follower"
	case CandidateRoleHandle:
		return "Candidate"
	case ExitRoleHandle:
		return "ExitRole"
	default:
		return fmt.Sprintf("Role-%b", rh)
	}
}

type Role interface {
	RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State)
}

type ExitRole struct{}

func (r *ExitRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	return ExitRoleHandle, state
}

var exitRoleInstance *ExitRole = new(ExitRole)
