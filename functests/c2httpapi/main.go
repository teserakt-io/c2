package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	c2t "gitlab.com/teserakt/c2/pkg/c2test"
	e4test "gitlab.com/teserakt/test-common"
)

func main() {

	var exitCode = 0
	defer func() {
		os.Exit(exitCode)
	}()

	const SERVER = "https://localhost:8888"
	const timeoutSeconds = 30

	var errc = make(chan *e4test.TestResult)
	var stopc = make(chan struct{})
	var waitdrunc = make(chan bool)

	pass := true

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

	c2binary, earlyerr := e4test.FindAndCheckPathFile("bin/c2")
	if earlyerr != nil {
		fmt.Fprintf(os.Stderr, "Error: .\n%s", earlyerr)
		exitCode = 1
		return
	}

	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", e4test.GetRandomDBName())

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)

	go e4test.RunDaemon(errc, stopc, waitdrunc, c2binary, []string{}, env, func() bool {
		return e4test.CheckC2Online(timeoutSeconds*time.Second, 5555, 8888, "127.0.0.1")
	})

	var daemonOK bool
	var err error

	daemonOK = <-waitdrunc

	if daemonOK {
		httpClient := e4test.ConstructHTTPSClient()

		go c2t.TestHTTPApi(errc, httpClient, SERVER)

		pass = true
	} else {
		errc <- &e4test.TestResult{
			Name:     "",
			Critical: true,
			Result:   false,
			Error:    fmt.Errorf("Daemon did not launch after timeout %s", timeoutSeconds),
		}
	}

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
			// Critical errors imply we stop and ignore any further action.
			// Not critical errors imply a warn state
			if result.Critical {
				pass = false
				fmt.Fprintf(os.Stderr, "%s", result.Error)
				break
			} else {
				fmt.Fprintf(os.Stderr, "%s", result.Error)
			}
		}
	}
	close(stopc)
	if !pass {
		fmt.Fprintf(os.Stdout, "One or more tests failed.\n%s\n", err)
		exitCode = 1
	} else {
		fmt.Fprintf(os.Stdout, "TESTS PASSED!\n")
	}
	fmt.Fprintf(os.Stderr, "\n")
	fmt.Fprintf(os.Stderr, "c2 output\n")
	<-waitdrunc
}
