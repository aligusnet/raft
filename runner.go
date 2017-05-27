package raft

import (
	"context"
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"time"
)

func Run(ctx context.Context, id int64, timeout time.Duration, endPointAddresses map[int64]string) {
	server := newServer(ctx)
	lis, err := net.Listen("tcp", endPointAddresses[id])
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}

	replicas := make(map[int64]*Replica)
	for replicaId, address := range endPointAddresses {
		if replicaId != id {
			replicas[replicaId] = newReplica(replicaId, address)
		}
	}

	server.roles[LeaderRoleHandle] = &LeaderRole{
		dispatcher: server.dispatcher,
		replicas:   replicas,
	}
	server.roles[FollowerRoleHandle] = &FollowerRole{dispatcher: server.dispatcher}
	server.roles[CandidateRoleHandle] = &CandidateRole{
		dispatcher: server.dispatcher,
		replicas:   replicas,
	}

	go server.run(FollowerRoleHandle, newState(id, timeout))

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
	glog.Flush()
}

func newReplica(id int64, address string) *Replica {
	r := &Replica{id, address, nil, 0, 0}
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		glog.Errorf("[Replica %v] failed to connect to %v", id, address)
	} else {
		glog.Infof("[Replica %v] connected to %v", id, address)
	}
	r.client = pb.NewRaftClient(conn)
	return r
}
