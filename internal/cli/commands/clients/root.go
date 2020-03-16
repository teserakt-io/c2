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
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
