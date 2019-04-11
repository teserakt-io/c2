package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	stdlog "log"

	"gitlab.com/teserakt/c2/internal/config"

	e4 "gitlab.com/teserakt/e4common"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/go-kit/kit/log"

	"github.com/olivere/elastic"
)

// variables set at build time
var gitCommit string
var buildDate string
var gitTag string

// C2 is the C2's state
type C2 struct {
	keyenckey      [e4.KeyLen]byte
	db             *gorm.DB
	mqttContext    MQTTContext
	logger         log.Logger
	configResolver *e4.AppPathResolver
	esClient       *elastic.Client
}

func main() {

	defer os.Exit(1)

	// our server
	var c2 C2

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
	c2.logger.Log("msg", "database initialized")

	// create critical error channel
	var errc = make(chan error)
	go func() {
		var c = make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// start MQTT client
	{
		if err := c2.createMQTTClient(&cfg.MQTT); err != nil {
			c2.logger.Log("msg", "MQTT client creation failed", "error", err)
			return
		}
		c2.logger.Log("msg", "MQTT client created")

		c2.mqttContext.qosPub = cfg.MQTT.QoSPub
		c2.mqttContext.qosSub = cfg.MQTT.QoSSub

		// subscribe to topics in the DB if not already done
		c2.subscribeToDBTopics()
	}

	// initialize ElasticSearch
	if cfg.ES.Enable {
		if err := c2.createESClient(cfg.ES.URL); err != nil {
			c2.logger.Log("msg", "ElasticSearch setup failed", "error", err)
		}
		c2.logger.Log("msg", "ElasticSearch setup successfully")
	} else {
		c2.esClient = nil
		c2.logger.Log("msg", "monitoring disabled: ElasticSearch not setup")
	}

	// initialize OpenCensus
	if err := setupOpencensusInstrumentation(cfg.IsProd); err != nil {
		c2.logger.Log("msg", "OpenCensus instrumentation setup failed", "error", err)
		return
	}
	c2.logger.Log("msg", "OpenCensus instrumentation setup successfully")

	go func() {
		errc <- c2.createGRPCServer(cfg.GRPC)
	}()

	go func() {
		errc <- c2.createHTTPServer(cfg.HTTP)
	}()

	c2.logger.Log("error", <-errc)
}

func (c2 *C2) createESClient(url string) error {
	var err error
	c2.esClient, err = elastic.NewClient(
		elastic.SetURL(url),
		elastic.SetSniff(false),
	)
	if err != nil {
		return err
	}
	ctx := context.Background()
	exists, err := c2.esClient.IndexExists("messages").Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		createIndex, err := c2.esClient.CreateIndex("messages").Do(ctx)
		if err != nil {
			return err
		}
		if !createIndex.Acknowledged {
			return fmt.Errorf("index creation not acknowledged")
		}
	}

	return nil
}
