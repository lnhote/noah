package core

import (
	"time"
)

// PersistentState is updated on stable storage before responding to RPC
type PersistentState struct {
	// latest term server has seen (initialized to 0 on first boot, increases monotonically)
	Term int `json:"term"`

	// candidateId that received vote in current term (or 0 if none)
	LastVotedServerId int `json:"last_voted_server_id"`

	// log entries; each entry contains command for state machine,
	// and term when entry was received by leader (first index is 1)
	Logs *LogRepo `json:"-"`
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

func NewPersistentState(term int, lastVotedServerId int, ents []*LogEntry) *PersistentState {
	logs := &LogRepo{logs: map[int]*LogEntry{}, lastIndex: 0}
	for _, ent := range ents {
		logs.SaveLogEntry(ent)
	}
	return &PersistentState{Term: term, LastVotedServerId: lastVotedServerId, Logs: logs}
}
