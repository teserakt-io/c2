package clients

import (
	"io/ioutil"
	"testing"

	"github.com/teserakt-io/c2/pkg/pb"

	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/cli"
)

func newResetCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewResetCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestReset(t *testing.T) {
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
			cmd := newResetCommand(c2ClientFactory)
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
		expectedClientName := "testClient1"
		expectedRequest := &pb.ResetClientRequest{
			Client: &pb.Client{Name: expectedClientName},
		}

		c2Client.EXPECT().ResetClient(gomock.Any(), expectedRequest).Return(&pb.ResetClientResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newResetCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("name", expectedClientName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
