package server

import (
	"fmt"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/server/raftrpc"
	"github.com/v2pro/plz/countlog"
	"math/rand"
	"time"
)

func becomeFollower(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleFollower
}

func becomeCandidate(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleCandidate
}

// waitForNextRoundElection sleep random time duration and try again
func waitForNextRoundElection(s *RaftServer) {
	duration := rand.Intn(s.RandomRangeInMs)
	time.Sleep(time.Millisecond * time.Duration(duration))
	countlog.Info(fmt.Sprintf("%s will sleep %d ms and vote again", s, duration))
}

func becomeLeader(s *RaftServer) {
	s.ServerConf.Info.Role = core.RoleLeader
	s.ServerConf.LeaderInfo = s.ServerConf.Info
	s.stableInfo.Term = s.stableInfo.Term + 1

	for _, server := range s.ServerConf.ClusterAddrList {
		if s.ServerConf.Info.ServerId == server.ServerId {
			continue
		}
		s.leaderState.NextIndex[server.ServerId] = s.stableInfo.Logs.GetLastIndex() + 1
		s.leaderState.MatchIndex[server.ServerId] = 0
		s.leaderState.LastRpcTime[server.ServerId] = time.Now()
	}
}

func shouldVote(req *raftrpc.RequestVoteRequest, s *RaftServer) bool {
	lastIndex := s.stableInfo.Logs.GetLastIndex()
	lastTerm := s.stableInfo.Logs.GetLastTerm()
	granted := false
	if req.LastLogTerm > lastTerm {
		granted = true
	} else if req.LastLogTerm < lastTerm {
		granted = false
	} else if req.LastLogIndex >= lastIndex {
		granted = true
	} else {
		granted = false
	}
	countlog.Debug(fmt.Sprintf("shouldVote %t req.LastLogTerm=%d, server.lastTerm=%d, req.LastLogIndex=%d, server.lastIndex=%d",
		granted, req.LastLogTerm, lastTerm, req.LastLogIndex, lastIndex))
	return granted
}

func buildAppendRPCRequest(s *RaftServer, follower *core.ServerInfo, prevLogIndex int) *raftrpc.AppendRPCRequest {
	req := &raftrpc.AppendRPCRequest{}
	req.LeaderNode = s.ServerConf.Info
	req.Term = s.stableInfo.Term
	req.CommitIndex = s.volatileInfo.CommitIndex
	newLogStartIndex := getFollwerNextIndex(s, follower.ServerId)
	if prevLogIndex == 0 {
		req.PrevLogIndex = 0
		req.PrevLogTerm = 0
	} else {
		if preLog, err := s.stableInfo.Logs.GetLogEntry(prevLogIndex); err == nil {
			req.PrevLogIndex = preLog.Index
			req.PrevLogTerm = preLog.Term
		} else {
			// too small
			countlog.Error(fmt.Sprintf("fail to get log entry: %s", err.Error()))
			req.PrevLogTerm = 0
			req.PrevLogIndex = 0
		}
	}
	// the new logs is from the follower's next index to the leader's last log index
	// prepare new logs for the follower
	req.LogEntries = s.stableInfo.Logs.GetLogList(newLogStartIndex)
	return req
}

// saveLogEntry add the log to log repo
func saveLogEntry(s *RaftServer, log *core.LogEntry) {
	s.stableInfo.Logs.SaveLogEntry(log)
}

func appendLogEntry(s *RaftServer, cmd *core.Command) {
	newLog := &core.LogEntry{Command: cmd, Index: getNextLogIndex(s), Term: s.stableInfo.Term}
	saveLogEntry(s, newLog)
}

func getLastLogTerm(s *RaftServer) int {
	return s.stableInfo.Logs.GetLastTerm()
}

func getLastLogIndex(s *RaftServer) int {
	return s.stableInfo.Logs.GetLastIndex()
}

func getNextLogIndex(s *RaftServer) int {
	return s.stableInfo.Logs.GetNextIndex()
}

func updateTerm(s *RaftServer, newTerm int) {
	s.stableInfo.Term = newTerm
}

func voteFor(s *RaftServer, candidateId int) {
	s.stableInfo.LastVotedServerId = candidateId
}

func resetVote(s *RaftServer) {
	voteFor(s, 0)
}

func getFollwerNextIndex(s *RaftServer, followerId int) int {
	return s.leaderState.NextIndex[followerId]
}
