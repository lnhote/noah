package common

import (
	"github.com/v2pro/plz/countlog"
	"os"

	"github.com/v2pro/plz/countlog/output"
	"github.com/v2pro/plz/countlog/output/compact"
	"github.com/v2pro/plz/countlog/output/lumberjack"
)

var (
	DEBUG   = true
	LOGFILE = "/tmp/test.log"
)

func SetupLog() {
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
