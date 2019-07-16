package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

type messageCommand struct {
	cobraCmd        *cobra.Command
	flags           messageCommandFlags
	c2ClientFactory cli.APIClientFactory
}

type messageCommandFlags struct {
	Topic   string
	Message string
}

var _ cli.Command = &messageCommand{}

// NewMessageCommand creates a new command allowing to
// send a message on a given topic
func NewMessageCommand(c2ClientFactory cli.APIClientFactory) cli.Command {
	messageCmd := &messageCommand{
		c2ClientFactory: c2ClientFactory,
	}

	cobraCmd := &cobra.Command{
		Use:   "message",
		Short: "Send a message on a topic",
		RunE:  messageCmd.run,
	}

	cobraCmd.Flags().SortFlags = false
	cobraCmd.Flags().StringVar(&messageCmd.flags.Topic, "topic", "", "The topic to send the message on")
	cobraCmd.Flags().StringVar(&messageCmd.flags.Message, "message", "", "The message to send")

	messageCmd.cobraCmd = cobraCmd

	return messageCmd
}

func (c *messageCommand) CobraCmd() *cobra.Command {
	return c.cobraCmd
}

func (c *messageCommand) run(cmd *cobra.Command, args []string) error {

	switch {
	case len(c.flags.Topic) <= 0:
		return fmt.Errorf("flag --topic is required")
	case len(c.flags.Message) <= 0:
		return fmt.Errorf("flag --message is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	c2Client, err := c.c2ClientFactory.NewClient(cmd)
	if err != nil {
		return fmt.Errorf("cannot create c2 api client: %v", err)
	}
	defer c2Client.Close()

	sendMessageReq := &pb.SendMessageRequest{
		Topic:   c.flags.Topic,
		Message: c.flags.Message,
	}

	_, err = c2Client.SendMessage(ctx, sendMessageReq)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}

	c.CobraCmd().Printf("Successfully sent message to topic %s\n", c.flags.Topic)
	return nil
}
