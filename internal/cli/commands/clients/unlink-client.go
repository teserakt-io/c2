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

type unlinkClientCommand struct {
	cobraCmd        *cobra.Command
	flags           unlinkClientCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type unlinkClientCommandFlags struct {
	SourceClientName string
	TargetClientName string
}

var _ cli.Command = (*unlinkClientCommand)(nil)

// NewUnlinkClientCommand returns a new command to unlink a client from another
func NewUnlinkClientCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	unlinkClientCommand := &unlinkClientCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "unlink-client",
		Short: "Unlink a client (source) from another client (target)",
		RunE:  unlinkClientCommand.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&unlinkClientCommand.flags.SourceClientName, "source", "", "The source client name")
	cobraCmd.Flags().StringVar(&unlinkClientCommand.flags.TargetClientName, "target", "", "The target client name")

	unlinkClientCommand.cobraCmd = cobraCmd

	return unlinkClientCommand
}

func (c *unlinkClientCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *unlinkClientCommand) run(cmd *cobra.Command, args []string) error {
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

	req := &pb.UnlinkClientRequest{
		SourceClient: &pb.Client{Name: c.flags.SourceClientName},
		TargetClient: &pb.Client{Name: c.flags.TargetClientName},
	}

	_, err = c2Client.UnlinkClient(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to link clients: %v", err)
	}

	c.CobraCmd().Printf("successfully unlinked client %s from client %s\n", c.flags.SourceClientName, c.flags.TargetClientName)

	return nil
}
