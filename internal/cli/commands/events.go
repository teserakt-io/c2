package commands

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type eventsCommand struct {
	cobraCmd        *cobra.Command
	flags           eventsCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type eventsCommandFlags struct {
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
