package server

import (
	"fmt"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/server/state"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type FollowerRole struct {
	dispatcher *Dispatcher
	timeout time.Duration
	electionTimer *time.Timer
}

func (r *FollowerRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	r.init(state.Timeout)
	defer r.dispose()
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := state.RequestVoteResponse(requestVote.in)
			if response.VoteGranted {
				r.resetElectionTimer()
			}
			requestVote.send(response)
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			r.resetElectionTimer()
			response, _ := state.AppendEntriesResponse(appendEntries.in)
			appendEntries.send(response)
			if response.Success {
				state.CommitUpTo(appendEntries.in.CommitIndex)
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			response := r.executeCommandResponse(state)
			if response != nil {
				executeCommand.send(response)
			} else {
				executeCommand.sendError(fmt.Errorf("Leader is unknown"))
			}

		case <- r.electionTimer.C:
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

func (r *FollowerRole) resetElectionTimer() {
	if !r.electionTimer.Stop() {
		<- r.electionTimer.C
	}
	r.electionTimer.Reset(r.timeout)
}

func (r *FollowerRole) init(d time.Duration) {
	r.timeout = d
	r.electionTimer = time.NewTimer(d)
}

func (r *FollowerRole) dispose() {
	r.electionTimer.Stop()
}
