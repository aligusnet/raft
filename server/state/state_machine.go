package state

type StateMachine interface {
	ExecuteCommand(command []byte) ([]byte, error)
	CommandToString(command []byte) (string, error)
	Debug() string
}
