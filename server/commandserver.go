package server

import (
	"errors"
	"net"
	"net/rpc"

	"github.com/lnhote/noaḥ/config"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/lnhote/noaḥ/server/core/errorcode"
	"github.com/v2pro/plz/countlog"
)

type NoahCommandServer struct {
}

func (ncs *NoahCommandServer) Get(cmd *core.Command, resp *core.ClientResponse) error {
	if cmd == nil {
		err := errors.New("EmptyCommand")
		countlog.Error("event!EmptyCommand", "err", err.Error())
		return err
	}
	if core.CurrentServerState.Role != core.RoleLeader {
		data := map[string]interface{}{
			"leader_address": config.GetLeaderAddr(),
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	val, err := AppendLog(cmd)
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
			"leader_address": config.GetLeaderAddr(),
		}
		resp.Code = errorcode.NotLeader
		resp.Data = data
		return nil
	}
	val, err := AppendLog(cmd)
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

	var address, _ = net.ResolveTCPAddr("tcp", addr)
	listener, err := net.ListenTCP("tcp", address)
	if err != nil {
		panic("Fail to start NoahCommandServer: " + err.Error())
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("StartNoahCommandServer failed to listen to port", "error", err, "addr", addr)
			continue
		}
		rpc.ServeConn(conn)
	}
}
