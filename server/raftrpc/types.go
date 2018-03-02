package raftrpc

import (
	"time"

	"github.com/lnhote/noaḥ/server/core"
)

type AppendRPCRequest struct {
	Addr         string
	LogEntries   []*core.Command
	NextIndex    int
	Term         int
	PrevLogTerm  int
	PrevLogIndex int
}

type AppendRPCResponse struct {
	Addr            string
	LastLogIndex    int
	LastLogTerm     int
	Time            time.Time
	UnmatchLogIndex int
}

type RequestVoteRequest struct {
	LastLogTerm      int
	LastLogIndex     int
	NextTerm         int
	CandidateAddress string
}

type RequestVoteResponse struct {
	Accept bool
}
