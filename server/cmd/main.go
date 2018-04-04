package main

import (
	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/server"
	"github.com/v2pro/plz/countlog"
)

var (
	clusters = []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8851"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:8852"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:8853"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:8854"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:8855")}

	leader = clusters[0]

	env = &server.Env{HeartBeatDurationInMs: 2000, LeaderElectionDurationInMs: 10000}
)

func init() {
	common.SetupLog()
}

// run a raft demo
func main() {
	countlog.Info("Start raft demo")
	s1 := server.NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env)
	s2 := server.NewRaftServerWithEnv(core.NewServerConf(clusters[2], leader, clusters), env)
	s3 := server.NewRaftServerWithEnv(core.NewServerConf(clusters[3], leader, clusters), env)
	s4 := server.NewRaftServerWithEnv(core.NewServerConf(clusters[4], leader, clusters), env)
	l1 := server.NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	go s1.Start()
	go s2.Start()
	go s3.Start()
	go s4.Start()
	l1.Start()
}
