package server

import (
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/server/state"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"math/rand"
	"time"
)

const requestKey int = 173

type CandidateRole struct {
	dispatcher *Dispatcher
	replicas   map[int64]*Replica
}

func newCandidateRole(dispatcher *Dispatcher) *CandidateRole {
	return &CandidateRole{
		dispatcher: dispatcher,
		replicas:   make(map[int64]*Replica),
	}
}

func (r *CandidateRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	state.EnterElectionRace()
	timeout := generateTimeout(state.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	ctx = context.WithValue(ctx, requestKey, state.RequestVoteRequest())
	defer cancel()

	result := make(chan *pb.RequestVoteResponse, len(r.replicas))

	for _, peer := range r.replicas {
		go requestVote(ctx, peer, result)
	}

	positiveVotes := 1
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.RequestVoteResponse(requestVote.in)
			requestVote.send(response)
			// step out
			if response.VoteGranted {
				return FollowerRoleHandle, state
			}
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, accepted := state.AppendEntriesResponse(appendEntries.in)
			appendEntries.send(response)
			if accepted {
				return FollowerRoleHandle, state
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.sendError(fmt.Errorf("Leader is unknown"))
		case res := <-result:
			if res.VoteGranted {
				positiveVotes++
			} else if res.Term > state.CurrentTerm { // step out
				state.CurrentTerm = res.Term
				return FollowerRoleHandle, state
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

func requestVote(ctx context.Context, peer *Replica, result chan *pb.RequestVoteResponse) {
	request := ctx.Value(requestKey).(*pb.RequestVoteRequest)
	for {
		resp, err := peer.client.RequestVote(ctx, request)
		if err == nil {
			glog.Infof("[Candidate] [peer thread: %v] got response: %v", peer.id, resp)
			result <- resp
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
