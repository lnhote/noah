package raftrpc

import (
	"github.com/lnhote/noaḥ/server/core"
)

type Raft interface {
	// ExecuteCommand save the command to sate machine
	ExecuteCommand(cmd *core.Command) ([]byte, error)

	// SendAppendEntryRPC is for leader only:
	// 1. append log.
	// 2. send heart beat.
	SendAppendEntryRPC(serverAddr string, req *AppendRPCRequest) (*AppendRPCResponse, error)

	// SendRequestVoteRPC is for candidate only:
	// 1. ask for vote for next leader election term
	SendRequestVoteRPC(serverAddr string, req *RequestVoteRequest) (*RequestVoteResponse, error)
}
