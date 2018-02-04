package server

import (
	"github.com/lnhote/noaḥ/server/command"
	"github.com/lnhote/noaḥ/server/core"
	"github.com/v2pro/plz/countlog"
	"net"
)

func StartLeaderHandler() {
	// TODO need to add a timer for heart beat

	addr := "127.0.0.1:8848"
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		countlog.Error("event!server failed to bind port", "err", err)
		return
	}
	defer listener.Close()
	countlog.Info("event!server started", "addr", addr)
	for {
		conn, err := listener.Accept()
		if err != nil {
			countlog.Error("event!server failed to accept outbound", "err", err)
			return
		}
		countlog.Info("event!client connected", "remote", conn.RemoteAddr(), "local", conn.LocalAddr())
		go handlerRequest(conn)
	}
}

func handlerRequest(conn net.Conn) {
	countlog.Info("event!client disconnected", "addr", conn.RemoteAddr())
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		countlog.Error("event!read error", err, err.Error())
		writeResp(conn, ReturnError(err.Error()))
		return
	}
	commandObj, err := command.ParseCommand(buf)
	if err != nil {
		writeResp(conn, ReturnError(err.Error()))
		return
	}
	raftImpl := &core.SimpleRaft{}
	val, err := core.HandleCommand(commandObj, raftImpl)
	writeResp(conn, NewResp(string(val), err))
}
