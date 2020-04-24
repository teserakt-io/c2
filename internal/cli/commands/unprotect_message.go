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

package commands

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type unprotectMessageCommand struct {
	cobraCmd        *cobra.Command
	flags           unprotectMessageCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type unprotectMessageCommandFlags struct {
	Topic            string
	ProtectedMessage []byte
}

var _ cli.Command = (*unprotectMessageCommand)(nil)

// NewUnprotectMessageCommand returns a new command to unprotect a given message with a topic key
func NewUnprotectMessageCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	unprotectMessageCommand := &unprotectMessageCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "unprotect-message",
		Short: "unprotect a message for a specific topic",
		RunE:  unprotectMessageCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&unprotectMessageCommand.flags.Topic, "topic", "", "The source topic")
	cobraCmd.Flags().BytesBase64Var(&unprotectMessageCommand.flags.ProtectedMessage, "message", nil, "A base64 encoded protected message")

	unprotectMessageCommand.cobraCmd = cobraCmd

	return unprotectMessageCommand
}

func (c *unprotectMessageCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *unprotectMessageCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.Topic) == 0 {
		return errors.New("--topic is required")
	}

	if len(c.flags.ProtectedMessage) == 0 {
		return errors.New("--message is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	resp, err := c2Client.UnprotectMessage(ctx, &pb.UnprotectMessageRequest{
		Topic:               c.flags.Topic,
		ProtectedBinaryData: c.flags.ProtectedMessage,
	})
	if err != nil {
		return fmt.Errorf("failed to unprotect message: %v", err)
	}

	c.CobraCmd().Print(base64.StdEncoding.EncodeToString(resp.BinaryData))

	return nil
}
