package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"
)

type startMQTTClientConfig struct {
	addr     string
	id       string
	password string
	username string
}

func (s *C2) createMQTTClient(scfg *startMQTTClientConfig) error {

	// TODO: secure connection to broker
	logger := log.With(s.logger, "protocol", "mqtt")
	logger.Log("addr", scfg.addr)
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(scfg.addr)
	mqOpts.SetClientID(scfg.id)
	mqOpts.SetPassword(scfg.password)
	mqOpts.SetUsername(scfg.username)
	mqttClient := mqtt.NewClient(mqOpts)
	logger.Log("msg", "mqtt parameters", "broker", mqttBroker, "id", mqttID, "username", mqttUsername)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		logger.Log("msg", "connection failed", "error", token.Error())
		return
	}
	logger.Log("msg", "connected to broker")
	// instantiate C2
	s.mqttClient = mqttClient
}
