package main

import (
	"fmt"
	"os"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/pkg/c2"
	e4 "gitlab.com/teserakt/e4common"
)

// variables set at build time
var gitCommit string
var buildDate string
var gitTag string

func main() {
	exitCode := 0
	defer os.Exit(exitCode)

	// show banner
	if len(gitTag) == 0 {
		fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit)
	} else {
		fmt.Printf("E4: C2 back-end - version %s (%s-%s)\n", gitTag, buildDate, gitCommit)
	}
	fmt.Println("Copyright (c) Teserakt AG, 2018-2019")
	// init logger
	logFileName := fmt.Sprintf("/var/log/e4_c2.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[ERROR] logs: unable to open file '%v' to write logs: %v\n", logFileName, err)
		fmt.Print("[WARN] logs: falling back to standard output only\n")
		logFile = os.Stdout
	} else { // we don't want to close os.Stdout
		defer logFile.Close()
	}

	logger := log.NewJSONLogger(logFile)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	defer logger.Log("msg", "goodbye")

	// set up config resolver
	configResolver := e4.NewAppPathResolver()
	configLoader := config.NewViperLoader("config", configResolver)

	logger.Log("msg", "load configuration and command args")

	cfg, err := configLoader.Load()
	if err != nil {
		logger.Log("error", err)
		exitCode = 1
		return
	}

	c2, err := c2.New(logger, cfg)
	if err != nil {
		logger.Log("error", err)
		exitCode = 1
		return
	}
	defer c2.Close()

	c2.EnableGRPCEndpoint()
	c2.EnableHTTPEndpoint()

	if err := c2.ListenAndServe(); err != nil {
		logger.Log("error", err)
		exitCode = 1
		return
	}
}
