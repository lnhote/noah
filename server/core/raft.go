package core

import (
	"github.com/lnhote/noaḥ/server/command"
	"github.com/lnhote/noaḥ/server/rpc"
)

type SimpleRaft struct {
}

// ExecuteCommand save the command to sate machine
func (s SimpleRaft) ExecuteCommand(cmd *command.Command) ([]byte, error) {
	return []byte("OK"), nil
}

// SendAppendEntryRPC is for leader only:
// 1. append log.
// 2. send heart beat.
func (s SimpleRaft) SendAppendEntryRPC(req *rpc.AppendRPCRequest) (*rpc.AppendRPCResponse, error) {
	return nil, nil
}

// SendRequestVoteRPC is for candidate only:
// 1. ask for vote for next leader election term
func (s SimpleRaft) SendRequestVoteRPC(req *rpc.RequestVoteRequest) (*rpc.RequestVoteResponse, error) {
	return nil, nil
}
