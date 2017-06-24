package server

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestCandidateRole(t *testing.T) {
	Convey("Replica should be elected given at least half of votes", t, func() {
		candidateState := newTestState(1, 30)
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), candidateState)
		So(rh, ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should respond on requestVote", t, func(c C) {
		candidateState := newTestState(1, 10)
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newTestState(2, 10)
			peerState.CurrentTerm = candidateState.CurrentTerm
			peerState.Log.Append(1, []byte("cmd1"))
			request := peerState.RequestVoteRequest()
			response, err := dispatcher.RequestVote(context.Background(), request)
			c.So(response.VoteGranted, ShouldBeTrue)
			c.So(err, ShouldBeNil)
		}()

		rh, _ := role.RunRole(context.Background(), candidateState)
		c.So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should step out on requestVote with bigger term", t, func(c C) {
		candidateState := newTestState(1, 10)
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newTestState(2, 10)
			peerState.CurrentTerm = candidateState.CurrentTerm+1
			peerState.Log.Append(1, []byte("cmd1"))
			request := peerState.RequestVoteRequest()
			response, err := dispatcher.RequestVote(context.Background(), request)
			c.So(response.VoteGranted, ShouldBeTrue)
			c.So(err, ShouldBeNil)
		}()

		rh, _ := role.RunRole(context.Background(), candidateState)
		c.So(rh, ShouldEqual, FollowerRoleHandle)
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		candidateState := newTestState(1, 10)
		dispatcher := newDispatcher()
		role := newCandidateRole(dispatcher)
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			role.replicas[i] = &Replica{client: client}
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newTestState(2, 10)
			peerState.CurrentTerm = candidateState.CurrentTerm + 1
			peerState.Log.Append(1, []byte("cmd1"))
			request := peerState.AppendEntriesRequest(1)
			response, err := dispatcher.AppendEntries(context.Background(), request)
			c.So(response, ShouldNotBeNil)
			c.So(err, ShouldBeNil)
		}()

		rh, _ := role.RunRole(context.Background(), candidateState)
		c.So(rh, ShouldEqual, FollowerRoleHandle)
	})

	Convey("Replica should respond on executeCommand", t, func(c C) {
		addresses := make(map[int64]string)
		addresses[2] = "address"
		candidateState := newTestStateWithAddresses(1, 10, addresses)
		candidateState.CurrentLeaderId = 2
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

		role.RunRole(context.Background(), candidateState)
	})

	Convey("Replica should not be elected given less than half of votes", t, func() {
		candidateState := newTestState(1, 10)
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(i == 0)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), candidateState)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given deadline passes", t, func() {
		candidateState := newTestState(1, 1)
		role := newCandidateRole(newDispatcher())
		for i := int64(0); i < 4; i++ {
			client := newRequestVoteClient(true)
			role.replicas[i] = &Replica{client: client}
		}

		rh, _ := role.RunRole(context.Background(), candidateState)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given cancelation", t, func() {
		candidateState := newTestState(1, 10)
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
		rh, _ := role.RunRole(ctx, candidateState)
		So(rh, ShouldEqual, ExitRoleHandle)
	})

	Convey("Candidate requires at least half of votes to be elected", t, func() {
		So(requiredVotes(4), ShouldEqual, 3)
		So(requiredVotes(5), ShouldEqual, 3)
		So(requiredVotes(6), ShouldEqual, 4)
	})
}
