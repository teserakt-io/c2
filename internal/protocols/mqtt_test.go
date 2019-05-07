package protocols

import (
	"errors"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/golang/mock/gomock"

	"gitlab.com/teserakt/c2/internal/analytics"
	"gitlab.com/teserakt/c2/internal/config"
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

	newMQTTPubSubClient := func() *mqttPubSubClient {
		return &mqttPubSubClient{
			mqtt:              mockMQTTClient,
			config:            config,
			logger:            log.NewNopLogger(),
			monitor:           mockMonitor,
			waitTimeout:       expectedTimeout,
			disconnectTimeout: expectedDisconnectTimeout,
		}
	}

	t.Run("Connect properly calls the MQTT library and handles the token", func(t *testing.T) {
		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Connect().Return(mockToken)

		pubSubClient := newMQTTPubSubClient()
		err := pubSubClient.Connect()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("Connect properly handle connection timeout", func(t *testing.T) {
		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Connect().Return(mockToken)

		pubSubClient := newMQTTPubSubClient()
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

		pubSubClient := newMQTTPubSubClient()
		err := pubSubClient.Connect()
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("Disconnect properly calls MQTT lib with proper timeout", func(t *testing.T) {
		mockMQTTClient.EXPECT().Disconnect(expectedDisconnectTimeout)

		pubSubClient := newMQTTPubSubClient()
		err := pubSubClient.Disconnect()
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})
}
