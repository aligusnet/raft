package server

import (
	pb "github.com/aligusnet/raft/raft"
	"github.com/aligusnet/raft/server/state"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"testing"
)

func TestServer(t *testing.T) {
	Convey("Given Initialized server and simple role", t, func() {
		testRole := RoleHandle(-100)
		server := newServer(context.Background())
		role := newSimpleRole(server)
		server.roles[testRole] = role
		go server.run(testRole, nil)
		defer server.Stop()

		Convey("when we send RequestVote request to a server we should receive a valid response", func() {
			_, err := server.dispatcher.RequestVote(nil, nil)
			So(err, ShouldBeNil)
		})

		Convey("when we send AppendEntries request to a server we should receive a valid response", func() {
			_, err := server.dispatcher.AppendEntries(nil, nil)
			So(err, ShouldBeNil)
		})

		Convey("when we send ExecuteCommand request to a server we should receive a valid response", func() {
			_, err := server.dispatcher.ExecuteCommand(nil, nil)
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

func TestServerStop(t *testing.T) {
	Convey("Server's Stop function should close ctx.Done channel", t, func() {
		server := newServer(context.Background())
		server.Stop()
		_, ok := <-server.ctx.Done()
		So(ok, ShouldBeFalse)
		So(server.ctx.Err(), ShouldEqual, context.Canceled)
	})
}

type simpleRole struct {
	dispatcher *Dispatcher
}

func newSimpleRole(s *Server) *simpleRole {
	return &simpleRole{dispatcher: s.dispatcher}
}

func (r *simpleRole) RunRole(ctx context.Context, state *state.State) (RoleHandle, *state.State) {
	for {
		select {
		case requestVote := <-r.dispatcher.requestVoteCh:
			requestVote.send(&pb.RequestVoteResponse{})
		case appendEntries := <-r.dispatcher.appendEntriesCh:
			appendEntries.send(&pb.AppendEntriesResponse{})
		case executeCommand := <-r.dispatcher.executeCommandCh:
			executeCommand.send(&pb.ExecuteCommandResponse{})
		case <-ctx.Done():
			return ExitRoleHandle, state
		}
	}
}
