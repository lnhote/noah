package main

import (
	"fmt"
	"github.com/lnhote/noaḥ/server"
)

func main() {
	fmt.Println("Start server")

	// start timers
	go server.StartHeartbeatTimer()
	go server.StartLeaderElectionTimer()

	// start servers on different ports, one for requests between servers, one for requests from clients
	go server.StartNoahClusterServer()
	server.StartNoahCommandServer()
}
