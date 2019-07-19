package commands

import (
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

func newTestDetachCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewDetachCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestDetach(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No name, no topic
			map[string]string{},
			// No topic
			map[string]string{
				"client": "client1",
			},
			// No name
			map[string]string{
				"topic": "topic1",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestDetachCommand(c2ClientFactory)
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
		expectedClientName := "testClient1"

		expectedRequest := &pb.RemoveTopicClientRequest{
			Topic:  expectedTopic,
			Client: &pb.Client{Name: expectedClientName},
		}

		c2Client.EXPECT().RemoveTopicClient(gomock.Any(), expectedRequest).Return(&pb.RemoveTopicClientResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestDetachCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("client", expectedClientName)
		cmd.CobraCmd().Flags().Set("topic", expectedTopic)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

}
