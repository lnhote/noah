package server

import (
	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/server/raftrpc"
	"github.com/stretchr/testify/assert"
	"log"
	"net/rpc"
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

	env        = &Env{HeartBeatDurationInMs: 1000, LeaderElectionDurationInMs: 5000, RandomRangeInMs: 3000}
	envNoTimer = &Env{HeartBeatDurationInMs: 0, LeaderElectionDurationInMs: 0, RandomRangeInMs: 50}
)

func init() {
	common.SetupLog()
}

func waitAfterStart() {
	time.Sleep(time.Millisecond * 20)
}

func waitAfterStop() {
	time.Sleep(time.Millisecond * 2000)
}

func TestStartHeartbeatTimer(t *testing.T) {
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
	waitAfterStart()
	s1Last := s1.leaderElectionTimer.LastFiredTime()
	s2Last := s2.leaderElectionTimer.LastFiredTime()
	s3Last := s3.leaderElectionTimer.LastFiredTime()
	s4Last := s4.leaderElectionTimer.LastFiredTime()
	for i := 0; i < 10; i++ {
		// make sure heart beat is triggered for leader
		time.Sleep(time.Millisecond * 1000)
		log.Printf("s5.leaderElectionTimer.TimeSinceLastFired() = %d", s5.leaderHeartBeatTimer.TimeSinceLastFired())
		assert.True(t, s5.leaderHeartBeatTimer.TimeSinceLastFired() < 50)
		assert.True(t, s5.ServerConf.Info.Role == core.RoleLeader)

		// make sure leader election timer is not fired for follower (get heart beat from leader)
		assert.True(t, s1.leaderElectionTimer.LastFiredTime() == s1Last)
		assert.True(t, s2.leaderElectionTimer.LastFiredTime() == s2Last)
		assert.True(t, s3.leaderElectionTimer.LastFiredTime() == s3Last)
		assert.True(t, s4.leaderElectionTimer.LastFiredTime() == s4Last)
		assert.True(t, s1.ServerConf.Info.Role == core.RoleFollower)
		assert.True(t, s2.ServerConf.Info.Role == core.RoleFollower)
		assert.True(t, s3.ServerConf.Info.Role == core.RoleFollower)
		assert.True(t, s4.ServerConf.Info.Role == core.RoleFollower)
	}
}

func TestVoteNoTimeout(t *testing.T) {
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
	waitAfterStart()
	assert.True(t, s1.collectVotes().Timeout < 1)
	assert.True(t, s2.collectVotes().Timeout < 1)
	assert.True(t, s3.collectVotes().Timeout < 1)
	assert.True(t, s4.collectVotes().Timeout < 1)
	assert.True(t, s5.collectVotes().Timeout < 1)
}

func TestStartAndStop(t *testing.T) {
	sLeader := NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	serverAddr := sLeader.ServerConf.Info.ServerAddr.String()
	// before start
	var _, err = rpc.Dial("tcp", serverAddr)
	assert.NotNil(t, err)

	// after start
	go sLeader.Start()
	waitAfterStart()
	client, err := rpc.Dial("tcp", serverAddr)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	client.Close()

	// stop
	sLeader.Stop()
	waitAfterStop()
	_, err1 := rpc.Dial("tcp", serverAddr)
	assert.NotNil(t, err1)
}

func TestLeaderElectionAfterLeaderDead(t *testing.T) {
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
	waitAfterStart()
	s5.Stop()
	waitAfterStop()
	s1Last := s1.leaderElectionTimer.LastFiredTime()
	s2Last := s2.leaderElectionTimer.LastFiredTime()
	s3Last := s3.leaderElectionTimer.LastFiredTime()
	s4Last := s4.leaderElectionTimer.LastFiredTime()
	time.Sleep(time.Second * 10)
	atLeastOne := s1.leaderElectionTimer.LastFiredTime() != s1Last || s2.leaderElectionTimer.LastFiredTime() != s2Last ||
		s3.leaderElectionTimer.LastFiredTime() != s3Last || s4.leaderElectionTimer.LastFiredTime() != s4Last
	assert.True(t, atLeastOne)
}

func TestVoteRejectCase(t *testing.T) {
	s1 := NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env)
	cmd := &core.Command{CommandType: core.CmdSet, Key: "name", Value: []byte("test")}
	s1.Logs.SaveLogEntry(&core.LogEntry{Command: cmd, Index: 1, Term: 1})
	s1.Logs.SaveLogEntry(&core.LogEntry{Command: cmd, Index: 2, Term: 2})
	s1.Logs.SaveLogEntry(&core.LogEntry{Command: cmd, Index: 3, Term: 3})
	s1.Logs.SaveLogEntry(&core.LogEntry{Command: cmd, Index: 4, Term: 4})
	s1.ServerInfo.NextIndex = 5
	s1.ServerInfo.Term = 4

	reqCandidate := &raftrpc.RequestVoteRequest{}
	reqCandidate.Candidate = clusters[2]
	res := raftrpc.RequestVoteResponse{}
	reqCandidate.NextTerm = 5

	// case 1: term is too small
	reqCandidate.LastLogIndex = 3
	reqCandidate.LastLogTerm = 3
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.False(t, res.Accept)

	// case 2: length is small
	reqCandidate.LastLogIndex = 3
	reqCandidate.LastLogTerm = 4
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.False(t, res.Accept)

	// case 3: good enough
	reqCandidate.LastLogIndex = 4
	reqCandidate.LastLogTerm = 4
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.True(t, res.Accept)
	assert.True(t, s1.ServerInfo.LastVotedTerm == reqCandidate.NextTerm)

	// case 4: term is too old
	reqCandidate.LastLogIndex = 4
	reqCandidate.LastLogTerm = 4
	reqCandidate.NextTerm = 3
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.False(t, res.Accept)

	// case 5: already voted this
	s1.ServerInfo.LastVotedServerId = clusters[2].ServerId
	reqCandidate.NextTerm = 5
	s1.ServerInfo.LastVotedTerm = 5
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.True(t, res.Accept)

	// case 6: already voted other
	s1.ServerInfo.LastVotedServerId = 100
	s1.ServerInfo.LastVotedTerm = 5
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(reqCandidate, &res))
	assert.False(t, res.Accept)

}
