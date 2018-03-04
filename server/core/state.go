package core

import (
	"fmt"
	"time"
)

const (
	RoleFollower  = 0
	RoleCandidate = 1
	RoleLeader    = 2
)

type FollowerState struct {
	// Addr is follower's ip
	Ip string

	// LastIndex is the last received log index for the server
	LastIndex int

	// LastTerm is the term for last received log
	LastTerm int

	// NextIndex is the next log index for the server
	NextIndex int

	// CommitIndex is the committed log index
	CommitIndex int

	// MatchedIndex is the last matched index for the follower
	MatchedIndex int

	LastRpcTime time.Time
}

type ServerState struct {
	// for all
	CommitIndex int
	NextIndex   int
	Term        int
	Role        int

	// for follower
	LeaderIp string

	// for leader election
	LastVotedTerm     int
	LastVotedServerIp string

	// for leader
	Followers map[string]*FollowerState
}

// LogsToCommit: logindex = command struct
var logsToCommit = map[int]*LogEntry{}

var CurrentServerState = NewServerState()

func NewServerState() *ServerState {
	state := &ServerState{}
	state.Followers = map[string]*FollowerState{}
	return state
}

func GetLogTerm(index int) (int, error) {
	entry, err := GetLogEntry(index)
	if err != nil {
		return 0, err
	}
	return entry.Term, nil
}

func GetLogEntry(index int) (*LogEntry, error) {
	if entry, ok := logsToCommit[index]; ok {
		return entry, nil
	} else {
		return nil, fmt.Errorf("InvalidLogIndex(%d)", index)
	}
}

func SaveLogEntry(log *LogEntry) {
	logsToCommit[log.Index] = log
}

func GetLastIndex() int {
	return CurrentServerState.NextIndex - 1
}

func IsAcceptedByMajority(index int) bool {
	counts := 1
	total := len(CurrentServerState.Followers) + 1
	for _, follower := range CurrentServerState.Followers {
		if follower.LastIndex >= index {
			counts = counts + 1
		}
	}
	return counts*2 > total
}
