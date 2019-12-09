package analytics

//go:generate mockgen -destination=monitor_mocks.go -package analytics -self_package github.com/teserakt-io/c2/internal/analytics github.com/teserakt-io/c2/internal/analytics MessageMonitor

import (
	"context"
	"fmt"
	"time"

	"github.com/olivere/elastic"
	log "github.com/sirupsen/logrus"
)

// MessageMonitor defines an interface able to monitor C2 messages
type MessageMonitor interface {
	OnMessage(ctx context.Context, msg LoggedMessage)
	Enabled() bool
}

type esMessageMonitor struct {
	esClient    *elastic.Client
	logger      log.FieldLogger
	enabled     bool
	esIndexName string
}

var _ MessageMonitor = (*esMessageMonitor)(nil)

// NewESMessageMonitor creates a new message monitor backed by elasticSearch
func NewESMessageMonitor(esClient *elastic.Client, logger log.FieldLogger, enabled bool, esIndexName string) MessageMonitor {
	return &esMessageMonitor{
		esClient:    esClient,
		logger:      logger,
		enabled:     enabled,
		esIndexName: esIndexName,
	}
}

func (m *esMessageMonitor) Enabled() bool {
	return m.enabled
}

func (m *esMessageMonitor) OnMessage(ctx context.Context, msg LoggedMessage) {
	if !m.enabled {
		m.logger.Warn("message monitoring is not enabled, skipping logging.")
		return
	}

	index := fmt.Sprintf("%s-%s", m.esIndexName, time.Now().Format("2006.01.02"))
	_, err := m.esClient.Index().Index(index).Type("message").
		BodyJson(msg).
		Do(context.Background())
	if err != nil {
		m.logger.WithFields(log.Fields{"error": err, "loggedMessage": msg}).Error("failed to send LoggedMessage to elasticSearch")
		return
	}
	// This log produce lots of entries when enabled over a busy broker. Enable with care :)
	m.logger.WithField("index", index).Debug("successfully logged message to elasticsearch")
}
