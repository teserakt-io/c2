package analytics

//go:generate mockgen -destination=monitor_mocks.go -package analytics -self_package gitlab.com/teserakt/c2/internal/analytics gitlab.com/teserakt/c2/internal/analytics MessageMonitor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/olivere/elastic"
	"gitlab.com/teserakt/c2/internal/config"
)

// MessageMonitor defines an interface able to monitor C2 messages
type MessageMonitor interface {
	OnMessage(msg LoggedMessage)
	Enabled() bool
}

type esMessageMonitor struct {
	esClient *elastic.Client
	cfg      config.ESCfg
}

var _ MessageMonitor = &esMessageMonitor{}

// NewESMessageMonitor creates a new message monitor backed by elasticSearch
func NewESMessageMonitor(cfg config.ESCfg) (MessageMonitor, error) {
	monitor := &esMessageMonitor{
		cfg: cfg,
	}

	if !cfg.Enable {
		return monitor, nil // Doesn't attempt to connect to ES when not enabled
	}

	esClient, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
	)

	if err != nil {
		return nil, err
	}
	monitor.esClient = esClient

	ctx := context.Background()

	// TODO: elastic usually work with index having a date in it, like log-YYYY-MM-DD
	// it make is easier to query / manage the indexes. Should we use it like this ?
	exists, err := esClient.IndexExists("messages").Do(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		createIndex, err := esClient.CreateIndex("messages").Do(ctx)
		if err != nil {
			return nil, err
		}
		if !createIndex.Acknowledged {
			return nil, fmt.Errorf("index creation not acknowledged")
		}
	}

	return monitor, nil
}

func (m *esMessageMonitor) Enabled() bool {
	return m.cfg.Enable
}

func (m *esMessageMonitor) OnMessage(msg LoggedMessage) {
	b, _ := json.Marshal(msg)
	ctx := context.Background()

	m.esClient.Index().Index("messages").Type("message").BodyString(string(b)).Do(ctx)
}
