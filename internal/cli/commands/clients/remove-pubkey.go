package clients

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type removePubKeyCommand struct {
	cobraCmd        *cobra.Command
	flags           removePubKeyCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type removePubKeyCommandFlags struct {
	SourceClientName string
	TargetClientName string
}

var _ cli.Command = (*removePubKeyCommand)(nil)

// NewRemovePubKeyCommand returns a new command to send a client pubkey to another client
func NewRemovePubKeyCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	removePubKeyCommand := &removePubKeyCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "remove-pubkey",
		Short: "Remove a client pubkey (source) from another client (target)",
		RunE:  removePubKeyCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&removePubKeyCommand.flags.SourceClientName, "source", "", "The source client name")
	cobraCmd.Flags().StringVar(&removePubKeyCommand.flags.TargetClientName, "target", "", "The target client name")

	removePubKeyCommand.cobraCmd = cobraCmd

	return removePubKeyCommand
}

func (c *removePubKeyCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *removePubKeyCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.SourceClientName) == 0 {
		return fmt.Errorf("flag --source is required")
	}
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

	req := &pb.RemoveClientPubKeyRequest{
		SourceClient: &pb.Client{Name: c.flags.SourceClientName},
		TargetClient: &pb.Client{Name: c.flags.TargetClientName},
	}

	_, err = c2Client.RemoveClientPubKey(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send pubkey removal command: %v", err)
	}

	c.CobraCmd().Printf("Command to remove client %s pubkey successfully sent to client %s\n", c.flags.SourceClientName, c.flags.TargetClientName)

	return nil
}
