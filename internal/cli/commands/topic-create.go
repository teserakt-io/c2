package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type topicCreateCommand struct {
	cobraCmd        *cobra.Command
	flags           topicCreateCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type topicCreateCommandFlags struct {
	Name string
}

// NewTopicCreateCommand returns a new command to create a new topic
func NewTopicCreateCommand(c2ClientFactory cli.APIClientFactory) Command {
	createCmd := &topicCreateCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new topic",
		RunE:  createCmd.run,
	}

	cobraCmd.Flags().StringVar(&createCmd.flags.Name, "name", "", "The topic name")

	createCmd.cobraCmd = cobraCmd

	return createCmd
}

func (c *topicCreateCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *topicCreateCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.Name) == 0 {
		return fmt.Errorf("flag --name is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
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

	fmt.Printf("Topic %s created successfully\n", c.flags.Name)

	return nil
}
