package config

import (
	"errors"
	"fmt"
)

var (
	// ErrNoAddr is returned when the server address is missing from configuration
	ErrNoAddr = errors.New("no address supplied")
	// ErrNoCert is returned when the certificate path is missing from configuration
	ErrNoCert = errors.New("no certificate path supplied")
	// ErrNoKey is returned when the key path is missing from configuration
	ErrNoKey = errors.New("no key path supplied")
	// ErrNoPassphrase is returned when the passphrase is missing from configuration
	ErrNoPassphrase = errors.New("no database passphrase supplied")
	// ErrNoDatabase is returned when the database name is missing
	ErrNoDatabase = errors.New("no database name supplied")
	// ErrUnsupportedDBType is returned when an invalid DB type is provided in configuration
	ErrUnsupportedDBType = errors.New("unknown or unsupported database type")
	// ErrNoDBFile is returned when no database file is provided in configuration
	ErrNoDBFile = errors.New("no database file supplied")
	// ErrNoUsername is returned when no username is provided in configuration
	ErrNoUsername = errors.New("no username supplied")
	// ErrNoPassword is returned when no password is provided in configuration
	ErrNoPassword = errors.New("no password supplied")
	// ErrInvalidSecureConnection is returned when an invalid secure connection mode is provided.
	// see available config.DBSecureConnectionType
	ErrInvalidSecureConnection = errors.New("invalid secure connection mode")
	// ErrNoSchema is returned when database configuration is missing a schema (postgres only)
	ErrNoSchema = errors.New("no schema supplied")
	// ErrAtLeastOneURLRequired is returned when a list of urls is empty but require at least one
	ErrAtLeastOneURLRequired = errors.New("at least one url is required")
	// ErrIndexNameRequired is returned when a index name is empty but required
	ErrIndexNameRequired = errors.New("index name is required")
)

// Validate check Config and returns an error if anything is invalid
func (c Config) Validate() error {
	if err := c.GRPC.Validate(); err != nil {
		return fmt.Errorf("GRPC configuration validation error: %v", err)
	}

	if err := c.HTTP.Validate(); err != nil {
		return fmt.Errorf("HTTP configuration validation error: %v", err)
	}

	if err := c.MQTT.Validate(); err != nil {
		return fmt.Errorf("MQTT configuration validation error: %v", err)
	}

	if err := c.ES.Validate(); err != nil {
		return fmt.Errorf("ES configuration validation error: %v", err)
	}

	if err := c.Kafka.Validate(); err != nil {
		return fmt.Errorf("Kafka configuration validation error: %v", err)
	}

	if err := c.DB.Validate(); err != nil {
		return fmt.Errorf("DB configuration validation error: %v", err)
	}

	return nil
}

// Validate checks ServerCfg and returns an error if anything is invalid
func (c ServerCfg) Validate() error {
	if len(c.Addr) == 0 {
		return ErrNoAddr
	}

	if len(c.Cert) == 0 {
		return ErrNoCert
	}

	if len(c.Key) == 0 {
		return ErrNoKey
	}

	return nil
}

// Validate checks MQTTCfg and returns an error if anything is invalid
func (c MQTTCfg) Validate() error {
	return nil
}

// Validate checks ESCfg and returns an error if anything is invalid
func (c ESCfg) Validate() error {
	if c.Enable && len(c.URLs) == 0 {
		return ErrAtLeastOneURLRequired
	}

	if c.IsC2LoggingEnabled() && len(c.C2LogsIndexName) == 0 {
		return ErrIndexNameRequired
	}

	if c.IsMessageLoggingEnabled() && len(c.MessageIndexName) == 0 {
		return ErrIndexNameRequired
	}

	return nil
}

func (c KafkaCfg) Validate() error {
	return nil
}

// Validate checks DBCfg and returns an error if anything is invalid
func (c DBCfg) Validate() error {
	if len(c.Passphrase) == 0 {
		return ErrNoPassphrase
	}

	switch c.Type {
	case DBTypePostgres:
		return c.validatePostgres()
	case DBTypeSQLite:
		return c.validateSQLite()
	default:
		return ErrUnsupportedDBType
	}
}

func (c DBCfg) validatePostgres() error {
	if len(c.Host) == 0 {
		return ErrNoAddr
	}

	if len(c.Database) == 0 {
		return ErrNoDatabase
	}

	if len(c.Username) == 0 {
		return ErrNoUsername
	}

	if len(c.Password) == 0 {
		return ErrNoPassword
	}

	if len(c.Schema) == 0 {
		return ErrNoSchema
	}

	if c.SecureConnection != DBSecureConnectionEnabled &&
		c.SecureConnection != DBSecureConnectionSelfSigned &&
		c.SecureConnection != DBSecureConnectionInsecure {
		return ErrInvalidSecureConnection
	}

	return nil
}

func (c DBCfg) validateSQLite() error {
	if len(c.File) == 0 {
		return ErrNoDBFile
	}

	return nil
}
