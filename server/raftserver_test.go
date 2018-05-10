package server

import (
	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/entity"
	"github.com/lnhote/noah/core/raftrpc"
	"github.com/lnhote/noah/core/store"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"log"
	"net"
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
		core.NewServerInfo(1, core.RoleCandidate, "127.0.0.1:8861"),
		core.NewServerInfo(2, core.RoleCandidate, "127.0.0.1:8862"),
		core.NewServerInfo(3, core.RoleCandidate, "127.0.0.1:8863"),
		core.NewServerInfo(4, core.RoleCandidate, "127.0.0.1:8864"),
		core.NewServerInfo(5, core.RoleCandidate, "127.0.0.1:8865")}

	leader = clusters[0]

	env        = &core.Env{HeartBeatDurationInMs: 1000, LeaderElectionDurationInMs: 5000, RandomRangeInMs: 3000}
	envNoTimer = &core.Env{HeartBeatDurationInMs: 0, LeaderElectionDurationInMs: 0, RandomRangeInMs: 50}
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
	s1.Stop()
	s2.Stop()
	s3.Stop()
	s4.Stop()
	s5.Stop()
}

func TestVoteNoTimeout(t *testing.T) {
	leader := clustersNoLeader[0]
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
	s1.Stop()
	s2.Stop()
	s3.Stop()
	s4.Stop()
	s5.Stop()

}

func TestStartAndStop(t *testing.T) {
	clusters := []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8881"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:8882"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:8883"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:8884"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:8885")}
	leader := clusters[0]

	sLeader := NewRaftServerWithEnv(core.NewServerConf(leader, leader, clusters), env)
	serverAddr := sLeader.ServerConf.Info.ServerAddr.String()
	// before start
	var _, err = net.DialTimeout("tcp", serverAddr, 50*time.Millisecond)
	assert.NotNil(t, err)

	// after start
	go sLeader.Start()
	waitAfterStart()
	client, err := net.DialTimeout("tcp", serverAddr, 50*time.Millisecond)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	client.Close()

	// stop
	sLeader.Stop()
	waitAfterStop()
	_, err1 := net.DialTimeout("tcp", serverAddr, 50*time.Millisecond)
	assert.NotNil(t, err1)
}

func TestLeaderElectionAfterLeaderDead(t *testing.T) {
	clusters := []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8871"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:8872"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:8873"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:8874"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:8875")}
	leader := clusters[0]
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
	atLeastOneLeader := s1.ServerConf.Info.Role == core.RoleLeader || s2.ServerConf.Info.Role == core.RoleLeader ||
		s3.ServerConf.Info.Role == core.RoleLeader || s4.ServerConf.Info.Role == core.RoleLeader
	assert.True(t, atLeastOneLeader)
	s1.Stop()
	s2.Stop()
	s3.Stop()
	s4.Stop()
	s5.Stop()

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
	clusters := []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8891"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:8892"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:8893"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:8894"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:8895")}
	leader := clusters[0]
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
	voteFor(s1, clusters[2].ServerID)
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 4), &res))
	assert.True(t, res.Accept)

	// case 6: already voted other
	voteFor(s1, 100)
	assert.Nil(t, s1.OnReceiveRequestVoteRPC(newTestVoteReq(5, 3, 4), &res))
	assert.False(t, res.Accept)
}

func TestRaftServer_AppendLog(t *testing.T) {
	clusters := []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:9001"),
		core.NewServerInfo(2, core.RoleFollower, "127.0.0.1:9002"),
		core.NewServerInfo(3, core.RoleFollower, "127.0.0.1:9003"),
		core.NewServerInfo(4, core.RoleFollower, "127.0.0.1:9004"),
		core.NewServerInfo(5, core.RoleFollower, "127.0.0.1:9005")}
	leader := clusters[0]
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
	assert.Equal(t, 1, getLastLogIndex(s1Leader))
	assert.Equal(t, 0, getLastLogTerm(s1Leader))
	assert.Equal(t, 1, getLastLogIndex(s2))
	assert.Equal(t, 0, getLastLogTerm(s2))
	assert.Equal(t, 1, getLastLogIndex(s3))
	assert.Equal(t, 0, getLastLogTerm(s3))
	assert.Equal(t, 1, getLastLogIndex(s4))
	assert.Equal(t, 0, getLastLogTerm(s4))
	assert.Equal(t, 1, getLastLogIndex(s5))
	assert.Equal(t, 0, getLastLogTerm(s5))
	logEntry, _ := s5.stableInfo.Logs.GetLogEntry(1)
	assert.Equal(t, "test", string(logEntry.Command.Value))
	s1Leader.Stop()
	s2.Stop()
	s3.Stop()
	s4.Stop()
	s5.Stop()
}

func ExampleNewRaftServer() {
	envExample := &core.Env{HeartBeatDurationInMs: 1000, LeaderElectionDurationInMs: 5000, RandomRangeInMs: 3000}
	serverConfig, err := core.GetServerConfFromFile("config.yml")
	if err != nil {
		panic(err)
	}
	raftNode := NewRaftServerWithEnv(serverConfig, envExample)
	raftNode.OpenRepo("test/ExampleNewRaftServer", store.DefaultPageSize, store.SegmentSizeBytes)
	raftNode.Start()
}

func TestLog(t *testing.T) {
	logger, _ := zap.NewDevelopmentConfig().Build() // or NewProduction, or NewDevelopment
	sugar := logger.Sugar()
	defer logger.Sync()

	const url = "http://example.com"

	// In most circumstances, use the SugaredLogger. It's 4-10x faster than most
	// other structured logging packages and has a familiar, loosely-typed API.
	sugar.Infof("Failed to fetch URL: %s", url)

}
