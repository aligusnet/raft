package raft

import (
	"container/list"
	"github.com/alexander-ignatyev/raft/log"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/alexander-ignatyev/raft/state"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

func Run(ctx context.Context, id int64, timeout time.Duration, log log.Log, endPointAddresses map[int64]string, machine state.StateMachine) {
	server := newServer(ctx)
	lis, err := net.Listen("tcp", endPointAddresses[id])
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}

	replicas := make(map[int64]*Replica)
	for replicaId, address := range endPointAddresses {
		if replicaId != id {
			replicas[replicaId] = newReplica(ctx, replicaId, address)
		}
	}

	server.roles[LeaderRoleHandle] = &LeaderRole{
		dispatcher: server.dispatcher,
		replicas:   replicas,
		waitlist:   list.New(),
	}
	server.roles[FollowerRoleHandle] = &FollowerRole{dispatcher: server.dispatcher}
	server.roles[CandidateRoleHandle] = &CandidateRole{
		dispatcher: server.dispatcher,
		replicas:   replicas,
	}

	state := state.New(id, timeout, endPointAddresses, log, machine)
	go server.run(FollowerRoleHandle, state)

	gs := grpc.NewServer()
	pb.RegisterRaftServer(gs, server.dispatcher)
	// Register reflection service on gRPC server.
	reflection.Register(gs)

	go func() {
		select {
		case <-ctx.Done():
			gs.GracefulStop()
		}
	}()

	if err := gs.Serve(lis); err != nil {
		if err == grpc.ErrServerStopped {
			glog.Info("The server has been stopped")
		} else {
			glog.Errorf("failed to serve: %v", err)
		}
	}
}

func newReplica(ctx context.Context, id int64, address string) *Replica {
	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure())
	if err != nil {
		glog.Errorf("[Replica %v] failed to connect to %v", id, address)
	} else {
		glog.Infof("[Replica %v] connected to %v", id, address)
	}
	return &Replica{
		id:      id,
		address: address,
		client:  pb.NewRaftClient(conn),
	}
}
