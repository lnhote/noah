package core

// ExecuteCommand execute the command and update the state machine
// state machine is rocksdb
func UpdateStateMachine(index int) ([]byte, error) {

	//
	return []byte("OK"), nil
}

func ExeStateMachineCmd(cmd *Command) ([]byte, error) {

	//
	return []byte("OK"), nil
}

// SaveToLogs add the command to uncommited log list
// return the log index for this entry
func SaveToLogs(cmd *Command) int {
	logIndex := CurrentServerState.NextIndex
	LogsToCommit[logIndex] = cmd
	CurrentServerState.NextIndex++
	return logIndex
}
