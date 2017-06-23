package state

import (
	"github.com/alexander-ignatyev/raft/log"
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestState(t *testing.T) {
	Convey("Given just created state", t, func() {
		s := newTestState(1, 10)

		Convey("LastLogIndexAndTerm", func() {
			lastLogIndex, lastLogTerm := s.LastLogIndexAndTerm()
			So(lastLogIndex, ShouldEqual, -1)
			So(lastLogTerm, ShouldEqual, -1)
		})
	})
	Convey("Given initialized aged state", t, func() {
		s := newTestState(1, 10)
		s.CurrentTerm = 10
		s.Log.Append(8, []byte("cmd1"))
		s.Log.Append(8, []byte("cmd2"))
		s.Log.Append(9, []byte("cmd3"))

		Convey("LastLogIndexAndTerm", func() {
			lastLogIndex, lastLogTerm := s.LastLogIndexAndTerm()
			So(lastLogIndex, ShouldEqual, 2)
			So(lastLogTerm, ShouldEqual, 9)
		})
	})
}

func TestState_LeaderAddress(t *testing.T) {
	Convey("No leader", t, func() {
		s := newTestState(1, 10)
		So(s.LeaderAddress(), ShouldBeEmpty)
	})

	Convey("No leader's address", t, func() {
		state := newTestState(1, 10)
		state.CurrentLeaderId = 2
		So(state.LeaderAddress(), ShouldBeEmpty)
	})

	Convey("We have leader and its address", t, func() {
		state := newTestState(1, 10)
		state.CurrentLeaderId = 2
		state.addresses[2] = "address898"
		So(state.LeaderAddress(), ShouldEqual, "address898")
	})
}

func TestState_RequestVote(t *testing.T) {
	Convey("Given initialized non-voted state", t, func() {
		state := newTestState(1, 10)
		state.CurrentTerm = 10
		state.Log.Append(8, []byte("cmd1"))
		state.Log.Append(8, []byte("cmd2"))
		state.Log.Append(9, []byte("cmd3"))

		Convey("we should create correct requestVote request", func() {
			lastLogIndex, lastLogTerm := state.LastLogIndexAndTerm()
			request := state.RequestVoteRequest()
			So(request.CandidateId, ShouldEqual, state.id)
			So(request.Term, ShouldEqual, state.CurrentTerm)
			So(request.LastLogIndex, ShouldEqual, lastLogIndex)
			So(request.LastLogTerm, ShouldEqual, lastLogTerm)
		})

		request := &pb.RequestVoteRequest{CandidateId: 10}
		Convey("Given requestVore request with in.Term > CurrentTerm", func() {
			request.Term = state.CurrentTerm + 1
			Convey("and in.LastLogTerm < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 4
				request.LastLogTerm = 8
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
			})
			Convey("and in.LastLogIndex > lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 4
				request.LastLogTerm = 9
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				request.LastLogTerm = 9
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 1
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldBeLessThanOrEqualTo, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})

		Convey("Given requestVore request with in.Term = CurrentTerm", func() {
			request.Term = state.CurrentTerm
			Convey("and in.LastLogIndex > lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 4
				request.LastLogTerm = 9
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex == lastLogIndex we should vote for the candidate", func() {
				request.LastLogIndex = 2
				request.LastLogTerm = 9
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeTrue)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldEqual, request.CandidateId)
			})
			Convey("and in.LastLogIndex < lastLogIndex we should not vote for the candidate", func() {
				request.LastLogIndex = 1
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldEqual, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})

		Convey("Given requestVore request with in.Term < CurrentTerm", func() {
			request.Term = state.CurrentTerm - 1
			Convey("we should not vote for the candidate", func() {
				request.LastLogIndex = 4
				response := state.RequestVoteResponse(request)
				So(response.VoteGranted, ShouldBeFalse)
				So(response.Term, ShouldBeGreaterThan, request.Term)
				So(state.votedFor, ShouldNotEqual, request.CandidateId)
			})
		})
	})

	Convey("Given initialized voted state", t, func() {
		state := newTestState(1, 10)
		state.CurrentTerm = 10
		state.Log.Append(8, []byte("cmd1"))
		state.Log.Append(8, []byte("cmd2"))
		state.Log.Append(9, []byte("cmd3"))
		state.votedFor = 5

		Convey("We should not vote for the candidate", func() {
			request := &pb.RequestVoteRequest{CandidateId: 10}
			request.Term = state.CurrentTerm + 1
			request.LastLogIndex = 4
			response := state.RequestVoteResponse(request)
			So(response.VoteGranted, ShouldBeFalse)
		})
	})
}

func TestState_AppendEntries(t *testing.T) {
	Convey("Given initialized state", t, func() {
		state := newTestState(1, 10)
		state.CurrentTerm = 10
		state.Log.Append(8, []byte("cmd1"))
		state.Log.Append(8, []byte("cmd2"))
		state.Log.Append(9, []byte("cmd3"))

		Convey("we should create hearbeat message", func() {
			request := state.AppendEntriesRequest(3)
			So(request.Term, ShouldEqual, state.CurrentTerm)
			So(request.PrevLogIndex, ShouldEqual, 2)
			So(request.PrevLogTerm, ShouldEqual, 9)
			So(request.CommitIndex, ShouldEqual, -1)
			So(request.Entries, ShouldBeEmpty)
		})

		Convey("we should create appendEntries request", func() {
			Convey("for peers behind the leader", func() {
				request := state.AppendEntriesRequest(1)
				So(request.Term, ShouldEqual, state.CurrentTerm)
				So(request.PrevLogIndex, ShouldEqual, 0)
				So(request.PrevLogTerm, ShouldEqual, 8)
				So(request.CommitIndex, ShouldEqual, -1)
				So(len(request.Entries), ShouldEqual, 2)
			})

			Convey("for peers with empty Log", func() {
				request := state.AppendEntriesRequest(0)
				So(request.Term, ShouldEqual, state.CurrentTerm)
				So(request.PrevLogIndex, ShouldEqual, -1)
				So(request.PrevLogTerm, ShouldEqual, -1)
				So(request.CommitIndex, ShouldEqual, -1)
				So(len(request.Entries), ShouldEqual, 3)
			})
		})

		peerState := newTestState(1, 10)
		peerState.Log.Append(8, []byte("cmd1"))
		peerState.Log.Append(8, []byte("cmd2"))
		peerState.Log.Append(9, []byte("cmd3"))

		Convey("we should reject appendEntries with outdated Term", func() {
			peerState.CurrentTerm = state.CurrentTerm - 1
			request := peerState.AppendEntriesRequest(1)
			response, accepted := state.AppendEntriesResponse(request)
			So(accepted, ShouldBeFalse)
			So(response.Term, ShouldEqual, state.CurrentTerm)
			So(response.Success, ShouldBeFalse)
		})
		Convey("we should accept correct heartbeat", func() {
			peerState.CurrentTerm = state.CurrentTerm
			request := peerState.AppendEntriesRequest(3)
			So(request.Entries, ShouldBeEmpty)
			response, accepted := state.AppendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.CurrentTerm)
			So(response.Success, ShouldBeTrue)
		})
		Convey("we should request for missed entries", func() {
			peerState.Log.Append(9, []byte("cmd4"))
			peerState.CurrentTerm = state.CurrentTerm
			request := peerState.AppendEntriesRequest(4)
			response, accepted := state.AppendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.CurrentTerm)
			So(response.Success, ShouldBeFalse)
		})
		Convey("we should apply new entries", func() {
			peerState.Log.EraseAfter(1)
			peerState.Log.Append(10, []byte("cmd10"))
			peerState.Log.Append(10, []byte("cmd11"))
			peerState.Log.Append(10, []byte("cmd12"))
			peerState.CurrentTerm = state.CurrentTerm
			request := peerState.AppendEntriesRequest(2)
			response, accepted := state.AppendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.CurrentTerm)
			So(response.Success, ShouldBeTrue)
			So(state.Log.Size(), ShouldEqual, 5)
			So(state.Log.Get(2).Command, ShouldResemble, []byte("cmd10"))
			So(state.Log.Get(3).Command, ShouldResemble, []byte("cmd11"))
			So(state.Log.Get(4).Command, ShouldResemble, []byte("cmd12"))
		})
		Convey("we should correctly process appendEntries with negative lastLogIndex", func() {
			peerState.Log.EraseAfter(-1)
			peerState.CurrentTerm = state.CurrentTerm
			request := peerState.AppendEntriesRequest(0)
			response, accepted := state.AppendEntriesResponse(request)
			So(accepted, ShouldBeTrue)
			So(response.Term, ShouldEqual, state.CurrentTerm)
			So(response.Success, ShouldBeTrue)
			So(state.Log.Size(), ShouldEqual, 0)
		})
	})
}

func newTestState(id int64, timeout int) *State {
	return New(1,
		time.Millisecond*time.Duration(timeout),
		make(map[int64]string),
		log.New(),
		&noOpStateMachine{})
}

type noOpStateMachine struct{}

func (m *noOpStateMachine) ExecuteCommand(command []byte) ([]byte, error) {
	return []byte("hi"), nil
}
func (m *noOpStateMachine) CommandToString(command []byte) (string, error) {
	return "hi", nil
}
func (m *noOpStateMachine) Debug() string {
	return "hi"
}
