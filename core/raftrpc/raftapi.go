package raftrpc

import (
	"github.com/lnhote/noah/core/entity"
)

type RaftApi interface {
	Get(cmd *entity.Command, resp *ClientResponse) error
	Set(cmd *entity.Command, resp *ClientResponse) error
	OnReceiveAppendRPC(req *AppendRPCRequest, resp *AppendRPCResponse) error
	OnReceiveRequestVoteRPC(req *RequestVoteRequest, resp *RequestVoteResponse) error
}
