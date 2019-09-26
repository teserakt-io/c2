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

		cfg := DBCfg{
			Type:     slibcfg.DBTypePostgres,
			Database: expectedDatabase,
			Host:     expectedHost,
			Username: expectedUsername,
			Password: expectedPassword,
		}

		expectedConnectionString := fmt.Sprintf(
			"host=%s dbname=%s user=%s password=%s %s",
			expectedHost,
			expectedDatabase,
			expectedUsername,
			expectedPassword,
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
