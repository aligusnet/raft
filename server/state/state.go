package state

import (
	"fmt"
	"github.com/aligusnet/raft/server/log"
	pb "github.com/aligusnet/raft/raft"
	"github.com/golang/glog"
	"time"
)

const notVoted int64 = -1
const unknownLeader int64 = -1

type State struct {
	id              int64
	CurrentTerm     int64
	votedFor        int64
	commitIndex     int64
	lastApplied     int64
	CurrentLeaderId int64
	Timeout         time.Duration
	Log             log.Log
	addresses       map[int64]string
	machine         StateMachine
}

func New(id int64,
	timeout time.Duration,
	addresses map[int64]string,
	log log.Log,
	machine StateMachine) *State {

	return &State{
		id:              id,
		Timeout:         timeout,
		Log:             log,
		votedFor:        notVoted,
		CurrentLeaderId: unknownLeader,
		addresses:       addresses,
		commitIndex:     -1,
		machine:         machine,
	}
}

func (s *State) EnterElectionRace() {
	s.CurrentTerm++
	s.votedFor = s.id
}

// Get address of current leader.
// Returns empty string if leader's address is unkwnown
func (s *State) LeaderAddress() string {
	if address, ok := s.addresses[s.CurrentLeaderId]; ok {
		return address
	}
	return ""
}

func (s *State) LastLogIndexAndTerm() (int64, int64) {
	index := s.Log.Size() - 1
	lastLogIndex := int64(-1)
	lastLogTerm := int64(-1)
	if index >= 0 {
		logEntry := s.Log.Get(index)
		lastLogIndex = logEntry.Index
		lastLogTerm = logEntry.Term
	}
	return lastLogIndex, lastLogTerm
}

func (s *State) RequestVoteRequest() *pb.RequestVoteRequest {
	lastLogIndex, lastLogTerm := s.LastLogIndexAndTerm()
	return &pb.RequestVoteRequest{
		CandidateId:  s.id,
		Term:         s.CurrentTerm,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
}

func (s *State) RequestVoteResponse(in *pb.RequestVoteRequest) *pb.RequestVoteResponse {
	response := &pb.RequestVoteResponse{Term: s.CurrentTerm}
	lastLogIndex, lastLogTerm := s.LastLogIndexAndTerm()
	if in.Term < s.CurrentTerm {
		// we don't vote for candidates with stale Term
		response.VoteGranted = false
	} else if in.Term == s.CurrentTerm && s.votedFor != notVoted {
		// we already voted in the current term
		response.VoteGranted = false
	} else if in.LastLogTerm < lastLogTerm || in.LastLogIndex < lastLogIndex {
		// Log's up-to-date checking
		response.VoteGranted = false
	} else {
		response.VoteGranted = true
	}

	if response.VoteGranted {
		s.votedFor = in.CandidateId
		s.CurrentTerm = in.Term
		s.abandoneLeader()
	}
	return response
}

func (s *State) AppendEntriesRequest(peerNextLogIndex int64) *pb.AppendEntriesRequest {
	prevLogIndex := peerNextLogIndex - 1
	prevLogTerm := int64(-1)
	if prevLogIndex >= 0 {
		prevLogTerm = s.Log.Get(prevLogIndex).Term
	}
	request := &pb.AppendEntriesRequest{
		Term:         s.CurrentTerm,
		LeaderId:     s.id,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		CommitIndex:  s.commitIndex,
		Entries:      make([]*pb.LogEntry, 0),
	}

	for i := peerNextLogIndex; i < s.Log.Size(); i++ {
		request.Entries = append(request.Entries, s.Log.Get(i))
	}

	return request
}

// returns AppendEntriesResponse, true if request is accepted otherwise false
func (s *State) AppendEntriesResponse(request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	response := &pb.AppendEntriesResponse{}
	if request.Term < s.CurrentTerm {
		response.Term = s.CurrentTerm
		response.Success = false
		response.NextLogIndex = s.Log.Size()
		return response, false
	} else {
		lastLogIndex := s.Log.Size() - 1
		s.CurrentTerm = request.Term
		response.Term = s.CurrentTerm

		response.Success = false
		if request.PrevLogIndex < 0 {
			response.Success = true
		} else if lastLogIndex >= request.PrevLogIndex {
			prevLogTerm := s.Log.Get(request.PrevLogIndex).Term
			response.Success = prevLogTerm == request.PrevLogTerm
		}

		if response.Success {
			if request.PrevLogIndex < s.commitIndex {
				// danger zone: check term and log index
				offset := s.commitIndex - request.PrevLogIndex - 1
				numEntries := int64(len(request.Entries))
				for i := int64(0); i < offset && i < numEntries; i++ {
					oldEntry := s.Log.Get(i+request.PrevLogIndex+1)
					newEntry := request.Entries[i]
					if oldEntry.Term != newEntry.Term || oldEntry.Index != newEntry.Index {
						glog.Errorf("[State] got inconsistency in AppendEntries request from leader id %v",
							request.LeaderId)
					}
				}
			}
			s.Log.EraseAfter(request.PrevLogIndex)
			for _, entry := range request.Entries {
				s.Log.Append(entry.Term, entry.Command)
			}
		}

		response.NextLogIndex = s.Log.Size()
		s.discoverLeader(request.LeaderId)
		return response, true
	}
}

func (s *State) CommitUpTo(index int64) ([]byte, error) {
	if index < 0 {
		return nil, nil
	}
	var err error = nil
	var res []byte = nil
	if index >= s.Log.Size() {
		err = fmt.Errorf("Got incorrect commit index: %v, Log's size: %v", index, s.Log.Size())
		glog.Error(err)
		glog.Flush()
		return nil, err
	}
	for i := s.commitIndex + 1; i <= index; i++ {
		res, err = s.machine.ExecuteCommand(s.Log.Get(i).Command)
		if err != nil {
			glog.Warningf("Got error after executing command # %v: %v", i, err)
		}
		s.commitIndex = i
		glog.Infof("State Machine after executed command # %v: %v", i, s.machine.Debug())
		glog.Flush()
	}
	return res, err
}

func (s *State) discoverLeader(id int64) {
	if s.CurrentLeaderId != id {
		s.CurrentLeaderId = id
		glog.Infof("[State] discover new leader: %v, commit index: %v", id, s.commitIndex)
	}
}

func (s *State) abandoneLeader() {
	if s.CurrentLeaderId != unknownLeader {
		glog.Infof("[State] abandoning old leader: %v %v", s.CurrentLeaderId, s.commitIndex)
		s.CurrentLeaderId = unknownLeader
	}
}
