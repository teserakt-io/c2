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

package topics

import (
	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
)

type rootCommand struct {
	cobraCmd *cobra.Command
}

var _ cli.Command = (*rootCommand)(nil)

// NewRootCommand returns a new Client Command, which
// only exists to group all client related sub commands.
func NewRootCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	topicListCommand := NewListCommand(c2ClientFactory)
	topicCreateCommand := NewCreateCommand(c2ClientFactory)
	topicRemoveCommand := NewRemoveCommand(c2ClientFactory)
	listClientsCommand := NewListClientsCommand(c2ClientFactory)

	cmd := &rootCommand{}
	cobraCmd := &cobra.Command{
		Use:   "topic",
		Short: "group commands to interact with c2 topics",
	}

	cobraCmd.AddCommand(
		topicListCommand.CobraCmd(),
		topicCreateCommand.CobraCmd(),
		topicRemoveCommand.CobraCmd(),
		listClientsCommand.CobraCmd(),
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
