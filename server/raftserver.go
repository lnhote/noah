package server

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/entity"
	"github.com/lnhote/noah/core/errmsg"
	"github.com/lnhote/noah/core/errno"
	"github.com/lnhote/noah/core/raftrpc"
	"github.com/lnhote/noah/core/store"
	"github.com/lnhote/noah/core/store/kvstore"
	"github.com/v2pro/plz/countlog"
)

const (
	// SUCCESS rpc returns success
	SUCCESS = 1

	// FAIL rpc returns fail
	FAIL = 2

	// TIMEOUT rpc times out
	TIMEOUT = 3
)

// Rules for Servers

// All Servers:
//• If commitIndex > lastApplied: increment lastApplied, apply
//log[lastApplied] to state machine (§5.3)
//• If RPC request or response contains term T > currentTerm:
//set currentTerm = T, convert to follower (§5.1)

//Followers (§5.2):
//• Respond to RPCs from candidates and leaders
//• If election timeout elapses without receiving AppendEntries
//RPC from current leader or granting vote to candidate:
//convert to candidate

//Candidates (§5.2):
//• On conversion to candidate, start election:
//• Increment currentTerm
//• Vote for self
//• Reset election timer
//• Send RequestVote RPCs to all other servers
//• If votes received from majority of servers: become leader
//• If AppendEntries RPC received from new leader: convert to follower
//• If election timeout elapses: start new election

//Leaders:
//• Upon election: send initial empty AppendEntries RPCs
//(heartbeat) to each server; repeat during idle periods to
//prevent election timeouts (§5.2)
//• If command received from client: append entry to local log,
//respond after entry applied to state machine (§5.3)
//• If last log index ≥ nextIndex for a follower: send
//AppendEntries RPC with log entries starting at nextIndex
//• If successful: update nextIndex and matchIndex for follower (§5.3)
//• If AppendEntries fails because of log inconsistency: decrement nextIndex and retry (§5.3)
//• If there exists an N such that N > commitIndex, a majority of matchIndex[i] ≥ N, and log[N].term == currentTerm:
//set commitIndex = N (§5.3, §5.4).

// RaftServer object
type RaftServer struct {
	ServerConf   *core.ServerConfig
	stableInfo   *core.PersistentState
	volatileInfo *core.VolatileState
	leaderState  *core.VolatileLeaderState

	leaderHeartBeatTimer *eventTimer
	leaderElectionTimer  *eventTimer
	RandomRangeInMs      int

	wgHandleCommand *sync.WaitGroup

	sigs       chan os.Signal
	stopSignal chan int

	clientPool *raftrpc.ClientPool
	dbStore    kvstore.KVStore
}

// NewRaftServer returns a raft server with default env
func NewRaftServer(conf *core.ServerConfig) *RaftServer {
	return NewRaftServerWithEnv(conf, core.DefaultEnv)
}

// NewRaftServerWithEnv returns a new raft server with env
func NewRaftServerWithEnv(conf *core.ServerConfig, env *core.Env) *RaftServer {
	newServer := &RaftServer{}
	newServer.dbStore = kvstore.NewRocksDB(env.DBDir)
	newServer.ServerConf = conf
	newServer.stableInfo = &core.PersistentState{Term: 0, LastVotedServerID: 0, Logs: core.NewLogRepo()}
	newServer.volatileInfo = &core.VolatileState{}
	newServer.leaderState = &core.VolatileLeaderState{}
	newServer.wgHandleCommand = &sync.WaitGroup{}
	newServer.RandomRangeInMs = env.RandomRangeInMs
	newServer.leaderElectionTimer = newEventTimer(newServer.LeaderElectionEventHandler, env.LeaderElectionDurationInMs)
	newServer.leaderHeartBeatTimer = newEventTimer(newServer.HeartbeatEventHandler, env.HeartBeatDurationInMs)
	newServer.leaderState.NextIndex = map[int]int{}
	newServer.leaderState.MatchIndex = map[int]int{}
	newServer.leaderState.LastRPCTime = map[int]time.Time{}
	for _, addr := range newServer.ServerConf.ClusterAddrList {
		if newServer.ServerConf.Info.ServerID == addr.ServerID {
			continue
		}
		newServer.leaderState.NextIndex[addr.ServerID] = 1
		newServer.leaderState.MatchIndex[addr.ServerID] = 0
		newServer.leaderState.LastRPCTime[addr.ServerID] = time.Now()
	}
	newServer.clientPool = raftrpc.NewClientPool()
	newServer.sigs = make(chan os.Signal, 1)
	// SIGINT：interrupt/Ctrl-C
	// SIGTERM: kill, can be block
	// SIGUSR1, SIGUSR2: user defined
	signal.Notify(newServer.sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
	newServer.stopSignal = make(chan int, 1)
	return newServer
}

// OpenRepo opens a wal file and read logs/states from it, set to memory
func (s *RaftServer) OpenRepo(dirpath string, pageSize int, segmentSize int64) {
	newRepo, err := store.OpenRepo(dirpath, pageSize, segmentSize)
	if err != nil {
		panic(err)
	}
	s.stableInfo = newRepo.State
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
	s.leaderElectionTimer.Stop()
	s.leaderHeartBeatTimer.Stop()
}

// startServe starts listening to port and handles requests
func (s *RaftServer) startServe() {
	rpcServer := rpc.NewServer()
	if err := rpcServer.Register(raftrpc.NewRaftService(s)); err != nil {
		panic(err)
	}
	listener, err := net.ListenTCP("tcp", s.ServerConf.Info.ServerAddr)
	if err != nil {
		panic("Fail to start RaftServer: " + err.Error())
	}
	//rpc.Accept(listener)
	countlog.Info(fmt.Sprintf("%s started", s))
stopServeLoop:
	for {
		select {
		case sig := <-s.sigs:
			s.handleSignal(sig)
			break stopServeLoop
		case <-s.stopSignal:
			break stopServeLoop
		default:
			conn, err := listener.Accept()
			if err != nil {
				countlog.Error("RaftServer is busy", "error", err)
				continue
			}
			countlog.Debug(fmt.Sprintf("%s localPort %s, remotePort %s", s, conn.LocalAddr(), conn.RemoteAddr()))
			go rpcServer.ServeConn(conn)
		}
	}
	if err := listener.Close(); err != nil {
		countlog.Error("event!listener close error", "err", err)
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
		if s.ServerConf.Info.ServerID == node.ServerID {
			continue
		}
		lastTime := s.leaderState.LastRPCTime[node.ServerID]
		if time.Since(lastTime) < time.Duration(s.leaderHeartBeatTimer.DurationInMs)/2*time.Millisecond {
			// Don't send heart beat if we already sent any log recently
			countlog.Info(fmt.Sprintf("No need to send heartbeat to node %s: lastTime = %v", node.String(), lastTime))
			continue
		}
		req := &raftrpc.AppendRPCRequest{}
		req.LogEntries = []*core.LogEntry{}
		req.Term = s.stableInfo.Term
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
	countlog.Info(fmt.Sprintf("HeartbeatResponse from %s", followerAddr.String()))
	serverID, _ := s.ServerConf.FindIDByAddr(followerAddr.String())
	s.leaderState.LastRPCTime[serverID] = time.Now()
}

// LeaderElectionEventHandler is the event triggered by leaderElectionTimer
// LeaderElectionEventHandler sends request vote rpc to other servers
func (s *RaftServer) LeaderElectionEventHandler() {
	if s.ServerConf.Info.Role == core.RoleLeader {
		return
	}
	countlog.Info(fmt.Sprintf("%s starts leader election for term %d, old leader %s is dead",
		s.String(), s.stableInfo.Term+1, s.ServerConf.LeaderInfo.String()))

	s.ServerConf.Info.Role.HandleMsg(core.LeaderElectionStartEvent)
	for {
		if s.ServerConf.Info.Role == core.RoleCandidate {
			voteResult := s.collectVotes()
			if voteResult.IsEnough() {
				s.ServerConf.Info.Role.HandleMsg(core.LeaderElectionSuccessEvent)
				becomeLeader(s)
				countlog.Info(fmt.Sprintf("%s becomes new leader", s))
			} else {
				countlog.Info(fmt.Sprintf("%s votes %s not enough", s.String(), voteResult.String()))
				s.stableInfo.LastVotedServerID = 0
				waitForNextRoundElection(s)
			}
		}
		if s.ServerConf.Info.Role != core.RoleCandidate {
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
	nextTerm := s.stableInfo.Term + 1
	req := &raftrpc.RequestVoteRequest{}
	req.Candidate = s.ServerConf.Info
	req.NextTerm = nextTerm
	s.stableInfo.LastVotedServerID = s.ServerConf.Info.ServerID
	voteResult := newVoteCounter(len(s.ServerConf.ClusterAddrList))
	voteResult.AddAccept()
	for _, node := range s.ServerConf.ClusterAddrList {
		if node.ServerID != s.ServerConf.Info.ServerID {
			voteResult.WG.Add(1)
			go s.sendRequestVoteToServer(node, req, voteResult)
		}
	}
	voteResult.WG.Wait()
	return voteResult
}

func (s *RaftServer) sendRequestVoteToServer(node *core.ServerInfo, req *raftrpc.RequestVoteRequest, counter *voteCounter) {
	countlog.Debug(fmt.Sprintf("%s sendRequestVoteToServer_start %s", s, node.String()))
	defer func() {
		counter.WG.Done()
	}()
	client, err := s.clientPool.Get(node.ServerAddr)
	if err != nil {
		countlog.Error("Connect Error", "error", err, "followerAddr", node.ServerAddr.String())
		counter.AddFail()
		return
	}
	done := make(chan int, 1)
	var resp raftrpc.RequestVoteResponse
	go func() {
		if err = client.Call("RaftService.OnReceiveRequestVoteRPC", req, &resp); err != nil {
			countlog.Error(fmt.Sprintf("%s RaftService.OnReceiveRequestVoteRPC_fail from %s: %s",
				s.String(), node.String(), err.Error()))
			done <- FAIL
			return
		}
		countlog.Debug(fmt.Sprintf("%s OnReceiveRequestVoteRPC_success from %s: %+v", s.String(), node.String(), resp.Accept))
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
}

// OnReceiveAppendRPC implementation:
// 1. Reply false if term < currentTerm (§5.1)
// 2. Reply false if log doesn’t contain an entry at prevLogIndex whose term matches prevLogTerm (§5.3)
// 3. If an existing entry conflicts with a new one (same index but different terms),
// delete the existing entry and all that follow it (§5.3)
// 4. Append any new entries not already in the log
// 5. If leaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
func (s *RaftServer) OnReceiveAppendRPC(req *raftrpc.AppendRPCRequest, resp *raftrpc.AppendRPCResponse) error {
	countlog.Info(fmt.Sprintf("%s OnReceiveAppendRPC from %s", s.ServerConf.Info, req.LeaderNode.ServerAddr))
	// this leader is too old, not valid
	if s.stableInfo.Term > req.Term {
		countlog.Error("request term %d is too old, server term is %d", req.Term, s.stableInfo.Term)
		resp.Success = false
		resp.Term = s.stableInfo.Term
		return nil
	} else if s.stableInfo.Term < req.Term {
		updateTerm(s, req.Term)
		s.ServerConf.Info.Role.HandleMsg(core.DiscoverNewTermEvent)
		resp.Term = s.stableInfo.Term
	} else {
		s.ServerConf.Info.Role.HandleMsg(core.ReceiveAppendRPCEvent)
	}
	// 1. During leader election, if the dead leader revived, candidate has to become follower
	// 2. After leader election, When the dead leader comes to live after a new leader is elected,
	// leader has to become follower

	// this is a valid leader, sync term/commitIndex/leaderId
	if req.CommitIndex > s.volatileInfo.CommitIndex {
		s.volatileInfo.CommitIndex = req.CommitIndex
	}
	if s.stableInfo.Logs.GetLastIndex() <= req.CommitIndex {
		s.volatileInfo.CommitIndex = s.stableInfo.Logs.GetLastIndex()
	}
	s.ServerConf.LeaderInfo = req.LeaderNode
	resetVote(s)

	// leader is alive, so reset the timer
	s.leaderElectionTimer.Reset()

	// this is a heart beat
	if len(req.LogEntries) == 0 {
		resp.Success = true
		return nil
	}
	// check log before accept new logs from leader
	if req.PrevLogIndex > 0 {
		prevLogTerm, err := s.stableInfo.Logs.GetLogTerm(req.PrevLogIndex)
		if err != nil {
			countlog.Info("Leader.PrevLogIndex does not exist here", "index", req.PrevLogIndex)
			resp.Success = false
			return nil
		}
		if prevLogTerm != req.PrevLogTerm {
			resp.Success = false
			return nil
		}
		resp.Success = true
	}
	resp.Success = true
	// prevlog matches leader's, replicate the new log entires
	lastReplicatedLogIndex := req.LogEntries[0].Index
	for i := 0; i < len(req.LogEntries); i++ {
		saveLogEntry(s, req.LogEntries[i])
		lastReplicatedLogIndex = lastReplicatedLogIndex + 1
	}
	// if follower contains more logs than leader, delete them
	for {
		if _, err := s.stableInfo.Logs.GetLogEntry(lastReplicatedLogIndex); err != nil {
			// index does not exist in follower
			break
		} else {
			s.stableInfo.Logs.Delete(lastReplicatedLogIndex)
		}
		lastReplicatedLogIndex++
	}
	return nil
}

// OnReceiveRequestVoteRPC will make the election decision: accept or reject
// 1. Reply false if term < currentTerm (§5.1)
// 2. If votedFor is null or candidateId,
// and candidate’s log is at least as up-to-date as receiver’s log, grant vote (§5.2, §5.4)
func (s *RaftServer) OnReceiveRequestVoteRPC(req *raftrpc.RequestVoteRequest, resp *raftrpc.RequestVoteResponse) error {
	countlog.Info(fmt.Sprintf("%s OnReceiveRequestVoteRPC from %s", s, req.Candidate))
	resp.Term = s.stableInfo.Term
	// the candidate is too old
	if s.stableInfo.Term >= req.NextTerm {
		resp.Accept = false
		return nil
	}
	if s.stableInfo.LastVotedServerID == 0 {
		resp.Accept = shouldVote(req, s)
		if resp.Accept {
			s.stableInfo.LastVotedServerID = req.Candidate.ServerID
		}
	} else {
		// already voted for other candidate
		if s.stableInfo.LastVotedServerID != req.Candidate.ServerID {
			resp.Accept = false
			return nil
		}
		// TODO voted for this candidate before, should I check the log again
		resp.Accept = true
		return nil
	}
	return nil
}

// Get handles client get command
func (s *RaftServer) Get(cmd *entity.Command, resp *raftrpc.ClientResponse) error {
	countlog.Info("Get", "cmd", cmd)
	if cmd == nil {
		resp.Errcode = errno.MissingParam
		countlog.Error(fmt.Sprintf("EmptyCommand"))
		return nil
	}
	if s.ServerConf.Info.Role != core.RoleLeader {
		resp.ServerAddr = s.ServerConf.Info.ServerAddr
		resp.Errcode = errno.NotLeader
		return nil
	}
	s.dbStore.Connect()
	defer s.dbStore.Close()
	switch cmd.CommandType {
	case entity.CmdGet:
		val, err := s.dbStore.Get(cmd.Key)
		if err != nil {
			countlog.Error("DBGet failed", "key", cmd.Key, "error", err)
			resp.Errcode = errno.InternalServerError
			return err
		}
		resp.Errcode = errno.Success
		resp.Data = val
	default:
		resp.Errcode = errno.ValueNotAccepcted
		countlog.Error(fmt.Sprintf("UnkownCommandType(%d)", cmd.CommandType))
	}
	return nil
}

// Set handles client set command
func (s *RaftServer) Set(cmd *entity.Command, resp *raftrpc.ClientResponse) error {
	countlog.Info("Set", "cmd", cmd)
	if cmd == nil {
		resp.Errcode = errno.MissingParam
		resp.Errmsg = "EmptyCommand"
		countlog.Error("EmptyCommand")
		return nil
	}
	if s.ServerConf.Info.Role != core.RoleLeader {
		resp.ServerAddr = s.ServerConf.Info.ServerAddr
		resp.Errcode = errno.NotLeader
		return nil
	}
	switch cmd.CommandType {
	case entity.CmdSet:
		val, err := s.ReplicateLog(cmd)
		if err != nil {
			countlog.Error("DBSet -> ReplicateLog failed", "key", cmd.Key, "error", err)
			resp.Errmsg = err.Error()
			resp.Errcode = errno.ReplicateLogFail
			return nil
		}
		resp.Errcode = errno.Success
		resp.Data = val
	default:
		resp.Errcode = errno.ValueNotAccepcted
		countlog.Error(fmt.Sprintf("UnkownCommandType(%d)", cmd.CommandType))
	}
	return nil
}

type replcateResult struct {
	*raftrpc.AppendRPCResponse
	followerID   int
	returnStatus int
}

// ReplicateLog replicates logs to follower
func (s *RaftServer) ReplicateLog(cmd *entity.Command) ([]byte, error) {
	prevLogIndex := getLastLogIndex(s)
	appendLogEntry(s, cmd)
	countlog.Info("ReplicateLog to follwers", "cmd", cmd)
	appendRPCMsg := make(chan replcateResult, len(s.ServerConf.ClusterAddrList)-1)
	for _, server := range s.ServerConf.ClusterAddrList {
		if server.ServerID == s.ServerConf.Info.ServerID {
			continue
		}
		s.wgHandleCommand.Add(1)
		go s.replicateLogToServer(server.ServerAddr, buildAppendRPCRequest(s, server, prevLogIndex), appendRPCMsg)
	}
	success := 1
	go func(appendRPCMsg chan replcateResult) {
		for res := range appendRPCMsg {
			if res.returnStatus == SUCCESS {
				if res.Success {
					s.leaderState.NextIndex[res.followerID] = s.stableInfo.Logs.GetNextIndex()
					s.leaderState.MatchIndex[res.followerID] = s.stableInfo.Logs.GetLastIndex()
					s.leaderState.LastRPCTime[res.followerID] = time.Now()
					success++
				}
			}
		}
	}(appendRPCMsg)
	s.wgHandleCommand.Wait()
	close(appendRPCMsg)

	if success*2 > len(s.ServerConf.ClusterAddrList) {
		// replicate success
		s.volatileInfo.CommitIndex = s.stableInfo.Logs.GetLastIndex()
		s.applyLogs()
		return s.dbStore.Get(cmd.Key)
	}
	return nil, errmsg.ReplicateLogFail
}

func (s *RaftServer) replicateLogToServer(followerAddr *net.TCPAddr, req *raftrpc.AppendRPCRequest, res chan replcateResult) {
	countlog.Debug(fmt.Sprintf("%s replicateLogToServer_start %s", s, followerAddr.String()))
	defer s.wgHandleCommand.Done()
	done := make(chan int, 1)
	var resp raftrpc.AppendRPCResponse
	followerID, _ := s.ServerConf.FindIDByAddr(followerAddr.String())
	followerInfo := s.ServerConf.ClusterAddrList[followerID]
	go func() {
		client, err := s.clientPool.Get(followerAddr)
		if err != nil {
			countlog.Error("Connect Error", "error", err, "followerAddr", followerAddr.String())
			done <- FAIL
			return
		}
		prevLogIndex := req.PrevLogIndex
		for {
			if err = client.Call("RaftService.OnReceiveAppendRPC", req, &resp); err != nil {
				countlog.Error(fmt.Sprintf("%s RaftService.OnReceiveAppendRPC_fail from %s: %s", followerAddr.String(), "error", err.Error()))
				done <- FAIL
				return
			}
			countlog.Debug(fmt.Sprintf("%s OnReceiveAppendRPC_success from %s", s.String(), followerAddr.String()))
			if !resp.Success {
				// need to decrease prevLogIndex and send again
				prevLogIndex = prevLogIndex - 1
				req = buildAppendRPCRequest(s, followerInfo, prevLogIndex)
			} else {
				break
			}
			if prevLogIndex < 0 {
				break
			}
		}
		done <- SUCCESS
	}()
	select {
	case <-time.After(time.Millisecond * 1000):
		countlog.Warn(fmt.Sprintf("%s OnReceiveRequestVoteRPC_timeout from %s", s, followerAddr.String()))
		res <- replcateResult{nil, followerID, TIMEOUT}
	case ret := <-done:
		if ret == FAIL {
			res <- replcateResult{nil, followerID, FAIL}
		} else {
			res <- replcateResult{&resp, followerID, SUCCESS}
		}
	}
}

// applyLogs execute logs from last applied index to last log index
func (s *RaftServer) applyLogs() {
	for i := s.volatileInfo.LastApplied + 1; i <= s.volatileInfo.CommitIndex; i++ {
		countlog.Info(fmt.Sprintf("apply log %d", i))
		if logToApply, err := s.stableInfo.Logs.GetLogEntry(i); err == nil {
			if dbErr := s.dbStore.Set(logToApply.Command.Key, logToApply.Command.Value); dbErr == nil {
				countlog.Info(fmt.Sprintf("apply log %d [%s] success", i, logToApply.Command.String()))
				s.volatileInfo.LastApplied++
			} else {
				countlog.Error(fmt.Sprintf("applyLogs fails to DBSet %s: %s", logToApply.Command, err.Error()))
				break
			}
		} else {
			countlog.Error(fmt.Sprintf("applyLogs fails to GetLogEntry %d: %s", i, err.Error()))
			break
		}
	}
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
