package main

import (
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	e4test "teserakt/e4go/pkg/e4test"
)

func testHTTPApi(errc chan error, httpClient http.Client, host string) {

	const TESTIDS = 4
	var testids [TESTIDS][]byte
	var testidkeys [TESTIDS][]byte
	var err error

	testids[0], err = e4test.GenerateID()
	if err != nil {
		errc <- fmt.Errorf("e4test.GenerateID failed. %s", err)
		return
	}

	testidkeys[0], err = e4test.GenerateKey()
	if err != nil {
		errc <- fmt.Errorf("e4test.GenerateKeys failed. %s", err)
		return
	}

	url := fmt.Sprintf("%s/e4/client/%s", host, hex.EncodeToString(testids[0]))
	req, err := http.NewRequest("PATCH", url, strings.NewReader(""))
	if err != nil {
		errc <- err
		return
	}
	resp, err := httpClient.Do(req)

	if err != nil {
		errc <- err
		return
	}
	if resp.StatusCode != 200 {
		errc <- fmt.Errorf("Request %s failed", url)
		return
	}

}

func main() {

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	var errc = make(chan error)
	var stopc = make(chan struct{})
	var waitc = make(chan struct{})

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

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, (fmt.Sprintf("E4C2_DB_FILE=%s", e4test.GetRandomDBName())))

	go e4test.RunDaemon(errc, stopc, waitc, c2binary, []string{}, env)

	<-waitc
	fmt.Println("Running tests")

	httpClient := e4test.ConstructHTTPSClient()

	go testHTTPApi(errc, httpClient, SERVER)

	var err error
	err = <-errc
	close(stopc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tests failed.\n%s\n", err)
		exitCode = 1
	}
	fmt.Fprintf(os.Stderr, "C2Backend Output\n")
	<-waitc
}
