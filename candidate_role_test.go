package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestCandidateRole(t *testing.T) {
	Convey("Replica should be elected given at least half of votes", t, func() {
		state := newState(1, time.Millisecond*30, NewLog())
		state.votedFor = state.id
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should respond on requestVote", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newState(2, time.Millisecond*10, NewLog())
			peerState.currentTerm = state.currentTerm + 1
			peerState.log.Append(1, []byte("cmd1"))
			request := peerState.requestVoteRequest()
			response, err := dispatcher.RequestVote(context.Background(), request)
			c.So(response.VoteGranted, ShouldBeTrue)
			c.So(err, ShouldBeNil)
		}()

		rh, _ := role.RunRole(context.Background(), state)
		c.So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		state := newState(1, time.Millisecond*30, NewLog())
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newState(2, time.Millisecond*10, NewLog())
			peerState.currentTerm = state.currentTerm + 1
			peerState.log.Append(1, []byte("cmd1"))
			request := peerState.appendEntriesRequest(1)
			response, err := dispatcher.AppendEntries(context.Background(), request)
			c.So(response, ShouldNotBeNil)
			c.So(err, ShouldBeNil)
		}()

		rh, _ := role.RunRole(context.Background(), state)
		c.So(rh, ShouldEqual, FollowerRoleHandle)
	})

	Convey("Replica should respond on executeCommand", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		state.currentLeaderId = 2
		state.addresses[2] = "address"
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			request := &pb.ExecuteCommandRequest{[]byte("Command1")}
			response, err := dispatcher.ExecuteCommand(context.Background(), request)
			c.So(response, ShouldBeNil)
			c.So(err, ShouldNotBeNil)
		}()

		role.RunRole(context.Background(), state)
	})

	Convey("Replica should not be elected given less than half of votes", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i == 0)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given deadline passes", t, func() {
		state := newState(1, time.Millisecond, NewLog())
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(true)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given cancelation", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(true)
			role.replicas[i] = &Replica{client: client}
		}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(2 * time.Millisecond)
			cancel()
		}()
		rh, _ := role.RunRole(ctx, state)
		So(rh, ShouldEqual, ExitRoleHandle)
	})

	Convey("Candidate requires at least half of votes to be elected", t, func() {
		So(requiredVotes(4), ShouldEqual, 3)
		So(requiredVotes(5), ShouldEqual, 3)
		So(requiredVotes(6), ShouldEqual, 4)
	})
}
