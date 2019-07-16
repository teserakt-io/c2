package clients

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"gitlab.com/teserakt/c2/internal/cli"
	"gitlab.com/teserakt/c2/pkg/pb"

	"github.com/golang/mock/gomock"
)

func newTestListCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewListCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestList(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly return all clients", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := int64(10)

		expectedClientRequest := &pb.GetClientsRequest{
			Count: expectedCount,
		}

		expectedNames := []string{
			"client1",
			"client2",
			"client3",
		}

		expectedClients := []*pb.Client{}
		for _, name := range expectedNames {
			expectedClients = append(expectedClients, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClients(gomock.Any(), &pb.CountClientsRequest{}).Return(&pb.CountClientsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetClients(gomock.Any(), expectedClientRequest).Return(&pb.GetClientsResponse{Clients: expectedClients}, nil)
		c2Client.EXPECT().Close()

		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedNames, "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute properly paginate calls to api", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := cli.MaxPageSize + int64(1)
		expectedUserOffset := int64(10)

		expectedClientRequest1 := &pb.GetClientsRequest{
			Count:  cli.MaxPageSize,
			Offset: expectedUserOffset,
		}

		expectedClientRequest2 := &pb.GetClientsRequest{
			Count:  expectedCount - cli.MaxPageSize,
			Offset: expectedUserOffset + cli.MaxPageSize,
		}

		expectedNames1 := []string{
			"client1",
			"client2",
			"client3",
		}

		expectedNames2 := []string{
			"client4",
			"client5",
			"client6",
		}

		expectedClients1 := []*pb.Client{}
		for _, name := range expectedNames1 {
			expectedClients1 = append(expectedClients1, &pb.Client{Name: name})
		}

		expectedClients2 := []*pb.Client{}
		for _, name := range expectedNames2 {
			expectedClients2 = append(expectedClients2, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClients(gomock.Any(), &pb.CountClientsRequest{}).Return(&pb.CountClientsResponse{Count: expectedCount}, nil)
		gomock.InOrder(
			c2Client.EXPECT().GetClients(gomock.Any(), expectedClientRequest1).Return(&pb.GetClientsResponse{Clients: expectedClients1}, nil),
			c2Client.EXPECT().GetClients(gomock.Any(), expectedClientRequest2).Return(&pb.GetClientsResponse{Clients: expectedClients2}, nil),
		)
		c2Client.EXPECT().Close()

		cmd.CobraCmd().Flags().Set("offset", fmt.Sprintf("%d", expectedUserOffset))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(append(expectedNames1, expectedNames2...), "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with count request only this amount to the api", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := int64(10)
		expectedUserCount := int64(9)

		expectedClientRequest := &pb.GetClientsRequest{
			Count: expectedUserCount,
		}

		expectedNames := []string{
			"client1",
			"client2",
			"client3",
		}

		expectedClients := []*pb.Client{}
		for _, name := range expectedNames {
			expectedClients = append(expectedClients, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClients(gomock.Any(), &pb.CountClientsRequest{}).Return(&pb.CountClientsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetClients(gomock.Any(), expectedClientRequest).Return(&pb.GetClientsResponse{Clients: expectedClients}, nil)
		c2Client.EXPECT().Close()

		cmd.CobraCmd().Flags().Set("count", fmt.Sprintf("%d", expectedUserCount))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedNames, "\n") + "\n")
		if bytes.Compare(buf.Bytes(), expectedOutput) != 0 {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})
}
