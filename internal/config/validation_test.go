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
