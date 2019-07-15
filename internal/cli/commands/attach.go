package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type attachCommand struct {
	cobraCmd        *cobra.Command
	flags           attachCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type attachCommandFlags struct {
	ClientName string
	Topic      string
}

var _ cli.Command = &attachCommand{}

// NewAttachCommand creates a new command allowing to
// attach a client to a topic
func NewAttachCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	attachCmd := &attachCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "attach",
		Short: "Link a client to a topic",
		RunE:  attachCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&attachCmd.flags.ClientName, "client", "", "The client name to be linked to the topic")
	cobraCmd.Flags().StringVar(&attachCmd.flags.Topic, "topic", "", "The topic to be linked to the client")

	attachCmd.cobraCmd = cobraCmd

	return attachCmd
}

func (c *attachCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *attachCommand) run(cmd *cobra.Command, args []string) error {

	switch {
	case len(c.flags.ClientName) <= 0:
		return fmt.Errorf("flag --client is required")
	case len(c.flags.Topic) <= 0:
		return fmt.Errorf("flag --topic is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	newTopicClientReq := &pb.NewTopicClientRequest{
		Client: &pb.Client{Name: c.flags.ClientName},
		Topic:  c.flags.Topic,
	}

	_, err = c2Client.NewTopicClient(ctx, newTopicClientReq)
	if err != nil {
		return fmt.Errorf("failed to attach client to topic: %v", err)
	}

	fmt.Printf("Successfully attached client %s to topic %s\n", c.flags.ClientName, c.flags.Topic)
	return nil
}
