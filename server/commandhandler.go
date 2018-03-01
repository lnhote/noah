package server

import (
	"errors"

	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
)

// HandleCommand is the main flow
func HandleCommand(cmd *core.Command, raftImpl raftrpc.Raft) ([]byte, error) {
	logIndex := core.SaveToLogs(cmd)

	go sendCommands(cmd, raftImpl)
	for {
		// TODO handle timeout
		if core.CurrentServerState.IsAcceptedByMajority(logIndex) {
			core.UpdateStateMachine(logIndex)
			return core.ExeStateMachineCmd(cmd)
		}
	}

	return []byte("Fail"), nil
}

func HandleCommandMock(cmd *core.Command, raftImpl raftrpc.Raft) (interface{}, error) {
	if cmd.CommandType == core.CmdGet {
		return "5", nil
	}
	if cmd.CommandType == core.CmdSet {
		return "success", nil
	}
	return nil, errors.New("UnknownCmd")
}

func sendCommands(cmd *core.Command, raftImpl raftrpc.Raft) error {
	for ip, _ := range core.CurrentServerState.Followers {
		req := &raftrpc.AppendRPCRequest{}
		req.Ip = ip
		req.Term = core.CurrentServerState.Term
		req.NextIndex = core.CurrentServerState.NextIndex
		req.Log, _ = cmd.ToLog()
		go sendCommand(raftImpl, req)
	}
	return nil
}

func sendCommand(raftImpl raftrpc.Raft, req *raftrpc.AppendRPCRequest) error {
	resp, err := raftImpl.SendAppendEntryRPC(req.Ip, req)
	if err != nil {
		// client may have to retry
		return err
	}
	core.CurrentServerState.Collect(resp.Ip, resp.LastLogIndex)
	return nil
}
