package raftrpc

import "net"

type ClientResponse struct {
	Errcode    int
	Errmsg     string
	ServerAddr *net.TCPAddr
	Data       interface{}
}
