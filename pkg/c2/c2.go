// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package c2

import (
	"context"
	"errors"
	"fmt"
	stdlog "log"
	"os"
	"os/signal"
	"syscall"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
	e4crypto "github.com/teserakt-io/e4go/crypto"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/api"
	"github.com/teserakt-io/c2/internal/commands"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/crypto"
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
	logger          log.FieldLogger
	e4Service       services.E4
	pubSubClient    protocols.PubSubClient
	eventDispatcher events.Dispatcher

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
func New(logger log.FieldLogger, cfg config.Config) (*C2, error) {
	var err error

	var e4Key crypto.E4Key
	switch cfg.Crypto.CryptoMode() {
	case config.SymKey:
		e4Key = crypto.NewE4SymKey()
		logger.Info("initialized E4Key in symmetric key mode")
	case config.PubKey:
		e4Key, err = crypto.NewE4PubKey(cfg.Crypto.C2PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create E4PubKey: %v", err)
		}

		logger.Info("initialized E4Key in public key mode")
	default:
		return nil, fmt.Errorf("unsupported crypto mode: %s", cfg.Crypto.CryptoMode())
	}

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

	switch {
	case cfg.DB.SecureConnection.IsInsecure():
		logger.Warn("Unencrypted database connection.")
		fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
	case cfg.DB.SecureConnection.IsSelfSigned():
		logger.Warn("Self signed certificate used. We do not recommend this setup.")
		fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
	}

	dbLogger := stdlog.New(logger.WithField("protocol", "db").WriterLevel(log.DebugLevel), "", 0)
	db, err := models.NewDB(cfg.DB, dbLogger)
	if err != nil {
		logger.WithError(err).Error("database creation failed")

		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	logger.Info("database connection opened")

	if err := db.Migrate(); err != nil {
		log.WithError(err).Error("database migration failed")

		return nil, fmt.Errorf("database migration failed: %v", err)
	}
	logger.Info("database initialized")

	monitor := analytics.NewESMessageMonitor(
		esClient,
		logger.WithField("protocol", "monitoring"),
		cfg.ES.IsMessageLoggingEnabled(),
		cfg.ES.MessageIndexName,
	)

	var pubSubClient protocols.PubSubClient
	switch {
	case cfg.MQTT.Enabled:
		pubSubClient = protocols.NewMQTTPubSubClient(cfg.MQTT, logger.WithField("protocol", "mqtt"), monitor)
		logger.Info("MQTT client created")
	case cfg.Kafka.Enabled:
		pubSubClient = protocols.NewKafkaPubSubClient(cfg.Kafka, logger.WithField("protocol", "kafka"), monitor)
		logger.Info("Kafka client created")
	default:
		return nil, errors.New("no pubSub client enabled from configuration, cannot start c2 without one")
	}

	if err := pubSubClient.Connect(); err != nil {
		return nil, fmt.Errorf("MQTT client connection failed: %v", err)
	}

	eventDispatcher := events.NewDispatcher(logger)

	dbEncKey, err := e4crypto.DeriveSymKey(cfg.DB.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to create key from passphrase: %v", err)
	}

	e4Service := services.NewE4(
		db,
		pubSubClient,
		commands.NewFactory(),
		eventDispatcher,
		events.NewFactory(),
		e4Key,
		logger.WithField("protocol", "e4"),
		dbEncKey,
		cfg.Crypto,
	)

	// initialize Observability
	if err := analytics.SetupObservability(cfg.OpencensusAddress, cfg.OpencensusSampleAll); err != nil {
		logger.WithError(err).Error("observability instrumentation setup failed")
		return nil, fmt.Errorf("observability instrumentation setup failed: %v", err)
	}
	logger.WithFields(log.Fields{
		"oc-agent":   cfg.OpencensusAddress,
		"sample-all": cfg.OpencensusSampleAll,
	}).Info("observability instrumentation setup successfully")

	return &C2{
		cfg:             cfg,
		db:              db,
		logger:          logger,
		e4Service:       e4Service,
		pubSubClient:    pubSubClient,
		eventDispatcher: eventDispatcher,
	}, nil
}

// Close closes all internal C2 connections
func (c *C2) Close() {
	c.db.Close()
	c.pubSubClient.Disconnect()
}

// EnableHTTPEndpoint will turn on C2 over HTTP
func (c *C2) EnableHTTPEndpoint() {
	c.endpoints = append(c.endpoints, api.NewHTTPServer(c.cfg.HTTP, c.cfg.GRPC.Cert, c.logger.WithField("protocol", "http")))
	c.logger.Info("enabled C2 HTTP server")
}

// EnableGRPCEndpoint will turn on C2 over GRPC
func (c *C2) EnableGRPCEndpoint() {
	c.endpoints = append(c.endpoints, api.NewGRPCServer(c.cfg.GRPC, c.e4Service, c.eventDispatcher, c.logger.WithField("protocol", "grpc")))
	c.logger.Info("enabled C2 GRPC server")
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
				c.logger.WithError(err).Error("failed to count topics")
				errc <- ErrSubscribeExisting
				return
			}

			offset := 0
			batchSize := 100
			for offset < topicCount {
				topics, err := c.e4Service.GetTopicsRange(ctx, offset, batchSize)
				if err != nil {
					c.logger.WithError(err).Error("failed to get topic batch")
					errc <- ErrSubscribeExisting
					return
				}

				if err := c.pubSubClient.SubscribeToTopics(ctx, topics); err != nil {
					c.logger.WithError(err).Error("subscribing to all existing topics failed")
					errc <- ErrSubscribeExisting
					return
				}

				offset += batchSize
			}

			c.logger.WithField("count", topicCount).Info("subscribed to all topics")
		}()
	} else {
		c.logger.Warn("message monitoring is not enabled, skipping global subscription")
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
