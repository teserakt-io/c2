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

package protocols

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	gomock "github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/models"
	e4 "github.com/teserakt-io/e4go"
)

func TestKafkaPubSubClient(t *testing.T) {
	if os.Getenv("C2TEST_KAFKA") == "" {
		t.Skip("C2TEST_KAFKA environment variable isn't set, skipping postgres tests")
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMonitor := analytics.NewMockMessageMonitor(mockCtrl)

	logger := log.New()
	logger.SetOutput(ioutil.Discard)

	cfg := config.KafkaCfg{
		Brokers: []string{"127.0.0.1:9092"},
	}

	t.Run("Connect initialize the client properly", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)

		if kafkaClient.connected {
			t.Errorf("Expected client to be disconnected")
		}
		if kafkaClient.consumer != nil {
			t.Errorf("Expected nil consumer, got %#v", kafkaClient.consumer)
		}
		if kafkaClient.producer != nil {
			t.Errorf("Expected nil producer, got %#v", kafkaClient.producer)
		}

		if err := kafkaClient.Connect(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !kafkaClient.connected {
			t.Errorf("Expected client to be connected")
		}
		if kafkaClient.consumer == nil {
			t.Errorf("Expected not nil consumer")
		}
		if kafkaClient.producer == nil {
			t.Errorf("Expected not nil producer")
		}
		if kafkaClient.subscribedTopics == nil {
			t.Errorf("Expected not nil subscribedTopics")
		}
	})

	t.Run("Connect returns already connected error if called twice", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)

		if err := kafkaClient.Connect(); err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}

		if err := kafkaClient.Connect(); err != ErrAlreadyConnected {
			t.Errorf("Expected error to be %v, got %v", ErrAlreadyConnected, err)
		}
	})

	t.Run("Disconnect properly close the client", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)

		stopChan := make(chan bool)
		kafkaClient.subscribedTopics["foo"] = stopChan

		if err := kafkaClient.Connect(); err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}

		if err := kafkaClient.Disconnect(); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		select {
		case <-stopChan:
		case <-time.After(10 * time.Millisecond):
			t.Errorf("Expected stopchan to be closed, got timeout")
		}

		if kafkaClient.connected {
			t.Errorf("Expected client to be disconnected")
		}
		if kafkaClient.consumer != nil {
			t.Errorf("Expected nil consumer, got %#v", kafkaClient.consumer)
		}
		if kafkaClient.producer != nil {
			t.Errorf("Expected nil producer, got %#v", kafkaClient.producer)
		}
		if len(kafkaClient.subscribedTopics) > 0 {
			t.Errorf("Expected empty subscribedTopics, got %#v", kafkaClient.subscribedTopics)
		}
	})

	t.Run("Disconnected a not connected client returns a not connected error", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)
		if err := kafkaClient.Disconnect(); err != ErrNotConnected {
			t.Errorf("Expected error to be %v, got %v", ErrNotConnected, err)
		}
	})

	t.Run("Publish and Subscribe properly creates and listen for messages", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)
		if err := kafkaClient.Connect(); err != nil {
			t.Fatalf("failed to connect client")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		client1 := models.Client{E4ID: []byte("client1")}
		client2 := models.Client{E4ID: []byte("client2")}
		client3 := models.Client{E4ID: []byte("client3")}

		expectedTopic1 := e4.TopicForID(client1.E4ID)
		expectedTopic2 := e4.TopicForID(client2.E4ID)
		expectedTopic3 := e4.TopicForID(client3.E4ID)

		expectedMessage1 := []byte("message_1")
		expectedMessage2 := []byte("message_2")
		expectedMessage3 := []byte("message_3")
		expectedMessage4 := []byte("message_4")

		if err := kafkaClient.SubscribeToTopic(ctx, expectedTopic1); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if err := kafkaClient.SubscribeToTopics(ctx, []string{expectedTopic2, expectedTopic3}); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedLoggedMessage1 := analytics.LoggedMessage{
			Topic:   expectedTopic1,
			Payload: expectedMessage1,
			IsUTF8:  true,
		}
		expectedLoggedMessage2 := analytics.LoggedMessage{
			Topic:   expectedTopic2,
			Payload: expectedMessage2,
			IsUTF8:  true,
		}
		expectedLoggedMessage3 := analytics.LoggedMessage{
			Topic:   expectedTopic3,
			Payload: expectedMessage3,
			IsUTF8:  true,
		}
		expectedLoggedMessage4 := analytics.LoggedMessage{
			Topic:   expectedTopic3,
			Payload: expectedMessage4,
			IsUTF8:  true,
		}

		mockMonitor.EXPECT().OnMessage(gomock.Any(), expectedLoggedMessage1)
		mockMonitor.EXPECT().OnMessage(gomock.Any(), expectedLoggedMessage2)
		mockMonitor.EXPECT().OnMessage(gomock.Any(), expectedLoggedMessage3)
		mockMonitor.EXPECT().OnMessage(gomock.Any(), expectedLoggedMessage4)

		if err := kafkaClient.Publish(ctx, expectedMessage1, client1, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage2, client2, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage3, client3, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage4, client3, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}

		// gives some time for the messages to get consumed
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Subscribe / Unsubscribe properly add / remove topic subscription", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)
		if err := kafkaClient.Connect(); err != nil {
			t.Fatalf("failed to connect client")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "e4_topic1"

		if err := kafkaClient.SubscribeToTopic(ctx, expectedTopic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if _, exists := kafkaClient.subscribedTopics[expectedTopic]; !exists {
			t.Errorf("Expected topic to be in subscription list, got %#v", kafkaClient.subscribedTopics)
		}

		if err := kafkaClient.UnsubscribeFromTopic(ctx, expectedTopic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if _, exists := kafkaClient.subscribedTopics[expectedTopic]; exists {
			t.Errorf("Expected topic to have been removed from subscription list, got %#v", kafkaClient.subscribedTopics)
		}
	})

	t.Run("ValidateTopic properly filter out invalid topics", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)

		testData := map[string]error{
			"kafka_valid":    nil,
			"kafka-valid":    nil,
			"kafka.valid":    nil,
			"kafka.valid123": nil,
			"#":              ErrInvalidTopic,
			"kafka/invalid":  ErrInvalidTopic,
			"+":              ErrInvalidTopic,
			"kafka$":         ErrInvalidTopic,
		}

		for topic, want := range testData {
			got := kafkaClient.ValidateTopic(topic)
			if got != want {
				t.Errorf("got error '%v', want '%v' when validating topic %s", got, want, topic)
			}
		}
	})
}
