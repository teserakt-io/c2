package commands

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"
)

func newTestCountCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewCountCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestCount(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No flags
			map[string]string{},
			// topics with topic
			map[string]string{
				"topics": "1",
				"topic":  "topic1",
			},
			// clients with client
			map[string]string{
				"clients": "1",
				"client":  "client1",
			},
			// topics and clients
			map[string]string{
				"clients": "1",
				"topics":  "1",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestCountCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute with --topics returns topics count", func(t *testing.T) {
		expectedCount := int64(42)
		c2Client.EXPECT().CountTopics(gomock.Any(), &pb.CountTopicsRequest{}).Return(&pb.CountTopicsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestCountCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("topics", "1")
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(fmt.Sprintf("%d\n", expectedCount))
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with --topics and --client returns topics for client count", func(t *testing.T) {
		expectedCount := int64(42)
		expectedClientName := "testClient1"

		c2Client.EXPECT().
			CountTopicsForClient(gomock.Any(), &pb.CountTopicsForClientRequest{Client: &pb.Client{Name: expectedClientName}}).
			Return(&pb.CountTopicsForClientResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestCountCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("topics", "1")
		cmd.CobraCmd().Flags().Set("client", expectedClientName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(fmt.Sprintf("%d\n", expectedCount))
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with --clients returns clients count", func(t *testing.T) {
		expectedCount := int64(42)
		c2Client.EXPECT().CountClients(gomock.Any(), &pb.CountClientsRequest{}).Return(&pb.CountClientsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestCountCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("clients", "1")
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(fmt.Sprintf("%d\n", expectedCount))
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with --clients and --topic returns clients for topic count", func(t *testing.T) {
		expectedCount := int64(42)
		expectedTopic := "testTopic1"

		c2Client.EXPECT().
			CountClientsForTopic(gomock.Any(), &pb.CountClientsForTopicRequest{Topic: expectedTopic}).
			Return(&pb.CountClientsForTopicResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestCountCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("clients", "1")
		cmd.CobraCmd().Flags().Set("topic", expectedTopic)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(fmt.Sprintf("%d\n", expectedCount))
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})
}
