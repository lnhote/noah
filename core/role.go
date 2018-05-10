package core

type raftRole int

const (
	RoleFollower raftRole = iota
	RoleCandidate
	RoleLeader
)

func (r raftRole) String() string {
	return [...]string{"Follower", "Candidate", "Leader"}[r]
}

func (r *raftRole) HandleMsg(msg raftEvent) {
	switch *r {
	case RoleFollower:
		r.handleFollowerEvent(msg)
	case RoleCandidate:
		r.handleCandidateEvent(msg)
	case RoleLeader:
		r.handleLeaderEvent(msg)
	default:

	}

}

func (r *raftRole) handleLeaderEvent(msg raftEvent) {
	switch msg {
	case DiscoverNewTermEvent:
		*r = RoleFollower
	default:

	}
}

func (r *raftRole) handleCandidateEvent(msg raftEvent) {
	switch msg {
	case DiscoverNewTermEvent:
		*r = RoleFollower
	case ReceiveAppendRPCEvent:
		*r = RoleFollower
	case LeaderElectionSuccessEvent:
		*r = RoleLeader
	default:

	}
}
func (r *raftRole) handleFollowerEvent(msg raftEvent) {
	switch msg {
	case LeaderElectionStartEvent:
		*r = RoleCandidate
	default:

	}
}
