package c2test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// Server defines a C2 server to use for testing
type Server interface {
	Start() error
	Stop() error
}

type server struct {
	dbPath       string
	mqttEndpoint string
	cmd          *exec.Cmd
}

var _ Server = (*server)(nil)

// NewServer creates a new C2 server to use for testing
func NewServer(mqttEndpoint string) Server {
	return &server{
		dbPath:       GetRandomDBName(),
		mqttEndpoint: mqttEndpoint,
	}
}

// Start will launch a C2 server and wait for it to be online
func (s *server) Start() error {
	// Start C2 server
	DBNAME := fmt.Sprintf("E4C2_DB_FILE=%s", s.dbPath)
	BROKER := fmt.Sprintf("E4C2_MQTT_BROKER=tcp://%s", s.mqttEndpoint)
	ESENABLE := "E4C2_ES_ENABLE=false"

	fmt.Fprintf(os.Stderr, "Database set to %s\n", DBNAME)
	fmt.Fprintf(os.Stderr, "Broker set to %s\n", BROKER)

	env := []string{"E4C2_DB_TYPE=sqlite3"}
	env = append(env, DBNAME)
	env = append(env, BROKER)
	env = append(env, ESENABLE)

	s.cmd = exec.Command("bin/c2")
	s.cmd.Env = append(os.Environ(), env...)

	s.cmd.Stdout = os.Stderr
	s.cmd.Stderr = os.Stderr

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Wait for server to be ready
	retryTimeout := 100 * time.Millisecond
	maxRetryCount := 100
	retryCount := 0

	ticker := time.NewTicker(retryTimeout)
	defer ticker.Stop()

	for range ticker.C {
		if CheckC2Online("127.0.0.1", 5555, 8888) {
			return nil
		}

		if retryCount > maxRetryCount {
			s.Stop()
			return errors.New("timeout while waiting for server to start")
		}

		retryCount++
	}

	return nil
}

func (s *server) Stop() error {
	if s.cmd != nil {
		if err := s.cmd.Process.Kill(); err != nil {
			return err
		}
	}

	if err := os.Remove(s.dbPath); err != nil {
		return err
	}

	return nil
}
