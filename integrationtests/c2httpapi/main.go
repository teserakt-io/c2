package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	e4test "teserakt/e4go/pkg/e4test"
)

func testHTTPReq(testname string, httpClient http.Client,
	verb string, url string, body string, responseCode int) (*http.Response, error) {

	fmt.Fprintf(os.Stderr, "%s %s\n", verb, url)
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

	fmt.Println(resp.StatusCode)

	return resp, nil
}

func testHTTPApi(errc chan error, httpClient http.Client, host string) {

	const TESTIDS = 4
	const TESTTOPICS = 4
	var testids [TESTIDS]e4test.TestIDKey
	var testtopics [TESTTOPICS]e4test.TestTopicKey
	var err error

	for i := 0; i < TESTIDS; i++ {
		err = testids[i].New()
		if err != nil {
			errc <- fmt.Errorf("e4test.GenerateID failed. %s", err)
			return
		}
	}
	for i := 0; i < TESTTOPICS; i++ {
		err = testtopics[i].New()
		if err != nil {
			errc <- fmt.Errorf("e4test.GenerateID failed. %s", err)
			return
		}
	}

	var resp *http.Response
	var url string

	for i := 0; i < TESTIDS; i++ {
		// Create a new client on the C2
		url = fmt.Sprintf("%s/e4/client/%s/key/%s", host,
			testids[i].GetHexID(), testids[i].GetHexKey())
		if _, err = testHTTPReq("Create Client", httpClient, "POST", url, "", 200); err != nil {
			errc <- err
			return
		}
	}
	/*
		url := fmt.Sprintf("%s/e4/client/%s", host, testids[0].GetHexID())
		if _, err = testHTTPReq("Reset Topic/New Key", httpClient, "PATCH", url, "", 200); err != nil {
			errc <- err
			return
		}
	*/

	for i := 0; i < TESTTOPICS; i++ {
		// Create a corresponding topics
		url = fmt.Sprintf("%s/e4/topic/%s", host,
			testtopics[i].TopicName)
		if _, err = testHTTPReq("Create Topic", httpClient, "POST", url, "", 200); err != nil {
			errc <- err
			return
		}
	}

	// Add the topic to the client.
	url = fmt.Sprintf("%s/e4/client/%s/topic/%s", host,
		testids[0].GetHexID(), testtopics[0].TopicName)
	if _, err = testHTTPReq("Add Topic to Client", httpClient, "PUT", url, "", 200); err != nil {
		errc <- err
		return
	}

	// Check the M2M link returns the topic we added
	url = fmt.Sprintf("%s/e4/client/%s/topics/0/10", host,
		testids[0].GetHexID())
	if resp, err = testHTTPReq("M2M Find Added Topic", httpClient, "GET", url, "", 200); err != nil {
		errc <- err
		return
	}
	var decodedtopics1 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics1)
	if err != nil {
		errc <- fmt.Errorf("Test M2M Find Added Topic: %s", err)
		return
	}
	if len(decodedtopics1) != 1 || decodedtopics1[0] != testtopics[0].TopicName {
		errc <- fmt.Errorf("Test M2M Find Added Topic: Incorrect topic returned, returned body is %s", decodedtopics1)
		return
	}

	// Remove the topic from the client (but not the C2)
	url = fmt.Sprintf("%s/e4/client/%s/topic/%s", host,
		testids[0].GetHexID(), testtopics[0].TopicName)
	if _, err = testHTTPReq("Remove Topic from Client", httpClient, "DELETE", url, "", 200); err != nil {
		errc <- err
		return
	}

	// Check Topic appears to have been removed from the client
	url = fmt.Sprintf("%s/e4/client/%s/topics/0/10", host,
		testids[0].GetHexID())
	if resp, err = testHTTPReq("Test M2M Doesn't Show Removed Topic", httpClient, "GET", url, "", 200); err != nil {
		errc <- err
		return
	}
	var decodedtopics2 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics2)
	if err != nil {
		errc <- fmt.Errorf("Test M2M Doesn't Show Removed Topic: %s", err)
		return
	}
	if len(decodedtopics2) != 0 {
		errc <- fmt.Errorf("Test M2M Doesn't Show Removed Topic: Topics found, returned body is %s", decodedtopics2)
		return
	}

	// Delete topic
	url = fmt.Sprintf("%s/e4/topic/%s", host,
		testtopics[0].TopicName)
	if _, err = testHTTPReq("Remove topic from C2", httpClient, "DELETE", url, "", 200); err != nil {
		errc <- err
		return
	}

	// Check double remove of topic fails
	url = fmt.Sprintf("%s/e4/topic/%s", host,
		testtopics[0].TopicName)
	if _, err = testHTTPReq("Check double remove fails", httpClient, "DELETE", url, "", 404); err != nil {
		errc <- err
		return
	}

	// Get topics list
	url = fmt.Sprintf("%s/e4/topic", host)
	if resp, err = testHTTPReq("Test Fetch Topics", httpClient, "GET", url, "", 200); err != nil {
		errc <- err
		return
	}
	var decodedtopics3 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedtopics3)
	if err != nil {
		errc <- fmt.Errorf("Test Fetch Topics: %s", err)
		return
	}
	if len(decodedtopics3) != TESTTOPICS-1 {
		errc <- fmt.Errorf("Test Fetch Topics: Incorrect number of returned topics, returned body is %s", decodedtopics3)
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
			errc <- fmt.Errorf("Test Fetch Topics: Created topic %s not found, topics are %s", testtopic, decodedtopics3)
			return
		}
	}

	// Get client list
	url = fmt.Sprintf("%s/e4/client", host)
	if resp, err = testHTTPReq("Test Fetch Client", httpClient, "GET", url, "", 200); err != nil {
		errc <- err
		return
	}
	var decodedIDs1 []string
	err = json.NewDecoder(resp.Body).Decode(&decodedIDs1)
	if err != nil {
		errc <- fmt.Errorf("Test Fetch Client: %s", err)
		return
	}
	if len(decodedIDs1) != TESTIDS {
		errc <- fmt.Errorf("Test Fetch Client: Incorrect number of clients, returned body is %s", decodedIDs1)
		return
	}
	for i := 0; i < TESTIDS; i++ {
		found := false
		testtopic := testids[i]
		for j := 0; j < len(decodedIDs1); j++ {
			if decodedIDs1[j] == testtopic.GetHexID() {
				found = true
				break
			}
		}
		if !found {
			errc <- fmt.Errorf("Test Fetch Topics: Created client %s not found, clients are %s", testtopic, decodedtopics3)
			return
		}
	}

	errc <- nil
}

func main() {

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	var errc = make(chan error)
	var stopc = make(chan struct{})
	var waitdrunc = make(chan struct{})

	go func() {
		var signalc = make(chan os.Signal, 1)
		signal.Notify(signalc, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-signalc)
	}()

	c2binary, earlyerr := e4test.FindAndCheckPathFile("src/teserakt/e4go/bin/c2backend")
	if earlyerr != nil {
		fmt.Fprintf(os.Stderr, "Error: .\n%s", earlyerr)
		exitCode = 1
		return
	}

	const SERVER = "https://localhost:8888"
	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", e4test.GetRandomDBName())

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)

	go e4test.RunDaemon(errc, stopc, waitdrunc, c2binary, []string{}, env)

	<-waitdrunc
	fmt.Println("Running tests")

	httpClient := e4test.ConstructHTTPSClient()

	go testHTTPApi(errc, httpClient, SERVER)

	var err error
	err = <-errc
	close(stopc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tests failed.\n%s\n", err)
		exitCode = 1
	} else {
		fmt.Fprintf(os.Stderr, "TESTS PASSED!\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "C2Backend Output\n")
	<-waitdrunc
}
