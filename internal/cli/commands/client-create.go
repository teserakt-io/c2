package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
	e4 "gitlab.com/teserakt/e4common"
)

type clientCreateCommand struct {
	cobraCmd        *cobra.Command
	flags           clientCreateCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type clientCreateCommandFlags struct {
	Name     string
	Password string
	Key      []byte
}

// NewClientCreateCommand returns a new command to create clients
func NewClientCreateCommand(c2ClientFactory cli.APIClientFactory) Command {
	createCmd := &clientCreateCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "create",
		Short: "Creates a new client",
		Long:  fmt.Sprintf("Creates a new client, require an unique name, and either a password or a %d bytes hexadecimal key", e4.KeyLenHex),
		RunE:  createCmd.run,
	}

	cobraCmd.Flags().StringVar(&createCmd.flags.Name, "name", "", "The client name")
	cobraCmd.Flags().StringVar(&createCmd.flags.Password, "password", "", "The client password")
	cobraCmd.Flags().BytesHexVar(&createCmd.flags.Key, "key", nil, fmt.Sprintf("The client %d bytes hexadecimal key", e4.KeyLenHex))

	createCmd.cobraCmd = cobraCmd

	return createCmd
}

func (c *clientCreateCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *clientCreateCommand) run(cmd *cobra.Command, args []string) error {
	switch {
	case len(c.flags.Name) == 0:
		return fmt.Errorf("flag --name is required")
	case len(c.flags.Password) == 0 && len(c.flags.Key) == 0:
		return fmt.Errorf("one of --password or --key is required")
	case len(c.flags.Password) > 0 && len(c.flags.Key) > 0:
		return fmt.Errorf("only one of --password or --key is allowed")
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create api client: %v", err)
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

	fmt.Printf("Client %s created successfully\n", c.flags.Name)

	return nil
}
