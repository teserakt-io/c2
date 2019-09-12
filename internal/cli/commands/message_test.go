package commands

import (
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestMessageCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewMessageCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No topic, no message
			map[string]string{},
			// No topic
			map[string]string{
				"message": "test message",
			},
			// No message
			map[string]string{
				"topic": "topic1",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestMessageCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute forward expected request to the c2Client when passing proper flags", func(t *testing.T) {
		expectedTopic := "testTopic1"
		expectedMessage := "test message"

		expectedRequest := &pb.SendMessageRequest{
			Topic:   expectedTopic,
			Message: expectedMessage,
		}

		c2Client.EXPECT().SendMessage(gomock.Any(), expectedRequest).Return(&pb.SendMessageResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestMessageCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("topic", expectedTopic)
		cmd.CobraCmd().Flags().Set("message", expectedMessage)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

}
