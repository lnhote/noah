package core

type raftEvent int

const (
	LeaderElectionStartEvent = iota
	LeaderElectionSuccessEvent
	LeaderElectionFailEvent
	DiscoverNewTermEvent
	ReceiveAppendRPCEvent
)
