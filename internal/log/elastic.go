package log

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/olivere/elastic"
)

type elasticLogger struct {
	logger   log.Logger
	esClient *elastic.Client
	index    string
}

// WithElasticSearch wrap around a gokit logger to make it forward
// logs to elasticSearch on given index
func WithElasticSearch(logger log.Logger, esClient *elastic.Client, index string) (log.Logger, error) {
	return &elasticLogger{
		logger:   logger,
		esClient: esClient,
		index:    index,
	}, nil
}

// Log starts a goroutine responsible of publishing the logged keyvals to elasticsearch.
// It then calls the wrapped logger if provided.
func (l *elasticLogger) Log(keyvals ...interface{}) error {
	go func() {
		buf := bytes.NewBuffer(nil)
		jsonLogger := log.NewJSONLogger(buf)
		jsonLogger = log.With(jsonLogger, "@timestamp", log.DefaultTimestamp)

		if err := jsonLogger.Log(keyvals...); err != nil {
			l.logger.Log("msg", "failed to log keyvals to buffer", "error", err, "data", keyvals)
		}

		index := fmt.Sprintf("%s-%s", l.index, time.Now().Format("2006.01.02"))
		_, err := l.esClient.Index().Index(index).Type("log").
			BodyString(string(buf.Bytes())).
			Refresh("true").
			Do(context.Background())

		if err != nil {
			l.logger.Log("msg", "failed to log to elasticsearch", "error", err, "data", keyvals)
			return
		}
	}()

	// Default gokit logger caller is fetched from 3 levels deep in callstack
	// we need 2 more to keep proper caller displaying.
	logger := log.With(l.logger, "caller", log.Caller(5))
	if err := logger.Log(keyvals...); err != nil {
		return err
	}

	return nil
}
