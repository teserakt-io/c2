package main

import (
	MQTT "github.com/eclipse/paho.mqtt.golang"
)

/*

In main.go we define protoclient like this:
type ProtoClient interface {
	SubscribeTopic(topic string)
	UnsubscribeTopic(topic string)
	SendMessageToTopic(topic string, payload []byte)
	Initialize() error
}
*/

// MqttTransport A struct to implement ProtoClient interface, containing MQTT.Client
type MqttTransport struct {
	client MQTT.Client
}

// SubscribeTopic subscribes to an MQTT topic with qos 0
func (t MqttTransport) SubscribeTopic(topic string) error {
	qos := 0
	token := t.client.Subscribe(topic, byte(qos), nil)
	token.Wait()
	return token.Error()
}

// UnsubscribeTopic unsubscribes from an MQTT topic
func (t MqttTransport) UnsubscribeTopic(topic string) error {
	token := t.client.Unsubscribe(topic)
	token.Wait()
	return token.Error()
}

// SendMessageToTopic sends a message to a given topic
func (t MqttTransport) SendMessageToTopic(topic string, encryptedpayload []byte) error {
	token := t.client.Publish(topic, 0, false, encryptedpayload)
	token.Wait()
	return token.Error()
}

// Initialize constructs the MqttTransport object by initializing the
// underlying MQTT.Client with optional config parameters. The recv
// channel should receive all events from the ProtoClient and the
// controlchannel string indicates what topic or client should be treated
// as sending E4 control messages.
func (t MqttTransport) Initialize(recv chan [2]string, controlchannel string,
	config map[string]interface{}) error {

	opts := MQTT.NewClientOptions()
	opts.AddBroker(config["broker"].(string))
	opts.SetClientID(config["clientid"].(string))
	opts.SetUsername(config["username"].(string))
	opts.SetPassword(config["password"].(string))
	opts.SetCleanSession(config["cleansession"].(bool))

	store := config["store"].(string)
	if store != ":memory:" {
		opts.SetStore(MQTT.NewFileStore(store))
	}

	opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		recv <- [2]string{msg.Topic(), string(msg.Payload())}
	})

	t.client = MQTT.NewClient(opts)
	if token := t.client.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	if token := t.client.Subscribe(controlchannel, byte(2), nil); token.Wait() && token.Error() != nil {
		return token.Error()
	}

	return nil
}
