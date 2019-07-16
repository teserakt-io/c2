package clients

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type listTopicsCommand struct {
	cobraCmd        *cobra.Command
	flags           listTopicsCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type listTopicsCommandFlags struct {
	Name   string
	Offset int64
	Count  int64
}

var _ cli.Command = &listTopicsCommand{}

// NewListTopicsCommand creates a new command allowing to
// list existing topics for a given client
func NewListTopicsCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	listTopicsCmd := &listTopicsCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "list-topics",
		Short: "List topics for a client",
		RunE:  listTopicsCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&listTopicsCmd.flags.Name, "name", "", "The client name")
	cobraCmd.Flags().Int64Var(&listTopicsCmd.flags.Offset, "offset", 0, "The offset to start listing topics from")
	cobraCmd.Flags().Int64Var(&listTopicsCmd.flags.Count, "count", 0, "The maximum number of topics to return, values <= 0 means all")

	listTopicsCmd.cobraCmd = cobraCmd

	return listTopicsCmd
}

func (c *listTopicsCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *listTopicsCommand) run(cmd *cobra.Command, args []string) error {
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

	countReq := &pb.CountTopicsForClientRequest{
		Client: &pb.Client{Name: c.flags.Name},
	}
	countResp, err := c2Client.CountTopicsForClient(ctx, countReq)
	if err != nil {
		return fmt.Errorf("failed to count topics for client: %v", err)
	}

	totalCount := countResp.Count
	if c.flags.Count > 0 {
		// Will fetch as many as requested by user, up to maximum number available
		totalCount = int64(math.Min(float64(totalCount), float64(c.flags.Count)))
	}

	currentOffset := c.flags.Offset
	for totalCount > 0 {
		count := int64(math.Min(float64(cli.MaxPageSize), float64(totalCount)))
		req := &pb.GetTopicsForClientRequest{
			Client: &pb.Client{Name: c.flags.Name},
			Count:  count,
			Offset: currentOffset,
		}

		resp, err := c2Client.GetTopicsForClient(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to fetch topics for client: %v", err)
		}

		currentOffset += count
		totalCount -= count
		for _, topic := range resp.Topics {
			c.CobraCmd().Println(topic)
		}
	}

	return nil
}
