package topics

import (
	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
)

type rootCommand struct {
	cobraCmd *cobra.Command
}

var _ cli.Command = &rootCommand{}

// NewRootCommand returns a new Client Command, which
// only exists to group all client related sub commands.
func NewRootCommand(c2ClientFactory cli.APIClientFactory) cli.Command {

	topicListCommand := NewListCommand(c2ClientFactory)
	topicCreateCommand := NewCreateCommand(c2ClientFactory)

	cmd := &rootCommand{}
	cobraCmd := &cobra.Command{
		Use:   "topic",
		Short: "group commands to interact with c2 topics",
	}

	cobraCmd.AddCommand(
		topicListCommand.CobraCmd(),
		topicCreateCommand.CobraCmd(),
	)

	cmd.cobraCmd = cobraCmd

	return cmd
}

func (c *rootCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
