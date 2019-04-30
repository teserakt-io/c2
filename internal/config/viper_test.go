package config

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/d4l3k/messagediff.v1"
)

type testPathResolver struct {
	ConfigDirFunc          func() string
	ConfigRelativePathFunc func(string) string
}

var _ PathResolver = &testPathResolver{}

func (t *testPathResolver) ConfigDir() string {
	return t.ConfigDirFunc()
}

func (t *testPathResolver) ConfigRelativePath(p string) string {
	return t.ConfigRelativePathFunc(p)
}

// getRootDir retrieve project root directory path from current test file
func getRootDir() string {
	_, filename, _, _ := runtime.Caller(0)

	return filepath.Join(filepath.Dir(filename), "..", "..")
}
func TestViperLoader(t *testing.T) {
	resolver := &testPathResolver{
		ConfigDirFunc: func() string {
			return filepath.Join(getRootDir(), "test/data/config/")
		},
		ConfigRelativePathFunc: func(p string) string {
			return p
		},
	}

	t.Run("Loader properly load and returns expected configuration", func(t *testing.T) {
		loader := NewViperLoader("_viper.config.valid", resolver)
		cfg, err := loader.Load()

		if err != nil {
			t.Errorf("expected err to be nil, got %s", err)
		}

		expectedCfg := Config{
			IsProd: true,
			MQTT: MQTTCfg{
				ID:     "mqttid",
				Broker: "tcp://mqtt.broker:1234",
				QoSPub: 1,
				QoSSub: 2,
			},
			DB: DBCfg{
				Logging:          true,
				Type:             DBTypeSQLite,
				File:             "/path/to/db/file",
				Passphrase:       "passphrase",
				SecureConnection: DBSecureConnectionSelfSigned,
			},
			GRPC: ServerCfg{
				Addr: "0.0.0.0:1234",
				Cert: "/path/to/grpc/cert",
				Key:  "/path/to/grpc/key",
			},
			HTTP: ServerCfg{
				Addr: "0.0.0.0:5678",
				Cert: "/path/to/http/cert",
				Key:  "/path/to/http/key",
			},
		}

		diff, equal := messagediff.PrettyDiff(expectedCfg, cfg)

		if !equal {
			t.Errorf("loaded configuration doesn't match expectation:\n%s", diff)
		}
	})

	t.Run("Loader validate configuration and returns errors when invalid", func(t *testing.T) {
		loader := NewViperLoader("_viper.config.invalid-no-passphrase", resolver)
		_, err := loader.Load()

		if !strings.Contains(err.Error(), ErrNoPassphrase.Error()) {
			t.Errorf("expected err to contains %s, got %s", ErrNoPassphrase, err)
		}
	})
}
