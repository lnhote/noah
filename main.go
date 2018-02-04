package main

import (
	"fmt"

	"github.com/lnhote/noaḥ/server"
)

func main() {
	fmt.Println("Start server")
	server.StartLeaderHandler()
	server.StartFollowerHandler()
}
