package main

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"go/build"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	e4 "teserakt/e4go/pkg/e4common"
	e4test "teserakt/e4go/pkg/e4test"
)

func testHTTPApi(errc chan error, httpClient http.Client) {

	const TESTIDS = 4
	var testids [TESTIDS][]byte
	var testidkeys [TESTIDS][]byte
	var err error

	testids[0], err = testGenerateID()
	if err != nil {
		errc <- fmt.Errorf("testGenerateID failed. %s", err)
	}

	testidkeys[0], err = testGenerateKey()
	if err != nil {
		errc <- fmt.Errorf("testGenerateKeys failed. %s", err)
	}


	req, _ := http.NewRequest("PATCH", fmt.Fprintf(""))
	resp, err := httpClient.Do()
}

func main() {

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	var errc = make(chan error)

	go func() {
		var signalc = make(chan os.Signal, 1)
		signal.Notify(signalc, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-signalc)
	}()

	c2binary, earlyerr := findAndCheckPathFile("teserakt/e4go/bin/c2backend")
	if earlyerr != nil {
		fmt.Errorf("Error: .\n%s", earlyerr)
		exitCode = 1
		return
	}

	const SERVER = "https://localhost:8888"
	// TODO: configure C2. Can we do this on the command line?
	
	go runDaemon(errc, c2binary, ["-c"])
	
	httpClient := e4test.ConstructHTTPSClient()
	
	go testHTTPApi(errc, httpClient)

	var err error
	err = <-errc
	if err != nil {
		fmt.Errorf("Test failed.\n%s", err)
		exitCode = 1
	}
}
