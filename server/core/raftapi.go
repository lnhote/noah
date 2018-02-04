package core

type Raft interface {
	// ExecuteCommand save the command to sate machine
	ExecuteCommand(cmd []byte);


	// SendAppendEntryRPC is for leader only:
	// 1. append log.
	// 2. send heart beat.
	SendAppendEntryRPC();

	// SendRequestVoteRPC is for candidate only:
	// 1. ask for vote for next leader election term
	SendRequestVoteRPC();
}