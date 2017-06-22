package raft

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
