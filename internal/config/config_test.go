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

package config

import (
	"fmt"
	"testing"

	slibcfg "github.com/teserakt-io/serverlib/config"
)

func TestDBCfg(t *testing.T) {
	t.Run("ConnectionString returns the proper connection string for Postgres type", func(t *testing.T) {
		expectedDatabase := "test"
		expectedHost := "some/host:port"
		expectedUsername := "username"
		expectedPassword := "password"
		expectedSchema := "schema"

		cfg := DBCfg{
			Type:     slibcfg.DBTypePostgres,
			Database: expectedDatabase,
			Host:     expectedHost,
			Username: expectedUsername,
			Password: expectedPassword,
			Schema:   expectedSchema,
		}

		expectedConnectionString := fmt.Sprintf(
			"host=%s dbname=%s user=%s password=%s search_path=%s %s",
			expectedHost,
			expectedDatabase,
			expectedUsername,
			expectedPassword,
			expectedSchema,
			slibcfg.PostgresSSLModeFull,
		)

		cnxStr, err := cfg.ConnectionString()

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if expectedConnectionString != cnxStr {
			t.Errorf("expected connectionString to be %s, got %s", expectedConnectionString, cnxStr)
		}
	})

	t.Run("ConnectionString returns the proper connection string for SQLite type", func(t *testing.T) {
		expectedFile := "some/db/file"

		cfg := DBCfg{
			Type: slibcfg.DBTypeSQLite,
			File: expectedFile,
		}

		cnxStr, err := cfg.ConnectionString()

		if err != nil {
			t.Errorf("expected no error, got %s", err)
		}

		if expectedFile != cnxStr {
			t.Errorf("expected connectionString to be %s, got %s", expectedFile, cnxStr)
		}
	})

	t.Run("ConnectionString returns an error on unsupported DB type", func(t *testing.T) {
		cfg := DBCfg{
			Type: slibcfg.DBType("unknown"),
		}

		_, err := cfg.ConnectionString()

		if err != ErrUnsupportedDBType {
			t.Errorf("Expected err to be %s, got %s", ErrUnsupportedDBType, err)
		}
	})

}
