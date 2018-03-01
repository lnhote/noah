package server

import (
	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/raftrpc"
)

func SendRequestVoteToAll() {
	impl := &raftrpc.SimpleRaft{}
	req := &raftrpc.RequestVoteRequest{}
	// TODO Don't send heart beat if we already sent any log recently
	// TODO Don't send to leader
	followerAddrList := config.GetAddrList()
	for _, followerAddr := range followerAddrList {
		req.CandidateAddress = config.GetThisIp()
		resp, err := impl.SendRequestVoteRPC(followerAddr, req)
		if err != nil {
			continue
		}
		if resp.Accept {
			// TODO vote++
		}
	}
	// TODO if vote count is more than 1/2, become leader
	// TODO if vote is not enough, sleep random duration 2-20 ms and vote again
	return
}
