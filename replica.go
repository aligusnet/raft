package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
)

type Replica struct {
	id         int64
	address    string
	client     pb.RaftClient
	nextIndex  int64
	matchIndex int64
}

// respond on peer's AppendEntriesRespond. Suitable if the current replica is leader.
// return AppendEntriesRequest if we need to send new request to the peer,
// return true if the leader has to be "demoted" to follower.
func (peer *Replica) appendEntriesRequest(state *State,
	in *pb.AppendEntriesResponse,
) (*pb.AppendEntriesRequest, bool) {

	if in.Success {
		return nil, false
	}

	if in.Term > state.currentTerm {
		return nil, true
	}

	peer.nextIndex--
	return state.appendEntriesRequest(peer.nextIndex), false
}
