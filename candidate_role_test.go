package raft

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCandidateRole(t *testing.T) {
	Convey("Replica should be elected given at least half of votes", t, func() {
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i%2 == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, state: &State{}}
		So(role.RunRole(), ShouldEqual, LeaderRoleHandle)
	})

	Convey("Replica should not be elected given less than half of votes", t, func() {
		var replicas []*Replica
		for i := 0; i < 4; i++ {
			client := newRequestVoteClient(i == 0)
			replicas = append(replicas, &Replica{client: client})
		}

		role := &CandidateRole{replicas: replicas, state: &State{}}
		So(role.RunRole(), ShouldEqual, CandidateRoleHandle)
	})

	Convey("Candidate requires at least half of votes to be elected", t, func() {
		So(requiredVotes(4), ShouldEqual, 3)
		So(requiredVotes(5), ShouldEqual, 3)
		So(requiredVotes(6), ShouldEqual, 4)
	})
}
