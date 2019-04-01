package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	stdlog "log"

	"gitlab.com/teserakt/c2backend/internal/config"

	e4 "gitlab.com/teserakt/e4common"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/go-kit/kit/log"

	"contrib.go.opencensus.io/exporter/ocagent"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// variables set at build time
var gitCommit string
var buildDate string
var gitTag string

// C2 is the C2's state
type C2 struct {
	keyenckey [e4.KeyLen]byte
	db        *gorm.DB

	mqttClient     mqtt.Client
	logger         log.Logger
	configResolver *e4.AppPathResolver
}

func main() {

	defer os.Exit(1)

	// our server
	var c2 C2

	// show banner
	if len(gitTag) == 0 {
		fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit[:4])
	} else {
		fmt.Printf("E4: C2 back-end - version %s (%s-%s)\n", gitTag, buildDate, gitCommit[:4])
	}
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

	configLoader := config.NewViperLoader("config", c2.configResolver)

	c2.logger.Log("msg", "load configuration and command args")

	cfg, err := configLoader.Load()
	if err != nil {
		c2.logger.Log("error", err)

		return
	}

	if cfg.DB.SecureConnection == config.DBSecureConnectionInsecure {
		c2.logger.Log("msg", "Unencrypted database connection.")
		fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
	} else if cfg.DB.SecureConnection == config.DBSecureConnectionSelfSigned {
		c2.logger.Log("msg", "Self signed certificate used. We do not recommend this setup.")
		fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
	}

	keyenckey := e4.HashPwd(cfg.DB.Passphrase)
	copy(c2.keyenckey[:], keyenckey)

	c2.logger.Log("msg", "config loaded")

	// open db
	dbConnectionString, err := cfg.DB.ConnectionString()
	if err != nil {
		c2.logger.Log("error", err)

		return
	}

	db, err := gorm.Open(cfg.DB.Type.String(), dbConnectionString)
	if err != nil {
		c2.logger.Log("msg", "database opening failed", "error", err)

		return
	}
	defer db.Close()

	c2.logger.Log("msg", "database open")
	db.LogMode(cfg.DB.Logging)
	db.SetLogger(stdloglogger)
	c2.db = db

	if cfg.DB.Type == config.DBTypePostgres {
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
	{
		logger := log.With(c2.logger, "protocol", "mqtt")
		logger.Log("addr", cfg.MQTT.Broker)

		mqOpts := mqtt.NewClientOptions()
		mqOpts.AddBroker(cfg.MQTT.Broker)
		mqOpts.SetClientID(cfg.MQTT.ID)
		mqOpts.SetPassword(cfg.MQTT.Password)
		mqOpts.SetUsername(cfg.MQTT.Username)

		mqttClient := mqtt.NewClient(mqOpts)
		logger.Log("msg", "mqtt parameters", "broker", cfg.MQTT.Broker, "id", cfg.MQTT.ID, "username", cfg.MQTT.Username)
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			logger.Log("msg", "connection failed", "error", token.Error())

			return
		}

		logger.Log("msg", "connected to broker")
		// instantiate C2
		c2.mqttClient = mqttClient
	}

	// initialize OpenCensus
	if err := setupOpencensusInstrumentation(cfg.IsProd); err != nil {
		c2.logger.Log("msg", "OpenCensus instrumentation setup failed", "error", err)

		return
	}

	go func() {
		errc <- c2.createGRPCServer(cfg.GRPC)
	}()

	go func() {
		errc <- c2.createHTTPServer(cfg.HTTP)
	}()

	c2.logger.Log("error", <-errc)
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		next.ServeHTTP(w, r)
	})
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
