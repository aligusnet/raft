package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type LeaderRole struct {
	dispatcher *Dispatcher
	replicas   map[int64]*Replica
}

func newLeaderRole(dispatcher *Dispatcher) *LeaderRole {
	return &LeaderRole{
		dispatcher: dispatcher,
		replicas:   make(map[int64]*Replica),
	}
}

const heartbeatTimeoutKey int = 174

func (r *LeaderRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	timeout := generateHeartbeetTimeout(state.timeout)
	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, heartbeatTimeoutKey, timeout)
	defer cancel()
	request := state.appendEntriesRequest(state.log.Size())
	for _, peer := range r.replicas {
		peer.nextIndex = state.log.Size()
		go peerThread(ctx, peer, request)
	}
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := r.requestVoteResponse(state, requestVote.in)
			requestVote.send(response)
			if response.VoteGranted {
				return FollowerRoleHandle, state
			}
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, accepted := r.appendEntriesResponse(state, appendEntries.in)
			appendEntries.send(response)
			if accepted {
				return FollowerRoleHandle, state
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			response := &pb.ExecuteCommandResponse{Success: true}
			executeCommand.send(response)
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}

func (r *LeaderRole) requestVoteResponse(state *State, in *pb.RequestVoteRequest) *pb.RequestVoteResponse {
	if in.Term <= state.currentTerm {
		return &pb.RequestVoteResponse{
			Term:        state.currentTerm,
			VoteGranted: false,
		}
	} else {
		return state.requestVoteResponse(in)
	}
}

func (r *LeaderRole) appendEntriesResponse(state *State, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	if in.Term == state.currentLeaderId {
		glog.Fatalf("[Leader] got appendEntries from another leader with the same Term: %v", in)
	}
	if in.Term < state.currentTerm {
		return &pb.AppendEntriesResponse{
			Term:    state.currentTerm,
			Success: false,
		}, false
	} else {
		return state.appendEntriesResponse(in)
	}
}

func peerThread(ctx context.Context, peer *Replica, r *pb.AppendEntriesRequest) {
	request := new(pb.AppendEntriesRequest)
	*request = *r
	glog.Infof("[Leader] [peer thread: %v] sending initial AppendEntries (heartbeat): %v", peer.id, request)
	peer.client.AppendEntries(ctx, request)

	timeout := ctx.Value(heartbeatTimeoutKey).(time.Duration)
	for {
		select {
		case <-time.After(timeout):
			glog.Infof("[Leader] [peer thread: %v] sending AppendEntries (heartbeat): %v", peer.id, request)
			_, err := peer.client.AppendEntries(ctx, request)
			if err != nil {
				glog.Warningf("[Leader] [peer thread: %v] failed to send AppendEntries to the peer: %v", peer.id, err)
			}
		case <-ctx.Done():
			glog.Infof("[Leader] [peer thread: %v] exiting", peer.id)
			return
		}
	}
}

func generateHeartbeetTimeout(timeout time.Duration) time.Duration {
	ns := timeout.Nanoseconds() * 5 / 10
	return time.Nanosecond * time.Duration(ns)
}
