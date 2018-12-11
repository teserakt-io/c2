package main

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/go-kit/kit/log"

	e4 "gitlab.com/teserakt/e4common"
	//gosmrf "gitlab.com/teserakt/smrf"
)

/* This class as with any client class should implement the
 * ProtocolClient interface described in main.go
 */
type JoynrClient struct {
	logger     log.Logger
	mqttClient mqtt.Client
}

func (j *JoynrClient) initialize(log log.Logger,
	//recv chan [2]string, controlchannel string,
	config map[string]interface{}) {

	j.logger = log

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

	j.mqttClient = mqtt.NewClient(mqOpts)
	log.Log("msg", "mqtt parameters", "broker", mqttBroker, "id", mqttID, "username", mqttUsername)
	if token := j.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Log("msg", "connection failed", "error", token.Error())
		return
	}
	log.Log("msg", "connected to broker")
	// instantiate C2
}

func (j *JoynrClient) publish(payload []byte, topic string, qos byte) error {

	return nil
}

func (j *JoynrClient) sendCommandToClient(id, payload []byte) error {

	topic := e4.TopicForID(id)
	qos := byte(2)

	return j.publish(payload, topic, qos)
}
