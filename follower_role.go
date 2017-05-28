package raft

import (
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
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
			response := r.executeCommandResponse(state)
			if response != nil {
				executeCommand.send(response)
			} else {
				executeCommand.sendError(fmt.Errorf("Leader is unknown"))
			}

		case <-time.After(state.timeout):
			glog.Info("[Follower] election timeout")
			return CandidateRoleHandle, state // timeout
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}

func (r *FollowerRole) executeCommandResponse(state *State) *pb.ExecuteCommandResponse {
	address := state.leaderAddress()
	if len(address) > 0 {
		response := &pb.ExecuteCommandResponse{
			Success:       false,
			ServerAddress: address,
		}
		return response
	}
	return nil
}
