package main

import (
	"fmt"
	"log"

	"github.com/teserakt-io/c2/internal/cli"

	"github.com/teserakt-io/c2/internal/cli/commands"
)

// Provided by build script
var gitCommit string
var gitTag string
var buildDate string

func main() {
	log.SetFlags(0)

	c2ClientFactory := cli.NewAPIClientFactory()

	rootCmd := commands.NewRootCommand(c2ClientFactory, getVersion())
	if err := rootCmd.CobraCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

func getVersion() string {
	var out string

	if len(gitTag) == 0 {
		out = fmt.Sprintf("E4: C2 cli - version %s-%s\n", buildDate, gitCommit)
	} else {
		out = fmt.Sprintf("E4: C2 cli - version %s (%s-%s)\n", gitTag, buildDate, gitCommit)
	}
	out += fmt.Sprintln("Copyright (c) Teserakt AG, 2018-2019")

	return out
}
