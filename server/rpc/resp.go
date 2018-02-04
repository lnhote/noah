package rpc

type AppendRPCRequest struct {
	Ip        string
	Log       string
	NextIndex int
	Term      int
}

type AppendRPCResponse struct {
	Ip string

	LastLogIndex int

	LastLogTerm int

	Error string
}

type RequestVoteRequest struct {
}

type RequestVoteResponse struct {
}
