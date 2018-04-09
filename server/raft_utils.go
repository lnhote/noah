package server

import (
	"fmt"
	"github.com/lnhote/noah/core"
	"github.com/v2pro/plz/countlog"
	"math/rand"
	"time"
)

func becomeFollower(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleFollower
	s.ServerInfo.Role = core.RoleFollower
}

func becomeCandidate(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleCandidate
	s.ServerInfo.Role = core.RoleCandidate
}

// waitForNextRoundElection sleep random time duration and try again
func waitForNextRoundElection(s *RaftServer) {
	duration := rand.Intn(s.RandomRangeInMs)
	time.Sleep(time.Millisecond * time.Duration(duration))
	countlog.Info(fmt.Sprintf("%s will sleep %d ms and vote again", s, duration))
}

func stopTimers(s *RaftServer) {
	s.leaderElectionTimer.Stop()
	s.leaderHeartBeatTimer.Stop()
}

func becomeLeader(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleLeader
	s.ServerConf.LeaderInfo = s.ServerConf.Info
	s.ServerInfo.Role = core.RoleLeader
	s.ServerInfo.Term = s.ServerInfo.Term + 1
	s.ServerInfo.LeaderId = s.ServerConf.Info.ServerId
	for _, server := range s.ServerConf.ClusterAddrList {
		if s.ServerConf.Info.ServerId == server.ServerId {
			continue
		}
		s.ServerInfo.Followers[server.ServerId] = &core.FollowerState{Node: server}
	}
}
