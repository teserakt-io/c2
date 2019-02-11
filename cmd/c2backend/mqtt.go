package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
)

func (s *C2) createMQTTClient() error {

	// TODO: secure connection to broker
	logger := log.With(c2.logger, "protocol", "mqtt")
	logger.Log("addr", mqttBroker)
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(mqttBroker)
	mqOpts.SetClientID(mqttID)
	mqOpts.SetPassword(mqttPassword)
	mqOpts.SetUsername(mqttUsername)
	mqttClient := mqtt.NewClient(mqOpts)
	logger.Log("msg", "mqtt parameters", "broker", mqttBroker, "id", mqttID, "username", mqttUsername)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Log("msg", "connection failed", "error", token.Error())
		return
	}
	logger.Log("msg", "connected to broker")
	// instantiate C2
	c2.mqttClient = mqttClient
}
