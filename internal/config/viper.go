package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// viperConfigLoader implements config.Loader
type viperConfigLoader struct {
	v            *viper.Viper
	pathResolver PathResolver
}

// PathResolver defines method for resolving application configuration paths
type PathResolver interface {
	ConfigDir() string
	ConfigRelativePath(string) string
}

// NewViperLoader creates a new configuration loader using Viper
func NewViperLoader(configName string, pathResolver PathResolver) Loader {
	v := viper.New()
	v.SetConfigName(configName)
	v.AddConfigPath(pathResolver.ConfigDir())

	return &viperConfigLoader{
		v:            v,
		pathResolver: pathResolver,
	}
}

type viperType int

const (
	viperInt viperType = iota
	viperString
	viperStringSlice
	viperBool
	viperDBType
	viperSecureConnection
)

type viperCfgField struct {
	target       interface{}
	keyName      string
	cfgType      viperType
	defaultValue interface{}
	envMapping   string
}

// Load configure viper and read the configuration
func (loader *viperConfigLoader) Load() (Config, error) {
	cfg := Config{}

	viperFields := []viperCfgField{
		{&cfg.IsProd, "production", viperBool, false, ""},

		{&cfg.GRPC.Addr, "grpc-host-port", viperString, "0.0.0.0:5555", "E4C2_GRPC_HOST_PORT"},
		{&cfg.GRPC.Cert, "grpc-cert", viperString, "", "E4C2_GRPC_CERT"},
		{&cfg.GRPC.Key, "grpc-key", viperString, "", "E4C2_GRPC_KEY"},

		{&cfg.HTTP.Addr, "http-host-port", viperString, "0.0.0.0:8888", "E4C2_HTTP_HOST_PORT"},
		{&cfg.HTTP.Cert, "http-cert", viperString, "", "E4C2_HTTP_CERT"},
		{&cfg.HTTP.Key, "http-key", viperString, "", "E4C2_HTTP_KEY"},

		{&cfg.MQTT.ID, "mqtt-id", viperString, "e4c2", "E4C2_MQTT_ID"},
		{&cfg.MQTT.Broker, "mqtt-broker", viperString, "tcp://localhost:1883", "E4C2_MQTT_BROKER"},
		{&cfg.MQTT.QoSPub, "mqtt-qos-pub", viperInt, 2, "E4C2_MQTT_QOS_PUB"},
		{&cfg.MQTT.QoSSub, "mqtt-qos-sub", viperInt, 1, "E4C2_MQTT_QOS_SUB"},
		{&cfg.MQTT.Username, "mqtt-username", viperString, "", ""},
		{&cfg.MQTT.Password, "mqtt-password", viperString, "", ""},

		{&cfg.Kafka.Brokers, "kafka-brokers", viperStringSlice, "", ""},

		{&cfg.DB.Logging, "db-logging", viperBool, false, ""},
		{&cfg.DB.Type, "db-type", viperDBType, "", "E4C2_DB_TYPE"},
		{&cfg.DB.File, "db-file", viperString, "", "E4C2_DB_FILE"},
		{&cfg.DB.Host, "db-host", viperString, "", ""},
		{&cfg.DB.Database, "db-database", viperString, "", ""},
		{&cfg.DB.Schema, "db-schema", viperString, "", ""},
		{&cfg.DB.Username, "db-username", viperString, "", "E4C2_DB_USERNAME"},
		{&cfg.DB.Password, "db-password", viperString, "", "E4C2_DB_PASSWORD"},
		{&cfg.DB.Passphrase, "db-encryption-passphrase", viperString, "", "E4C2_DB_ENCRYPTION_PASSPHRASE"},
		{&cfg.DB.SecureConnection, "db-secure-connection", viperSecureConnection, "enable", "E4C2_DB_SECURE_CONNECTION"},

		{&cfg.ES.Enable, "es-enable", viperBool, false, "E4C2_ES_ENABLE"},
		{&cfg.ES.URLs, "es-urls", viperStringSlice, "", "E4C2_ES_URLS"},
		{&cfg.ES.enableC2Logging, "es-c2-logging-enable", viperBool, true, ""},
		{&cfg.ES.C2LogsIndexName, "es-c2-logging-index", viperString, "logs", ""},
		{&cfg.ES.enableMessageLogging, "es-message-logging-enable", viperBool, true, ""},
		{&cfg.ES.MessageIndexName, "es-message-logging-index", viperString, "messages", ""},
	}

	if err := loader.loadFields(viperFields); err != nil {
		return cfg, err
	}

	if err := cfg.Validate(); err != nil {
		return cfg, err
	}

	cfg.GRPC.Cert = loader.pathResolver.ConfigRelativePath(cfg.GRPC.Cert)
	cfg.GRPC.Key = loader.pathResolver.ConfigRelativePath(cfg.GRPC.Key)
	cfg.HTTP.Cert = loader.pathResolver.ConfigRelativePath(cfg.HTTP.Cert)
	cfg.HTTP.Key = loader.pathResolver.ConfigRelativePath(cfg.HTTP.Key)

	return cfg, nil
}

func (loader *viperConfigLoader) loadFields(fields []viperCfgField) error {
	for _, field := range fields {
		loader.v.SetDefault(field.keyName, field.defaultValue)

		if field.envMapping != "" {
			loader.v.BindEnv(field.keyName, field.envMapping)
		}
	}

	if err := loader.v.ReadInConfig(); err != nil {
		return err
	}

	for _, field := range fields {
		switch field.cfgType {
		case viperInt:
			v := field.target.(*int)
			*v = loader.v.GetInt(field.keyName)
		case viperString:
			v := field.target.(*string)
			*v = loader.v.GetString(field.keyName)
		case viperStringSlice:
			v := field.target.(*[]string)
			*v = loader.v.GetStringSlice(field.keyName)
		case viperBool:
			v := field.target.(*bool)
			value := loader.v.GetBool(field.keyName)
			*v = value
		case viperDBType:
			v := field.target.(*DBType)
			*v = DBType(loader.v.GetString(field.keyName))
		case viperSecureConnection:
			v := field.target.(*DBSecureConnectionType)
			*v = DBSecureConnectionType(loader.v.GetString(field.keyName))
		default:
			return fmt.Errorf("unsupported configuration type %v for field %v", field.cfgType, field.keyName)
		}
	}

	return nil
}
