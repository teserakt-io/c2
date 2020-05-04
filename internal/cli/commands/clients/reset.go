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

type resetCommand struct {
	cobraCmd        *cobra.Command
	flags           resetCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type resetCommandFlags struct {
	Name string
}

var _ cli.Command = (*resetCommand)(nil)

// NewResetCommand returns a new command to reset a client
func NewResetCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	resetCmd := &resetCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset a client",
		RunE:  resetCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&resetCmd.flags.Name, "name", "", "The client name")

	resetCmd.cobraCmd = cobraCmd

	return resetCmd
}

func (c *resetCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *resetCommand) run(cmd *cobra.Command, args []string) error {
	if len(c.flags.Name) == 0 {
		return fmt.Errorf("flag --name is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	_, err = c2Client.ResetClient(ctx, &pb.ResetClientRequest{Client: &pb.Client{Name: c.flags.Name}})
	if err != nil {
		return fmt.Errorf("failed to reset client: %v", err)
	}

	c.CobraCmd().Printf("Client %s reset successfully\n", c.flags.Name)

	return nil
}
