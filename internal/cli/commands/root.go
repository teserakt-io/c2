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

package commands

import (
	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/internal/cli/commands/clients"
	"github.com/teserakt-io/c2/internal/cli/commands/topics"
)

type rootCommandFlags struct {
	Endpoint string
	Cert     string
}

type rootCommand struct {
	cobraCmd *cobra.Command
	flags    rootCommandFlags
}

var _ cli.Command = (*rootCommand)(nil)

// NewRootCommand creates and configure a new cli root command
func NewRootCommand(c2ClientFactory cli.APIClientFactory, version string) cli.Command {
	rootCmd := &rootCommand{}

	clientCommand := clients.NewRootCommand(c2ClientFactory)
	topicCommand := topics.NewRootCommand(c2ClientFactory)

	countCommand := NewCountCommand(c2ClientFactory)
	attachCommand := NewAttachCommand(c2ClientFactory)
	detachCommand := NewDetachCommand(c2ClientFactory)

	eventsCommand := NewEventsCommand(c2ClientFactory)

	newC2KeyCommand := NewNewC2KeyCommand(c2ClientFactory)

	// TODO: disabled for now as it need a fair bit of polish before being usable
	//interactiveCmd := NewInteractiveCommand(rootCmd, version)
	completionCmd := NewCompletionCommand(rootCmd)

	cobraCmd := &cobra.Command{
		Use:                    "c2cli",
		BashCompletionFunction: completionCmd.GenerateCustomCompletionFuncs(),
		Version:                version,
		SilenceUsage:           true,
		SilenceErrors:          true,
	}

	cobraCmd.PersistentFlags().StringVarP(
		&rootCmd.flags.Endpoint,
		cli.EndpointFlag,
		"e",
		"127.0.0.1:5555", "url to the c2 grpc api",
	)

	cobraCmd.PersistentFlags().StringVarP(
		&rootCmd.flags.Cert,
		cli.CertFlag,
		"c",
		"configs/c2-cert.pem", "path to the c2 grpc api certificate",
	)

	cobraCmd.AddCommand(
		clientCommand.CobraCmd(),
		topicCommand.CobraCmd(),

		countCommand.CobraCmd(),
		attachCommand.CobraCmd(),
		detachCommand.CobraCmd(),

		eventsCommand.CobraCmd(),

		newC2KeyCommand.CobraCmd(),

		// TODO: disabled for now as it need a fair bit of polish before being usable
		//interactiveCmd.CobraCmd(),

		// Autocompletion script generation command
		completionCmd.CobraCmd(),
	)

	cobraCmd.SetVersionTemplate(`{{printf "%s" .Version}}`)

	rootCmd.cobraCmd = cobraCmd

	return rootCmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
