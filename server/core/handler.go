package core

import (
	"github.com/lnhote/noaḥ/server/command"
	"github.com/lnhote/noaḥ/server/rpc"
)

// HandleCommand is the main flow
func HandleCommand(cmd *command.Command, raftImpl Raft) ([]byte, error) {
	logIndex := command.SaveToLogs(cmd)

	go sendCommands(cmd, raftImpl)
	// TODO handle timeout
	for {
		if CurrentServerState.isAcceptedByMajority(logIndex) {
			command.UpdateStateMachine(logIndex)
			return command.ExeStateMachineCmd(cmd)
		}
	}

	return []byte("Fail"), nil
}

func sendCommands(cmd *command.Command, raftImpl Raft) error {
	for ip, _ := range CurrentServerState.Followers {
		req := &rpc.AppendRPCRequest{}
		req.Ip = ip
		req.Term = CurrentServerState.Term
		req.NextIndex = CurrentServerState.NextIndex
		req.Log, _ = cmd.ToLog()
		go sendCommand(raftImpl, req)
	}
	return nil
}

func sendCommand(raftImpl Raft, req *rpc.AppendRPCRequest) error {
	resp, err := raftImpl.SendAppendEntryRPC(req)
	if err != nil {
		// client may have to retry
		return err
	}
	CurrentServerState.collect(resp.Ip, resp.LastLogIndex)
	return nil
}
