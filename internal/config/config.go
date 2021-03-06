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

	slibcfg "github.com/teserakt-io/serverlib/config"
)

// List of available log levels
const (
	LogLevelNone  = "none"
	LogLevelDebug = "debug"
	LogLevelInfo  = "info"
	LogLevelWarn  = "warn"
	LogLevelError = "error"
)

// Config type holds the application configuration
type Config struct {
	Monitor bool

	Crypto CryptoCfg

	GRPC ServerCfg
	HTTP HTTPServerCfg

	MQTT  MQTTCfg
	Kafka KafkaCfg
	GCP   GCPCfg

	DB DBCfg

	OpencensusSampleAll bool
	OpencensusAddress   string

	ES ESCfg

	LoggerLevel string
}

// New creates a fresh configuration.
func New() *Config {
	return &Config{}
}

// ViperCfgFields returns the list of configuration fields to be loaded by viper
func (cfg *Config) ViperCfgFields() []slibcfg.ViperCfgField {
	return []slibcfg.ViperCfgField{
		{&cfg.Crypto.mode, "crypto-mode", slibcfg.ViperString, "symkey", "E4C2_CRYPTO_MODE"},
		{&cfg.Crypto.C2PrivateKeyPath, "crypto-c2-private-key", slibcfg.ViperRelativePath, "", "E4C2_CRYPTO_KEY"},
		{&cfg.Crypto.NewClientKeySendPubkey, "crypto-new-client-key-send-pubkeys", slibcfg.ViperBool, true, ""},

		{&cfg.GRPC.Addr, "grpc-host-port", slibcfg.ViperString, "0.0.0.0:5555", "E4C2_GRPC_HOST_PORT"},
		{&cfg.GRPC.Cert, "grpc-cert", slibcfg.ViperRelativePath, "", "E4C2_GRPC_CERT"},
		{&cfg.GRPC.Key, "grpc-key", slibcfg.ViperRelativePath, "", "E4C2_GRPC_KEY"},

		{&cfg.HTTP.Addr, "http-host-port", slibcfg.ViperString, "0.0.0.0:8888", "E4C2_HTTP_HOST_PORT"},
		{&cfg.HTTP.GRPCAddr, "http-grpc-host-port", slibcfg.ViperString, "127.0.0.1:5555", "E4C2_HTTP_GRPC_HOST_PORT"},
		{&cfg.HTTP.Cert, "http-cert", slibcfg.ViperRelativePath, "", "E4C2_HTTP_CERT"},
		{&cfg.HTTP.Key, "http-key", slibcfg.ViperRelativePath, "", "E4C2_HTTP_KEY"},

		{&cfg.MQTT.Enabled, "mqtt-enabled", slibcfg.ViperBool, true, "E4C2_MQTT_ENABLED"},
		{&cfg.MQTT.ID, "mqtt-id", slibcfg.ViperString, "e4c2", "E4C2_MQTT_ID"},
		{&cfg.MQTT.Broker, "mqtt-broker", slibcfg.ViperString, "tcp://localhost:1883", "E4C2_MQTT_BROKER"},
		{&cfg.MQTT.QoSPub, "mqtt-qos-pub", slibcfg.ViperInt, 2, "E4C2_MQTT_QOS_PUB"},
		{&cfg.MQTT.QoSSub, "mqtt-qos-sub", slibcfg.ViperInt, 1, "E4C2_MQTT_QOS_SUB"},
		{&cfg.MQTT.Username, "mqtt-username", slibcfg.ViperString, "", ""},
		{&cfg.MQTT.Password, "mqtt-password", slibcfg.ViperString, "", ""},

		{&cfg.Kafka.Enabled, "kafka-enabled", slibcfg.ViperBool, false, "E4C2_KAFKA_ENABLED"},
		{&cfg.Kafka.Brokers, "kafka-brokers", slibcfg.ViperStringSlice, "", ""},

		{&cfg.GCP.Enabled, "gcp-enabled", slibcfg.ViperBool, false, "E4C2_GCP_ENABLED"},
		{&cfg.GCP.ProjectID, "gcp-project-id", slibcfg.ViperString, "", ""},
		{&cfg.GCP.Region, "gcp-region", slibcfg.ViperString, "", ""},
		{&cfg.GCP.RegistryID, "gcp-registry-id", slibcfg.ViperString, "", ""},
		{&cfg.GCP.CommandSubFolder, "gcp-command-subfolder", slibcfg.ViperString, "e4", ""},

		{&cfg.DB.Logging, "db-logging", slibcfg.ViperBool, false, ""},
		{&cfg.DB.Type, "db-type", slibcfg.ViperDBType, "", "E4C2_DB_TYPE"},
		{&cfg.DB.File, "db-file", slibcfg.ViperString, "", "E4C2_DB_FILE"},
		{&cfg.DB.Host, "db-host", slibcfg.ViperString, "", ""},
		{&cfg.DB.Database, "db-database", slibcfg.ViperString, "", ""},
		{&cfg.DB.Schema, "db-schema", slibcfg.ViperString, "", ""},
		{&cfg.DB.Username, "db-username", slibcfg.ViperString, "", "E4C2_DB_USERNAME"},
		{&cfg.DB.Password, "db-password", slibcfg.ViperString, "", "E4C2_DB_PASSWORD"},
		{&cfg.DB.Passphrase, "db-encryption-passphrase", slibcfg.ViperString, "", "E4C2_DB_ENCRYPTION_PASSPHRASE"},
		{&cfg.DB.SecureConnection, "db-secure-connection", slibcfg.ViperDBSecureConnection, slibcfg.DBSecureConnectionEnabled, "E4C2_DB_SECURE_CONNECTION"},

		{&cfg.OpencensusAddress, "oc-agent-addr", slibcfg.ViperString, "localhost:55678", "C2AE_OC_ENDPOINT"},
		{&cfg.OpencensusSampleAll, "oc-sample-all", slibcfg.ViperBool, true, ""},

		{&cfg.ES.Enable, "es-enable", slibcfg.ViperBool, false, "E4C2_ES_ENABLE"},
		{&cfg.ES.URLs, "es-urls", slibcfg.ViperStringSlice, "", "E4C2_ES_URLS"},
		{&cfg.ES.enableMessageLogging, "es-message-logging-enable", slibcfg.ViperBool, true, ""},
		{&cfg.ES.MessageIndexName, "es-message-logging-index", slibcfg.ViperString, "messages", ""},

		{&cfg.LoggerLevel, "log-level", slibcfg.ViperString, "debug", "E4C2_LOG_LEVEL"},
	}
}

// ServerCfg holds configuration for a server
type ServerCfg struct {
	Addr string
	Key  string
	Cert string
}

// HTTPServerCfg extends the ServerCfg for the HTTP
type HTTPServerCfg struct {
	ServerCfg
	GRPCAddr string
}

// MQTTCfg holds configuration for MQTT
type MQTTCfg struct {
	Enabled  bool
	ID       string
	Broker   string
	QoSPub   int
	QoSSub   int
	Username string
	Password string
}

// KafkaCfg holds configuration for Kafka
type KafkaCfg struct {
	Enabled bool
	Brokers []string
}

// GCPCfg holds configuration for GCP IoT Core
type GCPCfg struct {
	Enabled          bool
	ProjectID        string
	Region           string
	RegistryID       string
	CommandSubFolder string
}

// DBCfg holds configuration for databases
type DBCfg struct {
	Logging          bool
	Type             slibcfg.DBType
	File             string
	Host             string
	Database         string
	Username         string
	Password         string
	Passphrase       string
	Schema           string
	SecureConnection slibcfg.DBSecureConnectionType
}

// ESCfg holds ElasticSearch config
type ESCfg struct {
	Enable               bool
	URLs                 []string
	enableMessageLogging bool
	MessageIndexName     string
}

// CryptoMode defines the type of cryptography used by the C2 instance
type CryptoMode string

// List of crypto modes supported
const (
	SymKey CryptoMode = "symkey"
	PubKey CryptoMode = "pubkey"
)

// CryptoCfg holds the crypto configuration
type CryptoCfg struct {
	mode                   string
	C2PrivateKeyPath       string
	NewClientKeySendPubkey bool
}

// CryptoMode returns configured mode as CryptoMode
func (c CryptoCfg) CryptoMode() CryptoMode {
	return CryptoMode(c.mode)
}

// IsMessageLoggingEnabled indicate whenever broker message must be logged to elasticsearch
func (c ESCfg) IsMessageLoggingEnabled() bool {
	return c.Enable && c.enableMessageLogging
}

// ConnectionString returns the string to use to establish the db connection
func (c DBCfg) ConnectionString() (string, error) {
	switch slibcfg.DBType(c.Type) {
	case slibcfg.DBTypePostgres:
		return fmt.Sprintf(
			"host=%s dbname=%s user=%s password=%s search_path=%s %s",
			c.Host, c.Database, c.Username, c.Password, c.Schema, c.SecureConnection.PostgresSSLMode(),
		), nil
	case slibcfg.DBTypeSQLite:
		return c.File, nil
	default:
		return "", ErrUnsupportedDBType
	}
}
