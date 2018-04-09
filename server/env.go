package server

var (
	DefaultEnv = &Env{
		LeaderElectionDurationInMs: 50,
		HeartBeatDurationInMs:      5,
		// (0 - 50 ms)
		RandomRangeInMs: 50,
	}
)

type Env struct {
	HeartBeatDurationInMs      int
	LeaderElectionDurationInMs int
	RandomRangeInMs            int
}
