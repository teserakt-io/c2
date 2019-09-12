package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/teserakt-io/c2/pkg/pb"

	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/cli"
)

func newTestListTopicsCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewListTopicsCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestListTopics(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No name
			map[string]string{},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestListTopicsCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute forward expected request to the c2Client when passing a password", func(t *testing.T) {
		expectedClientName := "testClient1"
		expectedCount := int64(42)

		expectedCountRequest := &pb.CountTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
		}

		expectedListRequest := &pb.GetTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
			Count:  expectedCount,
		}

		expectedTopics := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		c2Client.EXPECT().CountTopicsForClient(gomock.Any(), expectedCountRequest).Return(&pb.CountTopicsForClientResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetTopicsForClient(gomock.Any(), expectedListRequest).Return(&pb.GetTopicsForClientResponse{Topics: expectedTopics}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestListTopicsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedClientName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedTopics, "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute paginate calls to the api", func(t *testing.T) {
		expectedClientName := "testClient1"
		expectedOffset := int64(1)

		expectedCountRequest := &pb.CountTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
		}

		expectedCount := int64(cli.MaxPageSize + 1)

		expectedListRequest1 := &pb.GetTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
			Count:  cli.MaxPageSize,
			Offset: expectedOffset,
		}
		expectedListRequest2 := &pb.GetTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
			Count:  expectedCount - cli.MaxPageSize,
			Offset: expectedListRequest1.Count + expectedOffset,
		}

		expectedTopics1 := []string{
			"topic1",
			"topic2",
			"topic3",
		}
		expectedTopics2 := []string{
			"topic4",
			"topic5",
		}

		c2Client.EXPECT().CountTopicsForClient(gomock.Any(), expectedCountRequest).Return(&pb.CountTopicsForClientResponse{Count: expectedCount}, nil)
		gomock.InOrder(
			c2Client.EXPECT().GetTopicsForClient(gomock.Any(), expectedListRequest1).Return(&pb.GetTopicsForClientResponse{Topics: expectedTopics1}, nil),
			c2Client.EXPECT().GetTopicsForClient(gomock.Any(), expectedListRequest2).Return(&pb.GetTopicsForClientResponse{Topics: expectedTopics2}, nil),
		)
		c2Client.EXPECT().Close()

		cmd := newTestListTopicsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedClientName)
		cmd.CobraCmd().Flags().Set("offset", fmt.Sprintf("%d", expectedOffset))

		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(append(expectedTopics1, expectedTopics2...), "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with count request only this amount to the api", func(t *testing.T) {
		expectedClientName := "testClient1"
		expectedCount := int64(42)
		expectedUserCount := int64(10)

		expectedCountRequest := &pb.CountTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
		}

		expectedListRequest := &pb.GetTopicsForClientRequest{
			Client: &pb.Client{Name: expectedClientName},
			Count:  expectedUserCount,
		}

		expectedTopics := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		c2Client.EXPECT().CountTopicsForClient(gomock.Any(), expectedCountRequest).Return(&pb.CountTopicsForClientResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetTopicsForClient(gomock.Any(), expectedListRequest).Return(&pb.GetTopicsForClientResponse{Topics: expectedTopics}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestListTopicsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedClientName)
		cmd.CobraCmd().Flags().Set("count", fmt.Sprintf("%d", expectedUserCount))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedTopics, "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})
}
