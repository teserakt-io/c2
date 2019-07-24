package topics

import (
	"context"
	"fmt"
	"math"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type listClientsCommand struct {
	cobraCmd        *cobra.Command
	flags           listClientsCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type listClientsCommandFlags struct {
	Name   string
	Offset int64
	Count  int64
}

var _ cli.Command = (*listClientsCommand)(nil)

// NewListClientsCommand creates a new command allowing to
// list existing clients for a given topic
func NewListClientsCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	listClientsCmd := &listClientsCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "list-clients",
		Short: "List clients for a topic",
		RunE:  listClientsCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&listClientsCmd.flags.Name, "name", "", "The topic name")
	cobraCmd.Flags().Int64Var(&listClientsCmd.flags.Offset, "offset", 0, "The offset to start listing clients from")
	cobraCmd.Flags().Int64Var(&listClientsCmd.flags.Count, "count", 0, "The maximum number of clients to return, values <= 0 means all")

	listClientsCmd.cobraCmd = cobraCmd

	return listClientsCmd
}

func (c *listClientsCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *listClientsCommand) run(cmd *cobra.Command, args []string) error {
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

	countReq := &pb.CountClientsForTopicRequest{
		Topic: c.flags.Name,
	}
	countResp, err := c2Client.CountClientsForTopic(ctx, countReq)
	if err != nil {
		return fmt.Errorf("failed to count clients for topic: %v", err)
	}

	totalCount := countResp.Count
	if c.flags.Count > 0 {
		// Will fetch as many as requested by user, up to maximum number available
		totalCount = int64(math.Min(float64(totalCount), float64(c.flags.Count)))
	}

	currentOffset := c.flags.Offset
	for totalCount > 0 {
		count := int64(math.Min(float64(cli.MaxPageSize), float64(totalCount)))
		req := &pb.GetClientsForTopicRequest{
			Topic:  c.flags.Name,
			Count:  count,
			Offset: currentOffset,
		}

		resp, err := c2Client.GetClientsForTopic(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to fetch clients for topic: %v", err)
		}

		currentOffset += count
		totalCount -= count
		for _, client := range resp.Clients {
			c.CobraCmd().Println(client.Name)
		}
	}

	return nil
}
