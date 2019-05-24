package config

import (
	"fmt"
)

// DBType defines the different supported database types
type DBType string

const (
	// DBTypePostgres defines the PostgreSQL database type
	DBTypePostgres DBType = "postgres"
	// DBTypeSQLite defines the SQLite database type
	DBTypeSQLite DBType = "sqlite3"
)

// DBSecureConnectionType defines the different database connection security options
type DBSecureConnectionType string

const (
	// DBSecureConnectionEnabled is used to enable SSL on the database connection
	DBSecureConnectionEnabled DBSecureConnectionType = "enabled"
	// DBSecureConnectionSelfSigned is used to allow SSL self signed certificates on the database connection
	DBSecureConnectionSelfSigned DBSecureConnectionType = "selfsigned"
	// DBSecureConnectionInsecure is used to disable SSL on database connection
	DBSecureConnectionInsecure DBSecureConnectionType = "insecure"

	// PostgresSSLModeFull is used to enable full certificate checks on postgres
	PostgresSSLModeFull = "sslmode=verify-full"
	// PostgresSSLModeRequire is used to allow self signed certificates on postgres
	PostgresSSLModeRequire = "sslmode=require"
	// PostgresSSLModeDisable is used to disable encryption on postgres
	PostgresSSLModeDisable = "sslmode=disable"
)

// Loader defines a service able to load configuration
type Loader interface {
	Load() (Config, error)
}

// Config type holds the application configuration
type Config struct {
	IsProd  bool
	Monitor bool

	GRPC ServerCfg
	HTTP ServerCfg

	MQTT  MQTTCfg
	Kafka KafkaCfg

	DB DBCfg

	ES ESCfg
}

// ServerCfg holds configuration for a server
type ServerCfg struct {
	Addr string
	Key  string
	Cert string
}

// MQTTCfg holds configuration for MQTT
type MQTTCfg struct {
	Enabled bool
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
	Type             DBType
	File             string
	Host             string
	Database         string
	Username         string
	Password         string
	Passphrase       string
	Schema           string
	SecureConnection DBSecureConnectionType
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

// IsC2LoggingEnabled indiquate whenever C2 logging is enabled in configuration
func (c ESCfg) IsC2LoggingEnabled() bool {
	return c.Enable && c.enableC2Logging
}

// IsMessageLoggingEnabled indiqate whenever broker message must be logged to elasticsearch
func (c ESCfg) IsMessageLoggingEnabled() bool {
	return c.Enable && c.enableMessageLogging
}

// ConnectionString returns the string to use to establish the db connection
func (c DBCfg) ConnectionString() (string, error) {
	switch DBType(c.Type) {
	case DBTypePostgres:
		return fmt.Sprintf(
			"host=%s dbname=%s user=%s password=%s %s",
			c.Host, c.Database, c.Username, c.Password, c.SecureConnection.SSLMode(),
		), nil
	case DBTypeSQLite:
		return c.File, nil
	default:
		return "", ErrUnsupportedDBType
	}
}

func (t DBType) String() string {
	return string(t)
}

// SSLMode returns the corresponding SSLMode from SecureConnectionType
// defaulting to the most secure one.
func (m DBSecureConnectionType) SSLMode() string {
	switch m {
	case DBSecureConnectionSelfSigned:
		return PostgresSSLModeRequire
	case DBSecureConnectionInsecure:
		return PostgresSSLModeDisable
	default: // DBSecureConnectionEnabled and anything else
		return PostgresSSLModeFull
	}
}

func (m DBSecureConnectionType) String() string {
	return string(m)
}
