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

// PersistentState is updated on stable storage before responding to RPC
type PersistentState struct {
	// latest term server has seen (initialized to 0 on first boot, increases monotonically)
	Term int

	// candidateId that received vote in current term (or 0 if none)
	LastVotedServerId int

	// log entries; each entry contains command for state machine,
	// and term when entry was received by leader (first index is 1)
	Logs *LogRepo
}

// VolatileState can lost, store on all servers
type VolatileState struct {
	// index of highest log entry known to be committed (initialized to 0, increases monotonically)
	CommitIndex int

	// index of highest log entry applied to state machine
	// initialized to 0, increases monotonically
	LastApplied int
}

// VolatileLeaderState can lost, store on leader server
type VolatileLeaderState struct {
	// for each server, index of the next log entry
	// to send to that server (initialized to leader
	// last log index + 1)
	NextIndex map[int]int

	// for each server, index of highest log entry known to be replicated on server
	// (initialized to 0, increases monotonically)
	MatchIndex map[int]int

	LastRpcTime map[int]time.Time
}

type LogRepo struct {
	// logs: logindex => log struct, start from 1
	logs      map[int]*LogEntry
	lastIndex int
}

func NewLogRepo() *LogRepo {
	return &LogRepo{logs: map[int]*LogEntry{}, lastIndex: 0}
}

func (l *LogRepo) GetLastIndex() int {
	return l.lastIndex
}

func (l *LogRepo) GetNextIndex() int {
	return l.GetLastIndex() + 1
}

func (l *LogRepo) GetLastTerm() int {
	if log, err := l.GetLogEntry(l.lastIndex); err == nil {
		return log.Term
	} else {
		return 0
	}
}

func (l *LogRepo) GetLogTerm(index int) (int, error) {
	entry, err := l.GetLogEntry(index)
	if err != nil {
		return 0, err
	}
	return entry.Term, nil
}

func (l *LogRepo) GetLogList(start int) []*LogEntry {
	newLogs := []*LogEntry{}
	for i := start; i <= l.GetLastIndex(); i++ {
		newLogs = append(newLogs, l.logs[i])
	}
	return newLogs
}

func (ll *LogRepo) GetLogEntry(index int) (*LogEntry, error) {
	if entry, ok := ll.logs[index]; ok {
		return entry, nil
	} else {
		return nil, fmt.Errorf("InvalidLogIndex(%d)", index)
	}
}

func (ll *LogRepo) SaveLogEntry(log *LogEntry) {
	ll.logs[log.Index] = log
	if log.Index > ll.lastIndex {
		ll.lastIndex = log.Index
	}
}

func (ll *LogRepo) Delete(index int) {
	delete(ll.logs, index)
}
