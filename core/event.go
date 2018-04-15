package core

type raftEvent int

const (
	InitEvent = iota
	LeaderElectionStartEvent
	LeaderElectionSuccessEvent
	LeaderElectionFailEvent
	DiscoverNewTermEvent
	ReceiveAppendRPCEvent
)
