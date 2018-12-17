package main

import (
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
		err = testtopics[i].New()
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
