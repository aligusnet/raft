package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os/exec"
	"time"
	"sync"
	"sync/atomic"
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
	p.Kill()
	p.cmd = exec.Command(p.progname, p.args...)
	if err := p.cmd.Start(); err != nil {
		glog.Fatalf("Failed to start program # %v. error message: %v", p.instance, err)
	}
}

func (p *ProcessInfo) CombinedOutput() ([]byte, error) {
	p.Kill()
	p.cmd = exec.Command(p.progname, p.args...)
	return p.cmd.CombinedOutput()
}

func (p *ProcessInfo) Kill() {
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd = nil
	}
}


type ProcessGroup struct {
	name string
	processes []*ProcessInfo
	wg sync.WaitGroup
	stopped int32 // atomic access only
}


func createProcessGroup(name string, n int,
	maker func (instance int64) *ProcessInfo) *ProcessGroup {
	group := &ProcessGroup{
		name: name,
		processes: make([]*ProcessInfo, n),
		stopped: 0,
	}

	for i := 0; i < n; i++ {
		group.processes[i] = maker(int64(i))
	}

	return group
}

func (group *ProcessGroup) run() {
	for i := range group.processes {
		glog.Infof("starting %v # %v", group.name, i)
		group.wg.Add(1)
		go func(process *ProcessInfo, index int) {
			output, err := process.CombinedOutput()
			for err != nil && atomic.LoadInt32(&group.stopped) == 0  {
				glog.Infof("restarting %v %v, output: %s\n\n\n", group.name, index, output)
				time.Sleep(500*time.Millisecond)
				output, err = process.CombinedOutput()
			}

			glog.Infof("\n\n%v %v output: %s\n\n\n", group.name, index, output)
			group.wg.Done()
		}(group.processes[i], i)
	}
}


func (group *ProcessGroup) stop() {
	atomic.StoreInt32(&group.stopped, 1)
	for _, p := range group.processes {
		glog.Infof("stopping %v # %v", group.name, p.instance)
		p.Kill()
	}
}

func (group *ProcessGroup) wait() {
	atomic.StoreInt32(&group.stopped, 1)
	group.wg.Wait()
}

func main() {
	flag.Parse()

	defer glog.Flush()

	clients := createProcessGroup("client", 10, newClientInfo)
	servers := createProcessGroup("server", 5, newServerInfo)

	servers.run()
	defer servers.stop()

	time.Sleep(100 * time.Millisecond)

	clients.run()


	time.Sleep(3*time.Second)
	servers.processes[0].Kill()
	servers.processes[2].Kill()

	time.Sleep(3*time.Second)
	servers.processes[1].Kill()
	servers.processes[3].Kill()

	time.Sleep(3*time.Second)
	servers.processes[2].Kill()
	servers.processes[4].Kill()

	glog.Info("waiting for client termination")
	clients.wait()

	time.Sleep(3*time.Second)
	glog.Flush()
}
