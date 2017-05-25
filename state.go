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
	response := &pb.RequestVoteResponse{Term: s.currentTerm}
	lastLogIndex, _ := s.lastLogIndexAndTerm()
	if s.votedFor != s.id {
		response.VoteGranted = false
	} else if in.Term < s.currentTerm {
		response.VoteGranted = false
	} else if in.LastLogIndex > lastLogIndex {
		response.VoteGranted = true
	} else if in.LastLogIndex == lastLogIndex {
		response.VoteGranted = true
	}

	if response.VoteGranted {
		s.votedFor = in.CandidateId
	}
	return response
}

func (s *State) appendEntriesRequestBuilder() func(LogReader, int64) *pb.AppendEntriesRequest {
	term := s.currentTerm
	leaderId := s.id

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
			CommitIndex:  0, // TODO: fill CommitIndex
		}

		for i := peerNextLogIndex; i < log.Size(); i++ {
			request.Entries = append(request.Entries, log.Get(i))
		}

		return request
	}
	return builder
}

// returns AppendEntriesResponse, true if request is accepted otherwise false
func (s *State) appendEntriesReesponse(request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, bool) {
	response := &pb.AppendEntriesResponse{}
	if request.Term < s.currentTerm {
		response.Term = s.currentTerm
		response.Success = false
		return response, false
	} else {
		lastLogIndex := s.log.Size() - 1
		s.currentTerm = request.Term
		response.Term = s.currentTerm
		response.Success = false
		if lastLogIndex >= request.PrevLogIndex {
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
