package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
)

type Replica struct {
	id         int64
	address    string
	client     pb.RaftClient
	nextIndex  int64
	matchIndex int64
}
