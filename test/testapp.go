package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os/exec"
)

type ProcessInfo struct {
	instance int64
	progname string
	args     []string
	cmd      *exec.Cmd
}

func newServerInfo(instance int64) *ProcessInfo {
	instanceArg := fmt.Sprintf("--instance=%v", instance)
	logDirArg := fmt.Sprintf("--log_dir=/Users/ai/go/src/github.com/alexander-ignatyev/raft/test/logs/server%v", instance)
	return &ProcessInfo{instance: instance,
		progname: "./server/server",
		args: []string{instanceArg,
			logDirArg,
			"--v=10"}}

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
	servers := make([]*ProcessInfo, 5)

	for i := int64(0); i < 5; i++ {
		glog.Infof("starting server # %v", i)
		servers[i] = newServerInfo(i)
		servers[i].Start()
	}

	servers[0].cmd.Wait()
}
