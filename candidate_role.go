package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
	"math/rand"
	"time"
)

const requestKey int = 173

type CandidateRole struct {
	replicas []*Replica
}

func (r *CandidateRole) RunRole(state *State) (RoleHandle, *State) {
	timeout := generateTimeout(state.timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	ctx = context.WithValue(ctx, requestKey, requestVoteRequest(state))
	defer cancel()

	result := make(chan bool, len(r.replicas))

	for _, peer := range r.replicas {
		go requestVote(peer, ctx, result)
	}

	positiveVotes := 1
	totalVotes := 0
	for {
		select {
		case res := <-result:
			totalVotes++
			if res {
				positiveVotes++
			}
			if positiveVotes >= requiredVotes(len(r.replicas)+1) {
				return LeaderRoleHandle, state
			}
			if totalVotes >= len(r.replicas) {
				return CandidateRoleHandle, state
			}
		case <-ctx.Done():
			return CandidateRoleHandle, state // timeout
		}
	}
}

func requestVote(peer *Replica, ctx context.Context, result chan bool) {
	request := ctx.Value(requestKey).(*pb.RequestVoteRequest)
	for {
		resp, err := peer.client.RequestVote(ctx, request)
		if err == nil {
			result <- resp.VoteGranted
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func requiredVotes(replicasNum int) int {
	return replicasNum/2 + 1
}

func generateTimeout(timeout time.Duration) time.Duration {
	ns := timeout.Nanoseconds()
	ns += rand.Int63n(ns)
	return time.Nanosecond * time.Duration(ns)
}
