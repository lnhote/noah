package server

import (
	"github.com/lnhote/noaḥ/common"
	"github.com/lnhote/noaḥ/core"
	"testing"
)

func TestStartHeartbeatTimer(t *testing.T) {
	common.SetupLog()
	clusters := []*core.ServerInfo{
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8851"),
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8852"),
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8853"),
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8854"),
		core.NewServerInfo(1, core.RoleLeader, "127.0.0.1:8855")}
	go NewRaftServer(core.NewServerConf(clusters[1], clusters[0], clusters)).Start()
	go NewRaftServer(core.NewServerConf(clusters[2], clusters[0], clusters)).Start()
	go NewRaftServer(core.NewServerConf(clusters[3], clusters[0], clusters)).Start()
	go NewRaftServer(core.NewServerConf(clusters[4], clusters[0], clusters)).Start()
	NewRaftServer(core.NewServerConf(clusters[0], clusters[0], clusters)).Start()
}
