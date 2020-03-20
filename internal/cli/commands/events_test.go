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
	"errors"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/ptypes"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestEventsCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewEventsCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestEvents(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	c2Client := cli.NewMockC2Client(mockCtrl)

	c2ClientFactory := cli.NewMockAPIClientFactory(mockCtrl)
	c2ClientFactory.EXPECT().NewClient(gomock.Any()).AnyTimes().Return(c2Client, nil)

	t.Run("Execute properly checks flags and return expected errors", func(t *testing.T) {
		badFlagsDataset := []map[string]string{}

		for _, flagData := range badFlagsDataset {
			cmd := newTestEventsCommand(c2ClientFactory)
			for name, value := range flagData {
				cmd.CobraCmd().Flags().Set(name, value)
			}
			err := cmd.CobraCmd().Execute()
			if err == nil {
				t.Error("Expected an error, got nil")
			}
		}
	})

	t.Run("Execute forward expected request to the c2Client when passing proper flags", func(t *testing.T) {
		expectedRequest := &pb.SubscribeToEventStreamRequest{}

		mockClient := pb.NewMockC2_SubscribeToEventStreamClient(mockCtrl)
		pbEvent := &pb.Event{
			Type:      pb.EventType_CLIENT_SUBSCRIBED,
			Source:    "client1",
			Target:    "topic1",
			Timestamp: ptypes.TimestampNow(),
		}

		cmd := newTestEventsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		mockClient.EXPECT().Recv().Return(pbEvent, nil)
		mockClient.EXPECT().Recv().Return(nil, errors.New("recv error test"))

		c2Client.EXPECT().SubscribeToEventStream(gomock.Any(), expectedRequest).Return(mockClient, nil)
		c2Client.EXPECT().Close()

		err := cmd.CobraCmd().Execute()
		if err == nil {
			t.Error("Expected an error, got nil")
		}

		if !strings.Contains(buf.String(), pb.EventType_CLIENT_SUBSCRIBED.String()) {
			t.Errorf("Expected output to have displayed the message type, got %s", buf.String())
		}
	})

}
