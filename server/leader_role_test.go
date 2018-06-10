package server

import (
	pb "github.com/aligusnet/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestLeaderRole(t *testing.T) {
	Convey("Replica should exit given cancelation", t, func() {
		leaderState := newTestState(1, 10)

		ctx, cancel := context.WithCancel(context.Background())
		role := newLeaderRole(newDispatcher())
		go func() {
			time.Sleep(2 * time.Millisecond)
			cancel()
		}()
		rh, _ := role.RunRole(ctx, leaderState)
		So(rh, ShouldEqual, ExitRoleHandle)
	})

	Convey("Replica should respond on requestVote", t, func(c C) {
		Convey("giving Term > currentTerm it should grant vote", func() {
			leaderState := newTestState(1, 10)
			leaderState.CurrentTerm = 10
			dispatcher := newDispatcher()

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newTestState(2, 10)
				peerState.CurrentTerm = leaderState.CurrentTerm + 1
				peerState.Log.Append(1, []byte("cmd1"))
				request := peerState.RequestVoteRequest()
				response, err := dispatcher.RequestVote(context.Background(), request)
				c.So(err, ShouldBeNil)
				c.So(response.VoteGranted, ShouldBeTrue)
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(context.Background(), leaderState)
		})

		Convey(" giving Term <= currentTerm it should reject", func() {
			leaderState := newTestState(1, 10)
			leaderState.CurrentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newTestState(2, 10)
				peerState.CurrentTerm = leaderState.CurrentTerm - 1
				peerState.Log.Append(1, []byte("cmd1"))
				request := peerState.RequestVoteRequest()
				response, err := dispatcher.RequestVote(context.Background(), request)
				c.So(response.VoteGranted, ShouldBeFalse)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(ctx, leaderState)
		})
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		Convey("Given stale Term it should reject", func() {
			leaderState := newTestState(1, 10)
			leaderState.CurrentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newTestState(2, 10)
				peerState.CurrentTerm = leaderState.CurrentTerm - 1
				peerState.Log.Append(1, []byte("cmd1"))
				request := peerState.AppendEntriesRequest(1)
				response, err := dispatcher.AppendEntries(context.Background(), request)
				c.So(response.Term, ShouldBeGreaterThan, peerState.CurrentTerm)
				c.So(response.Success, ShouldBeFalse)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(ctx, leaderState)
		})

		Convey("Given newer Term it should become Follower", func() {
			leaderState := newTestState(1, 10)
			leaderState.CurrentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newTestState(2, 10)
				peerState.CurrentTerm = leaderState.CurrentTerm + 1
				peerState.Log.Append(1, []byte("cmd1"))
				request := peerState.AppendEntriesRequest(1)
				response, err := dispatcher.AppendEntries(context.Background(), request)
				c.So(response.Term, ShouldEqual, peerState.CurrentTerm)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			rh, _ := role.RunRole(ctx, leaderState)
			c.So(rh, ShouldEqual, FollowerRoleHandle)
		})
	})
}


func TestLeader_stripToHeartbeet(t *testing.T) {
	Convey("stripToHeartbeet: empty Entries list", t, func() {
		request := &pb.AppendEntriesRequest{
			Term:         4,
			LeaderId:     1,
			PrevLogIndex: -1,
			PrevLogTerm:  -1,
			CommitIndex:  -1,
		}

		stripToHeartbeet(request)
		So(request.PrevLogIndex, ShouldEqual, -1)
		So(request.PrevLogTerm, ShouldEqual, -1)
		So(request.Entries, ShouldBeEmpty)
	})

	Convey("stripToHeartbeet: non-empty Entries list", t, func() {
		request := &pb.AppendEntriesRequest{
			Term:         4,
			LeaderId:     1,
			PrevLogIndex: -1,
			PrevLogTerm:  -1,
			CommitIndex:  -1,
			Entries:      []*pb.LogEntry{{Term: 2, Index: 0}, {Term: 3, Index: 1}, {Term: 4, Index: 2}},
		}

		stripToHeartbeet(request)
		So(request.PrevLogIndex, ShouldEqual, 2)
		So(request.PrevLogTerm, ShouldEqual, 4)
		So(request.Entries, ShouldBeEmpty)
	})
}
