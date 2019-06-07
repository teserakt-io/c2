package main

import (
	"fmt"
	"os"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/pkg/c2"
	slibcfg "gitlab.com/teserakt/serverlib/config"
	slibpath "gitlab.com/teserakt/serverlib/path"
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
	configResolver, err := slibpath.NewAppPathResolver(os.Args[0])
	if err != nil {
		logger.Log("msg", "failed to create configuration resolver", "error", err)
		exitCode = 1
		return
	}
	configLoader := slibcfg.NewViperLoader("config", configResolver)

	logger.Log("msg", "load configuration and command args")

	cfg := config.New()
	if err := configLoader.Load(cfg.ViperCfgFields()); err != nil {
		logger.Log("msg", "configuration loading failed", "error", err)
		exitCode = 1
		return
	}

	if err := cfg.Validate(); err != nil {
		logger.Log("msg", "configuration validation failed", "error", err)
		exitCode = 1
		return
	}

	c2instance, err := c2.New(logger, *cfg)
	if err != nil {
		logger.Log("msg", "failed to create C2", "error", err)
		exitCode = 1
		return
	}
	defer c2instance.Close()

	c2instance.EnableGRPCEndpoint()
	c2instance.EnableHTTPEndpoint()

	if err := c2instance.ListenAndServe(); err != nil {
		if _, ok := err.(*c2.C2Signal); ok {
			logger.Log("msg", err)
			logger.Log("msg", "Done")
			exitCode = 0
			return
		}
		logger.Log("error", err)
		exitCode = 1
		return
	}
}
