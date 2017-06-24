package server

import (
	"fmt"
	"github.com/alexander-ignatyev/raft/server/log"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/server/state"
	. "github.com/smartystreets/goconvey/convey"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"testing"
	"time"
)

type clientMock struct {
	requestVote    []*pb.RequestVoteResponse
	appendEntries  []*pb.AppendEntriesResponse
	executeCommand []*pb.ExecuteCommandResponse

	requestVoteIndex    int
	appendEntriesIndex  int
	executeCommandIndex int
}

func TestPeer_AppendEntriesResponse(t *testing.T) {
	peer := &Replica{
		nextIndex: 2,
	}

	state := &state.State{
		CurrentTerm: 4,
		Log:         log.New(),
	}
	state.Log.Append(4, []byte("command1"))

	Convey("Peer should correctly respond on AppendEntriesResponse", t, func() {
		Convey("it should do nothing if the initial request succeeded", func() {
			response := &pb.AppendEntriesResponse{
				Term:    state.CurrentTerm,
				Success: true,
			}

			request, demoted := peer.appendEntriesRequest(state, response)
			So(request, ShouldBeNil)
			So(demoted, ShouldBeFalse)
		})

		Convey("it should advise the leader to demote if Term > currentTerm", func() {
			response := &pb.AppendEntriesResponse{
				Term:    state.CurrentTerm + 1,
				Success: false,
			}

			request, demoted := peer.appendEntriesRequest(state, response)
			So(request, ShouldBeNil)
			So(demoted, ShouldBeTrue)
		})

		Convey("it should advise the leader to resend data if the peer requires it", func() {
			response := &pb.AppendEntriesResponse{
				Term:    state.CurrentTerm,
				Success: false,
			}

			request, demoted := peer.appendEntriesRequest(state, response)
			So(request, ShouldNotBeNil)
			So(demoted, ShouldBeFalse)
			So(peer.nextIndex, ShouldBeLessThan, 2)
		})
	})
}

var replicaTestTimeout time.Duration = 5 * time.Millisecond

func newRequestVoteClient(votes ...bool) *clientMock {
	var requestVote []*pb.RequestVoteResponse
	for _, vote := range votes {
		requestVote = append(requestVote, &pb.RequestVoteResponse{VoteGranted: vote})
	}
	return &clientMock{requestVote: requestVote}
}

func newAppendEntriesClient(success ...bool) *clientMock {
	var appendEntries []*pb.AppendEntriesResponse
	for _, s := range success {
		appendEntries = append(appendEntries, &pb.AppendEntriesResponse{Success: s})
	}
	return &clientMock{appendEntries: appendEntries}
}

func (c *clientMock) RequestVote(ctx context.Context,
	in *pb.RequestVoteRequest,
	opts ...grpc.CallOption) (*pb.RequestVoteResponse, error) {
	time.Sleep(replicaTestTimeout) // processing time
	if len(c.requestVote) > c.requestVoteIndex {
		response := c.requestVote[c.requestVoteIndex]
		c.requestVoteIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}

func (c *clientMock) AppendEntries(ctx context.Context,
	in *pb.AppendEntriesRequest,
	opts ...grpc.CallOption) (*pb.AppendEntriesResponse, error) {
	time.Sleep(replicaTestTimeout) // processing time
	if len(c.appendEntries) > c.appendEntriesIndex {
		response := c.appendEntries[c.appendEntriesIndex]
		response.NextLogIndex = in.PrevLogIndex + int64(len(in.Entries))
		c.appendEntriesIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}

func (c *clientMock) ExecuteCommand(ctx context.Context,
	in *pb.ExecuteCommandRequest,
	opts ...grpc.CallOption) (*pb.ExecuteCommandResponse, error) {
	time.Sleep(replicaTestTimeout) // processing time
	if len(c.executeCommand) > c.executeCommandIndex {
		response := c.executeCommand[c.executeCommandIndex]
		c.executeCommandIndex++
		return response, nil
	} else {
		return nil, fmt.Errorf("Nothing to respond")
	}
}
