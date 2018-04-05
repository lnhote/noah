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

const (
	SUCCESS = 1
	FAIL    = 2
)

type RaftServer struct {
	ServerConf   *core.ServerConfig
	ServerInfo   *core.ServerState
	FollowerInfo *core.FollowerState
	Logs         *core.LogList

	leaderHeartBeatTimer *eventTimer
	leaderElectionTimer  *eventTimer

	wgHandleCommand *sync.WaitGroup

	sigs       chan os.Signal
	stopSignal chan int

	clientPool *raftrpc.ClientPool
}

func NewRaftServer(conf *core.ServerConfig) *RaftServer {
	return NewRaftServerWithEnv(conf, DefaultEnv)
}

func NewRaftServerWithEnv(conf *core.ServerConfig, env *Env) *RaftServer {
	newServer := &RaftServer{}
	newServer.ServerConf = conf
	newServer.Logs = core.NewLogList()
	newServer.ServerInfo = core.NewServerState()
	newServer.wgHandleCommand = &sync.WaitGroup{}
	newServer.leaderElectionTimer = NewEventTimer(newServer.LeaderElectionEventHandler, env.LeaderElectionDurationInMs)
	newServer.leaderHeartBeatTimer = NewEventTimer(newServer.HeartbeatEventHandler, env.HeartBeatDurationInMs)
	for _, addr := range newServer.ServerConf.ClusterAddrList {
		if newServer.ServerConf.Info.ServerId == addr.ServerId {
			continue
		}
		newServer.ServerInfo.Followers[addr.ServerId] = &core.FollowerState{Node: addr}
	}
	newServer.clientPool = raftrpc.NewClientPool()
	newServer.sigs = make(chan os.Signal, 1)
	// SIGINT：interrupt/Ctrl-C
	// SIGTERM: kill, can be block
	// SIGKILL: kill
	// SIGUSR1, SIGUSR2: user defined
	// SIGSTOP
	signal.Notify(newServer.sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
	newServer.stopSignal = make(chan int, 1)
	return newServer
}

func (s *RaftServer) String() string {
	return s.ServerConf.Info.String()
}

// Start runs a raft server
func (s *RaftServer) Start() {
	s.leaderElectionTimer.Start()
	s.leaderHeartBeatTimer.Start()
	s.startServe()
}

// Stop shuts down a raft server
func (s *RaftServer) Stop() {
	s.stopSignal <- 1
	s.leaderElectionTimer.Start()
	s.leaderHeartBeatTimer.Start()
}

// startServe starts listening to port and handles requests
func (s *RaftServer) startServe() {
	rpcServer := rpc.NewServer()
	rpcServer.Register(NewRaftService(s))
	listener, err := net.ListenTCP("tcp", s.ServerConf.Info.ServerAddr)
	if err != nil {
		panic("Fail to start RaftServer: " + err.Error())
	}
	//rpc.Accept(listener)
	countlog.Info(fmt.Sprintf("%s started", s.ServerConf.Info))
stopServeLoop:
	for {
		select {
		case sig := <-s.sigs:
			s.handleSignal(sig)
			break stopServeLoop
		case <-s.stopSignal:
			stopTimers(s)
			break stopServeLoop
		default:
			conn, err := listener.Accept()
			if err != nil {
				countlog.Error("RaftServer is busy", "error", err)
				continue
			}
			//countlog.Debug(fmt.Sprintf("%s localPort %s, remotePort %s", s, conn.LocalAddr(), conn.RemoteAddr()))
			go rpcServer.ServeConn(conn)
		}
	}
	countlog.Info(fmt.Sprintf("%s stoped", s))
}

// HeartbeatEventHandler is triggered by leaderHeartBeatTicker, leader will send heart beat rpc to followers
// HeartbeatEventHandler sends heart beat rpc to followers
func (s *RaftServer) HeartbeatEventHandler() {
	if s.ServerConf.Info.Role != core.RoleLeader {
		return
	}
	countlog.Info(fmt.Sprintf("%s send heart beat to followers", s))
	for _, node := range s.ServerConf.ClusterAddrList {
		if s.ServerConf.Info.ServerId == node.ServerId {
			continue
		}
		lastTime := s.ServerInfo.Followers[node.ServerId].LastRpcTime
		if time.Now().Sub(lastTime) < time.Duration(s.leaderHeartBeatTimer.DurationInMs)/2*time.Millisecond {
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

// sendHeartbeatToServer sends heart beat rpc to one server
func (s *RaftServer) sendHeartbeatToServer(req *raftrpc.AppendRPCRequest, followerAddr *net.TCPAddr) {
	countlog.Info(fmt.Sprintf("%s sendHeartbeat to %s", s, followerAddr))
	client, err := s.clientPool.Get(followerAddr)
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

// LeaderElectionEventHandler is the event triggered by leaderElectionTimer
// LeaderElectionEventHandler sends request vote rpc to other servers
func (s *RaftServer) LeaderElectionEventHandler() {
	if s.ServerConf.Info.Role == core.RoleLeader {
		return
	}
	countlog.Info(fmt.Sprintf("%s started leader election for term %d, old leader %s is dead",
		s.String(), s.ServerInfo.Term+1, s.ServerConf.LeaderInfo))
	if s.ServerConf.Info.Role == core.RoleFollower {
		becomeCandidate(s)
	}
	for {
		if s.ServerInfo.Role == core.RoleCandidate {
			voteResult := s.collectVotes()
			if voteResult.IsEnough() {
				becomeLeader(s)
				countlog.Info(fmt.Sprintf("%s becomes new leader", s))
			} else {
				countlog.Info(fmt.Sprintf("%s votes %+v not enough", s, voteResult))
				waitForNextRoundElection(s)
			}
		}
		if s.ServerInfo.Role != core.RoleCandidate {
			// exit loop when:
			// 1. role becomes follower when receiving append log rpc
			// 2. role becomes leader when it win the election
			break
		}
	}
	if s.ServerConf.Info.Role == core.RoleLeader {
		s.HeartbeatEventHandler()
	}
}

// collectVotes counts the accepted counts and total counts
func (s *RaftServer) collectVotes() *voteCounter {
	countlog.Info(fmt.Sprintf("%s collectVotes", s))
	nextTerm := s.ServerInfo.Term + 1
	req := &raftrpc.RequestVoteRequest{}
	req.Candidate = s.ServerConf.Info
	req.NextTerm = nextTerm
	s.ServerInfo.LastVotedTerm = nextTerm
	s.ServerInfo.LastVotedServerId = s.ServerConf.Info.ServerId
	voteResult := NewVoteCounter(len(s.ServerConf.ClusterAddrList))
	voteResult.AddAccept()
	for _, node := range s.ServerConf.ClusterAddrList {
		if node.ServerId != s.ServerConf.Info.ServerId {
			voteResult.WG.Add(1)
			go s.sendRequestVoteToServer(node, req, voteResult)
		}
	}
	voteResult.WG.Wait()
	return voteResult
}

func (s *RaftServer) sendRequestVoteToServer(node *core.ServerInfo, req *raftrpc.RequestVoteRequest, counter *voteCounter) {
	countlog.Info(fmt.Sprintf("%s sendRequestVoteToServer_start %s", s, node))
	defer func() {
		countlog.Info(fmt.Sprintf("%s sendRequestVoteToServer_exit %s", s, node))
		counter.WG.Done()
	}()
	client, err := s.clientPool.Get(node.ServerAddr)
	if err != nil {
		countlog.Error("Connect Error", "error", err, "followerAddr", node.ServerAddr.String())
		counter.AddFail()
		return
	}
	var done chan int
	done = make(chan int, 1)
	var resp raftrpc.RequestVoteResponse
	go func() {
		if err = client.Call("RaftService.OnReceiveRequestVoteRPC", req, &resp); err != nil {
			countlog.Error(fmt.Sprintf("%s RaftService.OnReceiveRequestVoteRPC_fail from %s: %s", node, err.Error()))
			done <- FAIL
			return
		}
		countlog.Info(fmt.Sprintf("%s OnReceiveRequestVoteRPC_success from %s: %+v", s, node, resp.Accept))
		done <- SUCCESS
	}()
	select {
	case <-time.After(time.Millisecond * 1000):
		countlog.Warn(fmt.Sprintf("%s OnReceiveRequestVoteRPC_timeout from %s", s, node.ServerAddr))
		counter.AddTimeout()
	case ret := <-done:
		if ret == FAIL {
			counter.AddFail()
		} else {
			if resp.Accept {
				counter.AddAccept()
			} else {
				counter.AddReject()
			}
		}
	}
	return
}

func (s *RaftServer) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {
	countlog.Info(fmt.Sprintf("%s OnReceiveAppendRPC from %s", s.ServerConf.Info, req.LeaderNode.ServerAddr))
	// 1. During leader election, if the dead leader revived, candidate has to become follower
	// 2. After leader election, When the dead leader comes to live after a new leader is elected,
	// leader has to become follower
	becomeFollower(s)
	s.ServerConf.LeaderInfo = req.LeaderNode
	// leader is alive, so reset the timer
	s.leaderElectionTimer.Reset()

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
	lastIndex := s.ServerInfo.GetLastIndex()
	if lastIndex < 0 {
		resp.Accept = true
		return nil
	}
	lastTerm, err := s.Logs.GetLogTerm(lastIndex)
	if err != nil {
		return err
	}
	if req.LastLogTerm > lastTerm {
		resp.Accept = true
	} else if req.LastLogTerm < lastTerm {
		resp.Accept = false
	} else if req.LastLogIndex >= lastIndex {
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

// handleSignal handles system signals
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
