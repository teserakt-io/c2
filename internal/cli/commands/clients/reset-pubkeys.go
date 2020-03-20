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

// NewResetPubKeysCommand returns a new command to remove all pubkeys from a client
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
