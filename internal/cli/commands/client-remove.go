package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type clientRemoveCommand struct {
	cobraCmd        *cobra.Command
	flags           clientRemoveCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type clientRemoveCommandFlags struct {
	Name string
}

// NewClientRemoveCommand returns a new command to remove clients
func NewClientRemoveCommand(c2ClientFactory cli.APIClientFactory) Command {
	removeCmd := &clientRemoveCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove a client",
		RunE:  removeCmd.run,
	}

	cobraCmd.Flags().StringVar(&removeCmd.flags.Name, "name", "", "The client name")

	removeCmd.cobraCmd = cobraCmd

	return removeCmd
}

func (c *clientRemoveCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *clientRemoveCommand) run(cmd *cobra.Command, args []string) error {
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

	removeClientReq := &pb.RemoveClientRequest{
		Client: &pb.Client{
			Name: c.flags.Name,
		},
	}

	_, err = c2Client.RemoveClient(ctx, removeClientReq)
	if err != nil {
		return fmt.Errorf("failed to remove client: %v", err)
	}

	fmt.Printf("Client %s removed successfully\n", c.flags.Name)

	return nil
}
