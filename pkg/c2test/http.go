package c2test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func testHTTPReq(testname string, httpClient *http.Client,
	verb string, url string, body string, responseCode int) (*http.Response, error) {

	//fmt.Fprintf(os.Stderr, "%s %s\n", verb, url)
	req, err := http.NewRequest(verb, url, strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("Test %s: %s", testname, err)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Test %s: %s", testname, err)
	}
	if resp.StatusCode != responseCode {
		return nil, fmt.Errorf("Test %s: Request %s failed response code test, expected %d, received %d", testname, url, responseCode, resp.StatusCode)
	}

	return resp, nil
}

// HTTPApi tests the http api of the C2
func HTTPApi(resChan chan<- TestResult, httpClient *http.Client, host string) {
	const TESTCLIENTS = 4
	const TESTTOPICS = 4
	var testClients [TESTCLIENTS]TestClient
	var testtopics [TESTTOPICS]TestTopic
	var err error

	for i := 0; i < TESTCLIENTS; i++ {
		client, err := NewTestClient()
		if err != nil {
			resChan <- TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("GenerateID failed. %s", err),
			}
			return
		}
		testClients[i] = *client
	}
	for i := 0; i < TESTTOPICS; i++ {
		topic, err := NewTestTopic(true)
		if err != nil {
			resChan <- TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("GenerateTopic failed. %s", err),
			}
			return
		}
		testtopics[i] = *topic
	}

	var resp *http.Response
	var url string

	for i := 0; i < TESTCLIENTS; i++ {
		// Create a new client on the C2
		url = fmt.Sprintf("%s/e4/client/name/%s/key/%s", host,
			testClients[i].Name, testClients[i].GetHexKey())
		if _, err = testHTTPReq("Create Client", httpClient, "POST", url, "", 200); err != nil {
			resChan <- TestResult{
				Name:     "Create Client",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}
	resChan <- TestResult{Name: "Create Client", Result: true, Critical: false, Error: nil}

	for i := 0; i < TESTTOPICS; i++ {
		// Create a corresponding topics
		url = fmt.Sprintf("%s/e4/topic/%s", host,
			testtopics[i].TopicName)
		if _, err = testHTTPReq("Create Topic", httpClient, "POST", url, "", 200); err != nil {
			resChan <- TestResult{
				Name:     "Create Topic",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}
	resChan <- TestResult{Name: "Create Topic", Result: true, Critical: false, Error: nil}

	// Add the topic to the client.
	url = fmt.Sprintf("%s/e4/client/name/%s/topic/%s", host,
		testClients[0].Name, testtopics[0].TopicName)
	if _, err = testHTTPReq("Add Topic to Client", httpClient, "PUT", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Add Topic to Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	resChan <- TestResult{Name: "Add Topic to Client", Result: true, Critical: false, Error: nil}

	// Check the M2M link returns the topic we added
	url = fmt.Sprintf("%s/e4/client/name/%s/topics/0/10", host,
		testClients[0].Name)
	if resp, err = testHTTPReq("M2M Find Added Topic", httpClient, "GET", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	var decodedtopics1 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics1)
	if err != nil {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Find Added Topic: %s", err),
		}
		return
	}
	if len(decodedtopics1) != 1 || decodedtopics1[0] != testtopics[0].TopicName {
		resChan <- TestResult{
			Name:     "M2M Find Added Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Find Added Topic: Incorrect topic returned, returned body is %s", decodedtopics1),
		}
		return
	}
	resChan <- TestResult{Name: "M2M Find Added Topic", Result: true, Critical: false, Error: nil}

	// Remove the topic from the client (but not the C2)
	url = fmt.Sprintf("%s/e4/client/name/%s/topic/%s", host,
		testClients[0].Name, testtopics[0].TopicName)
	if _, err = testHTTPReq("Remove Topic from Client", httpClient, "DELETE", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Remove Topic from Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	resChan <- TestResult{Name: "Remove Topic from Client", Result: true, Critical: false, Error: nil}

	// Check Topic appears to have been removed from the client
	url = fmt.Sprintf("%s/e4/client/name/%s/topics/0/10", host,
		testClients[0].Name)
	if resp, err = testHTTPReq("Test M2M Doesn't Show Removed Topic", httpClient, "GET", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	var decodedtopics2 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics2)
	if err != nil {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Doesn't Show Removed Topic: %s", err),
		}
		return
	}
	if len(decodedtopics2) != 0 {
		resChan <- TestResult{
			Name:     "Test M2M Doesn't Show Removed Topic",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test M2M Doesn't Show Removed Topic: Topics found, returned body is %s", decodedtopics2),
		}
		return
	}
	resChan <- TestResult{Name: "Test M2M Doesn't Show Removed Topic", Result: true, Critical: false, Error: nil}

	// Delete topic
	url = fmt.Sprintf("%s/e4/topic/%s", host,
		testtopics[0].TopicName)
	if _, err = testHTTPReq("Remove topic from C2", httpClient, "DELETE", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Remove topic from C2",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	resChan <- TestResult{Name: "Remove topic from C2", Result: true, Critical: false, Error: nil}

	// Check double remove of topic fails
	url = fmt.Sprintf("%s/e4/topic/%s", host,
		testtopics[0].TopicName)
	if _, err = testHTTPReq("Check double remove fails", httpClient, "DELETE", url, "", 404); err != nil {
		resChan <- TestResult{
			Name:     "Check double remove fails",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	resChan <- TestResult{Name: "Check double remove fails", Result: true, Critical: false, Error: nil}

	// Get topics list
	url = fmt.Sprintf("%s/e4/topics/all", host)
	if resp, err = testHTTPReq("Test Fetch Topics", httpClient, "GET", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	var decodedtopics3 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics3)
	if err != nil {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Topics: %s", err),
		}
		return
	}
	if len(decodedtopics3) != TESTTOPICS-1 {
		resChan <- TestResult{
			Name:     "Test Fetch Topics",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Topics: Incorrect number of returned topics. Expected %d, got %d.\n returned body is %#v", TESTTOPICS-1, len(decodedtopics3), decodedtopics3),
		}
		return
	}
	for i := 1; i < TESTTOPICS; i++ {
		found := false
		testtopic := testtopics[i]
		for j := 0; j < len(decodedtopics3); j++ {
			if decodedtopics3[j] == testtopic.TopicName {
				found = true
				break
			}
		}
		if !found {
			resChan <- TestResult{
				Name:     "Test Fetch Topics",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("Test Fetch Topics: Created topic %s not found, topics are %s", testtopic.TopicName, decodedtopics3),
			}
			return
		}
	}
	resChan <- TestResult{Name: "Test Fetch Topics", Result: true, Critical: false, Error: nil}

	// Get client list
	url = fmt.Sprintf("%s/e4/clients/all", host)
	if resp, err = testHTTPReq("Test Fetch Client", httpClient, "GET", url, "", 200); err != nil {
		resChan <- TestResult{
			Name:     "Test Fetch Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	var decodedIDs1 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedIDs1)
	if err != nil {
		resChan <- TestResult{
			Name:     "Test Fetch Client",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Client: %s", err),
		}
		return
	}
	if len(decodedIDs1) != TESTCLIENTS {
		resChan <- TestResult{
			Name:     "Test Fetch Client",
			Result:   false,
			Critical: true,
			Error:    fmt.Errorf("Test Fetch Client: Incorrect number of clients, returned body is %s", decodedIDs1),
		}
		return
	}
	for i := 0; i < TESTCLIENTS; i++ {
		found := false
		testid := testClients[i]
		for j := 0; j < len(decodedIDs1); j++ {
			if decodedIDs1[j] == testid.Name {
				found = true
				break
			}
		}
		if !found {
			resChan <- TestResult{
				Name:     "Test Fetch Client",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("Test Fetch Client: Created client %s not found, clients are %s", testid, decodedtopics3),
			}
			return
		}
	}
	resChan <- TestResult{Name: "Test Fetch Client", Result: true, Critical: false, Error: nil}
}
