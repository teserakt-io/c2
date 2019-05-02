package analytics

import (
	"context"
	"fmt"

	"github.com/olivere/elastic"
	"gitlab.com/teserakt/c2/internal/config"
)

// NewElasticClient creates a new ElasticSearch client
// that is connected to the URL specified in cfg. If the
// "messages" index doesn't exist, it will try to create it.
func NewElasticClient(cfg config.ESCfg) (*elastic.Client, error) {
	if !cfg.Enable {
		return nil, nil
	}

	esClient, err := elastic.NewClient(
		elastic.SetURL(cfg.URL),
		elastic.SetSniff(false),
	)

	if err != nil {
		return nil, err
	}

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

	return esClient, nil
}
