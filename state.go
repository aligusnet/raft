package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"time"
)

type State struct {
	id          int64
	currentTerm int64
	votedFor    int64
	commitIndex int64
	lastApplied int64
	timeout     time.Duration
	log         Log
}

func newState(id int64, timeout time.Duration) *State {
	return &State{
		id:       id,
		timeout:  timeout,
		log:      NewLog(),
		votedFor: id,
	}
}

func (s *State) setTerm(term int64) {
	if s.currentTerm < term {
		s.votedFor = s.id
	} else if s.currentTerm > term {
		panic("We cannon go back in time")
	}
	s.currentTerm = term

}

func (s *State) lastLogIndexAndTerm() (int64, int64) {
	index := s.log.Size() - 1
	lastLogIndex := int64(-1)
	lastLogTerm := int64(-1)
	if index >= 0 {
		logEntry := s.log.Get(index)
		lastLogIndex = logEntry.Index
		lastLogTerm = logEntry.Term
	}
	return lastLogIndex, lastLogTerm
}

func (s *State) requestVoteRequest() *pb.RequestVoteRequest {
	lastLogIndex, lastLogTerm := s.lastLogIndexAndTerm()
	return &pb.RequestVoteRequest{
		CandidateId:  s.id,
		Term:         s.currentTerm,
		LastLogIndex: lastLogIndex,
		LastLogTerm:  lastLogTerm,
	}
}

func (s *State) requestVoteResponse(in *pb.RequestVoteRequest) *pb.RequestVoteResponse {
	// TODO: revise logic of granting votes
	response := &pb.RequestVoteResponse{Term: s.currentTerm}
	lastLogIndex, lastLogTerm := s.lastLogIndexAndTerm()
	if s.votedFor != s.id {
		// we shot not grant vote if we've voted
		response.VoteGranted = false
	} else if in.Term < s.currentTerm {
		// we don't vote for candidates with stale Term
		response.VoteGranted = false
	} else if in.LastLogTerm < lastLogTerm { // log's up-to-date checking
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

func (s *State) appendEntriesRequestBuilder() func(LogReader, int64) *pb.AppendEntriesRequest {
	term := s.currentTerm
	leaderId := s.id
	commitIndex := s.commitIndex

	builder := func(log LogReader, peerNextLogIndex int64) *pb.AppendEntriesRequest {
		prevLogIndex := peerNextLogIndex - 1
		prevLogTerm := int64(-1)
		if prevLogIndex >= 0 {
			prevLogTerm = s.log.Get(prevLogIndex).Term
		}
		request := &pb.AppendEntriesRequest{
			Term:         term,
			LeaderId:     leaderId,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			CommitIndex:  commitIndex,
			Entries:      make([]*pb.LogEntry, 0),
		}

		for i := peerNextLogIndex; i < log.Size(); i++ {
			request.Entries = append(request.Entries, log.Get(i))
		}

		return request
	}
	return builder
}

// returns AppendEntriesResponse, true if request is accepted otherwise false
func (s *State) appendEntriesResponse(request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	response := &pb.AppendEntriesResponse{}
	if request.Term < s.currentTerm {
		response.Term = s.currentTerm
		response.Success = false
		return response, false
	} else {
		lastLogIndex := s.log.Size() - 1
		s.currentTerm = request.Term
		response.Term = s.currentTerm

		// TODO: revise logic of processing AppendEntries
		response.Success = false
		if request.PrevLogIndex < 0 {
			response.Success = true
			s.log.EraseAfter(request.PrevLogIndex)
		} else if lastLogIndex >= request.PrevLogIndex {
			prevLogTerm := s.log.Get(request.PrevLogIndex).Term
			if prevLogTerm == request.PrevLogTerm {
				response.Success = true
				s.log.EraseAfter(request.PrevLogIndex)
				for _, entry := range request.Entries {
					s.log.Append(entry.Term, entry.Command)
				}
			}
		}
		return response, true
	}
}
