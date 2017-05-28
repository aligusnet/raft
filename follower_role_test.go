package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestFollowerRole(t *testing.T) {
	Convey("Replica should respond on requestVote", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		dispatcher := newDispatcher()

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

		role := &FollowerRole{dispatcher: dispatcher}
		rh, _ := role.RunRole(context.Background(), state)
		c.So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		dispatcher := newDispatcher()
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newState(2, time.Millisecond*10, NewLog())
			peerState.currentTerm = state.currentTerm + 1
			peerState.log.Append(1, []byte("cmd1"))
			request := peerState.appendEntriesRequestBuilder()(peerState.log, 1)
			response, err := dispatcher.AppendEntries(context.Background(), request)
			c.So(response, ShouldNotBeNil)
			c.So(err, ShouldBeNil)
			cancel()
		}()

		role := &FollowerRole{dispatcher: dispatcher}
		role.RunRole(ctx, state)
		c.So(state.currentLeaderId, ShouldEqual, 2)
	})

	Convey("Replica should respond on executeCommand", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		dispatcher := newDispatcher()
		state.currentLeaderId = 2
		state.addresses[2] = "address737"
		go func() {
			time.Sleep(2 * time.Millisecond)
			request := &pb.ExecuteCommandRequest{[]byte("Command1")}
			response, err := dispatcher.ExecuteCommand(context.Background(), request)
			c.So(response.Success, ShouldBeFalse)
			c.So(response.ServerAddress, ShouldEqual, "address737")
			c.So(err, ShouldBeNil)
		}()

		role := &FollowerRole{dispatcher: dispatcher}
		role.RunRole(context.Background(), state)
	})

	Convey("Replica should become Candidate given deadline passes", t, func() {
		state := newState(1, time.Millisecond, NewLog())

		role := &FollowerRole{dispatcher: newDispatcher()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should exit given cancelation", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())

		ctx, cancel := context.WithCancel(context.Background())
		role := &FollowerRole{dispatcher: newDispatcher()}
		go func() {
			time.Sleep(2 * time.Millisecond)
			cancel()
		}()
		rh, _ := role.RunRole(ctx, state)
		So(rh, ShouldEqual, ExitRoleHandle)
	})
}
