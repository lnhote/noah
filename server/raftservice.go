package server

import (
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/server/raftrpc"
)

type RaftService struct {
	server *RaftServer
}

func NewRaftService(server *RaftServer) *RaftService {
	return &RaftService{
		server: server,
	}
}

func (rs *RaftService) Get(cmd *core.Command, resp *core.ClientResponse) error {
	return rs.server.Get(cmd, resp)
}

func (rs *RaftService) Set(cmd *core.Command, resp *core.ClientResponse) error {
	return rs.server.Set(cmd, resp)
}

func (rs *RaftService) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {
	return rs.server.OnReceiveAppendRPC(req, resp)
}

func (rs *RaftService) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {
	return rs.server.OnReceiveRequestVoteRPC(req, resp)
}
