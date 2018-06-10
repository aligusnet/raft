package server

import (
	"container/list"
	"fmt"
	pb "github.com/aligusnet/raft/raft"
	"github.com/aligusnet/raft/server/state"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"time"
)

type LeaderRole struct {
	dispatcher    *Dispatcher
	replicas      map[int64]*Replica
	peerOutChList map[int64]chan *pb.AppendEntriesRequest
	waitlist      *list.List
}

func newLeaderRole(dispatcher *Dispatcher) *LeaderRole {
	return &LeaderRole{
		dispatcher: dispatcher,
		replicas:   make(map[int64]*Replica),
		waitlist:   list.New(),
	}
}

type appendEntriesPeerMessage struct {
	peerId       int64
	response     *pb.AppendEntriesResponse
}

type executeCommandItem struct {
	message              *executeCommandMessage
	logIndex             int64
	responses            map[int64]bool
	numPositiveResponses int
}

type heartbeatTimeoutKeyType int

const heartbeatTimeoutKey heartbeatTimeoutKeyType = 174

func (r *LeaderRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	timeout := generateHeartbeetTimeout(state.Timeout)
	ctx, cancel := context.WithCancel(ctx)
	ctx = context.WithValue(ctx, heartbeatTimeoutKey, timeout)
	defer cancel()
	defer r.clean()

	peersInCh := make(chan *appendEntriesPeerMessage, len(r.replicas))
	r.peerOutChList = make(map[int64]chan *pb.AppendEntriesRequest)
	for _, peer := range r.replicas {
		r.peerOutChList[peer.id] = make(chan *pb.AppendEntriesRequest, 1)
		peer.nextIndex = state.Log.Size()
		go peerThread(ctx, peer.id, peer.client, peersInCh, r.peerOutChList[peer.id])
	}

	// Initial heartbeat
	request := state.AppendEntriesRequest(state.Log.Size())
	for _, ch := range r.peerOutChList {
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
					out := r.peerOutChList[appendEntriesPeer.peerId]
					go func() {
						out <- request
					}()
				} else {
					r.processAppendEntriesResponse(state, appendEntriesPeer)
				}
			} else {
				glog.Warning("Got appendEntriesPeerMessage from unknown peer:", appendEntriesPeer)
			}
		case requestVote := <-r.dispatcher.requestVoteCh:
			response := r.requestVoteResponse(state, requestVote.in)
			requestVote.send(response)
			if response.VoteGranted {
				glog.Infof("[Leader] granted vote for %v, stepping out...", requestVote.in.CandidateId)
				return FollowerRoleHandle, state
			}
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			response, accepted := r.appendEntriesResponse(state, appendEntries.in)
			appendEntries.send(response)
			if accepted {
				glog.Infof("[Leader] accepted AppendEntries from %v, stepping out...", appendEntries.in.LeaderId)
				return FollowerRoleHandle, state
			}
		case executeCommand := <-r.dispatcher.executeCommandCh:
			r.executeCommand(state, executeCommand)
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}

func (r *LeaderRole) requestVoteResponse(state *state.State, in *pb.RequestVoteRequest) *pb.RequestVoteResponse {
	if in.Term <= state.CurrentTerm {
		return &pb.RequestVoteResponse{
			Term:        state.CurrentTerm,
			VoteGranted: false,
		}
	} else {
		return state.RequestVoteResponse(in)
	}
}

func (r *LeaderRole) appendEntriesResponse(state *state.State, in *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	if in.Term == state.CurrentTerm {
		glog.Fatalf("[Leader] got appendEntries from another leader with the same Term: %v", in)
	}
	return state.AppendEntriesResponse(in)
}

func (r *LeaderRole) executeCommand(state *state.State, message *executeCommandMessage) {
	logIndex := state.Log.Append(state.CurrentTerm, message.in.Command)
	item := &executeCommandItem{
		message:   message,
		responses: make(map[int64]bool),
		logIndex:  logIndex,
		numPositiveResponses: 1,
	}
	r.waitlist.PushBack(item)
	for _, peer := range r.replicas {
		request := state.AppendEntriesRequest(peer.nextIndex)
		if len(request.Entries) == 0 {
			glog.Warningf("[Leader] got empty request for peer id %v", peer.id)
			continue
		}
		ch, ok := r.peerOutChList[peer.id]
		if ok {
			go func() {
				ch <- request
			}()
		} else {
			glog.Warningf("[Leader] [peer: %v] cannot find the peer's channel, message won't be sent", peer.id)
		}
	}
}

func (r *LeaderRole) processAppendEntriesResponse(state *state.State, message *appendEntriesPeerMessage) {
	peerLastLogIndex := message.response.NextLogIndex - 1
	glog.Infof("[Leader] got response from peer %v %v", message.peerId, message.response)
	if peer, ok := r.replicas[message.peerId]; ok {
		peer.nextIndex = message.response.NextLogIndex
	} else {
		glog.Errorf("Got response from unknown peer: %v", message.peerId)
		return
	}

	var next *list.Element
	for e := r.waitlist.Front(); e != nil; e = next {
		next = e.Next()
		item, ok := e.Value.(*executeCommandItem)
		if !ok {
			glog.Errorf("[Leader] waitlist contains unexpected value: %v, %T", e.Value, e.Value)
			continue
		}
		if item.logIndex > peerLastLogIndex {
			break
		}
		if _, ok := item.responses[message.peerId]; !ok {
			item.responses[message.peerId] = message.response.Success
			if message.response.Success {
				item.numPositiveResponses++
				if item.numPositiveResponses >= requiredResponses(len(r.replicas)) {
					glog.Infof("[Leader] sending success message for Execute Command, log index: %v",
						peerLastLogIndex)
					res, err := state.CommitUpTo(item.logIndex)
					if err == nil {
						go item.message.send(&pb.ExecuteCommandResponse{Success: true, Answer: res})
					} else {
						go item.message.sendError(err)
					}

					r.waitlist.Remove(e)
				}
			}
			if len(item.responses) >= len(r.replicas) {
				glog.Fatalf("[Leader] Failed to execute command, log index: %v",
					peerLastLogIndex)
			}
		}
	}
}

func (r *LeaderRole) clean() {
	var next *list.Element
	for e := r.waitlist.Front(); e != nil; e = next {
		next = e.Next()
		item := e.Value.(*executeCommandItem)
		go item.message.sendError(fmt.Errorf("Sorry, I am not a master any more"))
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
					peerId:       id,
					response:     resp,
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

func requiredResponses(replicasNum int) int {
	return replicasNum/2 + 1
}
