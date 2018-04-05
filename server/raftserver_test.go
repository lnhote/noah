package server

import (
	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/stretchr/testify/assert"
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

	clustersNoLeader = []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleCandidate, "127.0.0.1:8851"),
		core.NewServerInfo(2, core.RoleCandidate, "127.0.0.1:8852"),
		core.NewServerInfo(3, core.RoleCandidate, "127.0.0.1:8853"),
		core.NewServerInfo(4, core.RoleCandidate, "127.0.0.1:8854"),
		core.NewServerInfo(5, core.RoleCandidate, "127.0.0.1:8855")}

	leader = clusters[0]

	env        = &Env{HeartBeatDurationInMs: 2000, LeaderElectionDurationInMs: 10000}
	envNoTimer = &Env{HeartBeatDurationInMs: 0, LeaderElectionDurationInMs: 0}
)

func init() {
	common.SetupLog()
}

func TestStartHeartbeatTimer(t *testing.T) {
	//done := make(chan bool, 1)
	go NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[2], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[3], leader, clusters), env).Start()
	go NewRaftServerWithEnv(core.NewServerConf(clusters[4], leader, clusters), env).Start()
	NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env).Start()
	//<-done
}

func TestVoteTimeout(t *testing.T) {
	s1 := NewRaftServerWithEnv(core.NewServerConf(clustersNoLeader[0], leader, clustersNoLeader), envNoTimer)
	s2 := NewRaftServerWithEnv(core.NewServerConf(clustersNoLeader[1], leader, clustersNoLeader), envNoTimer)
	s3 := NewRaftServerWithEnv(core.NewServerConf(clustersNoLeader[2], leader, clustersNoLeader), envNoTimer)
	s4 := NewRaftServerWithEnv(core.NewServerConf(clustersNoLeader[3], leader, clustersNoLeader), envNoTimer)
	s5 := NewRaftServerWithEnv(core.NewServerConf(clustersNoLeader[4], leader, clustersNoLeader), envNoTimer)
	go s1.Start()
	go s2.Start()
	go s3.Start()
	go s4.Start()
	go s5.Start()
	time.Sleep(time.Second)
	assert.True(t, s1.collectVotes().Timeout < 2)
	assert.True(t, s2.collectVotes().Timeout < 2)
	assert.True(t, s3.collectVotes().Timeout < 2)
	assert.True(t, s4.collectVotes().Timeout < 2)
	assert.True(t, s5.collectVotes().Timeout < 2)
}

func TestStop(t *testing.T) {
	sLeader := NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	go sLeader.Start()
	sLeader.Stop()
	time.Sleep(time.Second)
}
