package main

import (
	"github.com/lnhote/noaḥ/common"
	"github.com/v2pro/plz/countlog"
)

func main() {
	common.SetupLog()
	countlog.Info("Start server")

	// start timers

	// start servers on different ports, one for requests between servers, one for requests from clients

}
