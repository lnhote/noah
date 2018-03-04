package store

import (
	"fmt"

	"github.com/lnhote/noaḥ/server/core"
)

// SaveLogEntry add the command to uncommited log list
// return the log index for this entry
func SaveLogEntry(log *core.LogEntry) int {
	log.Index = core.CurrentServerState.NextIndex
	return SaveLogEntryAtIndex(log, log.Index)
}

func SaveLogEntryAtIndex(log *core.LogEntry, logIndex int) int {
	log.Index = logIndex
	core.SaveLogEntry(log)
	core.CurrentServerState.NextIndex = logIndex + 1
	return logIndex
}

// ExecuteLogAndUpdateStateMachine update the state machine, e.g., rocksdb
func ExecuteLogAndUpdateStateMachine(indexStart int, indexEnd int) ([]byte, error) {
	var result []byte
	for i := indexStart; i <= indexEnd; i++ {
		log, err := core.GetLogEntry(i)
		if err != nil {
			return nil, err
		}
		result, err = exeStateMachineCmd(log.Command)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// exeStateMachineCmd execute the command
func exeStateMachineCmd(cmd *core.Command) ([]byte, error) {
	switch cmd.CommandType {
	case core.CmdSet:
		return nil, DBSet(cmd.Key, cmd.Value)
	case core.CmdGet:
		return DBGet(cmd.Key)
	default:
		return nil, fmt.Errorf("UnkownCommandType(%d)", cmd.CommandType)
	}
}
