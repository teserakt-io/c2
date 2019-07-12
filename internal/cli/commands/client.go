package commands

import (
	"github.com/spf13/cobra"
	"gitlab.com/teserakt/c2/internal/cli"
)

type clientCommand struct {
	cobraCmd *cobra.Command
}

var _ Command = &clientCommand{}

// NewClientCommand returns a new Client Command, which
// only exists to group all client related sub commands.
func NewClientCommand(c2ClientFactory cli.APIClientFactory) Command {
	clientListCommand := NewClientListCommand(c2ClientFactory)
	clientCreateCommand := NewClientCreateCommand(c2ClientFactory)
	clientRemoveCommand := NewClientRemoveCommand(c2ClientFactory)
	clientListTopics := NewClientListTopicsCommand(c2ClientFactory)

	cmd := &clientCommand{}
	cobraCmd := &cobra.Command{
		Use:   "client",
		Short: "group commands to interact with c2 clients",
	}

	cobraCmd.AddCommand(
		clientListCommand.CobraCmd(),
		clientCreateCommand.CobraCmd(),
		clientRemoveCommand.CobraCmd(),
		clientListTopics.CobraCmd(),
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *clientCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
