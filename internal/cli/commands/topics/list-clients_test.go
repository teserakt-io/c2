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

package topics

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/cli"
	"github.com/teserakt-io/c2/pkg/pb"
)

func newTestListClientsCommand(clientFactory cli.APIClientFactory) cli.Command {
	cmd := NewListClientsCommand(clientFactory)
	cmd.CobraCmd().SetOutput(ioutil.Discard)
	cmd.CobraCmd().DisableFlagParsing = true

	return cmd
}

func TestListClients(t *testing.T) {
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
			cmd := newTestListClientsCommand(c2ClientFactory)
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
		expectedTopicName := "testTopic1"
		expectedCount := int64(42)

		expectedCountRequest := &pb.CountClientsForTopicRequest{
			Topic: expectedTopicName,
		}

		expectedListRequest := &pb.GetClientsForTopicRequest{
			Topic: expectedTopicName,
			Count: expectedCount,
		}

		expectedClientNames := []string{
			"client1",
			"client2",
			"client3",
		}
		expectedClients := []*pb.Client{}
		for _, name := range expectedClientNames {
			expectedClients = append(expectedClients, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClientsForTopic(gomock.Any(), expectedCountRequest).Return(&pb.CountClientsForTopicResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetClientsForTopic(gomock.Any(), expectedListRequest).Return(&pb.GetClientsForTopicResponse{Clients: expectedClients}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestListClientsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedTopicName)
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedClientNames, "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute paginate calls to the api", func(t *testing.T) {
		expectedTopicName := "testTopic1"
		expectedOffset := int64(1)

		expectedCountRequest := &pb.CountClientsForTopicRequest{
			Topic: expectedTopicName,
		}

		expectedCount := int64(cli.MaxPageSize + 1)

		expectedListRequest1 := &pb.GetClientsForTopicRequest{
			Topic:  expectedTopicName,
			Count:  cli.MaxPageSize,
			Offset: expectedOffset,
		}
		expectedListRequest2 := &pb.GetClientsForTopicRequest{
			Topic:  expectedTopicName,
			Count:  expectedCount - cli.MaxPageSize,
			Offset: expectedListRequest1.Count + expectedOffset,
		}

		expectedClientNames1 := []string{
			"topic1",
			"topic2",
			"topic3",
		}
		expectedClientNames2 := []string{
			"topic4",
			"topic5",
		}

		expectedClients1 := []*pb.Client{}
		for _, name := range expectedClientNames1 {
			expectedClients1 = append(expectedClients1, &pb.Client{Name: name})
		}

		expectedClients2 := []*pb.Client{}
		for _, name := range expectedClientNames2 {
			expectedClients2 = append(expectedClients2, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClientsForTopic(gomock.Any(), expectedCountRequest).Return(&pb.CountClientsForTopicResponse{Count: expectedCount}, nil)
		gomock.InOrder(
			c2Client.EXPECT().GetClientsForTopic(gomock.Any(), expectedListRequest1).Return(&pb.GetClientsForTopicResponse{Clients: expectedClients1}, nil),
			c2Client.EXPECT().GetClientsForTopic(gomock.Any(), expectedListRequest2).Return(&pb.GetClientsForTopicResponse{Clients: expectedClients2}, nil),
		)
		c2Client.EXPECT().Close()

		cmd := newTestListClientsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedTopicName)
		cmd.CobraCmd().Flags().Set("offset", fmt.Sprintf("%d", expectedOffset))

		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(append(expectedClientNames1, expectedClientNames2...), "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with count request only this amount to the api", func(t *testing.T) {
		expectedTopicName := "testTopic1"
		expectedCount := int64(42)
		expectedUserCount := int64(10)

		expectedCountRequest := &pb.CountClientsForTopicRequest{
			Topic: expectedTopicName,
		}

		expectedListRequest := &pb.GetClientsForTopicRequest{
			Topic: expectedTopicName,
			Count: expectedUserCount,
		}

		expectedClientNames := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		expectedClients := []*pb.Client{}
		for _, name := range expectedClientNames {
			expectedClients = append(expectedClients, &pb.Client{Name: name})
		}

		c2Client.EXPECT().CountClientsForTopic(gomock.Any(), expectedCountRequest).Return(&pb.CountClientsForTopicResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetClientsForTopic(gomock.Any(), expectedListRequest).Return(&pb.GetClientsForTopicResponse{Clients: expectedClients}, nil)
		c2Client.EXPECT().Close()

		cmd := newTestListClientsCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		cmd.CobraCmd().Flags().Set("name", expectedTopicName)
		cmd.CobraCmd().Flags().Set("count", fmt.Sprintf("%d", expectedUserCount))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedClientNames, "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})
}
