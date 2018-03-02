package server

import (
	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/lnhote/noaḥ/server/store"
	"github.com/v2pro/plz/countlog"
	"math"
	"net"
	"net/rpc"
	"time"
)

func init() {
	core.CurrentServerState.Role = core.RoleLeader
	for _, addr := range config.GetServerList() {
		core.CurrentServerState.Followers[addr] = &core.FollowerState{}
	}
}

type NoahClusterServer struct {
}

func (ncs *NoahClusterServer) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {
	// 1. During leader election, if the dead leader revived, candidate has to become follower
	// 2. After leader election, When the dead leader comes to live after a new leader is elected,
	// leader has to become follower
	core.CurrentServerState.Role = core.RoleFollower
	// leader is alive, so reset the timer
	ResetLeaderElectionTimer()

	core.CurrentServerState.Term = req.Term
	core.CurrentServerState.CommitIndex = req.CommitIndex
	resp.Addr = config.GetLocalAddr()
	resp.Time = time.Now()
	// check log before accept new logs from leader
	term, err := core.GetLogTerm(req.PrevLogIndex)
	if err != nil {
		countlog.Info("Leader.PrevLogIndex does not exist here", "index", req.PrevLogIndex)
		resp.UnmatchLogIndex = req.PrevLogIndex
		return nil
	}
	if term != req.PrevLogTerm {
		resp.UnmatchLogIndex = req.PrevLogIndex
		return nil
	}
	startLogIndex := req.PrevLogIndex + 1
	for _, log := range req.LogEntries {
		store.SaveLogEntryAtIndex(log, startLogIndex)
		startLogIndex++
	}
	resp.UnmatchLogIndex = math.MaxInt32
	resp.LastLogIndex = startLogIndex - 1
	return nil
}

// OnReceiveRequestVoteRPC will make the election decision: accept or reject
func (ncs *NoahClusterServer) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {
	if core.CurrentServerState.LastVotedTerm > req.NextTerm {
		resp.Accept = false
		return nil
	}
	if core.CurrentServerState.LastVotedTerm == req.NextTerm {
		if core.CurrentServerState.LastVotedServerIp == req.CandidateAddress {
			resp.Accept = true
		} else {
			resp.Accept = false
		}
		return nil
	}
	lastTerm, err := core.GetLogTerm(core.GetLastIndex())
	if err != nil {
		return err
	}
	if req.LastLogTerm > lastTerm {
		resp.Accept = true
	} else if req.LastLogTerm < lastTerm {
		resp.Accept = false
	} else if req.LastLogIndex >= core.GetLastIndex() {
		resp.Accept = true
	} else {
		resp.Accept = false
	}
	if resp.Accept {
		core.CurrentServerState.LastVotedTerm = req.NextTerm
		core.CurrentServerState.LastVotedServerIp = req.CandidateAddress
	}
	return nil
}

func StartNoahClusterServer() {
	addr := config.ClusterPort
	noahClusterServer := new(NoahClusterServer)
	rpc.Register(noahClusterServer)

	//rpc.HandleHTTP()
	//err := http.ListenAndServe(addr, nil)

	var address, _ = net.ResolveTCPAddr("tcp", addr)
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		panic("Fail to start NoahClusterServer: " + err.Error())
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("event!client handler failed to listen to port", "err", err, "addr", addr)
			continue
		}
		rpc.ServeConn(conn)
	}
}
