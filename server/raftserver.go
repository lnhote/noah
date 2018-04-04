package server

import (
	"errors"
	"fmt"
	"math"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/errorcode"
	"github.com/lnhote/noah/server/raftrpc"
	"github.com/lnhote/noah/server/store"
	"github.com/v2pro/plz/countlog"
)

type RaftServer struct {
	ServerConf   *core.ServerConfig
	ServerInfo   *core.ServerState
	FollowerInfo *core.FollowerState
	Logs         *core.LogList

	leaderHeartBeatTicker *time.Ticker
	HeartBeatDurationInMs int

	leaderElectionTimer        *time.Timer
	LeaderElectionDurationInMs int

	countMutex      *sync.Mutex
	wg              *sync.WaitGroup
	wgHandleCommand *sync.WaitGroup
	acceptedCount   int
	ackCount        int

	sigs     chan os.Signal
	stopSign chan int
}

func NewRaftServer(conf *core.ServerConfig) *RaftServer {
	return NewRaftServerWithEnv(conf, DefaultEnv)
}

func NewRaftServerWithEnv(conf *core.ServerConfig, env *Env) *RaftServer {
	newServer := &RaftServer{}
	newServer.ServerConf = conf
	newServer.Logs = core.NewLogList()
	newServer.ServerInfo = core.NewServerState()
	newServer.countMutex = &sync.Mutex{}
	newServer.wg = &sync.WaitGroup{}
	newServer.wgHandleCommand = &sync.WaitGroup{}
	newServer.acceptedCount = 0
	newServer.ackCount = 0

	newServer.LeaderElectionDurationInMs = env.LeaderElectionDurationInMs
	newServer.HeartBeatDurationInMs = env.HeartBeatDurationInMs
	newServer.leaderHeartBeatTicker = time.NewTicker(time.Duration(newServer.HeartBeatDurationInMs) * time.Millisecond)
	newServer.resetLeaderElectionTimer()

	for _, addr := range newServer.ServerConf.ClusterAddrList {
		if newServer.ServerConf.Info.ServerId == addr.ServerId {
			continue
		}
		newServer.ServerInfo.Followers[addr.ServerId] = &core.FollowerState{Node: addr}
	}
	newServer.sigs = make(chan os.Signal, 1)
	// SIGINT：interrupt/Ctrl-C
	// SIGTERM: kill, can be block
	// SIGKILL: kill
	// SIGUSR1, SIGUSR2: user defined
	// SIGSTOP
	signal.Notify(newServer.sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
	newServer.stopSign = make(chan int, 1)
	return newServer
}

func (s *RaftServer) Start() {
	go s.startLeaderElectionTimer()

	go s.startHeartbeatTimer()

	s.startServe()
}

func (s *RaftServer) Stop() {
	s.stopSign <- 1
}

func (s *RaftServer) handleSignal(sig os.Signal) {
	switch sig {
	case syscall.SIGINT:
		countlog.Info(fmt.Sprintf("handle signal SIGINT: %+v", sig))
	case syscall.SIGTERM:
		countlog.Info(fmt.Sprintf("handle signal SIGTERM: %+v", sig))
	case syscall.SIGKILL:
		countlog.Info(fmt.Sprintf("handle signal SIGKILL: %+v", sig))
	case syscall.SIGSTOP:
		countlog.Info(fmt.Sprintf("handle signal SIGSTOP: %+v", sig))
	case syscall.SIGUSR1:
		countlog.Info(fmt.Sprintf("handle signal START: %+v", sig))
	case syscall.SIGUSR2:
		countlog.Info(fmt.Sprintf("handle signal STOP: %+v", sig))
		panic(fmt.Sprintf("%s stop", s))
	default:
		countlog.Info(fmt.Sprintf("handle signal OTHER: %+v", sig))
	}
}

func (s *RaftServer) startServe() {
	rpcServer := rpc.NewServer()
	rpcServer.Register(NewRaftService(s))
	listener, err := net.ListenTCP("tcp", s.ServerConf.Info.ServerAddr)
	if err != nil {
		panic("Fail to start RaftServer: " + err.Error())
	}
	countlog.Info(fmt.Sprintf("%s started", s.ServerConf.Info))
	for {
		select {
		case sig := <-s.sigs:
			s.handleSignal(sig)
		case <-s.stopSign:
			countlog.Info(fmt.Sprintf("%s stoped", s))
			s.leaderHeartBeatTicker.Stop()
			s.leaderElectionTimer.Stop()
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				countlog.Error("RaftServer is busy", "error", err)
				continue
			}
			countlog.Debug(fmt.Sprintf("%s localPort %s, remotePort %s", s.String(), conn.LocalAddr(), conn.RemoteAddr()))
			rpcServer.ServeConn(conn)
		}
	}
}

// leader will send heart beat to followers periodically
func (s *RaftServer) startHeartbeatTimer() {
	for {
		<-s.leaderHeartBeatTicker.C
		if s.ServerConf.Info.Role == core.RoleLeader {
			s.SendHeartbeat()
		}
	}
}

// leader send empty AppendRPC request to followers
func (s *RaftServer) SendHeartbeat() {
	countlog.Info(fmt.Sprintf("%s SendHeartbeat", s))

	for _, node := range s.ServerConf.ClusterAddrList {
		if s.ServerConf.Info.ServerId == node.ServerId {
			continue
		}
		lastTime := s.ServerInfo.Followers[node.ServerId].LastRpcTime
		if time.Now().Sub(lastTime) < time.Duration(s.HeartBeatDurationInMs)/2*time.Millisecond {
			// Don't send heart beat if we already sent any log recently
			countlog.Info(fmt.Sprintf("No need to send heartbeat to node %s: lastTime = %v", node.String(), lastTime))
			continue
		}
		req := &raftrpc.AppendRPCRequest{}
		req.LogEntries = []*core.LogEntry{}
		req.Term = s.ServerInfo.Term
		req.LeaderNode = s.ServerConf.Info
		go func(r *raftrpc.AppendRPCRequest, followerAddr *net.TCPAddr) {
			s.sendHeartbeatToServer(r, followerAddr)
		}(req, node.ServerAddr)
	}
}

func (s *RaftServer) sendHeartbeatToServer(req *raftrpc.AppendRPCRequest, followerAddr *net.TCPAddr) {
	countlog.Info(fmt.Sprintf("%s sendHeartbeat to %s", s, followerAddr))
	client, err := rpc.Dial("tcp", followerAddr.String())
	if err != nil {
		countlog.Error("Connect Error", "error", err, "followerAddr", followerAddr.String())
		return
	}
	var resp raftrpc.AppendRPCResponse
	if err = client.Call("RaftService.OnReceiveAppendRPC", req, &resp); err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "followerAddr", followerAddr.String(), "error", err.Error())
		return
	}
	countlog.Info(fmt.Sprintf("HeartbeatResponse from %s", resp.Node))
	s.ServerInfo.Followers[resp.Node.ServerId].LastRpcTime = resp.Time
}

func (s *RaftServer) resetLeaderElectionTimer() {
	if s.leaderElectionTimer == nil {
		s.leaderElectionTimer = time.NewTimer(time.Duration(s.LeaderElectionDurationInMs) * time.Millisecond)
	}
	s.leaderElectionTimer.Stop()
	s.leaderElectionTimer.Reset(time.Duration(s.LeaderElectionDurationInMs) * time.Millisecond)
}

// follower will start leader election on time out
func (s *RaftServer) startLeaderElectionTimer() {
	for {
		<-s.leaderElectionTimer.C
		if s.ServerConf.Info.Role == core.RoleLeader {
			continue
		}
		s.resetLeaderElectionTimer()
		s.StartElection()
	}
}

func (s *RaftServer) StartElection() {
	countlog.Info(fmt.Sprintf("%s started leader election for term %d, old leader %s is dead",
		s.String(), s.ServerInfo.Term+1, s.ServerConf.LeaderInfo))
	if s.ServerConf.Info.Role == core.RoleFollower {
		becomeCandidate(s)
	}
	if s.ServerConf.Info.Role == core.RoleCandidate {
		s.sendRequestVoteToAll()
	}
}

func (s *RaftServer) sendRequestVoteToAll() {
	countlog.Info(fmt.Sprintf("%s sendRequestVoteToAll", s.String()))
	for s.ServerInfo.Role == core.RoleCandidate {
		req := &raftrpc.RequestVoteRequest{}
		req.Candidate = s.ServerConf.Info
		req.NextTerm = s.ServerInfo.Term + 1
		s.ServerInfo.LastVotedTerm = req.NextTerm
		s.ServerInfo.LastVotedServerId = s.ServerConf.Info.ServerId
		s.acceptedCount = 1
		s.ackCount = 1
		wg := &sync.WaitGroup{}
		for _, addr := range s.ServerConf.ClusterAddrList {
			if addr.ServerId == s.ServerConf.Info.ServerId {
				continue
			}
			if addr.ServerId == s.ServerConf.LeaderInfo.ServerId {
				continue
			}
			wg.Add(1)
			go s.getVoteFromServer(addr, req, wg)
		}
		countlog.Info("wait")
		wg.Wait()

		countlog.Info("wait done")
		if s.acceptedCount*2 > len(s.ServerConf.ClusterAddrList) {
			becomeLeader(s)
			countlog.Info(fmt.Sprintf("%s becomes new leader", s))
			s.SendHeartbeat()
		} else {
			sleepBeforeVote(s)
		}
		// role becomes Follower when receiving AppendLogRPC
	}
	return
}

func (s *RaftServer) getVoteFromServer(node *core.ServerInfo, req *raftrpc.RequestVoteRequest, wg *sync.WaitGroup) {
	countlog.Info(fmt.Sprintf("%s getVoteFromServer from %s", s, node))
	defer func() {
		countlog.Info(fmt.Sprintf("%s Done", s))
		wg.Done()
	}()
	resp, err := raftrpc.SendRequestVoteRPC(node.ServerAddr, req)
	if err != nil {
		countlog.Info(fmt.Sprintf("%s Err %s", s, err))
		return
	}
	countlog.Info(fmt.Sprintf("%s vote result from %s: %+v", s, node, resp))
	s.countMutex.Lock()
	s.ackCount++
	if resp.Accept {
		s.acceptedCount++
	}
	s.countMutex.Unlock()
	countlog.Info(fmt.Sprintf("%s finish", s))
	return
}

func (s *RaftServer) String() string {
	return s.ServerConf.Info.String()
}

func (s *RaftServer) Get(cmd *core.Command, resp *core.ClientResponse) error {
	countlog.Info("Get", "cmd", cmd)
	if cmd == nil {
		err := errors.New("EmptyCommand")
		countlog.Error("event!EmptyCommand", "err", err.Error())
		return err
	}
	if s.ServerConf.Info.Role != core.RoleLeader {
		data := map[string]interface{}{
			"leader_address": s.ServerConf.Info.ServerAddr,
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	switch cmd.CommandType {
	case core.CmdGet:
		val, err := store.DBGet(cmd.Key)
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"value": val,
		}
		resp.Code = errorcode.Success
		resp.Data = data
	default:
		return fmt.Errorf("UnkownCommandType(%d)", cmd.CommandType)
	}
	return nil
}

func (s *RaftServer) Set(cmd *core.Command, resp *core.ClientResponse) error {
	countlog.Info("Set", "cmd", cmd)
	*resp = core.ClientResponse{}
	if cmd == nil {
		err := errors.New("EmptyCommand")
		countlog.Error("event!EmptyCommand", "err", err.Error())
		return err
	}
	if s.ServerConf.Info.Role != core.RoleLeader {
		data := map[string]interface{}{
			"leader_address": s.ServerInfo.LeaderId,
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	val, err := s.AppendLog(cmd)
	if err != nil {
		return err
	}
	data := map[string]interface{}{
		"value": val,
	}
	resp.Code = errorcode.Success
	resp.Data = data
	return nil
}

func (s *RaftServer) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {
	countlog.Info(fmt.Sprintf("%s OnReceiveAppendRPC from %s", s.ServerConf.Info, req.LeaderNode.ServerAddr))
	// 1. During leader election, if the dead leader revived, candidate has to become follower
	// 2. After leader election, When the dead leader comes to live after a new leader is elected,
	// leader has to become follower
	becomeFollower(s)
	// leader is alive, so reset the timer
	s.resetLeaderElectionTimer()

	s.ServerInfo.Term = req.Term
	s.ServerInfo.CommitIndex = req.CommitIndex
	resp.Node = s.ServerConf.Info
	resp.Time = time.Now()
	// check log before accept new logs from leader
	term, err := s.Logs.GetLogTerm(req.PrevLogIndex)
	if err != nil && req.NextIndex != req.PrevLogIndex {
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
func (s *RaftServer) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {
	countlog.Info(fmt.Sprintf("%s OnReceiveRequestVoteRPC from %s", s, req.Candidate))
	if s.ServerInfo.LastVotedTerm > req.NextTerm {
		resp.Accept = false
		return nil
	}
	if s.ServerInfo.LastVotedTerm == req.NextTerm {
		if s.ServerInfo.LastVotedServerId == req.Candidate.ServerId {
			resp.Accept = true
		} else {
			resp.Accept = false
		}
		return nil
	}
	lastTerm, err := s.Logs.GetLogTerm(s.ServerInfo.GetLastIndex())
	if err != nil {
		return err
	}
	if req.LastLogTerm > lastTerm {
		resp.Accept = true
	} else if req.LastLogTerm < lastTerm {
		resp.Accept = false
	} else if req.LastLogIndex >= s.ServerInfo.GetLastIndex() {
		resp.Accept = true
	} else {
		resp.Accept = false
	}
	if resp.Accept {
		s.ServerInfo.LastVotedTerm = req.NextTerm
		s.ServerInfo.LastVotedServerId = req.Candidate.ServerId
	}
	return nil
}

// AppendLog is the main flow
func (s *RaftServer) AppendLog(cmd *core.Command) ([]byte, error) {
	countlog.Info("AppendLog", "cmd", cmd)
	newLog := &core.LogEntry{cmd, s.ServerInfo.NextIndex, s.ServerInfo.Term}
	logIndex := store.SaveLogEntry(newLog)
	s.wgHandleCommand.Add(len(s.ServerInfo.Followers))
	for _, follower := range s.ServerInfo.Followers {
		req := &raftrpc.AppendRPCRequest{}
		req.LeaderNode = s.ServerConf.Info
		req.Term = s.ServerInfo.Term
		req.PrevLogIndex = s.ServerInfo.NextIndex - 1
		req.PrevLogTerm, _ = s.Logs.GetLogTerm(req.PrevLogIndex)
		req.NextIndex = s.ServerInfo.NextIndex
		req.CommitIndex = s.ServerInfo.CommitIndex
		req.LogEntries = append(req.LogEntries, newLog)
		go s.replicateLogToServer(follower.Node.ServerAddr, req)
	}
	s.wgHandleCommand.Wait()
	if s.ServerInfo.IsAcceptedByMajority(logIndex) {
		newCommitIndex := logIndex
		lastCommitIndex := s.ServerInfo.CommitIndex
		val, err := store.ExecuteLogAndUpdateStateMachine(lastCommitIndex+1, newCommitIndex)
		if err != nil {
			return nil, err
		}
		s.ServerInfo.CommitIndex = newCommitIndex
		return val, nil
	} else {
		return nil, fmt.Errorf("ValueNotAccepcted")
	}
}

func (s *RaftServer) AppendLogMock(cmd *core.Command) (interface{}, error) {
	if cmd.CommandType == core.CmdGet {
		return "5", nil
	}
	if cmd.CommandType == core.CmdSet {
		return "success", nil
	}
	return nil, errors.New("UnknownCmd")
}

func (s *RaftServer) replicateLogToServer(followerAddr *net.TCPAddr, req *raftrpc.AppendRPCRequest) error {
	defer s.wgHandleCommand.Done()
	resp, err := raftrpc.SendAppendEntryRPC(followerAddr, req)
	if err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "followerAddr", followerAddr.String(), "error", err)
		return err
	}
	for resp.UnmatchLogIndex <= req.PrevLogIndex {
		req.PrevLogIndex = resp.UnmatchLogIndex - 1
		req.PrevLogTerm, _ = s.Logs.GetLogTerm(req.PrevLogIndex)
		logs := []*core.LogEntry{}
		for i := req.PrevLogIndex + 1; i < req.NextIndex; i++ {
			log, _ := s.Logs.GetLogEntry(i)
			logs = append(logs, log)
		}
		req.LogEntries = logs
		if resp, err = raftrpc.SendAppendEntryRPC(followerAddr, req); err != nil {
			countlog.Error("SendAppendEntryRPC Fail", "followerAddr", followerAddr.String(), "error", err)
			return err
		}
	}
	s.ServerInfo.Followers[resp.Node.ServerId].LastIndex = resp.LastLogIndex
	s.ServerInfo.Followers[resp.Node.ServerId].LastRpcTime = resp.Time
	s.ServerInfo.Followers[resp.Node.ServerId].MatchedIndex = resp.LastLogIndex
	return nil
}
