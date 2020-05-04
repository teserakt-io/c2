// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package commands

import (
	"bytes"
	"encoding/base64"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestUnprotectMessageCommand(clientFactory cli.APIClientFactory, output io.Writer) cli.Command {
	cmd := NewUnprotectMessageCommand(clientFactory)
	cmd.CobraCmd().SetOutput(output)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestUnprotectMessage(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{
			{
				"topic":   "",
				"message": base64.StdEncoding.EncodeToString([]byte("valid")),
			},
			{
				"topic":   "valid",
				"message": "",
			},
			{
				"topic":   "valid",
				"message": "invalid_b64",
			},
		}

		for _, flagData := range badFlagsDataset {
			cmd := newTestUnprotectMessageCommand(c2ClientFactory, ioutil.Discard)
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
		topic := "topic1"
		protectedMessage := []byte("protected-message")
		expectedRequest := &pb.UnprotectMessageRequest{
			Topic:               topic,
			ProtectedBinaryData: protectedMessage,
		}

		clearMessage := []byte("clear-message")
		expectedResponse := &pb.UnprotectMessageResponse{
			Topic:      topic,
			BinaryData: clearMessage,
		}

		c2Client.EXPECT().UnprotectMessage(gomock.Any(), expectedRequest).Return(expectedResponse, nil)
		c2Client.EXPECT().Close()

		output := bytes.NewBuffer(nil)
		cmd := newTestUnprotectMessageCommand(c2ClientFactory, output)
		cmd.CobraCmd().Flags().Set("topic", topic)
		cmd.CobraCmd().Flags().Set("message", base64.StdEncoding.EncodeToString(protectedMessage))

		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		b64UnprotectedMessage := base64.StdEncoding.EncodeToString(clearMessage)
		if g, w := output.String(), b64UnprotectedMessage; g != w {
			t.Fatalf("invalid output, got %s, want %s", g, w)
		}
	})
}
