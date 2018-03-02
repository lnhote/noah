package server

import (
	"time"

	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/v2pro/plz/countlog"
)

var (
	leaderHeartBeatTicker = time.NewTicker(5 * time.Millisecond)
)

// leader will send heart beat to followers periodically
func StartHeartbeatTimer() {
	for {
		<-leaderHeartBeatTicker.C
		if core.CurrentServerState.Role == core.RoleLeader {
			SendHearbeat()
		}
	}
}

// leader send empty AppendRPC request to followers
func SendHearbeat() {
	req := &raftrpc.AppendRPCRequest{}
	req.Term = core.CurrentServerState.Term
	for _, addr := range config.GetServerList() {
		if addr == config.GetLocalAddr() {
			continue
		}
		lastTime := core.CurrentServerState.Followers[addr].LastRpcTime
		if time.Now().Sub(lastTime) < time.Duration(10*time.Millisecond) {
			// Don't send heart beat if we already sent any log recently
			continue
		}
		req.Addr = addr
		go sendHeartbeatToServer(req)
	}
}

func sendHeartbeatToServer(req *raftrpc.AppendRPCRequest) {
	resp, err := raftrpc.SendAppendEntryRPC(req.Addr, req)
	if err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "addr", req.Addr, "error", err)
		return
	}
	core.CurrentServerState.Followers[resp.Addr].LastRpcTime = resp.Time
}
