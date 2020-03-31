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
