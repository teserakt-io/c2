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

package analytics

//go:generate mockgen -copyright_file ../../doc/COPYRIGHT_TEMPLATE.txt  -destination=monitor_mocks.go -package analytics -self_package github.com/teserakt-io/c2/internal/analytics github.com/teserakt-io/c2/internal/analytics MessageMonitor

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
