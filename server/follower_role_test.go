package server

import (
	pb "github.com/aligusnet/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestFollowerRole(t *testing.T) {
	Convey("Replica should respond on requestVote", t, func(c C) {
		followerState := newTestState(1, 10)
		dispatcher := newDispatcher()

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newTestState(2, 10)
			peerState.CurrentTerm = followerState.CurrentTerm + 1
			peerState.Log.Append(1, []byte("cmd1"))
			request := peerState.RequestVoteRequest()
			response, err := dispatcher.RequestVote(context.Background(), request)
			c.So(response.VoteGranted, ShouldBeTrue)
			c.So(err, ShouldBeNil)
		}()

		role := &FollowerRole{dispatcher: dispatcher}
		rh, _ := role.RunRole(context.Background(), followerState)
		c.So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		followerState := newTestState(1, 10)
		dispatcher := newDispatcher()
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newTestState(2, 10)
			peerState.CurrentTerm = followerState.CurrentTerm + 1
			peerState.Log.Append(1, []byte("cmd1"))
			request := peerState.AppendEntriesRequest(1)
			response, err := dispatcher.AppendEntries(context.Background(), request)
			c.So(response, ShouldNotBeNil)
			c.So(err, ShouldBeNil)
			cancel()
		}()

		role := &FollowerRole{dispatcher: dispatcher}
		role.RunRole(ctx, followerState)
		c.So(followerState.CurrentLeaderId, ShouldEqual, 2)
	})

	Convey("Replica should respond on executeCommand", t, func(c C) {
		addresses := make(map[int64]string)
		addresses[2] = "address737"
		followerState := newTestStateWithAddresses(1, 10, addresses)
		dispatcher := newDispatcher()
		followerState.CurrentLeaderId = 2
		go func() {
			time.Sleep(2 * time.Millisecond)
			request := &pb.ExecuteCommandRequest{[]byte("Command1")}
			response, err := dispatcher.ExecuteCommand(context.Background(), request)
			c.So(response.Success, ShouldBeFalse)
			c.So(response.ServerAddress, ShouldEqual, "address737")
			c.So(err, ShouldBeNil)
		}()

		role := &FollowerRole{dispatcher: dispatcher}
		role.RunRole(context.Background(), followerState)
	})

	Convey("Replica should become Candidate given deadline passes", t, func() {
		followerState := newTestState(1, 10)

		role := &FollowerRole{dispatcher: newDispatcher()}
		rh, _ := role.RunRole(context.Background(), followerState)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should exit given cancelation", t, func() {
		state := newTestState(1, 10)

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
