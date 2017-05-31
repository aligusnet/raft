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

type appendEntriesPeerMessage struct {
	peerId   int64
	response *pb.AppendEntriesResponse
}

type heartbeatTimeoutKeyType int

const heartbeatTimeoutKey heartbeatTimeoutKeyType = 174

func (r *LeaderRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	timeout := generateHeartbeetTimeout(state.timeout)
	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, heartbeatTimeoutKey, timeout)
	defer cancel()

	peersInCh := make(chan *appendEntriesPeerMessage, len(r.replicas))
	peerOutChList := make(map[int64]chan *pb.AppendEntriesRequest)
	for _, peer := range r.replicas {
		peerOutChList[peer.id] = make(chan *pb.AppendEntriesRequest, 1)
		peer.nextIndex = state.log.Size()
		go peerThread(ctx, peer.id, peer.client, peersInCh, peerOutChList[peer.id])
	}

	// Initial heartbeat
	request := state.appendEntriesRequest(state.log.Size())
	for _, ch := range peerOutChList {
		ch <- request
	}

	for {
		select {
		case appendEntriesPeer := <-peersInCh:
			if peer, ok := r.replicas[appendEntriesPeer.peerId]; ok {
				request, demoted := peer.appendEntriesRequest(state, appendEntriesPeer.response)
				if demoted {
					return FollowerRoleHandle, state
				} else if request != nil {
					out := peerOutChList[appendEntriesPeer.peerId]
					out <- request
				}
			} else {
				glog.Warning("Got appendEntriesPeerMessage from unknown peer:", appendEntriesPeer)
			}
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

func peerThread(ctx context.Context, id int64,
	client pb.RaftClient,
	out chan<- *appendEntriesPeerMessage,
	in <-chan *pb.AppendEntriesRequest) {

	var request *pb.AppendEntriesRequest
	timeout := ctx.Value(heartbeatTimeoutKey).(time.Duration)
	for {
		select {
		case request = <-in:
			if resp, err := client.AppendEntries(ctx, request); err == nil {
				stripToHeartbeet(request)
				msg := &appendEntriesPeerMessage{
					peerId:   id,
					response: resp,
				}
				out <- msg
			} else {
				glog.Warningf("[Leader] [peer thread: %v] got error message from the peer: %v", id, err)
			}
		case <-time.After(timeout):
			if request != nil {
				glog.Infof("[Leader] [peer thread: %v] sending AppendEntries (heartbeat): %v", id, request)
				_, err := client.AppendEntries(ctx, request)
				if err != nil {
					glog.Warningf("[Leader] [peer thread: %v] failed to send AppendEntries to the peer: %v", id, err)
				}
			} else {
				glog.Warningf("[Leader] [peer thread: %v] cannot send heartbeat due to empty request: %v", id)
			}
		case <-ctx.Done():
			glog.Infof("[Leader] [peer thread: %v] exiting", id)
			return
		}
	}
}

func generateHeartbeetTimeout(timeout time.Duration) time.Duration {
	ns := timeout.Nanoseconds() * 5 / 10
	return time.Nanosecond * time.Duration(ns)
}

func stripToHeartbeet(request *pb.AppendEntriesRequest) {
	if len(request.Entries) > 0 {
		lastEntry := request.Entries[len(request.Entries)-1]
		request.PrevLogIndex = lastEntry.Index
		request.PrevLogTerm = lastEntry.Term
		request.Entries = request.Entries[:0]
	}
}
