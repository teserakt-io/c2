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

package clients

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
	listCommand := NewListCommand(c2ClientFactory)
	createCommand := NewCreateCommand(c2ClientFactory)
	removeCommand := NewRemoveCommand(c2ClientFactory)
	listTopicsCommand := NewListTopicsCommand(c2ClientFactory)
	resetCommand := NewResetCommand(c2ClientFactory)
	newKeyCommand := NewNewKeyCommand(c2ClientFactory)
	sendPubKeyCommand := NewSendPubKeyCommand(c2ClientFactory)
	removePubKeyCommand := NewRemovePubKeyCommand(c2ClientFactory)
	resetPubKeysCommand := NewResetPubKeysCommand(c2ClientFactory)
	linkClientCommand := NewLinkClientCommand(c2ClientFactory)
	unlinkClientCommand := NewUnlinkClientCommand(c2ClientFactory)
	listLinkedClientsCommand := NewListLinkedClientsCommand(c2ClientFactory)

	cmd := &rootCommand{}
	cobraCmd := &cobra.Command{
		Use:   "client",
		Short: "group commands to interact with c2 clients",
	}

	cobraCmd.AddCommand(
		listCommand.CobraCmd(),
		createCommand.CobraCmd(),
		removeCommand.CobraCmd(),
		listTopicsCommand.CobraCmd(),
		resetCommand.CobraCmd(),
		newKeyCommand.CobraCmd(),
		sendPubKeyCommand.CobraCmd(),
		removePubKeyCommand.CobraCmd(),
		resetPubKeysCommand.CobraCmd(),
		linkClientCommand.CobraCmd(),
		unlinkClientCommand.CobraCmd(),
		listLinkedClientsCommand.CobraCmd(),
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
