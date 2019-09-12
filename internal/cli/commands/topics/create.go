package topics

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type createCommand struct {
	cobraCmd        *cobra.Command
	flags           createCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type createCommandFlags struct {
	Name string
}

var _ cli.Command = (*createCommand)(nil)

// NewCreateCommand returns a new command to create a new topic
func NewCreateCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	createCmd := &createCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new topic",
		RunE:  createCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&createCmd.flags.Name, "name", "", "The topic name")

	createCmd.cobraCmd = cobraCmd

	return createCmd
}

func (c *createCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *createCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.Name) == 0 {
		return fmt.Errorf("flag --name is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	_, err = c2Client.NewTopic(ctx, &pb.NewTopicRequest{Topic: c.flags.Name})
	if err != nil {
		return fmt.Errorf("failed to create topic: %v", err)
	}

	c.CobraCmd().Printf("Topic %s created successfully\n", c.flags.Name)

	return nil
}
