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

type protectMessageCommand struct {
	cobraCmd        *cobra.Command
	flags           protectMessageCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type protectMessageCommandFlags struct {
	Topic   string
	Message []byte
}

var _ cli.Command = (*protectMessageCommand)(nil)

// NewProtectMessageCommand returns a new command to protect a given message with a topic key
func NewProtectMessageCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	protectMessageCommand := &protectMessageCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "protect-message",
		Short: "Protect a message for a specific topic",
		RunE:  protectMessageCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&protectMessageCommand.flags.Topic, "topic", "", "The destination topic")
	cobraCmd.Flags().BytesBase64Var(&protectMessageCommand.flags.Message, "message", nil, "A base64 encoded message")

	protectMessageCommand.cobraCmd = cobraCmd

	return protectMessageCommand
}

func (c *protectMessageCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *protectMessageCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.Topic) == 0 {
		return errors.New("--topic is required")
	}

	if len(c.flags.Message) == 0 {
		return errors.New("--message is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	resp, err := c2Client.ProtectMessage(ctx, &pb.ProtectMessageRequest{
		Topic:      c.flags.Topic,
		BinaryData: c.flags.Message,
	})
	if err != nil {
		return fmt.Errorf("failed to protect message: %v", err)
	}

	c.CobraCmd().Print(base64.StdEncoding.EncodeToString(resp.ProtectedBinaryData))

	return nil
}
