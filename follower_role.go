package raft

import (
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type FollowerRole struct {
	dispatcher *Dispatcher
}

func (r *FollowerRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.requestVoteResponse(requestVote.in)
			requestVote.send(response)
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, _ := state.appendEntriesResponse(appendEntries.in)
			appendEntries.send(response)
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.sendError(fmt.Errorf("Not yest implemented"))
		case <-time.After(state.timeout):
			glog.Info("[Follower] election timeout")
			return CandidateRoleHandle, state // timeout
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}
