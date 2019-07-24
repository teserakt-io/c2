package clients

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
	e4 "gitlab.com/teserakt/e4common"
)

type createCommand struct {
	cobraCmd        *cobra.Command
	flags           createCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type createCommandFlags struct {
	Name     string
	Password string
	Key      []byte
}

var _ cli.Command = (*createCommand)(nil)

// NewCreateCommand returns a new command to create clients
func NewCreateCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	createCmd := &createCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new client",
		Long:  fmt.Sprintf("Creates a new client, require an unique name, and either a password or a %d bytes hexadecimal key", e4.KeyLenHex),
		RunE:  createCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&createCmd.flags.Name, "name", "", "The client name")
	cobraCmd.Flags().BytesHexVar(&createCmd.flags.Key, "key", nil, fmt.Sprintf("The client %d bytes hexadecimal key", e4.KeyLenHex))
	cobraCmd.Flags().StringVar(&createCmd.flags.Password, "password", "", "The client password")

	createCmd.cobraCmd = cobraCmd

	return createCmd
}

func (c *createCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *createCommand) run(cmd *cobra.Command, args []string) error {
	switch {
	case len(c.flags.Name) == 0:
		return errors.New("flag --name is required")
	case len(c.flags.Password) == 0 && len(c.flags.Key) == 0:
		return errors.New("one of --password or --key is required")
	case len(c.flags.Password) > 0 && len(c.flags.Key) > 0:
		return errors.New("only one of --password or --key is allowed")
	}

	if err := e4.IsValidName(c.flags.Name); err != nil {
		return fmt.Errorf("invalid name: %v", err)
	}

	key := c.flags.Key
	if len(key) == 0 {
		key = e4.HashPwd(c.flags.Password)
	}

	if err := e4.IsValidKey(key); err != nil {
		return fmt.Errorf("invalid key: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	newClientReq := &pb.NewClientRequest{
		Client: &pb.Client{
			Name: c.flags.Name,
		},
		Key: key,
	}

	_, err = c2Client.NewClient(ctx, newClientReq)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	c.CobraCmd().Printf("Client %s created successfully\n", c.flags.Name)

	return nil
}
