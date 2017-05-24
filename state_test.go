package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestState(t *testing.T) {
	Convey("Given just created state", t, func() {
		state := newState(1, time.Millisecond*10)

		Convey("lastLogIndexAndTerm", func() {
			lastLogIndex, lastLogTerm := state.lastLogIndexAndTerm()
			So(lastLogIndex, ShouldEqual, -1)
			So(lastLogTerm, ShouldEqual, -1)
		})
	})
	Convey("Given initialized aged state", t, func() {
		state := newState(1, time.Millisecond*10)
		state.currentTerm = 10
		state.log.Append(8, []byte("cmd1"))
		state.log.Append(8, []byte("cmd2"))
		state.log.Append(9, []byte("cmd3"))

		Convey("lastLogIndexAndTerm", func() {
			lastLogIndex, lastLogTerm := state.lastLogIndexAndTerm()
			So(lastLogIndex, ShouldEqual, 2)
			So(lastLogTerm, ShouldEqual, 9)
		})
	})
}

func TestState_RequestVote(t *testing.T) {
	Convey("Given initialized non-voted state", t, func() {
		state := newState(1, time.Millisecond*10)
		state.currentTerm = 10
		state.log.Append(8, []byte("cmd1"))
		state.log.Append(8, []byte("cmd2"))
		state.log.Append(9, []byte("cmd3"))

		Convey("we should create correct requestVote request", func() {
			lastLogIndex, lastLogTerm := state.lastLogIndexAndTerm()
			request := state.requestVoteRequest()
			So(request.CandidateId, ShouldEqual, state.id)
			So(request.Term, ShouldEqual, state.currentTerm)
			So(request.LastLogIndex, ShouldEqual, lastLogIndex)
			So(request.LastLogTerm, ShouldEqual, lastLogTerm)
		})

		request := &pb.RequestVoteRequest{CandidateId: 10}
		Convey("Given requestVore request with in.Term > currentTerm", func() {
			request.Term = state.currentTerm + 1
			Convey("and in.LastLogIndex > lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 4
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 1
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})

		Convey("Given requestVore request with in.Term = currentTerm", func() {
			request.Term = state.currentTerm
			Convey("and in.LastLogIndex > lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 4
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 1
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})

		Convey("Given requestVore request with in.Term < currentTerm", func() {
			request.Term = state.currentTerm - 1
			Convey("we should not vote for the candidate", func() {
				request.LastLogIndex = 4
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldBeGreaterThan, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})
	})

	Convey("Given initialized voted state", t, func() {
		state := newState(1, time.Millisecond*10)
		state.currentTerm = 10
		state.log.Append(8, []byte("cmd1"))
		state.log.Append(8, []byte("cmd2"))
		state.log.Append(9, []byte("cmd3"))
		state.votedFor = 5

		Convey("We should not vote for the candidate", func() {
			request := &pb.RequestVoteRequest{CandidateId: 10}
			request.Term = state.currentTerm + 1
			request.LastLogIndex = 4
			response := state.requestVoteResponse(request)
			So(response.VoteGranted, ShouldBeFalse)
		})
	})
}
