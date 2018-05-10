package common

import (
	"github.com/v2pro/plz/countlog"
	"go.uber.org/zap"

	"os"

	"github.com/v2pro/plz/countlog/output"
	"github.com/v2pro/plz/countlog/output/compact"
	"github.com/v2pro/plz/countlog/output/lumberjack"
)

var (
	DEBUG   = true
	LOGFILE = "/tmp/test.log"
	Sugar   *zap.SugaredLogger
)

func init() {
	logger, _ := zap.NewDevelopmentConfig().Build() // or NewProduction, or NewDevelopment
	Sugar = logger.Sugar()
	// In most circumstances, use the SugaredLogger. It's 4-10x faster than most
	// other structured logging packages and has a familiar, loosely-typed API.
}

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
