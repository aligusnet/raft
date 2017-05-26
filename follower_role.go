package raft

import (
	"context"
	"fmt"
)

type FollowerRole struct {
	dispatcher *Dispatcher
}

func (r *FollowerRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	ctx, _ = context.WithTimeout(ctx, state.timeout)

	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.requestVoteResponse(requestVote.in)
			requestVote.send(response)
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, _ := state.appendEntriesReesponse(appendEntries.in)
			appendEntries.send(response)
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.sendError(fmt.Errorf("Not yest implemented"))
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
