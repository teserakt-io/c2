package protocols

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"

	"github.com/go-kit/kit/log"
	gomock "github.com/golang/mock/gomock"
)

func TestKafkaPubSubClient(t *testing.T) {
	if os.Getenv("C2TEST_KAFKA") == "" {
		t.Skip("C2TEST_KAFKA environment variable isn't set, skipping postgress tests")
	}

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMonitor := analytics.NewMockMessageMonitor(mockCtrl)

	logger := log.NewNopLogger()

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
			t.Errorf("Epected no error, got %v", err)
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

		expectedTopic1 := "topic1"
		expectedTopic2 := "topic2"
		expectedTopic3 := "topic3"

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

		mockMonitor.EXPECT().OnMessage(ctx, expectedLoggedMessage1)
		mockMonitor.EXPECT().OnMessage(ctx, expectedLoggedMessage2)
		mockMonitor.EXPECT().OnMessage(ctx, expectedLoggedMessage3)
		mockMonitor.EXPECT().OnMessage(ctx, expectedLoggedMessage4)

		if err := kafkaClient.Publish(ctx, expectedMessage1, expectedTopic1, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage2, expectedTopic2, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage3, expectedTopic3, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}
		if err := kafkaClient.Publish(ctx, expectedMessage4, expectedTopic3, byte(0)); err != nil {
			t.Errorf("failed to publish, expected no error, got %v", err)
		}

		// gives some time for the messages to get consumed
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Publish / Subscribe handles '/' character in topic names", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)
		if err := kafkaClient.Connect(); err != nil {
			t.Fatalf("failed to connect client")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "e4/topicWithSlash"
		if err := kafkaClient.SubscribeToTopic(ctx, expectedTopic); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		expectedMessage := []byte("message")

		expectedLoggedMessage := analytics.LoggedMessage{
			Topic:   "e4-topicWithSlash",
			Payload: expectedMessage,
			IsUTF8:  true,
		}
		mockMonitor.EXPECT().OnMessage(ctx, expectedLoggedMessage)

		if err := kafkaClient.Publish(ctx, expectedMessage, expectedTopic, byte(0)); err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// gives some time for the message to get consumed
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("Subscribe / Unsubscribe properly add / remove topic subscription", func(t *testing.T) {
		kafkaClient := NewKafkaPubSubClient(cfg, logger, mockMonitor).(*kafkaPubSubClient)
		if err := kafkaClient.Connect(); err != nil {
			t.Fatalf("failed to connect client")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "e4/topic1"

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
}
