package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type clientResetCommand struct {
	cobraCmd        *cobra.Command
	flags           clientResetCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type clientResetCommandFlags struct {
	Name string
}

// NewClientResetCommand returns a new command to reset a client
func NewClientResetCommand(c2ClientFactory cli.APIClientFactory) Command {
	resetCmd := &clientResetCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset a client",
		RunE:  resetCmd.run,
	}

	cobraCmd.Flags().StringVar(&resetCmd.flags.Name, "name", "", "The client name")

	resetCmd.cobraCmd = cobraCmd

	return resetCmd
}

func (c *clientResetCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *clientResetCommand) run(cmd *cobra.Command, args []string) error {
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

	_, err = c2Client.ResetClient(ctx, &pb.ResetClientRequest{Client: &pb.Client{Name: c.flags.Name}})
	if err != nil {
		return fmt.Errorf("failed to reset client: %v", err)
	}

	fmt.Printf("Client %s reset successfully\n", c.flags.Name)

	return nil
}
