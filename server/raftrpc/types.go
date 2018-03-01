package raftrpc

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
	LastLogTerm  int
	LastLogIndex int
	NextTerm     int

	// ip address
	CandidateAddress string
}

type RequestVoteResponse struct {
	Accept bool
}
