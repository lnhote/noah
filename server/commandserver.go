package server

import (
	"errors"
	"fmt"
	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/core/errorcode"
	"github.com/lnhote/noaḥ/server/raftrpc"
	"github.com/v2pro/plz/countlog"
	"net"
	"net/rpc"
)

type NoahCommandServer struct {
}

func init() {
	core.CurrentServerState.Role = core.RoleLeader
}

func (ncs *NoahCommandServer) Get(cmd *core.Command, resp *core.ClientResponse) error {
	if cmd == nil {
		err := errors.New("EmptyCommand")
		countlog.Error("event!EmptyCommand", "err", err.Error())
		return err
	}
	if core.CurrentServerState.Role != core.RoleLeader {
		data := map[string]interface{}{
			"leader_address": config.GetLeaderIp(),
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	raftImpl := &raftrpc.SimpleRaft{}
	val, err := HandleCommand(cmd, raftImpl)
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

func (ncs *NoahCommandServer) Set(cmd *core.Command, resp *core.ClientResponse) error {
	*resp = core.ClientResponse{}
	if cmd == nil {
		err := errors.New("EmptyCommand")
		countlog.Error("event!EmptyCommand", "err", err.Error())
		return err
	}
	if core.CurrentServerState.Role != core.RoleLeader {
		data := map[string]interface{}{
			"leader_address": config.GetLeaderIp(),
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	raftImpl := &raftrpc.SimpleRaft{}
	val, err := HandleCommand(cmd, raftImpl)
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

func StartNoahCommandServer() {
	addr := config.ClientPort
	noahCmdServer := new(NoahCommandServer)

	rpc.Register(noahCmdServer)

	//rpc.HandleHTTP()
	//err := http.ListenAndServe(addr, nil)
	//if err != nil {
	//	return
	//}

	var address, _ = net.ResolveTCPAddr("tcp", addr)
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		fmt.Println("fail to start NoahCommandServer", err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("event!client handler failed to listen to port", "err", err, "addr", addr)
			continue
		}
		fmt.Println("get a request")
		rpc.ServeConn(conn)
	}
}
