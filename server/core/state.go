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
var LogsToCommit = map[int]*Command{}
var LogsToTerm = map[int]int{}

var CurrentServerState = NewServerState()

func NewServerState() *ServerState {
	state := &ServerState{}
	state.Followers = map[string]*FollowerState{}
	return state
}

func GetLogTerm(index int) (int, error) {
	if term, ok := LogsToTerm[index]; ok {
		return term, nil
	} else {
		return 0, fmt.Errorf("InvalidLogIndex=%d", index)
	}
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
