package main

import (
	"encoding/json"
	"fmt"
	e4c "teserakt/e4go/pkg/e4client"
)

// ClientCommand issues commands to the client over the command channel
type ClientCommand uint16

// EventCode represents an event type
type EventCode uint16

// Constants for client commands  we want to support
const (
	CMDSETID              ClientCommand = 0
	CMDSETKEY             ClientCommand = 1
	CMDGENKEY             ClientCommand = 2
	CMDSUBTOPIC           ClientCommand = 3
	CMDUNSUBTOPIC         ClientCommand = 4
	CMDSENDE4PROTECTEDMSG ClientCommand = 5
	CMDSENDUNPROTECTEDMSG ClientCommand = 6
)

// These EVT constants represent possible error codes from a
// controlled client.
const (
	EVTVERSION             EventCode = 0
	EVTERROR               EventCode = 1
	EVTE4COMMANDRECEIVED   EventCode = 2
	EVTE4MSGRECEIVED       EventCode = 3
	EVTINSECUREMSGRECEIVED EventCode = 4
)

// Command represents a {"Type": int, "Payload": "..."} command
// that can be inputted via stdin and control the client.
type command struct {
	Type    ClientCommand
	Payload string
}

// Event objects are reported as {"Code": int, "Properties": {}} where {} is an
// embedded json map (it can, but shouldn't, encode multiple levels of objects).
type event struct {
	Code       EventCode
	Properties map[string]interface{}
}

// ProtoClient implements methods of the protocol t be tested.
type ProtoClient interface {
	SubscribeTopic(topic string) error
	UnsubscribeTopic(topic string) error
	SendMessageToTopic(topic string, payload []byte) error
	Initialize(recv chan [2]string, controlchannel string,
		config map[string]interface{}) error
}

// E4ProtectedClient combines the E4 implementation detail from Golang
// with the ProtoClient implementation to be tested
type E4ProtectedClient struct {
	E4    *e4c.Client
	Proto ProtoClient
	recv  chan [2]string
}

func (c command) Encode() ([]byte, error) {
	return json.Marshal(c)
}

func (e event) Encode() ([]byte, error) {
	return json.Marshal(e)
}

func (e event) Report() {
	encoded, err := e.Encode()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", string(encoded))
}
