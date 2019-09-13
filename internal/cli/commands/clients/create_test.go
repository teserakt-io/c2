package clients

import (
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestCreateCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewCreateCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestCreate(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			// No name
			map[string]string{},
			// No password nor key
			map[string]string{
				"name": "testClient1",
			},
			// Both password and key
			map[string]string{
				"name":     "testClient1",
				"key":      "6162636465",
				"password": "testPassword",
			},
			// Invalid key
			map[string]string{
				"name": "testClient1",
				"key":  "6162636465",
			},
			// Invalid name - too long
			map[string]string{
				"name":     strings.Repeat("a", e4crypto.NameMaxLen+1),
				"password": "testPassword",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestCreateCommand(c2ClientFactory)
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
		expectedPassword := "testSuperSecretPassword"

		k, err := e4crypto.DeriveSymKey(expectedPassword)
		if err != nil {
			t.Fatalf("failed to derive symKey: %v", err)
		}

		expectedRequest := &pb.NewClientRequest{
			Client: &pb.Client{Name: expectedClientName},
			Key:    k,
		}

		c2Client.EXPECT().NewClient(gomock.Any(), expectedRequest).Return(&pb.NewClientResponse{}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestCreateCommand(c2ClientFactory)
		cmd.CobraCmd().Flags().Set("name", expectedClientName)
		cmd.CobraCmd().Flags().Set("password", expectedPassword)
		err = cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
