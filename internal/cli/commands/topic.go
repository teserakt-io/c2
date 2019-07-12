package commands

import (
	"github.com/spf13/cobra"
	"gitlab.com/teserakt/c2/internal/cli"
)

type topicCommand struct {
	cobraCmd *cobra.Command
}

var _ Command = &topicCommand{}

// NewTopicCommand returns a new Client Command, which
// only exists to group all client related sub commands.
func NewTopicCommand(c2ClientFactory cli.APIClientFactory) Command {

	topicListCommand := NewTopicListCommand(c2ClientFactory)
	topicCreateCommand := NewTopicCreateCommand(c2ClientFactory)

	cmd := &topicCommand{}
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

func (c *topicCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}
