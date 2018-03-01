package server

import (
	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/raftrpc"
)

// leader send empty AppendRPC request to followers
func SendHearbeat() {
	impl := &raftrpc.SimpleRaft{}
	req := &raftrpc.AppendRPCRequest{}
	// TODO Don't send heart beat if we already sent any log recently
	// TODO Don't send to leader
	followerAddrList := config.GetAddrList()
	for _, followerAddr := range followerAddrList {
		req.Ip = followerAddr
		impl.SendAppendEntryRPC(followerAddr, req)
	}
	return
}
