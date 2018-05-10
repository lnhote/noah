package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/lnhote/noah/common"
	"github.com/lnhote/noah/core"
	"github.com/lnhote/noah/core/store"
	"github.com/lnhote/noah/server"
)

func init() {
	common.DEBUG = true
	common.LOGFILE = "/tmp/noah/server.log"
	common.SetupLog()
}

func main() {
	envExample := &core.Env{HeartBeatDurationInMs: 1000, LeaderElectionDurationInMs: 5000, RandomRangeInMs: 3000}
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	defer common.Sugar.Sync()
	log.Print("Working Directory: ", wd)
	configFilename := filepath.Join(wd, "config/config.yml")
	log.Print("Read Config from: ", configFilename)
	serverConfig, err := core.GetServerConfFromFile(configFilename)
	if err != nil {
		panic(err)
	}
	repoDir := flag.String("data", "/tmp/noah/data", "directory path to store log entries/state data")
	flag.Parse()
	raftNode := server.NewRaftServerWithEnv(serverConfig, envExample)
	raftNode.OpenRepo(*repoDir, store.DefaultPageSize, store.SegmentSizeBytes)
	defer raftNode.Stop()
	raftNode.Start()
}
