package c2

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"
	"io/ioutil"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/olivere/elastic"
	e4crypto "github.com/teserakt-io/e4go/crypto"
	sliblog "github.com/teserakt-io/serverlib/log"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/api"
	"github.com/teserakt-io/c2/internal/commands"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/events"
	"github.com/teserakt-io/c2/internal/models"
	"github.com/teserakt-io/c2/internal/protocols"
	"github.com/teserakt-io/c2/internal/services"
)

// C2 Errors
var (
	ErrSubscribeExisting = errors.New("failed to subscribe to existing topics")
)

// APIEndpoint defines an interface that all C2 api endpoints must implement
type APIEndpoint interface {
	ListenAndServe(ctx context.Context) error
}

// C2 ...
type C2 struct {
	cfg             config.Config
	db              models.Database
	logger          log.Logger
	e4Service       services.E4
	pubSubClient    protocols.PubSubClient
	eventDispatcher events.Dispatcher

	privateKey []byte

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

	var privateKey []byte
	if cfg.Crypto.Mode == config.PubKey {
		privateKey, err = ioutil.ReadFile(cfg.Crypto.C2PrivateKeyPath)
		if err != nil {
			return nil, err
		}
		if g, w := len(privateKey), ed25519.PrivateKeySize; g != w {
			return nil, fmt.Errorf("invalid private key length, expected %d, got %d", g, w)
		}
	}

	if cfg.Crypto.Mode != config.SymKey {
		return nil, fmt.Errorf("invalid crypto mode")
	}

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
		logger.Log("msg", "unencrypted database connection.")
		fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
	case cfg.DB.SecureConnection.IsSelfSigned():
		logger.Log("msg", "self-signed certificate used. We do not recommend this setup.")
		fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
	}

	logger.Log("msg", "config loaded")

	db, err := models.NewDB(cfg.DB, stdloglogger)
	if err != nil {
		logger.Log("msg", "database creation failed", "error", err)

		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	logger.Log("msg", "database open")

	if err := db.Migrate(); err != nil {
		logger.Log("msg", "database setup failed", "error", err)

		return nil, fmt.Errorf("database migration failed: %v", err)
	}
	logger.Log("msg", "database initialized")

	monitor := analytics.NewESMessageMonitor(
		esClient,
		log.With(logger, "protocol", "monitoring"),
		cfg.ES.IsMessageLoggingEnabled(),
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

	eventDispatcher := events.NewDispatcher(logger)

	DBEncKey, err := e4crypto.DeriveSymKey(cfg.DB.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create key from passphrase: %v", err)
	}

	e4Service := services.NewE4(
		db,
		pubSubClient,
		commands.NewFactory(),
		eventDispatcher,
		events.NewFactory(),
		log.With(logger, "protocol", "c2"),
		DBEncKey,
	)

	// initialize Observability
	if err := analytics.SetupObservability(cfg.OpencensusAddress, cfg.OpencensusSampleAll); err != nil {
		logger.Log("msg", "Observability instrumentation setup failed", "error", err)

		return nil, fmt.Errorf("observability instrumentation setup failed: %v", err)
	}
	logger.Log("msg", "Observability instrumentation setup successfully", "oc-agent", cfg.OpencensusAddress, "sample-all", cfg.OpencensusSampleAll)

	return &C2{
		cfg:             cfg,
		db:              db,
		logger:          logger,
		e4Service:       e4Service,
		pubSubClient:    pubSubClient,
		eventDispatcher: eventDispatcher,
		privateKey:      privateKey,
	}, nil
}

// Close closes all internal C2 connections
func (c *C2) Close() {
	c.db.Close()
	c.pubSubClient.Disconnect()
}

// EnableHTTPEndpoint will turn on C2 over HTTP
func (c *C2) EnableHTTPEndpoint() {
	c.endpoints = append(c.endpoints, api.NewHTTPServer(c.cfg.HTTP, c.cfg.GRPC.Cert, c.e4Service, log.With(c.logger, "protocol", "http")))
	c.logger.Log("msg", "Enabled C2 HTTP server")
}

// EnableGRPCEndpoint will turn on C2 over GRPC
func (c *C2) EnableGRPCEndpoint() {
	c.endpoints = append(c.endpoints, api.NewGRPCServer(c.cfg.GRPC, c.e4Service, c.eventDispatcher, log.With(c.logger, "protocol", "grpc")))
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
