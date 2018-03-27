package raftrpc

import (
	"net/rpc"

	"github.com/lnhote/noaḥ/core"
	"github.com/v2pro/plz/countlog"
)

// ExecuteCommand save the command to sate machine
func ExecuteCommand(cmd *core.Command) ([]byte, error) {
	return []byte("OK"), nil
}

// SendAppendEntryRPC is for leader only:
// 1. append log.
// 2. send heart beat.
func SendAppendEntryRPC(node *core.ServerInfo, req *AppendRPCRequest) (*AppendRPCResponse, error) {
	var client, err = rpc.Dial("tcp", node.ServerAddr.String())
	if err != nil {
		countlog.Error("SendAppendEntryRPC Connect Error", "error", err.Error(), "serverAddr", node.ServerAddr.String())
		return nil, err
	}
	var resp AppendRPCResponse
	if err = client.Call("NoahClusterServer.OnReceiveAppendRPC", req, &resp); err != nil {
		countlog.Error("NoahClusterServer.OnReceiveAppendRPC Fail", "error", err.Error())
		return nil, err
	}
	return &resp, nil
}

// SendRequestVoteRPC is for candidate only:
// 1. ask for vote for next leader election term
func SendRequestVoteRPC(node *core.ServerInfo, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	serverAddr := node
	var client, err = rpc.Dial("tcp", node.ServerAddr.String())
	if err != nil {
		countlog.Error("SendRequestVoteRPC Connect Error", "error", err.Error(), "serverAddr", serverAddr)
	}
	var resp RequestVoteResponse
	err = client.Call("NoahClusterServer.OnReceiveRequestVoteRPC", req, &resp)
	if err != nil {
		countlog.Error("NoahClusterServer.OnReceiveRequestVoteRPC Fail", "error", err.Error())
		return nil, err
	}
	return &resp, nil
}
