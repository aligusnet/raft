package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os/exec"
	"time"
)

type ProcessInfo struct {
	instance int64
	progname string
	args     []string
	cmd      *exec.Cmd
}

func newClientInfo(instance int64) *ProcessInfo {
	logDirArg := fmt.Sprintf("--log_dir=logs/client%v", instance)
	return &ProcessInfo{progname: "./client/client",
		args: []string{
			logDirArg,
			"-alsologtostderr=true",
			"--v=10",
		},
	}
}

func newServerInfo(instance int64) *ProcessInfo {
	instanceArg := fmt.Sprintf("--instance=%v", instance)
	logDirArg := fmt.Sprintf("--log_dir=logs/server%v", instance)
	return &ProcessInfo{instance: instance,
		progname: "./server/server",
		args: []string{
			instanceArg,
			logDirArg,
			"--v=10",
		},
	}
}

func (p *ProcessInfo) Start() {
	p.Stop()
	p.cmd = exec.Command(p.progname, p.args...)
	if err := p.cmd.Start(); err != nil {
		glog.Fatalf("Failed to start program # %v. error message: %v", p.instance, err)
	}
}

func (p *ProcessInfo) CombinedOutput() []byte {
	p.Stop()
	p.cmd = exec.Command(p.progname, p.args...)
	if res, err := p.cmd.CombinedOutput(); err != nil {
		glog.Fatalf("Failed to start program # %v. error message: %v", p.instance, err)
		return res
	} else {
		return res
	}

}

func (p *ProcessInfo) Stop() {
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd = nil
	}
}

func main() {
	flag.Parse()
	clients := make([]*ProcessInfo, 1)
	servers := make([]*ProcessInfo, 5)

	for i := int64(0); i < 5; i++ {
		glog.Infof("starting server # %v", i)
		servers[i] = newServerInfo(i)
		servers[i].Start()
	}

	defer func() {
		for _, p := range servers {
			glog.Infof("stopping server # %v", p.instance)
			p.Stop()
		}
	}()

	time.Sleep(100 * time.Millisecond)
	glog.Info("staring client")
	clients[0] = newClientInfo(0)

	output := clients[0].CombinedOutput()

	fmt.Printf("%s\n", output)

	glog.Info("waiting for client termination")
	clients[0].cmd.Wait()

	servers[0].cmd.Wait()
}
