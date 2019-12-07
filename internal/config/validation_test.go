package config

import (
	"testing"

	slibcfg "github.com/teserakt-io/serverlib/config"
)

func TestServerCfgValidation(t *testing.T) {
	t.Run("Validate properly checks configuration and return errors", func(t *testing.T) {
		testData := map[ServerCfg]error{
			ServerCfg{}:                  ErrNoAddr,
			ServerCfg{Addr: "something"}: ErrNoCert,
			ServerCfg{Addr: "something", Cert: "cert/path"}:                  ErrNoKey,
			ServerCfg{Addr: "something", Cert: "cert/path", Key: "key/path"}: nil,
		}

		for cfg, expectedErr := range testData {
			err := cfg.Validate()
			if expectedErr != err {
				t.Errorf("expected error to be %s, got %s", expectedErr, err)
			}
		}
	})
}

func TestMQTTCfgValidation(t *testing.T) {
	t.Run("Validate properly checks configuration and return errors", func(t *testing.T) {
		testData := map[MQTTCfg]error{
			MQTTCfg{}: nil,
		}

		for cfg, expectedErr := range testData {
			err := cfg.Validate()
			if expectedErr != err {
				t.Errorf("expected error to be %s, got %s", expectedErr, err)
			}
		}
	})
}

func TestDBCfgValidation(t *testing.T) {
	t.Run("Validate properly checks configuration and return errors", func(t *testing.T) {
		testData := map[DBCfg]error{
			DBCfg{}:                        ErrNoPassphrase,
			DBCfg{Passphrase: "something"}: ErrUnsupportedDBType,
			DBCfg{Passphrase: "something", Type: slibcfg.DBType("something")}:                                                                                                                                     ErrUnsupportedDBType,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres}:                                                                                                                                          ErrNoAddr,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host"}:                                                                                                                            ErrNoDatabase,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo"}:                                                                                                           ErrNoUsername,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo", Username: "bar"}:                                                                                          ErrNoPassword,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo", Username: "bar", Password: "pwd"}:                                                                         ErrNoSchema,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo", Username: "bar", Password: "pwd", Schema: "foo"}:                                                          ErrInvalidSecureConnection,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo", Username: "bar", Password: "pwd", Schema: "foo", SecureConnection: slibcfg.DBSecureConnectionType("foo")}: ErrInvalidSecureConnection,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypePostgres, Host: "host", Database: "foo", Username: "bar", Password: "pwd", Schema: "foo", SecureConnection: slibcfg.DBSecureConnectionInsecure}:    nil,

			DBCfg{Passphrase: "something", Type: slibcfg.DBTypeSQLite}:                       ErrNoDBFile,
			DBCfg{Passphrase: "something", Type: slibcfg.DBTypeSQLite, File: "path/to/file"}: nil,
		}

		for cfg, expectedErr := range testData {
			err := cfg.Validate()
			if expectedErr != err {
				t.Errorf("expected error to be %s, got %s", expectedErr, err)
			}
		}
	})
}

func TestCryptoCfgValidation(t *testing.T) {
	t.Run("Validate properly checks configuration and return errors", func(t *testing.T) {
		testData := map[CryptoCfg]error{
			CryptoCfg{mode: "unknown"}:                                      ErrInvalidCryptoMode,
			CryptoCfg{mode: string(PubKey)}:                                 ErrNoKey,
			CryptoCfg{mode: string(PubKey), C2PrivateKeyPath: "/some/path"}: nil,
			CryptoCfg{mode: string(SymKey)}:                                 nil,
		}

		for cfg, expectedErr := range testData {
			err := cfg.Validate()
			if expectedErr != err {
				t.Errorf("expected error to be %s, got %s", expectedErr, err)
			}
		}
	})
}
