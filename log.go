package raft

import (
	pb "github.com/alexander-ignatyev/raft/raft"
)

type Log interface {
	Append(term int64, cmd []byte) int64 // returns index of appended entry
	Get(index int64) *pb.LogEntry
	Size() int64
	EraseAfter(index int64)
}

func NewLog() Log {
	return new(arrayLog)
}

type arrayLog struct {
	entries []*pb.LogEntry
}

func (l *arrayLog) Append(term int64, cmd []byte) int64 {
	index := l.Size()
	entry := &pb.LogEntry{Term: term, Index: index, Command: cmd}
	l.entries = append(l.entries, entry)
	return index
}

func (l *arrayLog) Get(index int64) *pb.LogEntry {
	return l.entries[index]
}

func (l *arrayLog) Size() int64 {
	return int64(len(l.entries))
}

func (l *arrayLog) EraseAfter(index int64) {
	l.entries = l.entries[0 : index+1]
}
