package raftrpc

import (
	"github.com/lnhote/noah/core/entity"
)

type RaftService struct {
	server RaftApi
}

func NewRaftService(server RaftApi) *RaftService {
	return &RaftService{
		server: server,
	}
}

func (rs *RaftService) Get(cmd *entity.Command, resp *ClientResponse) error {
	return rs.server.Get(cmd, resp)
}

func (rs *RaftService) Set(cmd *entity.Command, resp *ClientResponse) error {
	return rs.server.Set(cmd, resp)
}

func (rs *RaftService) OnReceiveAppendRPC(req *AppendRPCRequest, resp *AppendRPCResponse) error {
	return rs.server.OnReceiveAppendRPC(req, resp)
}

func (rs *RaftService) OnReceiveRequestVoteRPC(req *RequestVoteRequest, resp *RequestVoteResponse) error {
	return rs.server.OnReceiveRequestVoteRPC(req, resp)
}
