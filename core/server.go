package core

import (
	"fmt"
	"github.com/lnhote/noah/core/errmsg"
	"net"
)

type ServerInfo struct {
	ServerId   int
	Role       int
	ServerAddr *net.TCPAddr
}

func (s *ServerInfo) String() string {
	return fmt.Sprintf("Server[%d](%s)<%s>", s.ServerId, s.ServerAddr.String(),
		getRoleName(s.Role))
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
