package core

import (
	"fmt"
	"net"
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

func (c Command) ToLog() (string, error) {
	switch c.CommandType {
	case CmdGet:
		return fmt.Sprintf("GET|%s", c.Key), nil
	case CmdSet:
		return fmt.Sprintf("SET|%s|%s", c.Key, c.Value), nil
	default:
		return "", fmt.Errorf("NoSuchCommand||%d", c.CommandType)
	}
}

type LogEntry struct {
	Command *Command
	Index   int
	Term    int
}

type ClientResponse struct {
	Code int
	Data map[string]interface{}
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
	ClusterAddrList []*ServerInfo
}

func NewServerConf(info *ServerInfo, leader *ServerInfo, clusterAddrs []*ServerInfo) *ServerConfig {
	serverConf := &ServerConfig{}
	serverConf.Info = info
	serverConf.LeaderInfo = leader
	for _, cluster := range clusterAddrs {
		serverConf.ClusterAddrList = append(serverConf.ClusterAddrList, cluster)
	}
	return serverConf
}
