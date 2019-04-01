package config

import (
	"fmt"
	"testing"
)

func TestDBCfg(t *testing.T) {
	t.Run("ConnectionString returns the proper connection string for Postgres type", func(t *testing.T) {
		expectedDatabase := "test"
		expectedHost := "some/host:port"
		expectedUsername := "username"
		expectedPassword := "password"

		cfg := DBCfg{
			Type:     DBTypePostgres,
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
			PostgresSSLModeFull,
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
			Type: DBTypeSQLite,
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
			Type: DBType("unknow"),
		}

		_, err := cfg.ConnectionString()

		if err != ErrUnsupportedDBType {
			t.Errorf("Expected err to be %s, got %s", ErrUnsupportedDBType, err)
		}
	})

}

func TestSecureConnectionType(t *testing.T) {
	t.Run("SslMode returns the expected SSLMode from given SecureConnectionType", func(t *testing.T) {

		testData := map[DBSecureConnectionType]string{
			DBSecureConnectionEnabled:        PostgresSSLModeFull,
			DBSecureConnectionSelfSigned:     PostgresSSLModeRequire,
			DBSecureConnectionInsecure:       PostgresSSLModeDisable,
			DBSecureConnectionType("random"): PostgresSSLModeFull,
		}

		for secureCnxType, expectedSSLMode := range testData {
			if expectedSSLMode != secureCnxType.SSLMode() {
				t.Errorf("Expected SSLMode to be %s, got %s", expectedSSLMode, secureCnxType.SSLMode())
			}
		}
	})
}
