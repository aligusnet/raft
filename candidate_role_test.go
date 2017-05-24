package raft

import (
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

		role := &CandidateRole{replicas: replicas, balancer: newBalancer()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should response on requestVote", t, func(c C) {
		state := newState(1, time.Millisecond*10)
		balancer := newBalancer()
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		go func() {
			time.Sleep(2 * time.Millisecond)
			peerState := newState(2, time.Millisecond*10)
			peerState.currentTerm = state.currentTerm + 1
			peerState.log.Append(1, []byte("cmd1"))
			request := peerState.requestVoteRequest()
			response, err := balancer.RequestVote(context.Background(), request)
			c.So(response.VoteGranted, ShouldBeTrue)
			c.So(err, ShouldBeNil)
		}()

		role := &CandidateRole{replicas: replicas, balancer: balancer}
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

		role := &CandidateRole{replicas: replicas, balancer: newBalancer()}
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

		role := &CandidateRole{replicas: replicas, balancer: newBalancer()}
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
		role := &CandidateRole{replicas: replicas, balancer: newBalancer()}
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
