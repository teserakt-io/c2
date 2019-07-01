package c2

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/olivere/elastic"

	"gitlab.com/teserakt/c2/internal/analytics"
	"gitlab.com/teserakt/c2/internal/api"
	"gitlab.com/teserakt/c2/internal/commands"
	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/models"
	"gitlab.com/teserakt/c2/internal/protocols"
	"gitlab.com/teserakt/c2/internal/services"
	e4 "gitlab.com/teserakt/e4common"
	sliblog "gitlab.com/teserakt/serverlib/log"
)

// C2 Errors
var (
	ErrSubscribeExisting = errors.New("Failed to subscribe to existing topics")
)

// APIEndpoint defines an interface that all C2 api endpoints must implement
type APIEndpoint interface {
	ListenAndServe(ctx context.Context) error
}

// C2 ...
type C2 struct {
	cfg          config.Config
	db           models.Database
	logger       log.Logger
	e4Service    services.E4
	pubSubClient protocols.PubSubClient

	endpoints []APIEndpoint
}

// SignalError type system signals to an error
// Used to determine the proper exit code
type SignalError struct {
	text string
}

func (e SignalError) Error() string {
	return e.text
}

// New creates a new C2
func New(logger log.Logger, cfg config.Config) (*C2, error) {
	var err error
	var esClient *elastic.Client

	if cfg.ES.Enable {
		esClient, err = elastic.NewClient(
			elastic.SetURL(cfg.ES.URLs...),
			elastic.SetSniff(false),
		)
		if err != nil {
			return nil, err
		}
	}

	if cfg.ES.IsC2LoggingEnabled() {
		// extend logger to forward log to ES
		esLogger, err := sliblog.WithElasticSearch(logger, esClient, cfg.ES.C2LogsIndexName)
		logger = log.With(esLogger)
		if err != nil {
			return nil, fmt.Errorf("failed to create ES logger: %v", err)
		}

		logger.Log("msg", "elasticsearch log forwarding enabled")
	}

	// compatibility for packages that do not understand go-kit logger:
	stdloglogger := stdlog.New(log.NewStdlibAdapter(logger), "", 0)

	switch {
	case cfg.DB.SecureConnection.IsInsecure():
		logger.Log("msg", "Unencrypted database connection.")
		fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
	case cfg.DB.SecureConnection.IsSelfSigned():
		logger.Log("msg", "Self signed certificate used. We do not recommend this setup.")
		fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
	}

	logger.Log("msg", "config loaded")

	db, err := models.NewDB(cfg.DB, stdloglogger)
	if err != nil {
		logger.Log("msg", "database creation failed", "error", err)

		return nil, fmt.Errorf("failed to initialise database: %v", err)
	}

	logger.Log("msg", "database open")

	if err := db.Migrate(); err != nil {
		logger.Log("msg", "database setup failed", "error", err)

		return nil, fmt.Errorf("Database migration failed: %v", err)
	}
	logger.Log("msg", "database initialized")

	monitor := analytics.NewESMessageMonitor(
		esClient,
		log.With(logger, "protocol", "monitoring"),
		cfg.ES.IsC2LoggingEnabled(),
		cfg.ES.MessageIndexName,
	)

	// TODO switch between available protocols from config. Add config option to choose only 1.

	var pubSubClient protocols.PubSubClient
	switch {
	case cfg.MQTT.Enabled:
		pubSubClient = protocols.NewMQTTPubSubClient(cfg.MQTT, log.With(logger, "protocol", "mqtt"), monitor)
		logger.Log("msg", "MQTT client created")
	case cfg.Kafka.Enabled:
		pubSubClient = protocols.NewKafkaPubSubClient(cfg.Kafka, log.With(logger, "protocol", "kafka"), monitor)
		logger.Log("msg", "Kafka client created")
	default:
		return nil, errors.New("no pubSub client enabled from configuration, cannot start c2 without one")
	}

	if err := pubSubClient.Connect(); err != nil {
		return nil, fmt.Errorf("MQTT client connection failed: %v", err)
	}

	e4Service := services.NewE4(
		db,
		pubSubClient,
		commands.NewFactory(),
		log.With(logger, "protocol", "c2"),
		e4.HashPwd(cfg.DB.Passphrase),
	)

	// initialize Observability
	deploymentMode := analytics.Production
	if !cfg.IsProd {
		deploymentMode = analytics.Development
	}
	if err := deploymentMode.SetupObservability(); err != nil {
		logger.Log("msg", "Observability instrumentation setup failed", "error", err)

		return nil, fmt.Errorf("Observability instrumentation setup failed: %v", err)
	}
	logger.Log("msg", "Observability instrumentation setup successfully")

	return &C2{
		cfg:          cfg,
		db:           db,
		logger:       logger,
		e4Service:    e4Service,
		pubSubClient: pubSubClient,
	}, nil
}

// Close closes all internal C2 connections
func (c *C2) Close() {
	c.db.Close()
	c.pubSubClient.Disconnect()
}

// EnableHTTPEndpoint will turn on C2 over HTTP
func (c *C2) EnableHTTPEndpoint() {
	c.endpoints = append(c.endpoints, api.NewHTTPServer(c.cfg.HTTP, c.cfg.IsProd, c.e4Service, log.With(c.logger, "protocol", "http")))
	c.logger.Log("msg", "Enabled C2 HTTP server")
}

// EnableGRPCEndpoint will turn on C2 over GRPC
func (c *C2) EnableGRPCEndpoint() {
	c.endpoints = append(c.endpoints, api.NewGRPCServer(c.cfg.GRPC, c.e4Service, log.With(c.logger, "protocol", "grpc")))
	c.logger.Log("msg", "Enabled C2 GRPC server")
}

// ListenAndServe will start C2
func (c *C2) ListenAndServe(ctx context.Context) error {
	if len(c.endpoints) == 0 {
		return errors.New("no configured endpoints to serve C2")
	}

	// create critical error channel
	errc := make(chan error)

	if c.cfg.ES.IsMessageLoggingEnabled() {
		go func() {
			topicCount, err := c.e4Service.CountTopics(ctx)
			if err != nil {
				c.logger.Log("msg", "Failed to count topics", "error", err)
				errc <- ErrSubscribeExisting
				return
			}

			offset := 0
			batchSize := 100
			for offset < topicCount {
				topics, err := c.e4Service.GetTopicsRange(ctx, offset, batchSize)
				if err != nil {
					c.logger.Log("msg", "Failed to get topic batch", "error", err, "offset", offset, "batchSize", batchSize)
					errc <- ErrSubscribeExisting
					return
				}

				if err := c.pubSubClient.SubscribeToTopics(ctx, topics); err != nil {
					c.logger.Log("msg", "Subscribing to all existing topics failed", "error", err)
					errc <- ErrSubscribeExisting
					return
				}

				offset += batchSize
			}

			c.logger.Log("msg", "subscribed to all topics", "count", topicCount)
		}()
	} else {
		c.logger.Log("msg", "message monitoring is not enabled, skipping global subscription")
	}

	go func() {
		var sigc = make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)

		select {
		case errc <- SignalError{fmt.Sprintf("%v", <-sigc)}:
		case <-ctx.Done():
			return
		}
	}()

	for _, endpoint := range c.endpoints {
		go func(endpoint APIEndpoint) {
			errc <- endpoint.ListenAndServe(ctx)
		}(endpoint)
	}

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
