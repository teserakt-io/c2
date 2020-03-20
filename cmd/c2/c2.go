// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	slibcfg "github.com/teserakt-io/serverlib/config"
	slibpath "github.com/teserakt-io/serverlib/path"

	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/pkg/c2"
)

// variables set at build time
var gitCommit string
var buildDate string
var gitTag string

func main() {
	exitCode := 0
	defer os.Exit(exitCode)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// show banner
	if len(gitTag) == 0 {
		fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit)
	} else {
		fmt.Printf("E4: C2 back-end - version %s (%s-%s)\n", gitTag, buildDate, gitCommit)
	}
	fmt.Println("Copyright (c) Teserakt AG, 2018-2019")

	// init logger
	logger := log.NewEntry(log.New())
	logger.Logger.SetLevel(log.DebugLevel)
	logger.Logger.SetReportCaller(true)
	logger.Logger.SetFormatter(&log.JSONFormatter{})

	logFileName := "/var/log/e4_c2.log"
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[WARN] logs: unable to open file '%v' to write logs: %v\n", logFileName, err)
		fmt.Print("[WARN] logs: falling back to standard output only\n")
		logger.Logger.SetOutput(os.Stdout)
	} else { // we don't want to close os.Stdout
		logger.Logger.SetOutput(logFile)
		defer logFile.Close()
	}

	logger = logger.WithField("application", "c2")

	defer func() {
		if r := recover(); r != nil {
			logger.WithError(fmt.Errorf("%v", r)).Error("c2 panic")
		}

		logger.Warn("goodbye")
	}()

	// set up config resolver
	configResolver, err := slibpath.NewAppPathResolver(os.Args[0])
	if err != nil {
		logger.WithError(err).Error("failed to create configuration resolver")
		exitCode = 1
		return
	}
	configLoader := slibcfg.NewViperLoader("config", configResolver)

	logger.Info("load configuration and command args")

	cfg := config.New()
	if err := configLoader.Load(cfg.ViperCfgFields()); err != nil {
		logger.WithError(err).Error("configuration loading failed")
		exitCode = 1
		return
	}

	if err := cfg.Validate(); err != nil {
		logger.WithError(err).Error("configuration validation failed")
		exitCode = 1
		return
	}

	level, err := log.ParseLevel(cfg.LoggerLevel)
	if err != nil {
		logger.WithError(err).Warn("invalid logger level from configuration, falling back to debug")
		level = log.DebugLevel
	}
	logger.Logger.SetLevel(level)

	c2instance, err := c2.New(logger, *cfg)
	if err != nil {
		logger.WithError(err).Info("failed to create C2")
		exitCode = 1
		return
	}
	defer c2instance.Close()

	c2instance.EnableGRPCEndpoint()
	c2instance.EnableHTTPEndpoint()

	if err := c2instance.ListenAndServe(ctx); err != nil {
		if _, ok := err.(c2.SignalError); ok {
			logger.WithField("signal", err).Info("graceful shutdown")
			exitCode = 0
			return
		}
		logger.WithError(err).Error("failed to listen")
		exitCode = 1
		return
	}
}
