package clients

import (
	"io/ioutil"
	"os"
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

func createTempFile(t *testing.T, content []byte) (*os.File, func()) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "")
	if err != nil {
		t.Fatalf("failed to create temporary file: %v", err)
	}

	_, err = tmpFile.Write(content)
	if err != nil {
		t.Fatalf("failed to write content into file: %v", err)
	}

	return tmpFile, func() {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
	}
}

func TestCreate(t *testing.T) {
	expectedPassword := "veryLongSecretPassword"
	validPasswordFile, cleanup := createTempFile(t, []byte(expectedPassword))
	defer cleanup()
	validKeyFile, cleanup := createTempFile(t, e4crypto.RandomKey())
	defer cleanup()
	invalidPasswordFile, cleanup := createTempFile(t, []byte("tooShort"))
	defer cleanup()
	invalidKeyFile, cleanup := createTempFile(t, e4crypto.RandomKey()[:e4crypto.KeyLen-1])
	defer cleanup()

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
				"key":      validKeyFile.Name(),
				"password": validPasswordFile.Name(),
			},
			// Invalid key
			map[string]string{
				"name": "testClient1",
				"key":  invalidKeyFile.Name(),
			},
			// Invalid name - too long
			map[string]string{
				"name":     strings.Repeat("a", e4crypto.NameMaxLen+1),
				"password": invalidPasswordFile.Name(),
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
		cmd.CobraCmd().Flags().Set("password", validPasswordFile.Name())
		err = cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
