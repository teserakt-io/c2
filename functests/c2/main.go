package main

import (
	"bytes"
	"fmt"
	"os"
	"time"

	"gitlab.com/teserakt/c2/pkg/c2test"
)

// variables set at build time
var gitCommit string
var buildDate string

func runTest(errorChan chan<- error, testFunc func()) {
	mqttEndpoint := os.Getenv("C2TEST_MQTT")
	if len(mqttEndpoint) == 0 {
		mqttEndpoint = "127.0.0.1:1883"
	}

	server := c2test.NewServer(mqttEndpoint)
	if err := server.Start(); err != nil {
		errorChan <- fmt.Errorf("failed to start server: %v", err)
		return
	}
	defer server.Stop()

	fmt.Fprintln(os.Stderr, "C2 is online, launching tests...")

	s := time.Now()
	testFunc()
	fmt.Fprintf(os.Stderr, "Finished test suite (took %s)\n", time.Now().Sub(s))

	errorChan <- nil
}

func main() {
	fmt.Printf("E4: C2 functionnal tests - version %s-%s\n", buildDate, gitCommit)
	fmt.Println("Copyright (c) Teserakt AG, 2018-2019")

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	var grpcTotalCount, grpcFailureCount, httpTotalCount, httpFailureCount int

	grpcResChan := make(chan c2test.TestResult)
	grpcErrChan := make(chan error, 1)

	// Start a server and launch GRPC test suite
	go func() {
		runTest(grpcErrChan, func() {
			grpcClient, close, err := c2test.NewTestingGRPCClient("configs/c2-cert.pem", "127.0.0.1:5555")
			if err != nil {
				grpcErrChan <- fmt.Errorf("failed to create grpc client: %v", err)
				return
			}
			defer close()
			c2test.GRPCApi(grpcResChan, grpcClient)
		})
	}()

	// Process GRPC test results / handle errors
	grpcResults := bytes.NewBuffer(nil)
	done := false
	for !done {
		select {
		case result := <-grpcResChan: // print result
			grpcTotalCount++
			if !result.Result {
				grpcFailureCount++
			}
			result.Print(grpcResults)
		case err := <-grpcErrChan:
			if err != nil {
				exitCode = 1
				fmt.Fprintf(os.Stderr, "GRPC tests error: %v\n", err)
				return
			}
			done = true
		}
	}

	httpResChan := make(chan c2test.TestResult)
	httpErrChan := make(chan error, 1)

	// Start a server and launch HTTP test suite
	go func() {
		runTest(httpErrChan, func() {
			httpClient := c2test.NewTestingHTTPClient()
			c2test.HTTPApi(httpResChan, httpClient, "https://127.0.0.1:8888")
		})
	}()

	// Process HTTP results, handle errors
	httpResults := bytes.NewBuffer(nil)
	done = false
	for !done {
		select {
		case result := <-httpResChan: // print result
			httpTotalCount++
			if !result.Result {
				httpFailureCount++
			}
			result.Print(httpResults)
		case err := <-httpErrChan:
			if err != nil {
				exitCode = 1
				fmt.Fprintf(os.Stderr, "HTTP tests error: %v\n", err)
				return
			}
			done = true
		}
	}

	// Print results / Set exit code on failures
	fmt.Fprintf(os.Stderr, "\nGRPC Test Results:\n%s\n", grpcResults)
	fmt.Fprintf(os.Stderr, "HTTP Test Results:\n%s\n", httpResults)

	if grpcFailureCount > 0 || httpFailureCount > 0 {
		exitCode = 1
	}

	fmt.Fprintf(
		os.Stderr,
		"%d tests total (%d grpc, %d http), %d failures (%d grpc, %d http)\n",
		grpcTotalCount+httpTotalCount,
		grpcTotalCount,
		httpTotalCount,
		grpcFailureCount+httpFailureCount,
		grpcFailureCount,
		httpFailureCount,
	)
}
