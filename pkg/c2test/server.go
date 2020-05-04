// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	dbType := "E4C2_DB_TYPE=sqlite3"
	dbName := fmt.Sprintf("E4C2_DB_FILE=%s", s.dbPath)
	broker := fmt.Sprintf("E4C2_MQTT_BROKER=tcp://%s", s.mqttEndpoint)
	esEnable := "E4C2_ES_ENABLE=false"
	passphrase := "E4C2_DB_ENCRYPTION_PASSPHRASE=very_secure_testpass"
	cryptoMode := "E4C2_CRYPTO_MODE=symkey"

	fmt.Fprintf(os.Stderr, "Database set to %s\n", dbName)
	fmt.Fprintf(os.Stderr, "Broker set to %s\n", broker)

	env := []string{
		dbType,
		dbName,
		broker,
		esEnable,
		passphrase,
		cryptoMode,
	}

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
