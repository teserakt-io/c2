package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type clientNewKeyCommand struct {
	cobraCmd        *cobra.Command
	flags           clientNewKeyCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type clientNewKeyCommandFlags struct {
	Name string
}

// NewClientNewKeyCommand returns a new command to regenerate a client key
func NewClientNewKeyCommand(c2ClientFactory cli.APIClientFactory) Command {
	resetCmd := &clientNewKeyCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "new-key",
		Short: "Regenerate a client key",
		RunE:  resetCmd.run,
	}

	cobraCmd.Flags().StringVar(&resetCmd.flags.Name, "name", "", "The client name")

	resetCmd.cobraCmd = cobraCmd

	return resetCmd
}

func (c *clientNewKeyCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *clientNewKeyCommand) run(cmd *cobra.Command, args []string) error {
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

	_, err = c2Client.NewClientKey(ctx, &pb.NewClientKeyRequest{Client: &pb.Client{Name: c.flags.Name}})
	if err != nil {
		return fmt.Errorf("failed to regenerate client key: %v", err)
	}

	fmt.Printf("Client %s key regenerated successfully\n", c.flags.Name)

	return nil
}
