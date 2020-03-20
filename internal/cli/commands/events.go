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

	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

type eventsCommand struct {
	cobraCmd        *cobra.Command
	c2ClientFactory cli.APIClientFactory
}

var _ cli.Command = (*eventsCommand)(nil)

// NewEventsCommand creates a new command allowing to subscribe to C2 server event stream
// events will get printed on the command output (stdout by default)
func NewEventsCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	messageCmd := &eventsCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "events",
		Short: "Stream system events to stdout",
		RunE:  messageCmd.run,
	}

	messageCmd.cobraCmd = cobraCmd

	return messageCmd
}

func (c *eventsCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *eventsCommand) run(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	stream, err := c2Client.SubscribeToEventStream(ctx, &pb.SubscribeToEventStreamRequest{})
	if err != nil {
		return fmt.Errorf("failed to subscribe to event stream: %v", err)
	}

	c.CobraCmd().Println("Subscribed to server event stream...")
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		event, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("stream recv error: %v", err)
		}

		eventStr := fmt.Sprintf(
			"Type: %s, Source: %s, Target: %s, Timestamp: %s",
			event.Type.String(),
			event.Source,
			event.Target,
			ptypes.TimestampString(event.Timestamp),
		)

		c.CobraCmd().Printf("Received event: %s\n", eventStr)
	}
}
