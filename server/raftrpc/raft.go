package raftrpc

import (
	"fmt"
	"net/rpc"

	"github.com/lnhote/noaḥ/server/core"
)

type SimpleRaft struct {
}

// ExecuteCommand save the command to sate machine
func (s SimpleRaft) ExecuteCommand(cmd *core.Command) ([]byte, error) {
	return []byte("OK"), nil
}

// SendAppendEntryRPC is for leader only:
// 1. append log.
// 2. send heart beat.
func (s SimpleRaft) SendAppendEntryRPC(serverAddr string, req *AppendRPCRequest) (*AppendRPCResponse, error) {
	var client, err = rpc.Dial("tcp", serverAddr)
	if err != nil {
		println("Dial failed:", err.Error())
	}
	var resp AppendRPCResponse
	err = client.Call("NoahClusterServer.OnReceiveAppendRPC", req, &resp)
	if err != nil {
		fmt.Printf("%s Fail: %s", "NoahClusterServer.OnReceiveAppendRPC", err.Error())
		return nil, err
	}
	return &resp, nil
}

// SendRequestVoteRPC is for candidate only:
// 1. ask for vote for next leader election term
func (s SimpleRaft) SendRequestVoteRPC(serverAddr string, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	var client, err = rpc.Dial("tcp", serverAddr)
	if err != nil {
		println("Dial failed:", err.Error())
	}
	var resp RequestVoteResponse
	err = client.Call("NoahClusterServer.OnReceiveRequestVoteRPC", req, &resp)
	if err != nil {
		fmt.Printf("%s Fail: %s", "NoahClusterServer.OnReceiveRequestVoteRPC", err.Error())
		return nil, err
	}
	return &resp, nil
}
