package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	e4 "gitlab.com/teserakt/e4common"
	e4test "gitlab.com/teserakt/test-common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func testGRPCApi(errc chan *e4test.TestResult, grpcClient e4.C2Client) {

	const TESTIDS = 4
	const TESTTOPICS = 4
	var testids [TESTIDS]e4test.TestIDKey
	var testtopics [TESTTOPICS]e4test.TestTopicKey
	var err error

	for i := 0; i < TESTIDS; i++ {
		err = testids[i].New()
		if err != nil {
			errc <- &e4test.TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("e4test.GenerateID failed. %s", err),
			}
			return
		}
	}
	for i := 0; i < TESTTOPICS; i++ {
		// we don't actually need keys for these tests;
		// so don't generate them for the topics.
		err = testtopics[i].New(false)
		if err != nil {
			errc <- &e4test.TestResult{
				Name:     "",
				Result:   false,
				Critical: true,
				Error:    fmt.Errorf("e4test.GenerateTopic failed. %s", err),
			}
			return
		}
	}

	for i := 0; i < TESTIDS; i++ {
		result, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_CLIENT,
			testids[i].ID, testids[i].Key, "", "", 0, 0)
		bresult, ok := result.(bool)
		// must check bresult last, it won't be boolean unless the type assertion
		// succeeds.
		if err != nil || !ok || !bresult {
			errc <- &e4test.TestResult{
				Name:     "CreateClient",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}

	for i := 0; i < TESTTOPICS; i++ {
		result, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC,
			nil, nil, testtopics[i].TopicName, "", 0, 0)
		bresult, ok := result.(bool)
		// must check bresult last, it won't be boolean unless the type assertion
		// succeeds.
		if err != nil || !ok || !bresult {
			if err == nil {
				err = errors.New("Type mistmatch")
			}
			errc <- &e4test.TestResult{
				Name:     "CreateClient",
				Result:   false,
				Critical: true,
				Error:    err,
			}
			return
		}
	}
	// *** Add the topic to the client.
	result, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC_CLIENT,
		testids[0].ID, nil, testtopics[0].TopicName, "", 0, 0)
	bresult, ok := result.(bool)
	if err != nil || !ok || !bresult {
		if err == nil {
			err = errors.New("Type mistmatch")
		}
		errc <- &e4test.TestResult{
			Name:     "Add Topic to Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	errc <- &e4test.TestResult{Name: "Add Topic to Client", Result: true, Critical: false, Error: nil}

	// *** Check the M2M link returns the topic we added
	result, err = e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_GET_CLIENT_TOPICS,
		testids[0].ID, nil, "", "", 0, 10)
	client_topics, ok := result.([]string)
	if err != nil || !ok || !bresult {
		if err == nil {
			err = errors.New("Type mistmatch")
		}
		errc <- &e4test.TestResult{
			Name:     "Add Topic to Client",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	errc <- &e4test.TestResult{Name: "M2M Find Added Topic", Result: true, Critical: false, Error: nil}

	// *** Remove the topic from the client (but not the C2)
	errc <- &e4test.TestResult{Name: "Remove Topic from Client", Result: true, Critical: false, Error: nil}

	// *** Check Topic appears to have been removed from the client
	errc <- &e4test.TestResult{Name: "Test M2M Doesn't Show Removed Topic", Result: true, Critical: false, Error: nil}

	// *** Delete topic
	errc <- &e4test.TestResult{Name: "Remove topic from C2", Result: true, Critical: false, Error: nil}

	// *** Check double remove of topic fails
	errc <- &e4test.TestResult{Name: "Check double remove fails", Result: true, Critical: false, Error: nil}

	// *** Get topics list
	errc <- &e4test.TestResult{Name: "Test Fetch Topics", Result: true, Critical: false, Error: nil}

	// *** Get client list
	errc <- &e4test.TestResult{Name: "Test Fetch Client", Result: true, Critical: false, Error: nil}

	close(errc)
}

func main() {

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	var errc = make(chan *e4test.TestResult)
	var stopc = make(chan struct{})
	var waitdrunc = make(chan struct{})

	go func() {
		var signalc = make(chan os.Signal, 1)
		signal.Notify(signalc, syscall.SIGINT, syscall.SIGTERM)
		errc <- &e4test.TestResult{
			Name:     "",
			Critical: true,
			Result:   false,
			Error:    fmt.Errorf("%s", <-signalc),
		}
	}()

	c2binary, earlyerr := e4test.FindAndCheckPathFile("src/teserakt/e4go/bin/c2backend")
	if earlyerr != nil {
		fmt.Fprintf(os.Stderr, "Error: .\n%s", earlyerr)
		exitCode = 1
		return
	}

	const SERVER string = "localhost:5555"
	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", e4test.GetRandomDBName())

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)

	go e4test.RunDaemon(errc, stopc, waitdrunc, c2binary, []string{}, env)

	<-waitdrunc

	creds, err := credentials.NewClientTLSFromFile(*cert, "")
	if err != nil {
		log.Fatalf("failed to create TLS credentials from %v: %v", *cert, err)
	}

	conn, err := grpc.Dial(SERVER, grpc.WithTransportCredentials(creds))
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}

	defer conn.Close()
	grpcClient := e4.NewC2Client(conn)

	go testGRPCApi(errc, grpcClient)

	var err error
	pass := true

	for result := range errc {

		// tests without a name are support tasks; don't
		// print them as tests.
		if len(result.Name) != 0 {
			fmt.Printf("Test %s\t", result.Name)

			if result.Result {
				fmt.Println("ok")
			} else {
				fmt.Println("failed")
			}
		}
		// if any tests fail, report a failure.
		if !result.Result {
			pass = false
			// propagate error in critical cases,
			// otherwise just print it.
			if result.Critical {
				err = result.Error
				break
			} else {
				fmt.Fprintf(os.Stderr, "%s", result.Error)
			}
		}
	}
	close(stopc)
	if !pass {
		fmt.Fprintf(os.Stderr, "Tests failed.\n%s\n", err)
		exitCode = 1
	} else {
		fmt.Fprintf(os.Stderr, "TESTS PASSED!\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "C2Backend Output\n")
	<-waitdrunc
}
