package log

import (
	pb "github.com/alexander-ignatyev/raft/raft"
	"github.com/golang/glog"
)

type Log interface {
	Get(index int64) *pb.LogEntry
	Size() int64
	Append(term int64, cmd []byte) int64 // returns index of appended entry
	EraseAfter(index int64)
}

func New() Log {
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
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("log.Get() panicked on index: %v, size: %v", index, len(l.entries))
			panic(r)
		}
	}()
	return l.entries[index]
}

func (l *arrayLog) Size() int64 {
	return int64(len(l.entries))
}

func (l *arrayLog) EraseAfter(index int64) {
	l.entries = l.entries[0 : index+1]
}
