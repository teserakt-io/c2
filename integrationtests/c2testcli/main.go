package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	e4test "teserakt/e4go/pkg/e4test"
)

func testC2CLI(errc chan *e4test.TestResult,
	c2clipath string) {

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
				Error:    fmt.Errorf("e4test.GenerateID failed. %s", err),
			}
			return
		}
	}

	stdout, stderr, err := e4test.RunCommand(c2clipath,
		[]string{"-c", "nc", "-i", "\"testid\"", "-p", "\"testpwd\""}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "nt", "-t", "\"testtopic\""}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "ntc", "-t", "\"testtopic\"", "-i", "testid"}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "rsc", "-i", "testid"}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "nck", "-i", "testid"}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "ntc", "-t", "testtopic", "-i", "testid"}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

	stdout, stderr, err = e4test.RunCommand(c2clipath,
		[]string{"-c", "sm", "-t", "testtopic", "-m", "hello client"}, []string{})
	if err != nil {
		errc <- &e4test.TestResult{
			Name:     "Test",
			Result:   false,
			Critical: true,
			Error:    err,
		}
		return
	}
	fmt.Println(string(stdout[:]))
	fmt.Println(string(stderr[:]))

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

	c2clibin, earlyerr := e4test.FindAndCheckPathFile("src/teserakt/e4go/bin/c2cli")
	if earlyerr != nil {
		fmt.Fprintf(os.Stderr, "Error: .\n%s", earlyerr)
		exitCode = 1
		return
	}

	//const SERVER = "https://localhost:8888"
	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", e4test.GetRandomDBName())

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)

	go e4test.RunDaemon(errc, stopc, waitdrunc, c2binary, []string{}, env)

	<-waitdrunc
	fmt.Println("Running tests")

	go testC2CLI(errc, c2clibin)

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
