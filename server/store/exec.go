package store

import (
	"github.com/lnhote/noaḥ/server/core"
)

// ExecuteCommand execute the command
func ExeStateMachineCmd(cmd *core.Command) ([]byte, error) {

	//
	return []byte("OK"), nil
}

// UpdateCommitIndex update the state machine, e.g., rocksdb
func UpdateCommitIndex(index int) ([]byte, error) {

	core.CurrentServerState.CommitIndex = index
	return []byte("OK"), nil
}

// SaveLogEntry add the command to uncommited log list
// return the log index for this entry
func SaveLogEntry(log *core.LogEntry) int {
	logIndex := core.CurrentServerState.NextIndex
	core.LogsToCommit[logIndex] = log
	core.CurrentServerState.NextIndex = logIndex + 1
	return logIndex
}

func SaveLogEntryAtIndex(log *core.LogEntry, logIndex int) int {
	core.LogsToCommit[logIndex] = log
	core.CurrentServerState.NextIndex = logIndex + 1
	return logIndex
}
