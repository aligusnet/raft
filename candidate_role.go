package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
)

type CandidateRole struct {
	replicas []*Replica
}

func (r *CandidateRole) RunRole(state *State) (RoleHandle, *State) {
	request := requestVoteRequest(state)
	ctx := context.Background()
	result := make(chan bool, len(r.replicas))

	requestVote := func(peer *Replica, quit chan interface{}) {
		for {
			resp, err := peer.client.RequestVote(ctx, request)
			if err == nil {
				result <- resp.VoteGranted
				return
			}
			select {
			case <-quit:
				return
			default:
			}
		}
	}

	quits := make([]chan interface{}, 0)
	for _, peer := range r.replicas {
		quit := make(chan interface{}, 1)
		quits = append(quits, quit)
		go requestVote(peer, quit)
	}

	defer func() {
		for _, quit := range quits {
			quit <- true
		}
	}()

	positiveVotes := 1
	votes := 0
	for {
		select {
		case res := <-result:
			if res {
				positiveVotes++
			}
			votes++
			if positiveVotes >= requiredVotes(len(r.replicas)+1) {
				return LeaderRoleHandle, state
			}

			if votes >= len(r.replicas) {
				return CandidateRoleHandle, state
			}
		}
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
