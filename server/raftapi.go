package server

import (
	"github.com/lnhote/noaḥ/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
)

type RaftService interface {
	Start()
	Get(cmd *core.Command, resp *core.ClientResponse) error
	Set(cmd *core.Command, resp *core.ClientResponse) error
	OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error
	OnReceiveRequestVoteRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error
}
