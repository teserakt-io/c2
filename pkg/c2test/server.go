package c2test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"time"
)

// Server defines a C2 server to use for testing
type Server interface {
	Start() error
	Stop() error
	Output() ([]byte, error)
}

type server struct {
	dbPath string
	stdout io.Reader
	stderr io.Reader
	cancel context.CancelFunc
}

var _ Server = &server{}

// NewServer creates a new C2 server to use for testing
func NewServer() Server {
	return &server{
		dbPath: GetRandomDBName(),
	}
}

// Start will launch a C2 server and wait for it to be online
func (s *server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	// Start C2 server
	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", s.dbPath)
	BROKER := "E4C2_MQTT_BROKER=tcp://127.0.0.1:1883"
	ESENABLE := "E4C2_ES_ENABLE=false"

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)
	fmt.Fprintf(os.Stderr, "Broker set to %s\n", BROKER)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)
	env = append(env, BROKER)
	env = append(env, ESENABLE)

	cmd := exec.CommandContext(ctx, "bin/c2")
	cmd.Env = append(os.Environ(), env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	s.stdout = stdout
	s.stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Wait for server to be ready
	waitTimeout := 10 * time.Second
	retryTimeout := 100 * time.Millisecond

	online := false
	for !online {
		select {
		case <-time.After(retryTimeout):
			online = CheckC2Online("127.0.0.1", 5555, 8888)
		case <-time.After(waitTimeout):
			return errors.New("timeout while waiting for server to start")
		}
	}

	return nil
}

func (s *server) Stop() error {
	s.cancel()
	return os.Remove(s.dbPath)
}

func (s *server) Output() ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	out, err := ioutil.ReadAll(s.stdout)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(buf, "%s", string(out))

	out, err = ioutil.ReadAll(s.stderr)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(buf, "%s", string(out))

	return buf.Bytes(), nil
}
