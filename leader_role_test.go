package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
	"time"
)

func TestLeaderRole(t *testing.T) {
	Convey("Replica should exit given cancelation", t, func() {
		state := newState(1, time.Millisecond*10, NewLog())

		ctx, cancel := context.WithCancel(context.Background())
		role := newLeaderRole(newDispatcher())
		go func() {
			time.Sleep(2 * time.Millisecond)
			cancel()
		}()
		rh, _ := role.RunRole(ctx, state)
		So(rh, ShouldEqual, ExitRoleHandle)
	})

	Convey("Replica should respond on requestVote", t, func(c C) {
		Convey("giving Term > currentTerm it should grant vote", func() {
			state := newState(1, time.Millisecond*10, NewLog())
			state.currentTerm = 10
			dispatcher := newDispatcher()

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newState(2, time.Millisecond*10, NewLog())
				peerState.currentTerm = state.currentTerm + 1
				peerState.log.Append(1, []byte("cmd1"))
				request := peerState.requestVoteRequest()
				response, err := dispatcher.RequestVote(context.Background(), request)
				c.So(err, ShouldBeNil)
				c.So(response.VoteGranted, ShouldBeTrue)
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(context.Background(), state)
		})

		Convey(" giving Term <= currentTerm it should reject", func() {
			state := newState(1, time.Millisecond*10, NewLog())
			state.currentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newState(2, time.Millisecond*10, NewLog())
				peerState.currentTerm = state.currentTerm - 1
				peerState.log.Append(1, []byte("cmd1"))
				request := peerState.requestVoteRequest()
				response, err := dispatcher.RequestVote(context.Background(), request)
				c.So(response.VoteGranted, ShouldBeFalse)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(ctx, state)
		})
	})

	Convey("Replica should respond on appendEntries", t, func(c C) {
		Convey("Given stale Term it should reject", func() {
			state := newState(1, time.Millisecond*10, NewLog())
			state.currentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newState(2, time.Millisecond*10, NewLog())
				peerState.currentTerm = state.currentTerm - 1
				peerState.log.Append(1, []byte("cmd1"))
				request := peerState.appendEntriesRequest(1)
				response, err := dispatcher.AppendEntries(context.Background(), request)
				c.So(response.Term, ShouldBeGreaterThan, peerState.currentTerm)
				c.So(response.Success, ShouldBeFalse)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			role.RunRole(ctx, state)
		})

		Convey("Given newer Term it should become Follower", func() {
			state := newState(1, time.Millisecond*10, NewLog())
			state.currentTerm = 10
			dispatcher := newDispatcher()
			ctx, cancel := context.WithCancel(context.Background())

			go func() {
				time.Sleep(2 * time.Millisecond)
				peerState := newState(2, time.Millisecond*10, NewLog())
				peerState.currentTerm = state.currentTerm + 1
				peerState.log.Append(1, []byte("cmd1"))
				request := peerState.appendEntriesRequest(1)
				response, err := dispatcher.AppendEntries(context.Background(), request)
				c.So(response.Term, ShouldEqual, peerState.currentTerm)
				c.So(err, ShouldBeNil)
				cancel()
			}()

			role := newLeaderRole(dispatcher)
			rh, _ := role.RunRole(ctx, state)
			c.So(rh, ShouldEqual, FollowerRoleHandle)
		})
	})

	Convey("Replica should respond on executeCommand", t, func(c C) {
		state := newState(1, time.Millisecond*10, NewLog())
		dispatcher := newDispatcher()
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(2 * time.Millisecond)
			request := &pb.ExecuteCommandRequest{[]byte("Command1")}
			response, err := dispatcher.ExecuteCommand(context.Background(), request)
			c.So(response.Success, ShouldBeTrue)
			c.So(err, ShouldBeNil)
			cancel()
		}()

		role := newLeaderRole(dispatcher)
		role.RunRole(ctx, state)
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
			Entries:      []*pb.LogEntry{nil, nil, nil},
		}

		stripToHeartbeet(request)
		So(request.PrevLogIndex, ShouldEqual, 2)
		So(request.PrevLogTerm, ShouldEqual, 4)
		So(request.Entries, ShouldBeEmpty)
	})
}
