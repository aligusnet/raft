package raft

type State struct {
	currentTerm int64
	votedFor    int64
	commitIndex int64
	lastApplied int64
}
