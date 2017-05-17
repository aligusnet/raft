package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestServerSmoke(t *testing.T) {
	Convey("Given Initialized server and simple role", t, func() {
		server := newServer()
		role := newSimpleRole(server)
		server.currentRole = role
		go server.currentRole.RunRole()

		Convey("when we send RequestVote request to a server we should receive a valid response", func() {
			_, err := server.RequestVote(nil, nil)
			So(err, ShouldEqual, nil)
		})

		Convey("when we send AppendEntries request to a server we should receive a valid response", func() {
			_, err := server.AppendEntries(nil, nil)
			So(err, ShouldEqual, nil)
		})

		Convey("when we send ExecuteCommand request to a server we should receive a valid response", func() {
			_, err := server.ExecuteCommand(nil, nil)
			So(err, ShouldEqual, nil)
		})

		role.quitCh <- true
	})
}

type simpleRole struct {
	requestVoteCh    chan *requestVoteMessage
	appendEntriesCh  chan *appendEntriesMessage
	executeCommandCh chan *executeCommandMessage
	quitCh           chan bool
}

func newSimpleRole(s *Server) *simpleRole {
	return &simpleRole{quitCh: make(chan bool),
		requestVoteCh:    s.requestVoteCh,
		appendEntriesCh:  s.appendEntriesCh,
		executeCommandCh: s.executeCommandCh,
	}
}

func (r *simpleRole) RunRole() RoleHandle {
	for {
		select {
		case requestVote := <-r.requestVoteCh:
			requestVote.out <- struct {
				*pb.RequestVoteResponse
				error
			}{nil, nil}
		case appendEntries := <-r.appendEntriesCh:
			appendEntries.out <- struct {
				*pb.AppendEntriesResponse
				error
			}{nil, nil}
		case executeCommand := <-r.executeCommandCh:
			executeCommand.out <- struct {
				*pb.ExecuteCommandResponse
				error
			}{nil, nil}
		case <-r.quitCh:
			return LeaderRole
		}
	}
}
