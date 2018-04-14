package core

func getRoleName(role int) string {
	switch role {
	case RoleFollower:
		return "Follower"
	case RoleLeader:
		return "Leader"
	case RoleCandidate:
		return "Candidate"
	default:
		return "UnknownRole"
	}
}
