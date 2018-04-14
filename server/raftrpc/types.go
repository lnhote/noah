package raftrpc

import (
	"github.com/lnhote/noah/core"
)

type AppendRPCRequest struct {
	// leaderId
	LeaderNode *core.ServerInfo

	// log entries to store (empty for heartbeat; may send more than one for efficiency)
	LogEntries []*core.LogEntry

	// leader’s term
	Term int

	// term of prevLogIndex entry
	PrevLogTerm int

	// index of log entry immediately preceding new ones
	PrevLogIndex int

	// leader’s commitIndex
	CommitIndex int
}

type AppendRPCResponse struct {

	// currentTerm, for leader to update itself
	Term int

	// true if follower contained entry matching prevLogIndex and prevLogTerm
	Success bool
}

type RequestVoteRequest struct {
	// term of candidate’s last log entry
	LastLogTerm int

	// index of candidate’s last log entry
	LastLogIndex int

	// candidate’s term
	NextTerm int

	// candidate requesting vote
	Candidate *core.ServerInfo
}

type RequestVoteResponse struct {
	// currentTerm, for candidate to update itself
	Term int

	// true means candidate received vote
	Accept bool
}
