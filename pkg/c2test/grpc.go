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

package c2test

import (
	"context"
	"errors"
	"fmt"

	"github.com/teserakt-io/c2/pkg/pb"
)

// GRPCApi tests the GRPC api of the C2
func GRPCApi(ctx context.Context, resChan chan<- TestResult, grpcClient pb.C2Client) {
	const TESTCLIENTCOUNT = 4
	const TESTTOPICCOUNT = 4
	var testClients [TESTCLIENTCOUNT]TestClient
	var testTopics [TESTTOPICCOUNT]TestTopic
	var err error

	for i := 0; i < TESTCLIENTCOUNT; i++ {
		client, err := NewTestClient()
		if err != nil {
			resChan <- TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("e4test.GenerateID failed. %s", err),
			}
			return
		}
		testClients[i] = *client
	}
	for i := 0; i < TESTTOPICCOUNT; i++ {
		// we don't actually need keys for these tests;
		// so don't generate them for the topics.
		topic, err := NewTestTopic(false)
		if err != nil {
			resChan <- TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("e4test.GenerateTopic failed. %s", err),
			}
			return
		}
		testTopics[i] = *topic
	}

	for i := 0; i < TESTCLIENTCOUNT; i++ {
		_, err := grpcClient.NewClient(ctx, &pb.NewClientRequest{
			Client: &pb.Client{
				Name: testClients[i].Name,
			},
			Key: testClients[i].Key,
		})

		if err != nil {
			resChan <- TestResult{
				Name:     "Create Clients",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}
	resChan <- TestResult{Name: "Create Clients", Result: true, Critical: false, Error: nil}

	for i := 0; i < TESTTOPICCOUNT; i++ {
		_, err := grpcClient.NewTopic(ctx, &pb.NewTopicRequest{
			Topic: testTopics[i].TopicName,
		})

		if err != nil {
			resChan <- TestResult{
				Name:     "Create Topics",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}
	resChan <- TestResult{Name: "Create Topics", Result: true, Critical: false, Error: nil}

	// *** Add the topic to the client.
	_, err = grpcClient.NewTopicClient(ctx, &pb.NewTopicClientRequest{
		Client: &pb.Client{Name: testClients[0].Name},
		Topic:  testTopics[0].TopicName,
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "Add Topic to Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	resChan <- TestResult{Name: "Add Topic to Client", Result: true, Critical: false, Error: nil}

	// *** Check the M2M link returns the topic we added
	resp, err := grpcClient.GetTopicsForClient(ctx, &pb.GetTopicsForClientRequest{
		Client: &pb.Client{Name: testClients[0].Name},
		Count:  10,
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	clientTopics := resp.Topics
	if len(clientTopics) != 1 || clientTopics[0] != testTopics[0].TopicName {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("test M2M Find Added Topic: Incorrect topic returned, returned body is %s", clientTopics),
		}
		return
	}

	resChan <- TestResult{Name: "M2M Find Added Topic", Result: true, Critical: false, Error: nil}

	// *** Remove the topic from the client (but not the C2)
	_, err = grpcClient.RemoveTopicClient(ctx, &pb.RemoveTopicClientRequest{
		Client: &pb.Client{Name: testClients[0].Name},
		Topic:  testTopics[0].TopicName,
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "Remove Topic from Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	resChan <- TestResult{Name: "Remove Topic from Client", Result: true, Critical: false, Error: nil}

	// *** Check Topic appears to have been removed from the client
	resp, err = grpcClient.GetTopicsForClient(ctx, &pb.GetTopicsForClientRequest{
		Client: &pb.Client{Name: testClients[0].Name},
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	clientTopics = resp.Topics
	if len(clientTopics) != 0 {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("test M2M Doesn't Show Removed Topic: Topics found, returned body is %s", clientTopics),
		}
		return
	}
	resChan <- TestResult{Name: "Test M2M Doesn't Show Removed Topic", Result: true, Critical: false, Error: nil}

	// *** Delete topic
	_, err = grpcClient.RemoveTopic(ctx, &pb.RemoveTopicRequest{
		Topic: testTopics[0].TopicName,
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "Remove topic from C2",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	resChan <- TestResult{Name: "Remove topic from C2", Result: true, Critical: false, Error: nil}

	// *** Check double remove of topic fails
	_, err = grpcClient.RemoveTopic(ctx, &pb.RemoveTopicRequest{
		Topic: testTopics[0].TopicName,
	})
	if err == nil {
		resChan <- TestResult{
			Name:     "Check double remove fails",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("double remove should report an error via the API and did not"),
		}
		return
	}

	resChan <- TestResult{Name: "Check double remove fails", Result: true, Critical: false, Error: nil}

	// *** Get topics list
	getTopicsResp, err := grpcClient.GetTopics(ctx, &pb.GetTopicsRequest{
		Count: 10,
	})
	if err != nil {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	clientTopics = getTopicsResp.Topics
	if len(clientTopics) != TESTTOPICCOUNT-1 {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("test Fetch Topics: Incorrect number of returned topics, returned body is %s", clientTopics),
		}
		return
	}
	for i := 1; i < TESTTOPICCOUNT; i++ {
		found := false
		testtopic := testTopics[i]
		for j := 0; j < len(clientTopics); j++ {
			if clientTopics[j] == testtopic.TopicName {
				found = true
				break
			}
		}
		if !found {
			resChan <- TestResult{
				Name:     "Test Fetch Topics",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("test Fetch Topics: Created topic %s not found, topics are %s", testtopic, clientTopics),
			}
			return
		}
	}
	resChan <- TestResult{Name: "Test Fetch Topics", Result: true, Critical: false, Error: nil}

	// *** Get client list
	getClientsResp, err := grpcClient.GetClients(ctx, &pb.GetClientsRequest{Count: 10})
	if err != nil {
		if err == nil {
			err = errors.New("type mismatch")
		}
		resChan <- TestResult{
			Name:     "Test Fetch Clients",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}

	clients := getClientsResp.Clients
	if len(clients) != TESTCLIENTCOUNT {
		resChan <- TestResult{
			Name:     "Test Fetch Clients",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("test Fetch Clients: Incorrect number of returned clients, returned body is %s", clients),
		}
		return
	}
	for i := 0; i < TESTCLIENTCOUNT; i++ {
		found := false
		testclient := testClients[i]
		for j := 0; j < len(clients); j++ {
			if clients[j].Name == testclient.Name {
				found = true
				break
			}
		}
		if !found {
			resChan <- TestResult{
				Name:     "Test Fetch Client",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("test Fetch Client: Client s%s not found, clients are %s", testclient, clients),
			}
			return
		}
	}

	resChan <- TestResult{Name: "Test Fetch Client", Result: true, Critical: false, Error: nil}
}
