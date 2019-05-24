package analytics

//go:generate mockgen -destination=monitor_mocks.go -package analytics -self_package gitlab.com/teserakt/c2/internal/analytics gitlab.com/teserakt/c2/internal/analytics MessageMonitor

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/olivere/elastic"
)

// MessageMonitor defines an interface able to monitor C2 messages
type MessageMonitor interface {
	OnMessage(msg LoggedMessage)
	Enabled() bool
}

type esMessageMonitor struct {
	esClient    *elastic.Client
	logger      log.Logger
	enabled     bool
	esIndexName string
}

var _ MessageMonitor = &esMessageMonitor{}

// NewESMessageMonitor creates a new message monitor backed by elasticSearch
func NewESMessageMonitor(esClient *elastic.Client, logger log.Logger, enabled bool, esIndexName string) MessageMonitor {
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

func (m *esMessageMonitor) OnMessage(msg LoggedMessage) {
	if !m.enabled {
		m.logger.Log("msg", "message monitoring is not enabled, skipping logging.")
		return
	}

	index := fmt.Sprintf("%s-%s", m.esIndexName, time.Now().Format("2006.01.02"))
	_, err := m.esClient.Index().Index(index).Type("message").
		BodyJson(msg).
		Do(context.Background())
	if err != nil {
		m.logger.Log("msg", "failed to send LoggedMessage to elasticSearch", "error", err, "loggedMessage", msg)
		return
	}
	m.logger.Log("msg", "successfully logged message to elasticsearch", "index", index)
}
