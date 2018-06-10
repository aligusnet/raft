package client

import (
	"fmt"
	pb "github.com/aligusnet/raft/raft"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"math/rand"
	"time"
)

type Client struct {
	conn            *grpc.ClientConn
	client          pb.RaftClient
	serverAddresses []string
}

func New(serverAddresses []string) *Client {
	c := &Client{serverAddresses: serverAddresses}
	c.reconnect()
	return c
}

func (c *Client) IsConnected() bool {
	return c.conn != nil
}

func (c *Client) ExecuteCommand(cmd []byte) ([]byte, error) {
	if !c.IsConnected() {
		return nil, fmt.Errorf("client is not connected")
	}
	if resp, err := c.client.ExecuteCommand(context.Background(), &pb.ExecuteCommandRequest{cmd}); err == nil {
		if resp.Success == false && len(resp.ServerAddress) > 0 {
			glog.Infof("got instructions to reconnect to the new server: %v", resp.ServerAddress)
			c.Connect(resp.ServerAddress)
			return c.ExecuteCommand(cmd)
		}
		if resp.Success {
			glog.Info("command executed successfully")
			return resp.Answer, nil
		} else {
			glog.Info("command executed with error")
			return nil, fmt.Errorf("Failed to execute command on server due to unknown error")
		}
	} else {
		glog.Info("Failed to execute command, reconnecting and trying again...")
		c.reconnect()
		return c.ExecuteCommand(cmd)
	}
}

func (c *Client) CloseConnection() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.client = nil
	}
}

func (c *Client) reconnect() {
	var err error = fmt.Errorf("error")
	for err != nil {
		instance := rand.Intn(len(c.serverAddresses))
		err = c.Connect(c.serverAddresses[instance])
		if err != nil {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func (c *Client) Connect(address string) error {
	if conn, err := grpc.Dial(address, grpc.WithInsecure()); err != nil {
		glog.Fatalf("failed to connect: %v", err)
		return err
	} else {
		c.CloseConnection()
		c.conn = conn
		c.client = pb.NewRaftClient(c.conn)
		glog.Infof("connected to %v", address)
		return nil
	}
}
