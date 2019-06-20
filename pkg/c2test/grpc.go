package c2test

import (
	"context"
	"errors"
	"fmt"

	e4 "gitlab.com/teserakt/e4common"
)

// GRPCApi tests the GRPC api of the C2
func GRPCApi(resChan chan<- TestResult, grpcClient e4.C2Client) {
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
		result, err := grpcC2SendCommand(grpcClient, e4.C2Request_NEW_CLIENT,
			testClients[i].ID, testClients[i].Name, testClients[i].Key, "", "", 0, 0)
		bresult, ok := result.(bool)
		// must check bresult last, it won't be boolean unless the type assertion
		// succeeds.
		if err != nil || !ok || !bresult {
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
		result, err := grpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC,
			nil, "", nil, testTopics[i].TopicName, "", 0, 0)
		bresult, ok := result.(bool)
		// must check bresult last, it won't be boolean unless the type assertion
		// succeeds.
		if err != nil || !ok || !bresult {
			if err == nil {
				err = errors.New("Type mismatch")
			}
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
	result, err := grpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC_CLIENT,
		nil, testClients[0].Name, nil, testTopics[0].TopicName, "", 0, 0)
	bresult, ok := result.(bool)
	if err != nil || !ok || !bresult {
		if err == nil {
			err = errors.New("Type mismatch")
		}
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
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_GET_CLIENT_TOPICS,
		nil, testClients[0].Name, nil, "", "", 0, 10)
	clientTopics, ok := result.([]string)
	if err != nil || !ok {
		if err == nil {
			err = errors.New("Type mismatch")
		}
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	if len(clientTopics) != 1 || clientTopics[0] != testTopics[0].TopicName {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Find Added Topic: Incorrect topic returned, returned body is %s", clientTopics),
		}
		return
	}

	resChan <- TestResult{Name: "M2M Find Added Topic", Result: true, Critical: false, Error: nil}

	// *** Remove the topic from the client (but not the C2)
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_REMOVE_TOPIC_CLIENT,
		nil, testClients[0].Name, nil, testTopics[0].TopicName, "", 0, 10)
	bresult, ok = result.(bool)
	if err != nil || !ok || !bresult {
		if err == nil {
			err = errors.New("Type mismatch")
		}
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
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_GET_CLIENT_TOPICS,
		nil, testClients[0].Name, nil, "", "", 0, 10)
	clientTopics, ok = result.([]string)
	if err != nil || !ok {
		if err == nil {
			err = errors.New("Type mismatch")
		}
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	if len(clientTopics) != 0 {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Doesn't Show Removed Topic: Topics found, returned body is %s", clientTopics),
		}
		return
	}
	resChan <- TestResult{Name: "Test M2M Doesn't Show Removed Topic", Result: true, Critical: false, Error: nil}

	// *** Delete topic
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_REMOVE_TOPIC,
		nil, "", nil, testTopics[0].TopicName, "", 0, 10)
	bresult, ok = result.(bool)
	if err != nil || !ok || !bresult {
		if err == nil {
			err = errors.New("Type mismatch")
		}
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
	_, err = grpcC2SendCommand(grpcClient, e4.C2Request_REMOVE_TOPIC,
		nil, "", nil, testTopics[0].TopicName, "", 0, 10)
	//bresult, ok = result.(bool)
	if err == nil {
		if err == nil {
			err = errors.New("Type mismatch")
		}
		resChan <- TestResult{
			Name:     "Check double remove fails",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Double remove should report an error via the API and did not"),
		}
		return
	}

	resChan <- TestResult{Name: "Check double remove fails", Result: true, Critical: false, Error: nil}

	// *** Get topics list
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_GET_TOPICS,
		nil, testClients[0].Name, nil, "", "", 0, 10)
	clientTopics, ok = result.([]string)
	if err != nil || !ok {
		if err == nil {
			err = errors.New("Type mismatch")
		}
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	if len(clientTopics) == 0 || len(clientTopics) != TESTTOPICCOUNT-1 {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Topics: Incorrect number of returned topics, returned body is %s", clientTopics),
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
				Error:    fmt.Errorf("Test Fetch Topics: Created topic %s not found, topics are %s", testtopic, clientTopics),
			}
			return
		}
	}
	resChan <- TestResult{Name: "Test Fetch Topics", Result: true, Critical: false, Error: nil}

	// *** Get client list
	result, err = grpcC2SendCommand(grpcClient, e4.C2Request_GET_CLIENTS,
		nil, "", nil, "", "", 0, 10)
	clientClients, ok := result.([]string)
	if err != nil || !ok {
		if err == nil {
			err = errors.New("Type mismatch")
		}
		resChan <- TestResult{
			Name:     "Test Fetch Clients",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	if len(clientClients) == 0 || len(clientClients) != TESTCLIENTCOUNT {
		resChan <- TestResult{
			Name:     "Test Fetch Clients",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Clients: Incorrect number of returned clients, returned body is %s", clientClients),
		}
		return
	}
	for i := 0; i < TESTCLIENTCOUNT; i++ {
		found := false
		testclient := testClients[i]
		for j := 0; j < len(clientClients); j++ {
			if clientClients[j] == testclient.Name {
				found = true
				break
			}
		}
		if !found {
			resChan <- TestResult{
				Name:     "Test Fetch Client",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("Test Fetch Client: Client s%s not found, clients are %s", testclient, clientClients),
			}
			return
		}
	}

	resChan <- TestResult{Name: "Test Fetch Client", Result: true, Critical: false, Error: nil}
}

func grpcC2SendCommand(client e4.C2Client, commandcode e4.C2Request_Command, id []byte, name string, key []byte, topic, msg string, offset uint64, count uint64) (interface{}, error) {

	req := &e4.C2Request{
		Command: commandcode,
		Id:      id,
		Name:    name,
		Key:     key,
		Topic:   topic,
		Msg:     msg,
		Offset:  offset,
		Count:   count,
	}

	resp, err := client.C2Command(context.Background(), req)
	if err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, errors.New(resp.Err)
	}

	switch commandcode {
	case e4.C2Request_NEW_CLIENT:
		return true, nil
	case e4.C2Request_REMOVE_CLIENT:
		return true, nil
	case e4.C2Request_NEW_TOPIC:
		return true, nil
	case e4.C2Request_REMOVE_TOPIC:
		return true, nil
	case e4.C2Request_NEW_TOPIC_CLIENT:
		return true, nil
	case e4.C2Request_REMOVE_TOPIC_CLIENT:
		return true, nil
	case e4.C2Request_GET_CLIENTS:
		return resp.Names, nil
	case e4.C2Request_GET_TOPICS:
		return resp.Topics, nil
	case e4.C2Request_GET_CLIENT_TOPICS:
		return resp.Topics, nil
	case e4.C2Request_GET_CLIENT_TOPIC_COUNT:
		return resp.Count, nil
	case e4.C2Request_GET_TOPIC_CLIENTS:
		return resp.Topics, nil
	case e4.C2Request_GET_TOPIC_CLIENT_COUNT:
		return resp.Count, nil

	default:
		return nil, errors.New("No handler for that request type")
	}
}
