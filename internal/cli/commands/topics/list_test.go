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

	t.Run("Execute properly return all topics", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := int64(10)

		expectedTopicsRequest := &pb.GetTopicsRequest{
			Count: expectedCount,
		}

		expectedTopics := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		c2Client.EXPECT().CountTopics(gomock.Any(), &pb.CountTopicsRequest{}).Return(&pb.CountTopicsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetTopics(gomock.Any(), expectedTopicsRequest).Return(&pb.GetTopicsResponse{Topics: expectedTopics}, nil)
		c2Client.EXPECT().Close()

		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedTopics, "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute properly paginate calls to api", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := cli.MaxPageSize + int64(1)
		expectedUserOffset := int64(10)

		expectedTopicRequest1 := &pb.GetTopicsRequest{
			Count:  cli.MaxPageSize,
			Offset: expectedUserOffset,
		}

		expectedTopicRequest2 := &pb.GetTopicsRequest{
			Count:  expectedCount - cli.MaxPageSize,
			Offset: expectedUserOffset + cli.MaxPageSize,
		}

		expectedTopics1 := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		expectedTopics2 := []string{
			"topic4",
			"topic5",
			"topic6",
		}

		c2Client.EXPECT().CountTopics(gomock.Any(), &pb.CountTopicsRequest{}).Return(&pb.CountTopicsResponse{Count: expectedCount}, nil)
		gomock.InOrder(
			c2Client.EXPECT().GetTopics(gomock.Any(), expectedTopicRequest1).Return(&pb.GetTopicsResponse{Topics: expectedTopics1}, nil),
			c2Client.EXPECT().GetTopics(gomock.Any(), expectedTopicRequest2).Return(&pb.GetTopicsResponse{Topics: expectedTopics2}, nil),
		)
		c2Client.EXPECT().Close()

		cmd.CobraCmd().Flags().Set("offset", fmt.Sprintf("%d", expectedUserOffset))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(append(expectedTopics1, expectedTopics2...), "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})

	t.Run("Execute with count request only this amount to the api", func(t *testing.T) {
		cmd := newTestListCommand(c2ClientFactory)
		buf := bytes.NewBuffer(nil)
		cmd.CobraCmd().SetOutput(buf)

		expectedCount := int64(10)
		expectedUserCount := int64(9)

		expectedTopicRequest := &pb.GetTopicsRequest{
			Count: expectedUserCount,
		}

		expectedTopics := []string{
			"topic1",
			"topic2",
			"topic3",
		}

		c2Client.EXPECT().CountTopics(gomock.Any(), &pb.CountTopicsRequest{}).Return(&pb.CountTopicsResponse{Count: expectedCount}, nil)
		c2Client.EXPECT().GetTopics(gomock.Any(), expectedTopicRequest).Return(&pb.GetTopicsResponse{Topics: expectedTopics}, nil)
		c2Client.EXPECT().Close()

		cmd.CobraCmd().Flags().Set("count", fmt.Sprintf("%d", expectedUserCount))
		err := cmd.CobraCmd().Execute()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedOutput := []byte(strings.Join(expectedTopics, "\n") + "\n")
		if !bytes.Equal(buf.Bytes(), expectedOutput) {
			t.Errorf("Expected output to be %s, got %s", expectedOutput, buf.Bytes())
		}
	})
}
