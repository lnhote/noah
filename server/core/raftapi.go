package core

import (
	"github.com/lnhote/noaḥ/server/command"
	"github.com/lnhote/noaḥ/server/rpc"
)

type Raft interface {
	// ExecuteCommand save the command to sate machine
	ExecuteCommand(cmd *command.Command) ([]byte, error)

	// SendAppendEntryRPC is for leader only:
	// 1. append log.
	// 2. send heart beat.
	SendAppendEntryRPC(req *rpc.AppendRPCRequest) (*rpc.AppendRPCResponse, error)

	// SendRequestVoteRPC is for candidate only:
	// 1. ask for vote for next leader election term
	SendRequestVoteRPC(req *rpc.RequestVoteRequest) (*rpc.RequestVoteResponse, error)
}
