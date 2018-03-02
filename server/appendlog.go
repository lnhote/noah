package server

import (
	"errors"
	"fmt"
	"sync"

	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/lnhote/noaḥ/server/store"
	"github.com/v2pro/plz/countlog"
)

var (
	wgHandleCommand = &sync.WaitGroup{}
)

// AppendLog is the main flow
func AppendLog(cmd *core.Command) ([]byte, error) {
	logIndex := store.SaveLogEntry(cmd)
	wgHandleCommand.Add(len(core.CurrentServerState.Followers))
	for addr, _ := range core.CurrentServerState.Followers {
		req := &raftrpc.AppendRPCRequest{}
		req.Addr = addr
		req.Term = core.CurrentServerState.Term
		req.PrevLogIndex = core.CurrentServerState.NextIndex - 1
		req.PrevLogTerm, _ = core.GetLogTerm(req.PrevLogIndex)
		req.NextIndex = core.CurrentServerState.NextIndex
		req.LogEntries = append(req.LogEntries, cmd)
		go replicateLogToServer(req)
	}
	wgHandleCommand.Wait()
	if core.IsAcceptedByMajority(logIndex) {
		core.CurrentServerState.CommitIndex = logIndex
		return store.ExeStateMachineCmd(cmd)
	} else {
		return nil, fmt.Errorf("ValueNotAccepcted")
	}
}

func AppendLogMock(cmd *core.Command) (interface{}, error) {
	if cmd.CommandType == core.CmdGet {
		return "5", nil
	}
	if cmd.CommandType == core.CmdSet {
		return "success", nil
	}
	return nil, errors.New("UnknownCmd")
}

func replicateLogToServer(req *raftrpc.AppendRPCRequest) error {
	defer wgHandleCommand.Done()
	resp, err := raftrpc.SendAppendEntryRPC(req.Addr, req)
	if err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "addr", req.Addr, "error", err)
		return err
	}
	for resp.UnmatchLogIndex <= req.PrevLogIndex {
		req.PrevLogIndex = resp.UnmatchLogIndex - 1
		req.PrevLogTerm, _ = core.GetLogTerm(req.PrevLogIndex)
		cmds := []*core.Command{}
		for i := req.PrevLogIndex + 1; i < req.NextIndex; i++ {
			cmds = append(cmds, core.LogsToCommit[i])
		}
		req.LogEntries = cmds
		if resp, err = raftrpc.SendAppendEntryRPC(req.Addr, req); err != nil {
			countlog.Error("SendAppendEntryRPC Fail", "addr", req.Addr, "error", err)
			return err
		}
	}
	core.CurrentServerState.Followers[resp.Addr].LastIndex = resp.LastLogIndex
	core.CurrentServerState.Followers[resp.Addr].LastRpcTime = resp.Time
	return nil
}
