package server

import (
	"math/rand"
	"sync"
	"time"

	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/v2pro/plz/countlog"
)

var (
	countMutex          = &sync.Mutex{}
	wg                  = sync.WaitGroup{}
	acceptedCount       = 0
	ackCount            = 0
	leaderElectionTimer = time.NewTimer(time.Duration(20) * time.Millisecond)
)

func ResetLeaderElectionTimer() {
	leaderElectionTimer.Reset(time.Duration(20) * time.Millisecond)
}

// follower will start leader election on time out
func StartLeaderElectionTimer() {
	for {
		<-leaderElectionTimer.C
		ResetLeaderElectionTimer()
		startElection()
	}
}

func startElection() {
	if core.CurrentServerState.Role == core.RoleFollower {
		countlog.Info("leader is dead, start leader election")
		core.CurrentServerState.Role = core.RoleCandidate
		SendRequestVoteToAll()
	}
}

func SendRequestVoteToAll() {
	for core.CurrentServerState.Role == core.RoleCandidate {
		req := &raftrpc.RequestVoteRequest{}
		req.CandidateAddress = config.GetLocalAddr()
		req.NextTerm = core.CurrentServerState.Term + 1
		core.CurrentServerState.LastVotedTerm = req.NextTerm
		core.CurrentServerState.LastVotedServerIp = req.CandidateAddress
		serverList := config.GetServerList()
		acceptedCount = 0
		ackCount = 0
		wg.Add(len(serverList))
		for _, addr := range serverList {
			if addr == req.CandidateAddress {
				continue
			}
			go getVoteFromServer(addr, req)
		}
		wg.Wait()
		if acceptedCount*2 > len(serverList) {
			core.CurrentServerState.Role = core.RoleLeader
			core.CurrentServerState.Term = core.CurrentServerState.Term + 1
			core.CurrentServerState.LeaderIp = config.GetLocalAddr()
			SendHearbeat()
		} else {
			// sleep random time duration (0 - 50 ms) and try again
			duration := rand.Intn(50)
			time.Sleep(time.Millisecond * time.Duration(duration))
		}
		// role becomes Follower when receiving AppendLogRPC
	}
	return
}

func getVoteFromServer(serverAddr string, req *raftrpc.RequestVoteRequest) {
	resp, err := raftrpc.SendRequestVoteRPC(serverAddr, req)
	countMutex.Lock()
	defer countMutex.Unlock()
	ackCount++
	if err != nil {
		return
	}
	if resp.Accept {
		acceptedCount++
	}
	wg.Done()
}
