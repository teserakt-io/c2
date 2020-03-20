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
	"fmt"

	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type detachCommand struct {
	cobraCmd        *cobra.Command
	flags           detachCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type detachCommandFlags struct {
	ClientName string
	Topic      string
}

var _ cli.Command = (*detachCommand)(nil)

// NewDetachCommand creates a new command allowing to
// detach a client from a topic
func NewDetachCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	detachCmd := &detachCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "detach",
		Short: "Unlink a client to a topic",
		RunE:  detachCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&detachCmd.flags.ClientName, "client", "", "The client name to be unlinked to the topic")
	cobraCmd.Flags().StringVar(&detachCmd.flags.Topic, "topic", "", "The topic to be unlinked to the client")

	detachCmd.cobraCmd = cobraCmd

	return detachCmd
}

func (c *detachCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *detachCommand) run(cmd *cobra.Command, args []string) error {
	switch {
	case len(c.flags.ClientName) <= 0:
		return fmt.Errorf("flag --client is required")
	case len(c.flags.Topic) <= 0:
		return fmt.Errorf("flag --topic is required")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	removeTopicClientReq := &pb.RemoveTopicClientRequest{
		Client: &pb.Client{Name: c.flags.ClientName},
		Topic:  c.flags.Topic,
	}

	_, err = c2Client.RemoveTopicClient(ctx, removeTopicClientReq)
	if err != nil {
		return fmt.Errorf("failed to detach client from topic: %v", err)
	}

	c.CobraCmd().Printf("Successfully detached client %s from topic %s\n", c.flags.ClientName, c.flags.Topic)
	return nil
}
