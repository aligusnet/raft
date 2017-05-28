package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestState(t *testing.T) {
	Convey("Given just created state", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())

		Convey("lastLogIndexAndTerm", func() {
			lastLogIndex, lastLogTerm := state.lastLogIndexAndTerm()
			So(lastLogIndex, ShouldEqual, -1)
			So(lastLogTerm, ShouldEqual, -1)
		})
	})
	Convey("Given initialized aged state", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
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

func TestState_LeaderAddress(t *testing.T) {
	Convey("No leader", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		So(state.leaderAddress(), ShouldBeEmpty)
	})

	Convey("No leader's address", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		state.currentLeaderId = 2
		So(state.leaderAddress(), ShouldBeEmpty)
	})

	Convey("We have leader and its address", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		state.currentLeaderId = 2
		state.addresses[2] = "address898"
		So(state.leaderAddress(), ShouldEqual, "address898")
	})
}

func TestState_RequestVote(t *testing.T) {
	Convey("Given initialized non-voted state", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
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
			Convey("and in.LastLogTerm < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 4
				request.LastLogTerm = 8
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
			})
			Convey("and in.LastLogIndex > lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 4
				request.LastLogTerm = 9
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				request.LastLogTerm = 9
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
				request.LastLogTerm = 9
				response := state.requestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				request.LastLogTerm = 9
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
		state := newState(1, time.Millisecond*10, NewLog())
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

func TestState_AppendEntries(t *testing.T) {
	Convey("Given initialized state", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		state.currentTerm = 10
		state.log.Append(8, []byte("cmd1"))
		state.log.Append(8, []byte("cmd2"))
		state.log.Append(9, []byte("cmd3"))

		build := state.appendEntriesRequestBuilder()

		Convey("we should create hearbeat message", func() {
			request := build(state.log, 3)
			So(request.Term, ShouldEqual, state.currentTerm)
			So(request.PrevLogIndex, ShouldEqual, 2)
			So(request.PrevLogTerm, ShouldEqual, 9)
			So(request.CommitIndex, ShouldEqual, 0)
			So(request.Entries, ShouldBeEmpty)
		})

		Convey("we should create appendEntries request", func() {
			Convey("for peers behind the leader", func() {
				request := build(state.log, 1)
				So(request.Term, ShouldEqual, state.currentTerm)
				So(request.PrevLogIndex, ShouldEqual, 0)
				So(request.PrevLogTerm, ShouldEqual, 8)
				So(request.CommitIndex, ShouldEqual, 0)
				So(len(request.Entries), ShouldEqual, 2)
			})

			Convey("for peers with empty log", func() {
				request := build(state.log, 0)
				So(request.Term, ShouldEqual, state.currentTerm)
				So(request.PrevLogIndex, ShouldEqual, -1)
				So(request.PrevLogTerm, ShouldEqual, -1)
				So(request.CommitIndex, ShouldEqual, 0)
				So(len(request.Entries), ShouldEqual, 3)
			})
		})

		peerState := newState(2, time.Millisecond*10, NewLog())
		peerState.log.Append(8, []byte("cmd1"))
		peerState.log.Append(8, []byte("cmd2"))
		peerState.log.Append(9, []byte("cmd3"))

		Convey("we should reject appendEntries with outdated Term", func() {
			peerState.currentTerm = state.currentTerm - 1
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 1)
			response, accepted := state.appendEntriesResponse(request)
			So(accepted, ShouldBeFalse)
			So(response.Term, ShouldEqual, state.currentTerm)
			So(response.Success, ShouldBeFalse)
		})
		Convey("we should accept correct heartbeat", func() {
			peerState.currentTerm = state.currentTerm
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 3)
			So(request.Entries, ShouldBeEmpty)
			response, accepted := state.appendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.currentTerm)
			So(response.Success, ShouldBeTrue)
		})
		Convey("we should request for missed entries", func() {
			peerState.log.Append(9, []byte("cmd4"))
			peerState.currentTerm = state.currentTerm
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 4)
			response, accepted := state.appendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.currentTerm)
			So(response.Success, ShouldBeFalse)
		})
		Convey("we should apply new entries", func() {
			peerState.log.EraseAfter(1)
			peerState.log.Append(10, []byte("cmd10"))
			peerState.log.Append(10, []byte("cmd11"))
			peerState.log.Append(10, []byte("cmd12"))
			peerState.currentTerm = state.currentTerm
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 2)
			response, accepted := state.appendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.currentTerm)
			So(response.Success, ShouldBeTrue)
			So(state.log.Size(), ShouldEqual, 5)
			So(state.log.Get(2).Command, ShouldResemble, []byte("cmd10"))
			So(state.log.Get(3).Command, ShouldResemble, []byte("cmd11"))
			So(state.log.Get(4).Command, ShouldResemble, []byte("cmd12"))
		})
		Convey("we should correctly process appendEntries with negative lastLogIndex", func() {
			peerState.log.EraseAfter(-1)
			peerState.currentTerm = state.currentTerm
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 0)
			response, accepted := state.appendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.currentTerm)
			So(response.Success, ShouldBeTrue)
			So(state.log.Size(), ShouldEqual, 0)
		})
	})
}
