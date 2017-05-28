package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/alexander-ignatyev/raft"
	"github.com/golang/glog"
	"time"
)

func main() {
	var instance = flag.Int64("instance", -1, "instance number: >=0")
	flag.Parse()

	glog.Info("starting server")

	index := *instance
	if index < 0 {
		return
	}
	ports := [...]int{50100, 50101, 50102, 50103, 50104}

	fmt.Printf("Hello %v\n", index)

	endpoints := make(map[int64]string, 0)
	for id, port := range ports {
		endpoints[int64(id)] = fmt.Sprintf("localhost:%v", port)
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*10)

	raft.Run(ctx, int64(index), 200*time.Millisecond, raft.NewLog(), endpoints)

	glog.Info("Application is shutting down")
	glog.Flush()
}
