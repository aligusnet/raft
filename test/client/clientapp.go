package main

import (
	"flag"
	"github.com/alexander-ignatyev/raft"
	"github.com/golang/glog"
)

func main() {
	serverAddresses := []string{"localhost:50100", "localhost:50101", "localhost:50102", "localhost:50103", "localhost:50104"}
	flag.Parse()
	glog.Info("Hello from client")

	raftClient := raft.NewClient(serverAddresses)
	defer raftClient.CloseConnection()

	raftClient.ExecuteCommand([]byte("Hi!"))

	glog.Info("Client application is shutting down...")
	glog.Flush()
}
