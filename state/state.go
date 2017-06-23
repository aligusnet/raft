package state

import (
	"fmt"
	"github.com/alexander-ignatyev/raft/log"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/golang/glog"
	"time"
)

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
		votedFor:        id,
		CurrentLeaderId: -1,
		addresses:       addresses,
		commitIndex:     -1,
		machine:         machine,
	}
}

func (s *State) SetTerm(term int64) {
	if s.CurrentTerm < term {
		s.votedFor = s.id
	} else if s.CurrentTerm > term {
		panic("We cannon go back in time")
	}
	s.CurrentTerm = term

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
	// TODO: revise logic of granting votes
	response := &pb.RequestVoteResponse{Term: s.CurrentTerm}
	lastLogIndex, lastLogTerm := s.LastLogIndexAndTerm()
	if s.votedFor != s.id {
		// we shot not grant vote if we've voted
		response.VoteGranted = false
	} else if in.Term < s.CurrentTerm {
		// we don't vote for candidates with stale Term
		response.VoteGranted = false
	} else if in.LastLogTerm < lastLogTerm { // Log's up-to-date checking
		response.VoteGranted = false
	} else if in.LastLogTerm > lastLogTerm {
		response.VoteGranted = true
	} else if in.LastLogIndex >= lastLogIndex {
		response.VoteGranted = true
	} else if in.LastLogIndex < lastLogIndex {
		response.VoteGranted = false
	}

	if response.VoteGranted {
		s.votedFor = in.CandidateId
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
		return response, false
	} else {
		s.CurrentLeaderId = request.LeaderId
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
			s.Log.EraseAfter(request.PrevLogIndex)
			for _, entry := range request.Entries {
				s.Log.Append(entry.Term, entry.Command)
			}
		}

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
	}
	if s.commitIndex != index {
		s.commitIndex = index
		glog.Infof("State Machine after executed command # %v: %v", index, s.machine.Debug())
		glog.Flush()
	}
	return res, err
}
