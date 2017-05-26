package raft

import (
	"context"
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"math/rand"
	"time"
)

const requestKey int = 173

type CandidateRole struct {
	replicas   []*Replica
	dispatcher *Dispatcher
}

func (r *CandidateRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	state.currentTerm++
	state.votedFor = state.id
	timeout := generateTimeout(state.timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	ctx = context.WithValue(ctx, requestKey, state.requestVoteRequest())
	defer cancel()

	result := make(chan bool, len(r.replicas))

	for _, peer := range r.replicas {
		go requestVote(ctx, peer, result)
	}

	positiveVotes := 1
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.requestVoteResponse(requestVote.in)
			if response.VoteGranted {
				positiveVotes--
			}
			requestVote.send(response)
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, accepted := state.appendEntriesResponse(appendEntries.in)
			appendEntries.send(response)
			if accepted {
				return FollowerRoleHandle, state
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.sendError(fmt.Errorf("Leader is unknown"))
		case res := <-result:
			if res {
				positiveVotes++
			}
			if positiveVotes >= requiredVotes(len(r.replicas)+1) {
				return LeaderRoleHandle, state
			}
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return CandidateRoleHandle, state // timeout
			} else if ctx.Err() == context.Canceled {
				return ExitRoleHandle, state
			} else {
				panic(fmt.Sprintf("Unexpected Context.Err: %v", ctx.Err()))
			}
		}
	}
}

func requestVote(ctx context.Context, peer *Replica, result chan bool) {
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
