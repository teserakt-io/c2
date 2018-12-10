package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"

	e4 "gitlab.com/teserakt/e4common"
)

/* This class as with any client class should implement the
 * ProtocolClient interface described in main.go
 */
type MqttClient struct {
	logger     log.Logger
	mqttClient mqtt.Client
}

func (m *MqttClient) initialize(log log.Logger,
	//recv chan [2]string, controlchannel string,
	config map[string]interface{}) {

	m.logger = log

	mqttBroker := config["mqttBroker"].(string)
	mqttID := config["mqttID"].(string)
	mqttPassword := config["mqttPassword"].(string)
	mqttUsername := config["mqttUsername"].(string)

	log.Log("addr", mqttBroker)
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(mqttBroker)
	mqOpts.SetClientID(mqttID)
	mqOpts.SetPassword(mqttPassword)
	mqOpts.SetUsername(mqttUsername)

	mqttClient := mqtt.NewClient(mqOpts)
	log.Log("msg", "mqtt parameters", "broker", mqttBroker, "id", mqttID, "username", mqttUsername)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Log("msg", "connection failed", "error", token.Error())
		return
	}
	log.Log("msg", "connected to broker")
	// instantiate C2
}

func (m *MqttClient) publish(payload []byte, topic string, qos byte) error {

	payloadstring := string(payload)

	logger := log.With(m.logger, "protocol", "mqtt")

	if token := m.mqttClient.Publish(topic, qos, true, payloadstring); token.Wait() && token.Error() != nil {
		logger.Log("msg", "publish failed", "topic", topic, "error", token.Error())
		return token.Error()
	}
	logger.Log("msg", "publish succeeded", "topic", topic)

	return nil
}

func (m *MqttClient) sendCommandToClient(id, payload []byte) error {

	topic := e4.TopicForID(id)
	qos := byte(2)

	return m.publish(payload, topic, qos)
}
