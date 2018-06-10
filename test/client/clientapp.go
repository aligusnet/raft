package main

import (
	"flag"
	"fmt"
	"github.com/aligusnet/raft/client"
	gsmpb "github.com/aligusnet/raft/test/protobuf"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"math/rand"
)

type GranaryClientError struct {
	message string
}

func (e *GranaryClientError) Error() string {
	return e.message
}

type GranaryClient struct {
	client *client.Client
}

func (gc *GranaryClient) send(request *gsmpb.Request) (*gsmpb.Response, error) {
	glog.Info("sending request", request)
	if bytes, err := proto.Marshal(request); err == nil {
		res, err := gc.client.ExecuteCommand(bytes)
		if err == nil {
			response := &gsmpb.Response{}
			if err := proto.Unmarshal(res, response); err == nil {
				glog.Info("got response", response)
				return response, nil
			} else {
				glog.Info("Failed to decode response: ", err)
				return nil, err
			}
		} else {
			glog.Info("rift execute command error", err)
			return nil, err
		}
	} else {
		glog.Infof("Failed to encode request: %v", err)
		return nil, err
	}
}

func (gc *GranaryClient) GetInfo() (int64, int64, error) {
	request := &gsmpb.Request{&gsmpb.Request_GranaryInfo{&gsmpb.GranaryInfoRequest{}}}
	var response *gsmpb.Response
	var err error
	response, err = gc.send(request)
	if err != nil {
		return -1, -1, err
	}
	if infoResponse := response.GetGranaryInfo(); infoResponse != nil {
		return infoResponse.Size, infoResponse.Capacity, nil
	} else {
		return -2, -2, &GranaryClientError{fmt.Sprintf("Unexpected response: %T, %v", response, response.String())}
	}
}

func (gc *GranaryClient) PutGrain(n int64) error {
	request := &gsmpb.Request{&gsmpb.Request_PutGrain{&gsmpb.PutGrainRequest{n}}}
	var response *gsmpb.Response
	var err error
	response, err = gc.send(request)
	if err != nil {
		return err
	}
	if putResponse := response.GetPutGrain(); putResponse != nil {
		if putResponse.Success {
			return nil
		} else {
			return &GranaryClientError{fmt.Sprintf("Unsuccessfull response: %v", putResponse.String())}
		}
	} else {
		return &GranaryClientError{fmt.Sprintf("Unexpected response: %T, %v", response.GetResponse(), response.String())}
	}
}

func (gc *GranaryClient) TakeGrain(n int64) error {
	request := &gsmpb.Request{&gsmpb.Request_TakeGrain{&gsmpb.TakeGrainRequest{n}}}
	var response *gsmpb.Response
	var err error
	response, err = gc.send(request)
	if err != nil {
		return err
	}
	if takeResponse := response.GetTakeGrain(); takeResponse != nil {
		if takeResponse.Success {
			return nil
		} else {
			return &GranaryClientError{fmt.Sprintf("Unsuccessfull response: %v", takeResponse.String())}
		}
	} else {
		return &GranaryClientError{fmt.Sprintf("Unexpected response: %T, %v", response, response.String())}
	}
}

func getInfo(client *GranaryClient) {
	if size, capacity, err := client.GetInfo(); err == nil {
		glog.Infof("Granary info: %v, %v", size, capacity)
	} else {
		glog.Infof("Granary info's error: %v", err)
	}
}

func putGrain(client *GranaryClient) {
	n := rand.Int63n(1000)
	if err := client.PutGrain(n); err == nil {
		glog.Infof("Put %v grain in the granary", n)
	} else {
		glog.Infof("Failed to put %v grain in the granary: %v", n, err)
	}
}

func takeGrain(client *GranaryClient) {
	n := rand.Int63n(1000)
	if err := client.TakeGrain(n); err == nil {
		glog.Infof("Taken %v grain in the granary", n)
	} else {
		glog.Infof("Failed to take %v grain from the granary: %v", n, err)
	}
}

func main() {
	serverAddresses := []string{"localhost:50100", "localhost:50101", "localhost:50102", "localhost:50103", "localhost:50104"}
	flag.Parse()
	defer glog.Flush()

	glog.Info("Hello from client")

	raftClient := client.New(serverAddresses)
	defer raftClient.CloseConnection()

	client := &GranaryClient{client: raftClient}
	getInfo(client)

	for i := 0; i < 10; i++ {
		putGrain(client)
		getInfo(client)
		takeGrain(client)
		getInfo(client)
	}

	glog.Info("Client application is shutting down...")
}
