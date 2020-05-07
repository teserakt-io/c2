// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clients

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"

	"golang.org/x/crypto/ed25519"

	"github.com/spf13/cobra"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type createCommand struct {
	cobraCmd        *cobra.Command
	flags           createCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type createCommandFlags struct {
	Name         string
	PasswordPath string
	KeyPath      string
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
		Long:  fmt.Sprintf("Creates a new client, require an unique name, and a file containing either a password or a %d bytes key", e4crypto.KeyLen),
		RunE:  createCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&createCmd.flags.Name, "name", "", "The client name")
	cobraCmd.Flags().StringVar(&createCmd.flags.KeyPath, "key", "", fmt.Sprintf("Filepath to a %d bytes key", e4crypto.KeyLen))
	cobraCmd.Flags().StringVar(&createCmd.flags.PasswordPath, "password", "", "Filepath to a plaintext password file")

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
	case len(c.flags.PasswordPath) == 0 && len(c.flags.KeyPath) == 0:
		return errors.New("one of --password or --key is required")
	case len(c.flags.PasswordPath) > 0 && len(c.flags.KeyPath) > 0:
		return errors.New("only one of --password or --key is allowed")
	}

	if err := e4crypto.ValidateName(c.flags.Name); err != nil {
		return fmt.Errorf("invalid name: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	resp, err := c2Client.GetCryptoMode(ctx, &pb.GetCryptoModeRequest{})
	if err != nil {
		return fmt.Errorf("failed to get crypto-mode: %v", err)
	}

	var key []byte
	if len(c.flags.KeyPath) > 0 {
		var err error
		key, err = ioutil.ReadFile(c.flags.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read symKey from file: %v", err)
		}
	} else {
		var err error
		password, err := ioutil.ReadFile(c.flags.PasswordPath)
		if err != nil {
			return fmt.Errorf("failed to read password from file: %v", err)
		}

		switch resp.CryptoMode {
		case pb.CryptoMode_CRYPTOMODE_SYMKEY:
			key, err = e4crypto.DeriveSymKey(string(password))
			if err != nil {
				return fmt.Errorf("failed to derive symKey from password: %v", err)
			}
			c.CobraCmd().Println("Derived symmetric key from password")
		case pb.CryptoMode_CRYPTOMODE_PUBKEY:
			privateKey, err := e4crypto.Ed25519PrivateKeyFromPassword(string(password))
			if err != nil {
				return fmt.Errorf("failed to derive ed25519 private key from password: %v", err)
			}
			publicKey := ed25519.PrivateKey(privateKey).Public()
			key = publicKey.(ed25519.PublicKey)
			c.CobraCmd().Println("Derivated Ed25519 key from password")
		default:
			return fmt.Errorf("unsupported crypto-mode: %v", resp.CryptoMode)
		}
	}

	switch resp.CryptoMode {
	case pb.CryptoMode_CRYPTOMODE_SYMKEY:
		if err := e4crypto.ValidateSymKey(key); err != nil {
			return fmt.Errorf("invalid key: %v", err)
		}
	case pb.CryptoMode_CRYPTOMODE_PUBKEY:
		if err := e4crypto.ValidateEd25519PubKey(key); err != nil {
			return fmt.Errorf("invalid key: %v", err)
		}
	default:
		return fmt.Errorf("unsupported crypto-mode: %v", resp.CryptoMode)
	}

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
