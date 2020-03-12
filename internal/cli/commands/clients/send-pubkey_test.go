package clients

import (
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestSendPubKeyCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewSendPubKeyCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestSendPubKey(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			map[string]string{},
			map[string]string{"source": "clientA"},
			map[string]string{"target": "clientB"},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestSendPubKeyCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute forward expected request to the c2Client", func(t *testing.T) {
		expectedSourceClientName := "testClient1"
		expectedTargetClientName := "testClient2"
		expectedRequest := &pb.SendClientPubKeyRequest{
			SourceClient: &pb.Client{Name: expectedSourceClientName},
			TargetClient: &pb.Client{Name: expectedTargetClientName},
		}

		c2Client.EXPECT().SendClientPubKey(gomock.Any(), expectedRequest).Return(&pb.SendClientPubKeyResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestSendPubKeyCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("source", expectedSourceClientName)
		cmd.CobraCmd().Flags().Set("target", expectedTargetClientName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
