package server

import (
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/server/raftrpc"
)

type RaftApi interface {
	Get(cmd *core.Command, resp *core.ClientResponse) error
	Set(cmd *core.Command, resp *core.ClientResponse) error
	OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error
	OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error
}
