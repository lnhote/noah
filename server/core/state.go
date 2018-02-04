package core

import "github.com/lnhote/noaḥ/server/command"

const (
	RoleFollower  = 0
	RolwCandidate = 1
	RoleLeader    = 2
)

type FollowerState struct {
	// Ip is follower's ip
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
var LogsToCommit = map[int]*command.Command{}

func (s *ServerState) collect(ip string, index int) {
	// TODO concurrent, use channel
	s.Followers[ip].LastIndex = index
}

func (s *ServerState) isAcceptedByMajority(index int) bool {
	// TODO concurrent, use channel
	counts := 1
	total := len(s.Followers) + 1
	for _, follower := range s.Followers {
		if follower.LastIndex >= index {
			counts = counts + 1
		}
	}
	return counts*2 > total
}

var CurrentServerState = NewServerState()

func NewServerState() *ServerState {
	state := &ServerState{}
	state.Followers = map[string]*FollowerState{}
	return state
}
