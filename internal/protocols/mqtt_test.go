package protocols

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"
)

func TestMQTTPubSubClient(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockMQTTClient := NewMockMQTTClient(mockCtrl)
	mockMonitor := analytics.NewMockMessageMonitor(mockCtrl)

	config := config.MQTTCfg{
		ID:       "id",
		Broker:   "broker",
		QoSPub:   1,
		QoSSub:   2,
		Username: "username",
		Password: "password",
	}

	expectedTimeout := 10 * time.Millisecond
	expectedDisconnectTimeout := uint(1000)

	pubSubClient := &mqttPubSubClient{
		mqtt:              mockMQTTClient,
		config:            config,
		logger:            log.NewNopLogger(),
		monitor:           mockMonitor,
		waitTimeout:       expectedTimeout,
		disconnectTimeout: expectedDisconnectTimeout,
	}

	t.Run("Connect properly calls the MQTT library and handles the token", func(t *testing.T) {
		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Connect().Return(mockToken)

		err := pubSubClient.Connect()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Connect properly handle connection timeout", func(t *testing.T) {
		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Connect().Return(mockToken)

		err := pubSubClient.Connect()
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("Connect properly handle token errors", func(t *testing.T) {
		expectedError := errors.New("token-error")

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().Connect().Return(mockToken)

		err := pubSubClient.Connect()
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("Disconnect properly calls MQTT lib with proper timeout", func(t *testing.T) {
		mockMQTTClient.EXPECT().Disconnect(expectedDisconnectTimeout)

		err := pubSubClient.Disconnect()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("SubscribeToTopics does nothing when monitoring isn't enabled", func(t *testing.T) {
		mockMonitor.EXPECT().Enabled().Return(false)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pubSubClient.SubscribeToTopics(ctx, []string{"topic1", "topic2"})
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("SubscribeToTopics does nothing when no topics are provided", func(t *testing.T) {
		mockMonitor.EXPECT().Enabled().Return(true)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err := pubSubClient.SubscribeToTopics(ctx, []string{})
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("SubscribeToTopics properly subscribe to given topics", func(t *testing.T) {
		expectedTopics := []string{"topic1", "topic2"}
		expectedFilter := map[string]byte{
			"topic1": byte(config.QoSSub),
			"topic2": byte(config.QoSSub),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().SubscribeMultiple(expectedFilter, gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopics(ctx, expectedTopics)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("SubscribeToTopics handle broker timeout", func(t *testing.T) {
		expectedTopics := []string{"topic1", "topic2"}
		expectedFilter := map[string]byte{
			"topic1": byte(config.QoSSub),
			"topic2": byte(config.QoSSub),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().SubscribeMultiple(expectedFilter, gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopics(ctx, expectedTopics)
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("SubscribeToTopics handle token errors", func(t *testing.T) {
		expectedTopics := []string{"topic1", "topic2"}
		expectedFilter := map[string]byte{
			"topic1": byte(config.QoSSub),
			"topic2": byte(config.QoSSub),
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)

		expectedError := errors.New("token-error")
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().SubscribeMultiple(expectedFilter, gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopics(ctx, expectedTopics)
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("SubscribeToTopic don't do anything when monitoring isn't enabled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(false)

		err := pubSubClient.SubscribeToTopic(ctx, expectedTopic)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("SubscribeToTopic properly call subscribe", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Subscribe(expectedTopic, byte(config.QoSSub), gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopic(ctx, expectedTopic)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("SubscribeToTopic properly handle broker timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Subscribe(expectedTopic, byte(config.QoSSub), gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopic(ctx, expectedTopic)
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("SubscribeToTopic properly handle token error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)

		expectedError := errors.New("token-error")
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().Subscribe(expectedTopic, byte(config.QoSSub), gomock.Any()).Return(mockToken)

		err := pubSubClient.SubscribeToTopic(ctx, expectedTopic)
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("UnsubscribeFromTopic does nothing when monitoring isn't enabled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(false)

		err := pubSubClient.UnsubscribeFromTopic(ctx, expectedTopic)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("UnsubscribeFromTopic properly unsubscribe from broker", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Unsubscribe(expectedTopic).Return(mockToken)

		err := pubSubClient.UnsubscribeFromTopic(ctx, expectedTopic)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("UnsubscribeFromTopic properly handle broker timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Unsubscribe(expectedTopic).Return(mockToken)

		err := pubSubClient.UnsubscribeFromTopic(ctx, expectedTopic)
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("UnsubscribeFromTopic properly handle token error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedTopic := "topic1"

		mockMonitor.EXPECT().Enabled().Return(true)

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)

		expectedError := errors.New("token-error")
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().Unsubscribe(expectedTopic).Return(mockToken)

		err := pubSubClient.UnsubscribeFromTopic(ctx, expectedTopic)
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("Publish properly send message to broker", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedPayload := []byte("payload")
		expectedTopic := "topic1"
		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, expectedTopic, expectedQos)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("Publish properly handles broker timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedPayload := []byte("payload")
		expectedTopic := "topic1"
		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, expectedTopic, expectedQos)
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("Publish properly handles token error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedPayload := []byte("payload")
		expectedTopic := "topic1"
		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)

		expectedError := errors.New("token-error")
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, expectedTopic, expectedQos)
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})
}
