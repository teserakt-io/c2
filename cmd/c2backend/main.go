package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	stdlog "log"

	e4 "gitlab.com/teserakt/e4common"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/go-kit/kit/log"
	"github.com/spf13/viper"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
)

// variables set at build time
var gitCommit string
var buildDate string

// globally defined constants
const configfilename = "c2"

// C2 is the C2's state
type C2 struct {
	keyenckey      [e4.KeyLen]byte
	db             *gorm.DB
	mqttContext    MQTTContext
	logger         log.Logger
	configResolver *e4.AppPathResolver
}

// startServerConfig: settings required for server init of any type
type startServerConfig struct {
	addr     string
	certFile string
	keyFile  string
}

func main() {

	defer os.Exit(1)

	// our server
	var c2 C2

	// show banner
	fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit[:4])
	fmt.Println("Copyright (c) Teserakt AG, 2018-2019")

	// init logger
	logFileName := fmt.Sprintf("/var/log/e4_c2backend.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[ERROR] logs: unable to open file '%v' to write logs: %v\n", logFileName, err)
		fmt.Print("[WARN] logs: falling back to standard output only\n")
		logFile = os.Stdout
	}

	defer logFile.Close()

	c2.logger = log.NewJSONLogger(logFile)
	{
		c2.logger = log.With(c2.logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}
	defer c2.logger.Log("msg", "goodbye")

	// compatibility for packages that do not understand go-kit logger:
	stdloglogger := stdlog.New(log.NewStdlibAdapter(c2.logger), "", 0)

	// set up config resolver
	c2.configResolver = e4.NewAppPathResolver()

	// load config
	c := config(log.With(c2.logger, "unit", "config"), c2.configResolver)
	var (
		isProd       = c.GetBool("production")
		grpcAddr     = c.GetString("grpc-host-port")
		httpAddr     = c.GetString("http-host-port")
		mqttBroker   = c.GetString("mqtt-broker")
		mqttPassword = c.GetString("mqtt-password")
		mqttUsername = c.GetString("mqtt-username")
		mqttID       = c.GetString("mqtt-ID")
		mqttQoSPub   = c.GetInt("mqtt-QoS-pub")
		mqttQoSSub   = c.GetInt("mqtt-QoS-sub")
		dbLogging    = c.GetBool("db-logging")
		dbType       = c.GetString("db-type")
		dbPassphrase = c.GetString("db-encryption-passphrase")
		grpcCertCfg  = c.GetString("grpc-cert")
		grpcKeyCfg   = c.GetString("grpc-key")
		httpCertCfg  = c.GetString("http-cert")
		httpKeyCfg   = c.GetString("http-key")
	)

	// parse all filepaths from the config file.
	var grpcCert, grpcKey, httpCert, httpKey string

	if len(grpcCertCfg) == 0 {
		c2.logger.Log("msg", "No GRPC Certificate path supplied")
		return
	}
	if len(grpcKeyCfg) == 0 {
		c2.logger.Log("msg", "No GRPC Key path supplied")
		return
	}
	if len(httpCertCfg) == 0 {
		c2.logger.Log("msg", "No HTTP Certificate path supplied")
		return
	}
	if len(httpKeyCfg) == 0 {
		c2.logger.Log("msg", "No HTTP Key path supplied")
		return
	}
	grpcCert = c2.configResolver.ConfigRelativePath(grpcCertCfg)
	grpcKey = c2.configResolver.ConfigRelativePath(grpcKeyCfg)
	httpCert = c2.configResolver.ConfigRelativePath(httpCertCfg)
	httpKey = c2.configResolver.ConfigRelativePath(httpKeyCfg)

	if dbPassphrase == "" {
		c2.logger.Log("msg", "no passphrase supplied")
		fmt.Fprintf(os.Stderr, "ERROR: No passphrase supplied. Refusing to start with an empty passphrase.\n")
		return
	}
	keyenckey := e4.HashPwd(dbPassphrase)
	copy(c2.keyenckey[:], keyenckey)

	var dbConnectionString string

	if dbType == "postgres" {
		var (
			dbUsername         = c.GetString("db-username")
			dbPassword         = c.GetString("db-password")
			dbHost             = c.GetString("db-host")
			dbDatabase         = c.GetString("db-database")
			dbSecureConnection = c.GetString("db-secure-connection")
		)
		var sslstring string
		fmt.Println("db-secure-connection", dbSecureConnection)
		if dbSecureConnection == "enable" {
			sslstring = "sslmode=verify-full"
		} else if dbSecureConnection == "selfsigned" {
			sslstring = "sslmode=require"
			c2.logger.Log("msg", "Self signed certificate used. We do not recommend this setup.")
			fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
		} else if dbSecureConnection == "insecure" {
			sslstring = "sslmode=disable"
			c2.logger.Log("msg", "Unencrypted database connection.")
			fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
		} else {
			c2.logger.Log("msg", "Invalid option for db-secure-connection")
			return
		}

		dbConnectionString = fmt.Sprintf("host=%s dbname=%s user=%s password=%s %s",
			dbHost, dbDatabase, dbUsername, dbPassword, sslstring)
	} else if dbType == "sqlite3" {

		c2.logger.Log("msg", "SQLite3 selected as database")

		if isProd {
			fmt.Fprintf(os.Stderr, "ERROR: SQLite3 not supported in production environments\n")
			return
		}

		var (
			dbPath = c.GetString("db-file")
		)
		dbConnectionString = fmt.Sprintf("%s", dbPath)

	} else {
		// defensive coding:
		c2.logger.Log("msg", "unknown or unsupported database type", "db-type", dbType)
		return
	}

	c2.logger.Log("msg", "config loaded")

	// open db
	db, err := gorm.Open(dbType, dbConnectionString)

	if err != nil {
		c2.logger.Log("msg", "database opening failed", "error", err)
		return
	}

	defer db.Close()

	c2.logger.Log("msg", "database open")
	db.LogMode(dbLogging)
	db.SetLogger(stdloglogger)
	c2.db = db

	if dbType == "postgres" {
		c2.db.Exec("SET search_path TO e4_c2_test;")
	}

	// ensure the database schema is ready to use:
	err = c2.dbInitialize()
	if err != nil {
		c2.logger.Log("msg", "database setup failed", "error", err)
		return
	}

	// create critical error channel
	var errc = make(chan error)
	go func() {
		var c = make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// start mqtt client
	mqttStarter := &startMQTTClientConfig{
		addr:     mqttBroker,
		id:       mqttID,
		password: mqttPassword,
		username: mqttUsername,
	}

	if err := c2.createMQTTClient(mqttStarter); err != nil {
		c2.logger.Log("msg", "MQTT client creation failed", "error", err)
		return
	}

	c2.mqttContext.qosPub = mqttQoSPub
	c2.mqttContext.qosSub = mqttQoSSub

	// subscribe to topics in the DB if not already done
	c2.subscribeToDBTopics()

	// initialize OpenCensus
	if err := setupOpencensusInstrumentation(isProd); err != nil {
		c2.logger.Log("msg", "OpenCensus instrumentation setup failed", "error", err)
		return
	}

	// create grpc server
	grpcStarter := &startServerConfig{
		addr:     grpcAddr,
		certFile: grpcCert,
		keyFile:  grpcKey,
	}

	go func() {
		errc <- c2.createGRPCServer(grpcStarter)
	}()

	// create http server
	httpStarter := &startServerConfig{
		addr:     httpAddr,
		certFile: httpCert,
		keyFile:  httpKey,
	}

	go func() {
		errc <- c2.createHTTPServer(httpStarter)
	}()

	c2.logger.Log("error", <-errc)
}

func config(logger log.Logger, pathResolver *e4.AppPathResolver) *viper.Viper {

	logger.Log("msg", "load configuration and command args")

	var v = viper.New()

	v.SetConfigName("config")

	v.AddConfigPath(pathResolver.ConfigDir())

	v.SetDefault("production", false)

	v.SetDefault("mqtt-broker", "tcp://localhost:1883")
	v.SetDefault("mqtt-ID", "e4c2")
	v.SetDefault("mqtt-username", "")
	v.SetDefault("mqtt-password", "")

	v.SetDefault("db-logging", false)
	v.SetDefault("db-host", "localhost")
	v.SetDefault("db-database", "e4")
	v.SetDefault("db-secure-connection", "enable")
	v.SetDefault("grpc-host-port", "0.0.0.0:5555")
	v.SetDefault("http-host	-port", "0.0.0.0:8888")

	// Allow the whole environment to be configured by
	// env variables for testing.
	v.BindEnv("mqtt-broker", "E4C2_MQTT_BROKER")
	v.BindEnv("mqtt-ID", "E4C2_MQTT_ID")
	v.BindEnv("mqtt-QoS-pub", "E4C2_MQTT_QOS_PUB")
	v.BindEnv("mqtt-QoS-sub", "E4C2_MQTT_QOS_SUB")
	v.BindEnv("db-type", "E4C2_DB_TYPE")
	v.BindEnv("db-file", "E4C2_DB_FILE")
	v.BindEnv("db-username", "E4C2_DB_USERNAME")
	v.BindEnv("db-password", "E4C2_DB_PASSWORD")

	// This one in particular should survive even if others are subsequently
	// removed:
	v.BindEnv("db-encryption-passphrase", "E4C2_DB_ENCRYPTION_PASSPHRASE")

	v.BindEnv("db-secure-connection", "E4C2_DB_SECURE_CONNECTION")
	v.BindEnv("grpc-host-port", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("grpc-cert", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("grpc-key", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("http-host-port", "E4C2_HTTP_HOST_PORT")
	v.BindEnv("http-cert", "E4C2_HTTP_HOST_PORT")
	v.BindEnv("http-key", "E4C2_HTTP_HOST_PORT")

	// Now
	//pflag.Parse()
	//viper.BindPFlags(pflag.CommandLine)

	err := v.ReadInConfig()
	if err != nil {
		logger.Log("error", err)
	}

	return v
}

func setupOpencensusInstrumentation(isProd bool) error {
	oce, err := ocagent.NewExporter(
		// TODO: (@odeke-em), enable ocagent-exporter.WithCredentials option.
		ocagent.WithInsecure(),
		ocagent.WithServiceName("c2backend"))

	if err != nil {
		return fmt.Errorf("failed to create the OpenCensus Agent exporter: %v", err)
	}

	// and now finally register it as a Trace Exporter
	trace.RegisterExporter(oce)
	view.RegisterExporter(oce)

	if isProd == false {
		// setting trace sample rate to 100%
		trace.ApplyConfig(trace.Config{
			DefaultSampler: trace.AlwaysSample(),
		})
	}

	return nil
}
