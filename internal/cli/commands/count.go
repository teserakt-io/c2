package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type countCommand struct {
	cobraCmd        *cobra.Command
	flags           countCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type countCommandFlags struct {
	CountClients bool
	CountTopics  bool
	ClientName   string
	Topic        string
}

var _ cli.Command = (*countCommand)(nil)

// NewCountCommand creates a new command allowing to
// count clients, topics, clients for topics or topics for clients
func NewCountCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	countCmd := &countCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "count",
		Short: "Retrieve counts of clients, topics, topics for client or clients for topic",
		RunE:  countCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().BoolVar(&countCmd.flags.CountClients, "clients", false, "Count clients")
	cobraCmd.Flags().StringVar(&countCmd.flags.ClientName, "client", "", "Use with --topics to restrict count to topics belonging to client")
	cobraCmd.Flags().BoolVar(&countCmd.flags.CountTopics, "topics", false, "Count topics")
	cobraCmd.Flags().StringVar(&countCmd.flags.Topic, "topic", "", "Use with --clients to restrict count to clients belonging to topic")

	countCmd.cobraCmd = cobraCmd

	return countCmd
}

func (c *countCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *countCommand) run(cmd *cobra.Command, args []string) error {

	switch {
	case !c.flags.CountClients && !c.flags.CountTopics:
		return fmt.Errorf("one of --clients or --topics is required")
	case c.flags.CountClients && c.flags.CountTopics:
		return fmt.Errorf("only one of --clients or --topics is required")
	case c.flags.CountClients && len(c.flags.ClientName) > 0:
		return fmt.Errorf("can't use --client when counting clients")
	case c.flags.CountTopics && len(c.flags.Topic) > 0:
		return fmt.Errorf("can't use --topic when counting topics")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	switch {
	case c.flags.CountClients:
		if len(c.flags.Topic) > 0 {
			resp, err := c2Client.CountClientsForTopic(ctx, &pb.CountClientsForTopicRequest{Topic: c.flags.Topic})
			if err != nil {
				return fmt.Errorf("failed to count clients for topic: %v", err)
			}

			c.CobraCmd().Println(resp.Count)

			return nil
		}

		resp, err := c2Client.CountClients(ctx, &pb.CountClientsRequest{})
		if err != nil {
			return fmt.Errorf("failed to count clients: %v", err)
		}

		c.CobraCmd().Println(resp.Count)

		return nil
	case c.flags.CountTopics:
		if len(c.flags.ClientName) > 0 {
			resp, err := c2Client.CountTopicsForClient(ctx, &pb.CountTopicsForClientRequest{Client: &pb.Client{Name: c.flags.ClientName}})
			if err != nil {
				return fmt.Errorf("failed to count topics for client: %v", err)
			}

			c.CobraCmd().Println(resp.Count)

			return nil
		}

		resp, err := c2Client.CountTopics(ctx, &pb.CountTopicsRequest{})
		if err != nil {
			return fmt.Errorf("failed to count topics: %v", err)
		}

		c.CobraCmd().Println(resp.Count)

		return nil
	default:
		return fmt.Errorf("unknown operation")
	}
}
