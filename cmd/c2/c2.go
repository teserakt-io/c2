package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitlab.com/teserakt/c2/internal/api"
	"gitlab.com/teserakt/c2/internal/models"

	stdlog "log"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/services"

	e4 "gitlab.com/teserakt/e4common"

	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/go-kit/kit/log"
)

// variables set at build time
var gitCommit string
var buildDate string
var gitTag string

func main() {

	defer os.Exit(1)

	// show banner
	if len(gitTag) == 0 {
		fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit)
	} else {
		fmt.Printf("E4: C2 back-end - version %s (%s-%s)\n", gitTag, buildDate, gitCommit)
	}
	fmt.Println("Copyright (c) Teserakt AG, 2018-2019")

	// init logger
	logFileName := fmt.Sprintf("/var/log/e4_c2.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[ERROR] logs: unable to open file '%v' to write logs: %v\n", logFileName, err)
		fmt.Print("[WARN] logs: falling back to standard output only\n")
		logFile = os.Stdout
	}

	defer logFile.Close()

	logger := log.NewJSONLogger(logFile)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	defer logger.Log("msg", "goodbye")

	// compatibility for packages that do not understand go-kit logger:
	stdloglogger := stdlog.New(log.NewStdlibAdapter(logger), "", 0)

	// set up config resolver
	configResolver := e4.NewAppPathResolver()
	configLoader := config.NewViperLoader("config", configResolver)

	logger.Log("msg", "load configuration and command args")

	cfg, err := configLoader.Load()
	if err != nil {
		logger.Log("error", err)

		return
	}

	if cfg.DB.SecureConnection == config.DBSecureConnectionInsecure {
		logger.Log("msg", "Unencrypted database connection.")
		fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
	} else if cfg.DB.SecureConnection == config.DBSecureConnectionSelfSigned {
		logger.Log("msg", "Self signed certificate used. We do not recommend this setup.")
		fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
	}

	logger.Log("msg", "config loaded")

	db, err := models.NewDB(cfg.DB, stdloglogger)
	if err != nil {
		logger.Log("error", "db", err)

		return
	}
	defer db.Close()

	logger.Log("msg", "database open")

	if err := db.Migrate(); err != nil {
		logger.Log("msg", "database setup failed", "error", err)

		return
	}
	logger.Log("msg", "database initialized")

	esClient, err := services.NewElasticClient(cfg.ES)
	if err != nil {
		logger.Log("msg", "ElasticSearch setup failed", "error", err)

		return
	}

	logger.Log("msg", "ElasticSearch setup successfully (or disabled by configuration)")

	mqttClient, err := services.NewMQTTClient(cfg.MQTT, log.With(logger, "protocol", "mqtt"), esClient)
	if err != nil {
		logger.Log("msg", "MQTT client creation failed", "error", err)

		return
	}

	logger.Log("msg", "MQTT client created")

	c2Service := services.NewC2(
		db,
		mqttClient,
		log.With(logger, "protocol", "c2"),
		e4.HashPwd(cfg.DB.Passphrase),
	)

	// create critical error channel
	var errc = make(chan error)
	go func() {
		var c = make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// subscribe to topics in the DB if not already done
	topics, err := c2Service.GetTopicList()
	if err != nil {
		logger.Log("msg", "Failed to fetch all existing topics", "error", err)
	}

	if err := mqttClient.SubscribeToTopics(topics); err != nil {
		logger.Log("msg", "Subscribing to all existing topics failed", "error", err)

		return
	}

	// initialize OpenCensus
	oc := services.NewOpenSensus(cfg.IsProd)
	if err := oc.Setup(); err != nil {
		logger.Log("msg", "OpenCensus instrumentation setup failed", "error", err)
		return
	}

	logger.Log("msg", "OpenCensus instrumentation setup successfully")

	grpcServer := api.NewGRPCServer(cfg.GRPC, c2Service, log.With(logger, "protocol", "grpc"))

	httpServer := api.NewHTTPServer(cfg.HTTP, c2Service, log.With(logger, "protocol", "http"))

	go func() {
		errc <- grpcServer.ListenAndServe()
	}()

	go func() {
		errc <- httpServer.ListenAndServe()
	}()

	logger.Log("error", <-errc)
}
