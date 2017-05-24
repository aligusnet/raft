package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestServerSmoke(t *testing.T) {
	Convey("Given Initialized server and simple role", t, func() {
		testRole := RoleHandle(-100)
		server := newServer()
		role := newSimpleRole(server)
		server.roles[testRole] = role
		ctx, cancel := context.WithCancel(context.Background())
		go server.run(ctx, testRole, nil)
		defer cancel()

		Convey("when we send RequestVote request to a server we should receive a valid response", func() {
			_, err := server.RequestVote(nil, nil)
			So(err, ShouldBeNil)
		})

		Convey("when we send AppendEntries request to a server we should receive a valid response", func() {
			_, err := server.AppendEntries(nil, nil)
			So(err, ShouldBeNil)
		})

		Convey("when we send ExecuteCommand request to a server we should receive a valid response", func() {
			_, err := server.ExecuteCommand(nil, nil)
			So(err, ShouldBeNil)
		})

		Convey("server should return correct role", func() {
			So(server.getRole(testRole), ShouldEqual, role)
		})
		Convey("server should return exitRoleInstance on unknown errorHandle", func() {
			So(server.getRole(RoleHandle(-111)), ShouldEqual, exitRoleInstance)
		})
	})
}

type simpleRole struct {
	balancer *Balancer
}

func newSimpleRole(s *Server) *simpleRole {
	return &simpleRole{balancer: s.Balancer}
}

func (r *simpleRole) RunRole(ctx context.Context, state *State) (RoleHandle, *State) {
	for {
		select {
		case requestVote := <-r.balancer.requestVoteCh:
			requestVote.out <- struct {
				*pb.RequestVoteResponse
				error
			}{nil, nil}
		case appendEntries := <-r.balancer.appendEntriesCh:
			appendEntries.out <- struct {
				*pb.AppendEntriesResponse
				error
			}{nil, nil}
		case executeCommand := <-r.balancer.executeCommandCh:
			executeCommand.out <- struct {
				*pb.ExecuteCommandResponse
				error
			}{nil, nil}
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}
