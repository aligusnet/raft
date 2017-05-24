package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestCandidateRole(t *testing.T) {
	Convey("Replica should be elected given at least half of votes", t, func() {
		state := newState(1, time.Millisecond*10)
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, channels: newChannelSet()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should response on requestVote", t, func(c C) {
		state := newState(1, time.Millisecond*10)
		channels := newChannelSet()
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		go func() {
			time.Sleep(2*time.Millisecond)
			peerState := newState(2, time.Millisecond*10)
			peerState.currentTerm=state.currentTerm+1
			peerState.log.Append(1, []byte("cmd1"))
			request := peerState.requestVoteRequest()
			resultCh := make(chan struct {
				*pb.RequestVoteResponse
				error
			}, 1)
			msg := &requestVoteMessage{context.Background(), request, resultCh}
			channels.requestVoteCh <- msg
			result := <-resultCh
			c.So(result.VoteGranted, ShouldBeTrue)
			c.So(result.error, ShouldBeNil)
		}()

		role := &CandidateRole{replicas: replicas, channels: channels}
		rh, _ := role.RunRole(context.Background(), state)
		c.So(rh, ShouldEqual, CandidateRoleHandle)
	})


	Convey("Replica should not be elected given less than half of votes", t, func() {
		state := newState(1, time.Millisecond*10)
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, channels: newChannelSet()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given deadline passes", t, func() {
		state := newState(1, time.Millisecond)
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(true)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, channels: newChannelSet()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, CandidateRoleHandle)
	})

	Convey("Replica should not be elected given cancelation", t, func() {
		state := newState(1, time.Millisecond*10)
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(true)
			replicas = append(replicas, &Replica{client: client})
		}

		ctx, cancel := context.WithCancel(context.Background())
		role := &CandidateRole{replicas: replicas, channels: newChannelSet()}
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
