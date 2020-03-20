package clients

import (
	"io/ioutil"
	"testing"

	"github.com/teserakt-io/c2/pkg/pb"

	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/cli"
)

func newTestUnlinkClientCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewUnlinkClientCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestUnlinkClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No source or target
			map[string]string{},
			// No source
			map[string]string{
				"target": "client2",
			},
			// No target
			map[string]string{
				"source": "client1",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestUnlinkClientCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute forward expected request to the c2Client when passing a name", func(t *testing.T) {
		expectedSourceName := "testClient1"
		expectedTargetName := "testClient2"
		expectedRequest := &pb.UnlinkClientRequest{
			SourceClient: &pb.Client{Name: expectedSourceName},
			TargetClient: &pb.Client{Name: expectedTargetName},
		}

		c2Client.EXPECT().UnlinkClient(gomock.Any(), expectedRequest).Return(&pb.UnlinkClientResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestUnlinkClientCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("source", expectedSourceName)
		cmd.CobraCmd().Flags().Set("target", expectedTargetName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
