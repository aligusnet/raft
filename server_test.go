package raft

import (
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
		go server.run(testRole, nil)
		defer server.Stop()

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
	channels *channelSet
}

func newSimpleRole(s *Server) *simpleRole {
	return &simpleRole{channels: s.channels}
}

func (r *simpleRole) RunRole(state *State) (RoleHandle, *State) {
	for {
		select {
		case requestVote := <-r.channels.requestVoteCh:
			requestVote.out <- struct {
				*pb.RequestVoteResponse
				error
			}{nil, nil}
		case appendEntries := <-r.channels.appendEntriesCh:
			appendEntries.out <- struct {
				*pb.AppendEntriesResponse
				error
			}{nil, nil}
		case executeCommand := <-r.channels.executeCommandCh:
			executeCommand.out <- struct {
				*pb.ExecuteCommandResponse
				error
			}{nil, nil}
		case <-r.channels.quitCh:
			return ExitRoleHandle, state
		}
	}
}
