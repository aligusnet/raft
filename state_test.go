package raft

import (
	"github.com/alexander-ignatyev/raft/log"
	"github.com/alexander-ignatyev/raft/state"
	"time"
)

type noOpStateMachine struct{}

func (m *noOpStateMachine) ExecuteCommand(command []byte) ([]byte, error) {
	return []byte("hi"), nil
}
func (m *noOpStateMachine) CommandToString(command []byte) (string, error) {
	return "hi", nil
}
func (m *noOpStateMachine) Debug() string {
	return "hi"
}

func newTestState(id int64, timeout int) *state.State {
	return newTestStateWithAddresses(id, timeout, make(map[int64]string))
}

func newTestStateWithAddresses(id int64, timeout int, addresses map[int64]string) *state.State {
	return state.New(id,
		time.Millisecond*time.Duration(timeout),
		addresses,
		log.New(),
		&noOpStateMachine{})
}
