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
	"errors"
	"io/ioutil"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"

	"github.com/teserakt-io/c2/internal/analytics"
	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/models"
	e4 "github.com/teserakt-io/e4go"
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

	logger := log.New()
	logger.SetOutput(ioutil.Discard)

	pubSubClient := &mqttPubSubClient{
		mqtt:              mockMQTTClient,
		config:            config,
		logger:            logger,
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

		client := models.Client{E4ID: []byte("client1")}
		expectedTopic := e4.TopicForID(client.E4ID)

		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)
		mockToken.EXPECT().Error().Return(nil)

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, client, expectedQos)
		if err != nil {
			t.Errorf("Expected error to be nil, got %v", err)
		}
	})

	t.Run("Publish properly handles broker timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedPayload := []byte("payload")

		client := models.Client{E4ID: []byte("client1")}
		expectedTopic := e4.TopicForID(client.E4ID)

		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(false)

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, client, expectedQos)
		if err != ErrMQTTTimeout {
			t.Errorf("Expected error to be %v, got %v", ErrMQTTTimeout, err)
		}
	})

	t.Run("Publish properly handles token error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		expectedPayload := []byte("payload")

		client := models.Client{E4ID: []byte("client1")}
		expectedTopic := e4.TopicForID(client.E4ID)

		expectedQos := QoSExactlyOnce

		mockToken := NewMockMQTTToken(mockCtrl)
		mockToken.EXPECT().WaitTimeout(expectedTimeout).Return(true)

		expectedError := errors.New("token-error")
		mockToken.EXPECT().Error().Return(expectedError).AnyTimes()

		mockMQTTClient.EXPECT().Publish(expectedTopic, expectedQos, true, string(expectedPayload)).Return(mockToken)

		err := pubSubClient.Publish(ctx, expectedPayload, client, expectedQos)
		if err != expectedError {
			t.Errorf("Expected error to be %v, got %v", expectedError, err)
		}
	})

	t.Run("ValidateTopic properly filter out invalid topics", func(t *testing.T) {
		testData := map[string]error{
			"/mqtt/topic": nil,
			"mqttTopic":   nil,
			"mqtt/$SYS":   nil,
			"mqtt/123":    nil,
			"#":           ErrInvalidTopic,
			"/mqtt/#":     ErrInvalidTopic,
			"/mqtt/+":     ErrInvalidTopic,
			"+":           ErrInvalidTopic,
			"$SYS":        ErrInvalidTopic,
			"$SYS/foo":    ErrInvalidTopic,
		}

		for topic, want := range testData {
			got := pubSubClient.ValidateTopic(topic)
			if got != want {
				t.Errorf("got error '%v', want '%v' when validating topic %s", got, want, topic)
			}
		}
	})
}
