package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

type Replica struct {
	id         int64
	address    string
	client     pb.RaftClient
	nextIndex  int64
	matchIndex int64
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
