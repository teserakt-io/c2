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

func runTest(errorChan chan<- error, logChan chan<- []byte, testFunc func()) {
	server := c2test.NewServer()
	if err := server.Start(); err != nil {
		errorChan <- fmt.Errorf("failed to start server: %v", err)
		return
	}

	fmt.Fprintln(os.Stderr, "C2 is online, launching tests...")

	s := time.Now()
	testFunc()
	fmt.Fprintf(os.Stderr, "Finished grpc test suite (took %s)\n", time.Now().Sub(s))

	if err := server.Stop(); err != nil {
		errorChan <- fmt.Errorf("failed to stop server: %v", err)
	}

	serverOut, err := server.Output()
	if err != nil {
		errorChan <- fmt.Errorf("failed to retrieve server output: %v", err)
		return
	}

	logChan <- serverOut
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
	serverLogs := bytes.NewBuffer(nil)

	grpcServerOutputChan := make(chan []byte)
	grpcResChan := make(chan c2test.TestResult)
	grpcErrChan := make(chan error)

	// Start a server and launch GRPC test suite
	go func() {
		runTest(grpcErrChan, grpcServerOutputChan, func() {
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
		case out := <-grpcServerOutputChan:
			fmt.Fprintf(serverLogs, "\n\nGRPC Test Server Output:\n\n%s\n", string(out))
		case err := <-grpcErrChan:
			if err != nil {
				exitCode = 1
				fmt.Fprintf(os.Stderr, "GRPC tests error: %v", err)
				return
			}
			done = true
		}
	}

	httpServerOutputChan := make(chan []byte)
	httpResChan := make(chan c2test.TestResult)
	httpErrChan := make(chan error)

	// Start a server and launch HTTP test suite
	go func() {
		runTest(httpErrChan, httpServerOutputChan, func() {
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
		case out := <-httpServerOutputChan:
			fmt.Fprintf(serverLogs, "\n\nHTTP Test Server Output:\n\n%s\n", string(out))
		case err := <-httpErrChan:
			if err != nil {
				exitCode = 1
				fmt.Fprintf(os.Stderr, "HTTP tests error: %v", err)
				return
			}
			done = true
		}
	}

	// Print results / Set exit code on failures
	fmt.Fprintf(os.Stderr, "\n%s", string(serverLogs.Bytes()))
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
