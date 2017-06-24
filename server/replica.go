package server

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/server/state"
	"github.com/golang/glog"
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
	if peer.nextIndex > in.NextLogIndex {
		peer.nextIndex = in.NextLogIndex
	}
	if peer.nextIndex < 0 {
		glog.Errorf("[Replica] got negative next peer index, peer id: %v", peer.id)
		peer.nextIndex = 0
	}
	return state.AppendEntriesRequest(peer.nextIndex), false
}
