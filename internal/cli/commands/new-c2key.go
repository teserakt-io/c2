package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type newC2KeyCommand struct {
	cobraCmd        *cobra.Command
	flags           newC2KeyCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type newC2KeyCommandFlags struct {
	Force bool
}

var _ cli.Command = (*newC2KeyCommand)(nil)

// NewNewC2KeyCommand returns a new command to change the C2 keys, and send the new public key to all clients
func NewNewC2KeyCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	newC2KeyCommand := &newC2KeyCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "new-c2key",
		Short: "Generate a new C2 key pair and send the new public key to all clients",
		RunE:  newC2KeyCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().BoolVar(&newC2KeyCommand.flags.Force, "force", false, "Force the execution")

	newC2KeyCommand.cobraCmd = cobraCmd

	return newC2KeyCommand
}

func (c *newC2KeyCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *newC2KeyCommand) run(cmd *cobra.Command, args []string) error {
	if !c.flags.Force {
		return errors.New("this command is potentially destructive. Please rerun it with --force to confirm")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	_, err = c2Client.NewC2Key(ctx, &pb.NewC2KeyRequest{Force: c.flags.Force})
	if err != nil {
		return fmt.Errorf("failed to set new C2 key: %v", err)
	}

	c.CobraCmd().Printf("New C2 key set with success\n")

	return nil
}
