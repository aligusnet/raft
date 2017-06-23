package raft

import (
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/state"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type FollowerRole struct {
	dispatcher *Dispatcher
}

func (r *FollowerRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.RequestVoteResponse(requestVote.in)
			requestVote.send(response)
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, ok := state.AppendEntriesResponse(appendEntries.in)
			appendEntries.send(response)
			if ok {
				state.CommitUpTo(appendEntries.in.CommitIndex)
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			response := r.executeCommandResponse(state)
			if response != nil {
				executeCommand.send(response)
			} else {
				executeCommand.sendError(fmt.Errorf("Leader is unknown"))
			}

		case <-time.After(state.Timeout):
			glog.Info("[Follower] election timeout")
			return CandidateRoleHandle, state // timeout
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}

func (r *FollowerRole) executeCommandResponse(state *state.State) *pb.ExecuteCommandResponse {
	address := state.LeaderAddress()
	if len(address) > 0 {
		response := &pb.ExecuteCommandResponse{
			Success:       false,
			ServerAddress: address,
		}
		return response
	}
	return nil
}
