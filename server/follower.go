package server

import (
	"github.com/v2pro/plz/countlog"
	"net"
)

func StartFollowerHandler() {
	// TODO need to add a timer for heart beat, when timeout, trigger RequestVoteRequest
	addr := "127.0.0.1:8849"
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
		go handlerFollowerRequest(conn)
	}
}

func handlerFollowerRequest(conn net.Conn) {
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
	// TODO we can get two req here: AppendRPCRequest, RequestVoteRequest
	val := ""
	writeResp(conn, NewResp(string(val), err))
}
