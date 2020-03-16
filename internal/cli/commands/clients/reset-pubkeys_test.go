package clients

import (
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestResetPubKeysCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewResetPubKeysCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestResetPubKeys(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			map[string]string{},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestResetPubKeysCommand(c2ClientFactory)
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
		expectedTargetClientName := "testClient2"
		expectedRequest := &pb.ResetClientPubKeysRequest{
			TargetClient: &pb.Client{Name: expectedTargetClientName},
		}

		c2Client.EXPECT().ResetClientPubKeys(gomock.Any(), expectedRequest).Return(&pb.ResetClientPubKeysResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestResetPubKeysCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("target", expectedTargetClientName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
