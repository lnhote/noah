package server

import (
	"fmt"

	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/v2pro/plz/countlog"

	"testing"
	"time"
)

var (
	clusters = []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8851"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:8852"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:8853"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:8854"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:8855")}

	leader = clusters[0]

	env = &Env{HeartBeatDurationInMs: 2000, LeaderElectionDurationInMs: 10000}
)

func init() {
	common.SetupLog()
}

func TestStartHeartbeatTimer(t *testing.T) {
	go NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[2], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[3], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[4], leader, clusters), env).Start()
	NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env).Start()
}

func TestLeaderElection(t *testing.T) {
	done := make(chan bool, 1)
	s1 := NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env)
	s2 := NewRaftServerWithEnv(core.NewServerConf(clusters[2], leader, clusters), env)
	s3 := NewRaftServerWithEnv(core.NewServerConf(clusters[3], leader, clusters), env)
	s4 := NewRaftServerWithEnv(core.NewServerConf(clusters[4], leader, clusters), env)
	s5 := NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	go s1.Start()
	go s2.Start()
	go s3.Start()
	go s4.Start()
	go s5.Start()
	countlog.Info("kill leader server " + s5.String())
	s5.Stop()
	<-done
	fmt.Println("exiting")
}

func TestResetTimer(t *testing.T) {
	s := NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	go s.startLeaderElectionTimer()
	countlog.Info(fmt.Sprintf("start at %s\n", time.Now().String()))
	for i := 0; i < 20; i++ {
		countlog.Info(fmt.Sprintf("sleep 1s at %s\n", time.Now().String()))
		time.Sleep(1 * time.Second)
		// this will prevent startLeaderElectionTimer from being fired
		s.resetLeaderElectionTimer()
	}
}
