package server

type Response struct {
	ErrorMsg string
	Data     interface{}
}

func NewResp(val string, err error) *Response {
	if err != nil {
		return ReturnError(err.Error())
	} else {
		return ReturnSuccess(val)
	}
}

func ReturnSuccess(val string) *Response {
	return &Response{
		ErrorMsg: "",
		Data:     val,
	}
}

func ReturnError(val string) *Response {
	return &Response{
		ErrorMsg: val,
		Data:     "",
	}
}

func writeResp(conn net.Conn, resp *Response) {
	// Send a response back to person contacting us.
	countlog.Info("event!write response")
	// Close the connection when you're done with it.
	respStr, err := json.Marshal(resp)
	if err != nil {
		conn.Write([]byte(""))
		countlog.Error("event!parse json error", "err", err, "resp", resp)
	} else {
		conn.Write(respStr)
	}
	conn.Close()
	countlog.Info("event!close connection")
}
