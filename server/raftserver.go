package server

import (
	"github.com/lnhote/noaḥ/core"
	"github.com/lnhote/noaḥ/core/errorcode"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/lnhote/noaḥ/server/store"
	"github.com/v2pro/plz/countlog"

	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
)

type RaftServer struct {
	ServerConf   *core.ServerConfig
	ServerInfo   *core.ServerState
	FollowerInfo *core.FollowerState
	Logs         *core.LogList

	// private
	leaderHeartBeatTicker *time.Ticker
	countMutex            *sync.Mutex
	wg                    *sync.WaitGroup
	wgHandleCommand       *sync.WaitGroup
	acceptedCount         int
	ackCount              int
	leaderElectionTimer   *time.Timer
}

func NewRaftServer(conf *core.ServerConfig) *RaftServer {
	newServer := &RaftServer{}
	newServer.ServerConf = conf
	newServer.Logs = core.NewLogList()
	newServer.ServerInfo = core.NewServerState()
	newServer.leaderHeartBeatTicker = time.NewTicker(5 * time.Millisecond)
	newServer.countMutex = &sync.Mutex{}
	newServer.wg = &sync.WaitGroup{}
	newServer.wgHandleCommand = &sync.WaitGroup{}
	newServer.acceptedCount = 0
	newServer.ackCount = 0
	newServer.leaderElectionTimer = time.NewTimer(time.Duration(20) * time.Millisecond)
	return newServer
}

func (s *RaftServer) Start() {
	go s.StartHeartbeatTimer()
	addr := s.ServerConf.Info.ServerAddr
	rpc.Register(s)
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		panic("Fail to start NoahCommandServer: " + err.Error())
	}
	countlog.Info(fmt.Sprintf("StartNoahCommandServer %d started", s.ServerConf.Info.ServerId), "addr", addr.String())
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("StartNoahCommandServer failed to listen to port", "error", err, "addr", addr)
			continue
		}
		rpc.ServeConn(conn)
	}
}

// leader will send heart beat to followers periodically
func (s *RaftServer) StartHeartbeatTimer() {
	for {
		<-s.leaderHeartBeatTicker.C
		if s.ServerConf.Info.Role == core.RoleLeader {
			s.SendHearbeat()
		}
	}
}

// leader send empty AppendRPC request to followers
func (s *RaftServer) SendHearbeat() {
	countlog.Info("SendHearbeat", "server", s.String())
	req := &raftrpc.AppendRPCRequest{}
	req.Term = s.ServerInfo.Term
	for _, addr := range s.ServerConf.ClusterAddrList {
		if s.ServerConf.Info.ServerId == addr.ServerId {
			continue
		}
		lastTime := s.ServerInfo.Followers[addr.ServerId].LastRpcTime
		if time.Now().Sub(lastTime) < time.Duration(10*time.Millisecond) {
			// Don't send heart beat if we already sent any log recently
			continue
		}
		req.Node = addr
		go s.sendHeartbeatToServer(req)
	}
}

func (s *RaftServer) sendHeartbeatToServer(req *raftrpc.AppendRPCRequest) {
	countlog.Info("sendHeartbeatToServer", "server", s.String())
	resp, err := raftrpc.SendAppendEntryRPC(req.Node, req)
	if err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "node", req.Node.String(), "error", err)
		return
	}
	s.ServerInfo.Followers[resp.Node.ServerId].LastRpcTime = resp.Time
}

func (s *RaftServer) ResetLeaderElectionTimer() {
	s.leaderElectionTimer.Reset(time.Duration(20) * time.Millisecond)
}

// follower will start leader election on time out
func (s *RaftServer) StartLeaderElectionTimer() {
	for {
		<-s.leaderElectionTimer.C
		s.ResetLeaderElectionTimer()
		s.startElection()
	}
}

func (s *RaftServer) startElection() {
	countlog.Info("startElection", "server", s.String())
	if s.ServerConf.Info.Role == core.RoleFollower {
		countlog.Info("leader is dead, start leader election")
		s.ServerConf.Info.Role = core.RoleCandidate
		s.SendRequestVoteToAll()
	}
}

func (s *RaftServer) SendRequestVoteToAll() {
	countlog.Info("SendRequestVoteToAll error", "server", s.String())
	for s.ServerInfo.Role == core.RoleCandidate {
		req := &raftrpc.RequestVoteRequest{}
		req.Candidate = s.ServerConf.Info
		req.NextTerm = s.ServerInfo.Term + 1
		s.ServerInfo.LastVotedTerm = req.NextTerm
		s.ServerInfo.LastVotedServerId = s.ServerConf.Info.ServerId
		s.acceptedCount = 0
		s.ackCount = 0
		s.wg.Add(len(s.ServerConf.ClusterAddrList))
		for _, addr := range s.ServerConf.ClusterAddrList {
			if addr.ServerId == req.Candidate.ServerId {
				continue
			}
			go s.getVoteFromServer(addr, req)
		}
		s.wg.Wait()
		if s.acceptedCount*2 > len(s.ServerConf.ClusterAddrList) {
			s.ServerInfo.Role = core.RoleLeader
			s.ServerInfo.Term = s.ServerInfo.Term + 1
			s.ServerInfo.LeaderId = s.ServerConf.Info.ServerId
			s.SendHearbeat()
		} else {
			// sleep random time duration (0 - 50 ms) and try again
			duration := rand.Intn(50)
			time.Sleep(time.Millisecond * time.Duration(duration))
		}
		// role becomes Follower when receiving AppendLogRPC
	}
	return
}

func (s *RaftServer) getVoteFromServer(node *core.ServerInfo, req *raftrpc.RequestVoteRequest) {
	countlog.Info("getVoteFromServer", "server", s.String())
	resp, err := raftrpc.SendRequestVoteRPC(node, req)
	s.countMutex.Lock()
	defer s.countMutex.Unlock()
	s.ackCount++
	if err != nil {
		return
	}
	if resp.Accept {
		s.acceptedCount++
	}
	s.wg.Done()
}

func (s *RaftServer) String() string {
	return s.ServerConf.Info.String()
}

func (s *RaftServer) Get(cmd *core.Command, resp *core.ClientResponse) error {
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
	// 1. During leader election, if the dead leader revived, candidate has to become follower
	// 2. After leader election, When the dead leader comes to live after a new leader is elected,
	// leader has to become follower
	s.ServerConf.Info.Role = core.RoleFollower
	// leader is alive, so reset the timer
	s.ResetLeaderElectionTimer()

	s.ServerInfo.Term = req.Term
	s.ServerInfo.CommitIndex = req.CommitIndex
	resp.Node = s.ServerConf.Info
	resp.Time = time.Now()
	// check log before accept new logs from leader
	term, err := s.Logs.GetLogTerm(req.PrevLogIndex)
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
func (s *RaftServer) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {
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
	newLog := &core.LogEntry{cmd, s.ServerInfo.NextIndex, s.ServerInfo.Term}
	logIndex := store.SaveLogEntry(newLog)
	s.wgHandleCommand.Add(len(s.ServerInfo.Followers))
	for _, follower := range s.ServerInfo.Followers {
		req := &raftrpc.AppendRPCRequest{}
		req.Node = follower.Node
		req.Term = s.ServerInfo.Term
		req.PrevLogIndex = s.ServerInfo.NextIndex - 1
		req.PrevLogTerm, _ = s.Logs.GetLogTerm(req.PrevLogIndex)
		req.NextIndex = s.ServerInfo.NextIndex
		req.CommitIndex = s.ServerInfo.CommitIndex
		req.LogEntries = append(req.LogEntries, newLog)
		go s.replicateLogToServer(req)
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

func (s *RaftServer) replicateLogToServer(req *raftrpc.AppendRPCRequest) error {
	defer s.wgHandleCommand.Done()
	resp, err := raftrpc.SendAppendEntryRPC(req.Node, req)
	if err != nil {
		countlog.Error("SendAppendEntryRPC Fail", "node", req.Node.String(), "error", err)
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
		if resp, err = raftrpc.SendAppendEntryRPC(req.Node, req); err != nil {
			countlog.Error("SendAppendEntryRPC Fail", "addr", req.Node.String(), "error", err)
			return err
		}
	}
	s.ServerInfo.Followers[resp.Node.ServerId].LastIndex = resp.LastLogIndex
	s.ServerInfo.Followers[resp.Node.ServerId].LastRpcTime = resp.Time
	s.ServerInfo.Followers[resp.Node.ServerId].MatchedIndex = resp.LastLogIndex
	return nil
}
