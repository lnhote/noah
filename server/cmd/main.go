package main

import (
	"os"

	"github.com/lnhote/noaḥ/server"
	"github.com/v2pro/plz/countlog"
	"github.com/v2pro/plz/countlog/output"
	"github.com/v2pro/plz/countlog/output/compact"
	"github.com/v2pro/plz/countlog/output/lumberjack"
)

var (
	DEBUG   = true
	LOGFILE = "/tmp/test.log"
)

func main() {
	initLogger()
	countlog.Info("Start server")

	// start timers
	go server.StartHeartbeatTimer()
	go server.StartLeaderElectionTimer()

	// start servers on different ports, one for requests between servers, one for requests from clients
	go server.StartNoahClusterServer()

	server.StartNoahCommandServer()
}

func initLogger() {
	if DEBUG {
		countlog.EventWriter = output.NewEventWriter(output.EventWriterConfig{
			Format: &compact.Format{},
			Writer: os.Stdout,
		})
	} else {
		logFile := &lumberjack.Logger{
			Filename: LOGFILE,
		}
		defer logFile.Close()
		countlog.EventWriter = output.NewEventWriter(output.EventWriterConfig{
			Format: &compact.Format{},
			Writer: logFile,
		})
	}
}
