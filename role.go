package raft

type RoleHandle int

const (
	LeaderRoleHandle    RoleHandle = iota
	FollowerRoleHandle  RoleHandle = iota
	CandidateRoleHandle RoleHandle = iota
	ExitRoleHandle      RoleHandle = iota
)

type Role interface {
	RunRole() RoleHandle
}

type ExitRole struct{}

func (r *ExitRole) RunRole() RoleHandle {
	return ExitRoleHandle
}

var exitRoleInstance *ExitRole = new(ExitRole)
