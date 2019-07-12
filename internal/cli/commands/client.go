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

	cmd := &clientCommand{}
	cobraCmd := &cobra.Command{
		Use: "client",
	}

	cobraCmd.AddCommand(
		clientListCommand.CobraCmd(),
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *clientCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
