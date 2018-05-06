package core

import (
	"io/ioutil"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
)

func TestLoadYaml(t *testing.T) {
	b, err := ioutil.ReadFile("testdata/singleconfig.yml")
	assert.Nil(t, err)
	yc := &ymalConfig{}
	err = yaml.Unmarshal(b, yc)
	assert.Nil(t, err)
	assert.True(t, yc.ServerId == 1)
	addr, err := net.ResolveTCPAddr("tcp", yc.ServerAddr)
	assert.Nil(t, err)
	s := &ServerInfo{yc.ServerId, yc.Role, addr}
	assert.True(t, yc.ServerAddr == s.ServerAddr.String())

	b1, err := ioutil.ReadFile("testdata/listconfig.yml")
	assert.Nil(t, err)
	cluster := &ymalClusterConfig{}
	var thisServer *ymalConfig
	var leader *ymalConfig
	err = yaml.Unmarshal(b1, cluster)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(cluster.Cluster))
	assert.Equal(t, 1, cluster.Cluster[0].ServerId)
	assert.Equal(t, 2, cluster.Cluster[1].ServerId)
	for _, server := range cluster.Cluster {
		if server.ServerId == cluster.ServerId {
			thisServer = server
		}
		if server.ServerId == cluster.LeaderId {
			leader = server
		}
	}
	assert.Equal(t, cluster.ServerId, thisServer.ServerId)
	assert.Equal(t, cluster.LeaderId, leader.ServerId)
}

func TestGetServerConfFromFile(t *testing.T) {
	serverConfig, err := GetServerConfFromFile("testdata/listconfig.yml")
	assert.Nil(t, err)
	assert.Equal(t, 5, len(serverConfig.ClusterAddrList))
	assert.Equal(t, "127.0.0.1:8881", serverConfig.Info.ServerAddr.String())
	assert.Equal(t, RoleFollower, serverConfig.Info.Role)
	assert.Equal(t, 1, serverConfig.Info.ServerId)
	assert.Equal(t, "127.0.0.1:8885", serverConfig.LeaderInfo.ServerAddr.String())
	assert.Equal(t, RoleLeader, serverConfig.LeaderInfo.Role)
	assert.Equal(t, 5, serverConfig.LeaderInfo.ServerId)
}
