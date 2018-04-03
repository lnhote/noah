package raftrpc

import (
	"time"

	"github.com/lnhote/noaḥ/core"
)

type AppendRPCRequest struct {
	LeaderNode *core.ServerInfo

	LogEntries   []*core.LogEntry
	NextIndex    int
	Term         int
	PrevLogTerm  int
	PrevLogIndex int
	CommitIndex  int
}

type AppendRPCResponse struct {
	Node            *core.ServerInfo
	LastLogIndex    int
	LastLogTerm     int
	Time            time.Time
	UnmatchLogIndex int
}

type RequestVoteRequest struct {
	LastLogTerm  int
	LastLogIndex int
	NextTerm     int
	Candidate    *core.ServerInfo
}

type RequestVoteResponse struct {
	Accept bool
}
