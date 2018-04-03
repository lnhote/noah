package raftrpc

import (
	"fmt"
	"net"
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
func SendAppendEntryRPC(followerAddr *net.TCPAddr, req *AppendRPCRequest) (*AppendRPCResponse, error) {
	countlog.Info(fmt.Sprintf("%s SendAppendEntryRPC to %s", req.LeaderNode, followerAddr.String()))
	var client, err = rpc.Dial("tcp", followerAddr.String())
	if err != nil {
		countlog.Error("SendAppendEntryRPC Connect Error", "error", err.Error(), "serverAddr", followerAddr.String())
		return nil, err
	}
	var resp AppendRPCResponse
	if err = client.Call("RaftService.OnReceiveAppendRPC", req, &resp); err != nil {
		countlog.Error("RaftService.OnReceiveAppendRPC Fail", "error", err.Error())
		return nil, err
	}
	return &resp, nil
}

// SendRequestVoteRPC is for candidate only:
// 1. ask for vote for next leader election term
func SendRequestVoteRPC(serverAddr *net.TCPAddr, req *RequestVoteRequest) (*RequestVoteResponse, error) {
	countlog.Info(fmt.Sprintf("%s SendRequestVoteRPC to %s", req.Candidate, serverAddr))
	var client, err = rpc.Dial("tcp", serverAddr.String())
	if err != nil {
		countlog.Error("SendRequestVoteRPC Connect Error", "error", err.Error(), "serverAddr", serverAddr.String())
		return nil, err
	}
	var resp RequestVoteResponse
	if err = client.Call("RaftService.OnReceiveRequestVoteRPC", req, &resp); err != nil {
		countlog.Error("RaftService.OnReceiveRequestVoteRPC Fail", "error", err.Error())
		return nil, err
	}
	return &resp, nil
}
