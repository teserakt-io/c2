package main

import (
	b64 "encoding/base64"
	"fmt"
	"strings"
)

// TODO: this is a bit quick and dirty. Maybe need to improve.
func parseHumanCmdLine(cmdline string) (*Command, error) {

	commandpayload := cmdline[2:]
	components := strings.Split(commandpayload, " ")

	numcomponents := len(components)

	command := HumanNameToCommandMap[components[0]]

	switch command {
	case CMDSETID:
		if numcomponents != 2 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		return &Command{Type: command, Payload: components[1]}, nil
	case CMDSETKEY:
		if numcomponents != 2 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		return &Command{Type: command, Payload: components[1]}, nil
	case CMDGENKEY:
		if numcomponents != 1 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		return &Command{Type: command, Payload: ""}, nil
	case CMDSUBTOPIC:
		if numcomponents != 2 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		return &Command{Type: command, Payload: components[1]}, nil
	case CMDUNSUBTOPIC:
		if numcomponents != 2 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		return &Command{Type: command, Payload: components[1]}, nil
	case CMDSENDE4PROTECTEDMSG:
		if numcomponents != 3 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		topic := components[1]
		message := b64.StdEncoding.EncodeToString([]byte(components[2]))
		payload := fmt.Sprintf("topic:%s;payload:%s", topic, message)
		return &Command{Type: command, Payload: payload}, nil
	case CMDSENDUNPROTECTEDMSG:
		if numcomponents != 3 {
			return nil, fmt.Errorf("Incorrect number of arguments, %d provided", numcomponents)
		}
		topic := components[1]
		message := b64.StdEncoding.EncodeToString([]byte(components[2]))
		payload := fmt.Sprintf("topic:%s;payload:%s", topic, message)
		return &Command{Type: command, Payload: payload}, nil
	default:
		return nil, fmt.Errorf("Unknown command %s", components[0])
	}
}
