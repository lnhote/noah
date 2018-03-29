package server

var (
	DefaultEnv = &Env{
		LeaderElectionDurationInMs: 50,
		HeartBeatDurationInMs:      5,
	}
)

type Env struct {
	HeartBeatDurationInMs      int
	LeaderElectionDurationInMs int
}
