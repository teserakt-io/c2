package commands

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

const (
	// MaxPageSize defines the maximum number of clients to fetch per api query
	MaxPageSize int64 = 100
)

type clientListCommand struct {
	cobraCmd        *cobra.Command
	flags           clientListCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type clientListCommandFlags struct {
	Offset int64
	Count  int64
}

var _ Command = &clientListCommand{}

// NewClientListCommand creates a new command allowing to
// list existing clients
func NewClientListCommand(c2ClientFactory cli.APIClientFactory) Command {
	listCmd := &clientListCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "list",
		Short: "List all clients",
		RunE:  listCmd.run,
	}

	cobraCmd.Flags().Int64Var(&listCmd.flags.Offset, "offset", 0, "The offset to start listing clients from.")
	cobraCmd.Flags().Int64Var(&listCmd.flags.Count, "count", 0, "The maximum number of clients to return, values <= 0 means all")

	listCmd.cobraCmd = cobraCmd

	return listCmd
}

func (c *clientListCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *clientListCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	countResp, err := c2Client.CountClients(ctx, &pb.CountClientsRequest{})
	if err != nil {
		return fmt.Errorf("failed to get client count: %v", err)
	}

	totalCount := countResp.Count
	if c.flags.Count > 0 {
		// Will fetch as many as requested by user, up to maximum number of available clients
		totalCount = int64(math.Min(float64(totalCount), float64(c.flags.Count)))
	}

	currentOffset := c.flags.Offset
	for totalCount > 0 {
		count := int64(math.Min(float64(MaxPageSize), float64(totalCount)))
		req := &pb.GetClientsRequest{
			Count:  count,
			Offset: currentOffset,
		}

		resp, err := c2Client.GetClients(ctx, req)
		if err != nil {
			return fmt.Errorf("failed to fetch clients: %v", err)
		}

		currentOffset += count
		totalCount -= count
		for _, client := range resp.Clients {
			fmt.Println(client.Name)
		}
	}

	return nil
}
