package clients

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type resetPubKeysCommand struct {
	cobraCmd        *cobra.Command
	flags           resetPubKeysCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type resetPubKeysCommandFlags struct {
	TargetClientName string
}

var _ cli.Command = (*resetPubKeysCommand)(nil)

// NewResetPubKeysCommand returns a new command to send a client pubkey to another client
func NewResetPubKeysCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	resetPubKeysCommand := &resetPubKeysCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "reset-pubkeys",
		Short: "Remove all pubkeys from target client",
		RunE:  resetPubKeysCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&resetPubKeysCommand.flags.TargetClientName, "target", "", "The target client name")

	resetPubKeysCommand.cobraCmd = cobraCmd

	return resetPubKeysCommand
}

func (c *resetPubKeysCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *resetPubKeysCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.TargetClientName) == 0 {
		return fmt.Errorf("flag --target is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	req := &pb.ResetClientPubKeysRequest{
		TargetClient: &pb.Client{Name: c.flags.TargetClientName},
	}

	_, err = c2Client.ResetClientPubKeys(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send reset pubkeys command: %v", err)
	}

	c.CobraCmd().Printf("Command to reset pubkeys successfully sent to client %s\n", c.flags.TargetClientName)

	return nil
}
