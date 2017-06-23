package server

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/server/state"
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
func (peer *Replica) appendEntriesRequest(state *state.State,
	in *pb.AppendEntriesResponse,
) (*pb.AppendEntriesRequest, bool) {

	if in.Success {
		return nil, false
	}

	if in.Term > state.CurrentTerm {
		return nil, true
	}

	peer.nextIndex--
	return state.AppendEntriesRequest(peer.nextIndex), false
}
