package server

import (
	"fmt"
	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/v2pro/plz/countlog"
	"net"
	"net/rpc"
	"time"
)

var (
	leaderElectionTimer   = time.NewTimer(time.Duration(20) * time.Millisecond)
	leaderHeartBeatTicker = time.NewTicker(500 * time.Millisecond)
)

// leader will send heart beat to followers periodically
func StartHeartbeatTimer() {
	for {
		<-leaderHeartBeatTicker.C
		if core.CurrentServerState.Role == core.RoleLeader {
			SendHearbeat()
		}
	}
}

// follower will start leader election on time out
func StartLeaderElectionTimer() {
	for {
		<-leaderElectionTimer.C
		leaderElectionTimer.Reset(time.Duration(20) * time.Millisecond)
		if core.CurrentServerState.Role == core.RoleFollower {
			countlog.Info("event!can't hear from leader, convert to candidate")
			core.CurrentServerState.Role = core.RoleCandidate
			// 20 - 100ms
			SendRequestVoteToAll()
		}
	}
}

type NoahClusterServer struct {
}

func (ncs *NoahClusterServer) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {

	// TODO check log before accept new logs from leader
	return nil
}

func (ncs *NoahClusterServer) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {

	// TODO accept or reject
	return nil
}

func StartNoahClusterServer() {
	addr := config.ClusterPort
	noahClusterServer := new(NoahClusterServer)
	rpc.Register(noahClusterServer)

	//rpc.HandleHTTP()
	//err := http.ListenAndServe(addr, nil)
	//if err != nil {
	//	return
	//}

	var address, _ = net.ResolveTCPAddr("tcp", addr)
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		fmt.Println("fail to start NoahClusterServer", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("event!client handler failed to listen to port", "err", err, "addr", addr)
			continue
		}
		fmt.Println("get a request")
		rpc.ServeConn(conn)
	}
}
