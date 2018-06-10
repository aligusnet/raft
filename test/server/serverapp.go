package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/aligusnet/raft/server"
	"github.com/aligusnet/raft/server/log"
	gsmpb "github.com/aligusnet/raft/test/protobuf"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"time"
)

type GranaryError struct {
	message string
}

func (e *GranaryError) Error() string {
	return e.message
}

type GranaryStateMachine struct {
	size     int64
	capacity int64
}

func (gsm *GranaryStateMachine) putGrain(n int64) error {
	if gsm.size+n > gsm.capacity {
		return &GranaryError{"not enough capacity"}
	}
	gsm.size += n
	return nil
}

func (gsm *GranaryStateMachine) takeGrain(n int64) error {
	if gsm.size-n < 0 {
		return &GranaryError{"not enough size"}
	}
	gsm.size -= n
	return nil
}

func (gsm *GranaryStateMachine) ExecuteCommand(cmd []byte) ([]byte, error) {
	request := &gsmpb.Request{}
	if err := proto.Unmarshal(cmd, request); err != nil {
		glog.Errorf("failed to decode request: %v", err)
		return nil, err
	}

	response := &gsmpb.Response{}

	if request.GetGranaryInfo() != nil {
		resp := &gsmpb.GranaryInfoResponse{Size: gsm.size,
			Capacity: gsm.capacity}
		response.Response = &gsmpb.Response_GranaryInfo{resp}
		glog.Infof("GranaryStateMachine.getGranaryInfo: %v", response.Response)
	}

	if putGrain := request.GetPutGrain(); putGrain != nil {
		err := gsm.putGrain(putGrain.Number)
		resp := &gsmpb.PutGrainResponse{Success: err == nil}
		if err != nil {
			glog.Infof("GranaryStateMachine.putGrain error: %v", err)
			resp.Message = err.Error()
		}
		response.Response = &gsmpb.Response_PutGrain{resp}
	}

	if takeGrain := request.GetTakeGrain(); takeGrain != nil {
		err := gsm.takeGrain(takeGrain.Number)
		resp := &gsmpb.TakeGrainResponse{Success: err == nil}
		if err != nil {
			glog.Infof("GranaryStateMachine.takeGrain error: %v", err)
			resp.Message = err.Error()
		}
		response.Response = &gsmpb.Response_TakeGrain{resp}
	}

	if res, err := proto.Marshal(response); err != nil {
		glog.Errorf("failed to encode response: %v", err)
		return nil, err
	} else {
		return res, nil
	}
}

func (gsm *GranaryStateMachine) CommandToString(cmd []byte) (string, error) {
	request := &gsmpb.Request{}
	if err := proto.Unmarshal(cmd, request); err != nil {
		glog.Errorf("failed to decode request: %v", err)
		return "??", err
	}
	if request.GetGranaryInfo() != nil {
		return "#", nil
	}

	if putGrain := request.GetPutGrain(); putGrain != nil {
		return fmt.Sprintf(">>%v", putGrain.Number), nil
	}

	if takeGrain := request.GetTakeGrain(); takeGrain != nil {
		return fmt.Sprintf("<<%v", takeGrain.Number), nil
	}
	return "??", &GranaryError{"unknown command"}
}

func (gsm *GranaryStateMachine) Debug() string {
	return fmt.Sprintf("%v", gsm)
}

func main() {
	var instance = flag.Int64("instance", -1, "instance number: >=0")
	flag.Parse()
	defer glog.Flush()

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

	ctx := context.Background()

	server.Run(ctx, int64(index), 200*time.Millisecond, log.New(), endpoints, &GranaryStateMachine{size: 100, capacity: 1000})

	glog.Info("Application is shutting down")

}
