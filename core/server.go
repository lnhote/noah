package core

import (
	"fmt"
	"github.com/lnhote/noah/core/errmsg"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net"
)

type ServerInfo struct {
	ServerId   int
	Role       raftRole
	ServerAddr *net.TCPAddr
}

type ymalClusterConfig struct {
	ServerId int           `yaml:"server_id"`
	LeaderId int           `yaml:"leader_id"`
	Cluster  []*ymalConfig `yaml:"cluster"`
}

type ymalConfig struct {
	ServerId   int      `yaml:"server_id"`
	Role       raftRole `yaml:"role"`
	ServerAddr string   `yaml:"server_addr"`
}

func (s *ServerInfo) String() string {
	return fmt.Sprintf("Server[%d](%s)<%s>", s.ServerId, s.ServerAddr.String(),
		s.Role.String())
}

func NewServerInfo(id int, role raftRole, serverAddr string) *ServerInfo {
	serverAddrObj, _ := net.ResolveTCPAddr("tcp", serverAddr)
	serverInfo := &ServerInfo{
		ServerId:   id,
		Role:       role,
		ServerAddr: serverAddrObj,
	}
	return serverInfo
}

type ServerConfig struct {
	Info       *ServerInfo
	LeaderInfo *ServerInfo

	// id => info
	ClusterAddrList map[int]*ServerInfo

	// addr => info
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

func GetServerConfFromFile(filename string) (*ServerConfig, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cluster := &ymalClusterConfig{}
	err = yaml.Unmarshal(content, cluster)
	if err != nil {
		return nil, err
	}
	var leaderInfo *ServerInfo
	var thisServer *ServerInfo
	clusterInfoList := make([]*ServerInfo, 0)
	for _, server := range cluster.Cluster {
		serverAddr, err := net.ResolveTCPAddr("tcp", server.ServerAddr)
		if err != nil {
			return nil, err
		}
		nodeInfo := &ServerInfo{server.ServerId, server.Role, serverAddr}
		clusterInfoList = append(clusterInfoList, nodeInfo)
		if server.ServerId == cluster.ServerId {
			thisServer = nodeInfo
		}
		if server.ServerId == cluster.LeaderId {
			leaderInfo = nodeInfo
		}
	}
	return NewServerConf(thisServer, leaderInfo, clusterInfoList), nil
}
