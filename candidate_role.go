package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
)

type CandidateRole struct {
	replicas []*Replica
	state    *State
}

func (r *CandidateRole) RunRole() RoleHandle {
	request := requestVoteRequest(r.state)
	votes := 1
	for _, peer := range r.replicas {
		resp, err := peer.client.RequestVote(context.Background(), request)
		if err == nil {
			if resp.VoteGranted {
				votes++
			}
		}
	}
	if votes >= requiredVotes(len(r.replicas)+1) {
		return LeaderRoleHandle
	} else {
		return CandidateRoleHandle
	}
}

func requiredVotes(replicasNum int) int {
	return replicasNum/2 + 1
}

func requestVoteRequest(s *State) *pb.RequestVoteRequest {
	return &pb.RequestVoteRequest{
		CandidateId:  0,
		Term:         0,
		LastLogIndex: 0,
		LastLogTerm:  0,
	}

}
