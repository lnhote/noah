package raftrpc

import (
	"net"
	"net/rpc"
	"sync"
)

type ClientPool struct {
	pool   map[string]*rpc.Client
	wMutex *sync.Mutex
}

func NewClientPool() *ClientPool {
	return &ClientPool{
		pool:   map[string]*rpc.Client{},
		wMutex: &sync.Mutex{},
	}
}

func (c *ClientPool) Get(serverAddr *net.TCPAddr) (*rpc.Client, error) {
	if conn, ok := c.pool[serverAddr.String()]; ok {
		return conn, nil
	}
	client, err := rpc.Dial("tcp", serverAddr.String())
	if err != nil {
		return nil, err
	}
	c.wMutex.Lock()
	c.pool[serverAddr.String()] = client
	c.wMutex.Unlock()
	return client, nil
}
