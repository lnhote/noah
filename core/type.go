package core

import (
	"fmt"
	"net"

	"github.com/lnhote/noah/core/errmsg"
)

const (
	CmdSet = iota
	CmdGet
)

type Command struct {
	CommandType int
	Key         string
	Value       []byte
}

func (c Command) String() string {
	switch c.CommandType {
	case CmdGet:
		return fmt.Sprintf("Get|%s", c.Key)
	case CmdSet:
		return fmt.Sprintf("Set|%s|%s", c.Key, c.Value)
	default:
		return errmsg.NoSuchCommand.Error()
	}
}

type LogEntry struct {
	Command *Command
	Index   int
	Term    int
}

type ClientResponse struct {
	Errcode    int
	Errmsg     string
	ServerAddr *net.TCPAddr
	Data       interface{}
}

type ServerInfo struct {
	ServerId   int
	Role       int
	ServerAddr *net.TCPAddr
}

func (s *ServerInfo) String() string {
	return fmt.Sprintf("Server[%d](%s)<%s>", s.ServerId, s.ServerAddr.String(),
		getRoleName(s.Role))
}

func getRoleName(role int) string {
	switch role {
	case RoleFollower:
		return "Follower"
	case RoleLeader:
		return "Leader"
	case RoleCandidate:
		return "Candidate"
	default:
		return "UnknownRole"
	}
}

func NewServerInfo(id int, role int, serverAddr string) *ServerInfo {
	serverAddrObj, _ := net.ResolveTCPAddr("tcp", serverAddr)
	serverInfo := &ServerInfo{
		ServerId:   id,
		Role:       role,
		ServerAddr: serverAddrObj,
	}
	return serverInfo
}

type ServerConfig struct {
	Info            *ServerInfo
	LeaderInfo      *ServerInfo
	ClusterAddrList map[int]*ServerInfo

	clusterMapByAddr map[string]*ServerInfo
}

func (sc *ServerConfig) FindIdByAddr(addr string) (int, error) {
	if server, ok := sc.clusterMapByAddr[addr]; ok {
		return server.ServerId, nil
	} else {
		return 0, errmsg.ServerNotFound
	}
}

func NewServerConf(info *ServerInfo, leader *ServerInfo, clusterAddrs []*ServerInfo) *ServerConfig {
	serverConf := &ServerConfig{}
	serverConf.Info = info
	serverConf.LeaderInfo = leader
	serverConf.ClusterAddrList = make(map[int]*ServerInfo, len(clusterAddrs))
	serverConf.clusterMapByAddr = make(map[string]*ServerInfo, len(clusterAddrs))
	for _, server := range clusterAddrs {
		serverConf.ClusterAddrList[server.ServerId] = server
		serverConf.clusterMapByAddr[server.ServerAddr.String()] = server
	}
	return serverConf
}
