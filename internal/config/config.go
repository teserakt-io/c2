package config

import (
	"fmt"

	slibcfg "github.com/teserakt-io/serverlib/config"
)

// Config type holds the application configuration
type Config struct {
	IsProd  bool
	Monitor bool

	Crypto CryptoCfg

	GRPC ServerCfg
	HTTP HTTPServerCfg

	MQTT  MQTTCfg
	Kafka KafkaCfg

	DB DBCfg

	ES ESCfg
}

// New creates a fresh configuration.
func New() *Config {
	return &Config{}
}

// ViperCfgFields returns the list of configuration fields to be loaded by viper
func (cfg *Config) ViperCfgFields() []slibcfg.ViperCfgField {
	return []slibcfg.ViperCfgField{
		{&cfg.IsProd, "production", slibcfg.ViperBool, false, ""},

		{&cfg.IsProd, "crypto-mode", slibcfg.ViperString, "symkey", "E4C2_CRYPTO_MODE"},
		{&cfg.IsProd, "crypto-key", slibcfg.ViperRelativePath, "symkey", "E4C2_CRYPTO_KEY"},

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

		{&cfg.ES.Enable, "es-enable", slibcfg.ViperBool, false, "E4C2_ES_ENABLE"},
		{&cfg.ES.URLs, "es-urls", slibcfg.ViperStringSlice, "", "E4C2_ES_URLS"},
		{&cfg.ES.enableC2Logging, "es-c2-logging-enable", slibcfg.ViperBool, true, ""},
		{&cfg.ES.C2LogsIndexName, "es-c2-logging-index", slibcfg.ViperString, "logs", ""},
		{&cfg.ES.enableMessageLogging, "es-message-logging-enable", slibcfg.ViperBool, true, ""},
		{&cfg.ES.MessageIndexName, "es-message-logging-index", slibcfg.ViperString, "messages", ""},
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
	enableC2Logging      bool
	enableMessageLogging bool
	C2LogsIndexName      string
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
	Mode             CryptoMode
	C2PrivateKeyPath string
}

// IsC2LoggingEnabled indicate whenever C2 logging is enabled in configuration
func (c ESCfg) IsC2LoggingEnabled() bool {
	return c.Enable && c.enableC2Logging
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
			"host=%s dbname=%s user=%s password=%s %s",
			c.Host, c.Database, c.Username, c.Password, c.SecureConnection.PostgresSSLMode(),
		), nil
	case slibcfg.DBTypeSQLite:
		return c.File, nil
	default:
		return "", ErrUnsupportedDBType
	}
}
