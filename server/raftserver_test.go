package server

import (
	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/entity"
	"github.com/lnhote/noah/core/raftrpc"
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
	time.Sleep(time.Second * 20)
	atLeastOne := s1.leaderElectionTimer.LastFiredTime() != s1Last || s2.leaderElectionTimer.LastFiredTime() != s2Last ||
		s3.leaderElectionTimer.LastFiredTime() != s3Last || s4.leaderElectionTimer.LastFiredTime() != s4Last
	assert.True(t, atLeastOne)
	atLeastOneLeader := s1.ServerConf.Info.Role == core.RoleLeader || s2.ServerConf.Info.Role == core.RoleLeader ||
		s3.ServerConf.Info.Role == core.RoleLeader || s4.ServerConf.Info.Role == core.RoleLeader
	assert.True(t, atLeastOneLeader)

}

func newTestCmd() *entity.Command {
	return &entity.Command{CommandType: entity.CmdSet, Key: "name", Value: []byte("test")}
}

func pushTestLog(s *RaftServer, index int, term int) {
	saveLogEntry(s, &core.LogEntry{Command: newTestCmd(), Index: index, Term: term})
}

func newTestVoteReq(term, lastLogIndex, lastLogTerm int) *raftrpc.RequestVoteRequest {
	reqCandidate := &raftrpc.RequestVoteRequest{}
	reqCandidate.Candidate = clusters[2]
	reqCandidate.NextTerm = term
	reqCandidate.LastLogIndex = lastLogIndex
	reqCandidate.LastLogTerm = lastLogTerm
	return reqCandidate
}

func TestVoteRejectCase(t *testing.T) {
	s1 := NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), env)
	// add some logs first
	pushTestLog(s1, 1, 1)
	pushTestLog(s1, 2, 2)
	pushTestLog(s1, 3, 3)
	pushTestLog(s1, 4, 4)
	resetVote(s1)
	assert.Equal(t, 4, getLastLogTerm(s1))
	assert.Equal(t, 4, getLastLogIndex(s1))
	res := raftrpc.RequestVoteResponse{}
	updateTerm(s1, 4)

	// case 1: last log term is too small
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 3), &res))
	assert.False(t, res.Accept)

	// case 2: length is small
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 4), &res))
	assert.False(t, res.Accept)

	// case 3: good enough
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 4, 4), &res))
	assert.True(t, res.Accept)

	// case 4: term is too old
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(3, 4, 4), &res))
	assert.False(t, res.Accept)

	// case 5: already voted this
	voteFor(s1, clusters[2].ServerId)
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 4), &res))
	assert.True(t, res.Accept)

	// case 6: already voted other
	voteFor(s1, 100)
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 4), &res))
	assert.False(t, res.Accept)
}

func TestRaftServer_AppendLog(t *testing.T) {
	s1Leader := NewRaftServerWithEnv(core.NewServerConf(clusters[0], leader, clusters), envNoTimer)
	s2 := NewRaftServerWithEnv(core.NewServerConf(clusters[1], leader, clusters), envNoTimer)
	s3 := NewRaftServerWithEnv(core.NewServerConf(clusters[2], leader, clusters), envNoTimer)
	s4 := NewRaftServerWithEnv(core.NewServerConf(clusters[3], leader, clusters), envNoTimer)
	s5 := NewRaftServerWithEnv(core.NewServerConf(clusters[4], leader, clusters), envNoTimer)
	go s1Leader.Start()
	go s2.Start()
	go s3.Start()
	go s4.Start()
	go s5.Start()
	waitAfterStart()

	val, err := s1Leader.ReplicateLog(newTestCmd())
	assert.Nil(t, err)
	assert.Equal(t, []byte("test"), val)
	// make sure followers get this log
	assert.Equal(t, 1, getLastLogIndex(s2))
	assert.Equal(t, 1, getLastLogTerm(s2))
	assert.Equal(t, 1, getLastLogIndex(s3))
	assert.Equal(t, 1, getLastLogTerm(s3))
	assert.Equal(t, 1, getLastLogIndex(s4))
	assert.Equal(t, 1, getLastLogTerm(s4))
	assert.Equal(t, 1, getLastLogIndex(s5))
	assert.Equal(t, 1, getLastLogTerm(s5))
	logEntry, _ := s5.stableInfo.Logs.GetLogEntry(1)
	assert.Equal(t, "test", string(logEntry.Command.Value))
}
