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
	Node *ServerInfo

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
	LeaderId int

	// for leader election
	LastVotedTerm     int
	LastVotedServerId int

	// for leader
	Followers map[int]*FollowerState
}

func NewServerState() *ServerState {
	state := &ServerState{}
	state.Followers = map[int]*FollowerState{}
	return state
}

func (ss *ServerState) GetLastIndex() int {
	return ss.NextIndex - 1
}

func (ss *ServerState) IsAcceptedByMajority(index int) bool {
	counts := 1
	total := len(ss.Followers) + 1
	for _, follower := range ss.Followers {
		if follower.LastIndex >= index {
			counts = counts + 1
		}
	}
	return counts*2 > total
}

type LogList struct {
	logs map[int]*LogEntry
}

func NewLogList() *LogList {
	logList := &LogList{logs: map[int]*LogEntry{}}
	return logList
}

// LogsToCommit: logindex = command struct

func (ll *LogList) GetLogTerm(index int) (int, error) {
	entry, err := ll.GetLogEntry(index)
	if err != nil {
		return 0, err
	}
	return entry.Term, nil
}

func (ll *LogList) GetLogEntry(index int) (*LogEntry, error) {
	if entry, ok := ll.logs[index]; ok {
		return entry, nil
	} else {
		return nil, fmt.Errorf("InvalidLogIndex(%d)", index)
	}
}

func (ll *LogList) SaveLogEntry(log *LogEntry) {
	ll.logs[log.Index] = log
}
