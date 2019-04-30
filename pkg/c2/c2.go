package c2

import (
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"

	"gitlab.com/teserakt/c2/internal/analytics"
	"gitlab.com/teserakt/c2/internal/api"
	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/models"
	"gitlab.com/teserakt/c2/internal/protocols"
	"gitlab.com/teserakt/c2/internal/services"
	e4 "gitlab.com/teserakt/e4common"
)

// APIEndpoint defines an interface that all C2 api endpoints must implement
type APIEndpoint interface {
	ListenAndServe() error
}

// C2 ...
type C2 struct {
	cfg        config.Config
	db         models.Database
	logger     log.Logger
	e4Service  services.E4
	mqttClient protocols.MQTTClient

	endpoints []APIEndpoint
}

// New creates a new C2
func New(logger log.Logger, cfg config.Config) (*C2, error) {

	// compatibility for packages that do not understand go-kit logger:
	stdloglogger := stdlog.New(log.NewStdlibAdapter(logger), "", 0)

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
		logger.Log("msg", "database creation failed", "error", err)

		return nil, fmt.Errorf("failed to initialise database: %s", err)
	}

	logger.Log("msg", "database open")

	if err := db.Migrate(); err != nil {
		logger.Log("msg", "database setup failed", "error", err)

		return nil, fmt.Errorf("Database migration failed: %s", err)
	}
	logger.Log("msg", "database initialized")

	esClient, err := analytics.NewElasticClient(cfg.ES)
	if err != nil {
		logger.Log("msg", "ElasticSearch setup failed", "error", err)

		return nil, fmt.Errorf("ElasticSearch setup failed: %s", err)
	}

	logger.Log("msg", "ElasticSearch setup successfully (or disabled by configuration)")

	mqttClient, err := protocols.NewMQTTClient(cfg.MQTT, log.With(logger, "protocol", "mqtt"), esClient)
	if err != nil {
		logger.Log("msg", "MQTT client creation failed", "error", err)

		return nil, fmt.Errorf("MQTT client creation failed: %s", err)
	}

	logger.Log("msg", "MQTT client created")

	e4Service := services.NewE4(
		db,
		mqttClient,
		log.With(logger, "protocol", "c2"),
		e4.HashPwd(cfg.DB.Passphrase),
	)

	// initialize OpenCensus
	oc := analytics.NewOpenCensus(cfg.IsProd)
	if err := oc.Setup(); err != nil {
		logger.Log("msg", "OpenCensus instrumentation setup failed", "error", err)

		return nil, fmt.Errorf("OpenCensus instrumentation setup failed: %s", err)
	}

	logger.Log("msg", "OpenCensus instrumentation setup successfully")

	return &C2{
		cfg:        cfg,
		db:         db,
		logger:     logger,
		e4Service:  e4Service,
		mqttClient: mqttClient,
	}, nil
}

// Close closes all internal C2 connections
func (c *C2) Close() {
	c.db.Close()
}

// EnableHTTPEndpoint will turn on C2 over HTTP
func (c *C2) EnableHTTPEndpoint() {
	c.endpoints = append(c.endpoints, api.NewHTTPServer(c.cfg.HTTP, c.e4Service, log.With(c.logger, "protocol", "http")))
	c.logger.Log("msg", "Enabled C2 HTTP server")
}

// EnableGRPCEndpoint will turn on C2 over GRPC
func (c *C2) EnableGRPCEndpoint() {
	c.endpoints = append(c.endpoints, api.NewGRPCServer(c.cfg.GRPC, c.e4Service, log.With(c.logger, "protocol", "grpc")))
	c.logger.Log("msg", "Enabled C2 GRPC server")
}

// ListenAndServe will start C2
func (c *C2) ListenAndServe() error {

	if len(c.endpoints) == 0 {
		return errors.New("no configured endpoints to serve C2")
	}

	// subscribe to topics in the DB if not already done
	topics, err := c.e4Service.GetTopicList()
	if err != nil {
		c.logger.Log("msg", "Failed to fetch all existing topics", "error", err)

		return fmt.Errorf("Failed to fetch all existing topics: %s", err)
	}

	if err := c.mqttClient.SubscribeToTopics(topics); err != nil {
		c.logger.Log("msg", "Subscribing to all existing topics failed", "error", err)

		return fmt.Errorf("Subscribing to all existing topics failed: %s", err)
	}

	// create critical error channel
	errc := make(chan error)
	go func() {
		var sigc = make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-sigc)
	}()

	for _, endpoint := range c.endpoints {
		go func(endpoint APIEndpoint) {
			errc <- endpoint.ListenAndServe()
		}(endpoint)
	}

	return <-errc
}
