package raft

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestCandidateRole(t *testing.T) {
	state := newState(1, time.Millisecond*10)
	Convey("Replica should be elected given at least half of votes", t, func() {
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, channels: newChannelSet()}
		rh, _ := role.RunRole(context.Background(), state)
		So(rh, ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should not be elected given less than half of votes", t, func() {
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
		state.timeout = time.Millisecond
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
		state.timeout = 10 * time.Millisecond
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
