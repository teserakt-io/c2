package main

import (
	"bufio"
	"crypto/rand"
	b64 "encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"runtime/debug"
	e4c "teserakt/e4go/pkg/e4client"
	e4 "teserakt/e4go/pkg/e4common"
)

// variables set at build time
var gitCommit string
var buildDate string

func eventloop(errc chan error,
	controlc chan command,
	cli E4ProtectedClient) {

	for cmd := range controlc {
		switch cmd.Type {
		case CMDSETID:
			id := e4.HashIDAlias(cmd.Payload)
			copy(cli.E4.ID, id)
		case CMDSETKEY:
			key := []byte(cmd.Payload)
			cli.E4.SetIDKey(key)
		case CMDGENKEY:
			key := e4.RandomKey()
			cli.E4.SetIDKey(key)
		case CMDSUBTOPIC:
			topic := cmd.Payload
			cli.Proto.SubscribeTopic(topic)
		case CMDUNSUBTOPIC:
			topic := cmd.Payload
			cli.Proto.UnsubscribeTopic(topic)
		case CMDSENDE4PROTECTEDMSG:
			// Payload should be set to:
			// topic=?;payload=base64
			payloadparts := strings.Split(cmd.Payload, ";")
			if len(payloadparts) != 2 {
				continue
			}

			var topic string
			var payload []byte
			var err error

			for _, part := range payloadparts {
				subparts := strings.Split(part, "=")
				if len(subparts) != 2 {
					continue
				}
				key := subparts[0]
				value := subparts[1]
				switch key {
				case "topic":
					topic = value
				case "payload":
					payload, err = b64.StdEncoding.DecodeString(value)
					if err != nil {
						continue
					}
				default:
					continue
				}
			}

			e4protectedpayload, err := cli.E4.Protect(payload, topic)
			if err != nil {
				//
			}

			cli.Proto.SendMessageToTopic(topic, e4protectedpayload)

		case CMDSENDUNPROTECTEDMSG:

			payloadparts := strings.Split(cmd.Payload, ";")
			if len(payloadparts) != 2 {
				continue
			}

			var topic string
			var payload []byte
			var err error

			for _, part := range payloadparts {
				subparts := strings.Split(part, "=")
				if len(subparts) != 2 {
					continue
				}
				key := subparts[0]
				value := subparts[1]
				switch key {
				case "topic":
					topic = value
				case "payload":
					payload, err = b64.StdEncoding.DecodeString(value)
					if err != nil {
						continue
					}
				default:
					continue
				}
			}

			cli.Proto.SendMessageToTopic(topic, payload)

		default:

		}
	}
}

// ReceiveLoop receives and listens for events from the ProtoClient
// and reports events to the event channel as required.
func ReceiveLoop(errc chan error, cli E4ProtectedClient) {

	for incoming := range cli.recv {

		topic := incoming[0]
		payload := incoming[1]

		// E4: if topic is E4/<id>, the process as a command
		if topic == cli.E4.ReceivingTopic {
			cmd, err := cli.E4.ProcessCommand([]byte(payload))
			if err != nil {

				evt := event{Code: EVTERROR,
					Properties: map[string]interface{}{
						"context": "ProcessCommand",
						"message": err.Error(),
					}}
				evt.Report()
			} else {
				evt := event{Code: EVTE4COMMANDRECEIVED,
					Properties: map[string]interface{}{
						"command": cmd,
					}}
				evt.Report()
			}
		} else {
			// E4: attempt to decrypt
			message, err := cli.E4.Unprotect([]byte(payload), topic)
			if err == nil {

				evt := event{Code: EVTE4MSGRECEIVED,
					Properties: map[string]interface{}{
						"topic":   topic,
						"message": message,
					}}
				evt.Report()
			} else if err == e4c.ErrTopicKeyNotFound {

				evt := event{Code: EVTINSECUREMSGRECEIVED,
					Properties: map[string]interface{}{
						"topic":   topic,
						"message": message,
					}}
				evt.Report()
			} else {
				evt := event{Code: EVTERROR,
					Properties: map[string]interface{}{
						"context": "Unproect",
						"message": err.Error(),
					}}
				evt.Report()
			}
		}
	}
}

func readStdin(errc chan error, controlc chan command) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		stdintext := scanner.Text()
		var cmd command
		if err := json.Unmarshal([]byte(stdintext), &cmd); err != nil {
			continue
		}

		controlc <- cmd
	}
}

/*
func grpcreader() {

}
*/

func generateClientFileName() (string, error) {
	bytes := [28]byte{}
	_, err := rand.Read(bytes[:])
	if err != nil {
		return "", err
	}
	tCandidate := b64.StdEncoding.EncodeToString(bytes[:])
	tCleaned1 := strings.Replace(tCandidate, "+", "", -1)
	tCleaned2 := strings.Replace(tCleaned1, "/", "", -1)
	tCleaned3 := strings.Replace(tCleaned2, "=", "", -1)

	dbPath := fmt.Sprintf("/tmp/e4storage_%s", tCleaned3[0:32])
	return dbPath, nil
}

// All output that is non-fatal should be encoded using an event.Report()
// All critical errors should be reported to os.Stderr and change exitCode
func main() {

	var exitCode = 0
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "Panic: %s\n", r)
			debug.PrintStack()
			exitCode = 1
		}
		os.Exit(exitCode)
	}()

	evt := event{Code: EVTVERSION,
		Properties: map[string]interface{}{
			"GitCommit": gitCommit,
			"BuildDate": buildDate,
		}}
	evt.Report()

	var err error
	var e4clientid string
	var e4clientkey string
	var e4storagepath string
	//var protobroker string

	// TODO replace with spf13/cobra.
	clientid := flag.String("id", "", "The Client ID Alias for this client")
	clientkey := flag.String("key", "", "A password from which the client key will be derived")
	clientfilepath := flag.String("e4storage", "", "Location for the E4 client storage file")
	broker := flag.String("broker", "tcp://127.0.0.1:1883", "Specify Broker server")

	flag.Parse()

	if clientid == nil || clientkey == nil || broker == nil {
		fmt.Fprintf(os.Stderr, "Client id, key or broker not specified\n")
		exitCode = 1
		return
	}

	if *clientid == "" ||
		*clientkey == "" {
		fmt.Fprintf(os.Stderr, "Client id and/or key not specified\n")
		exitCode = 1
		return
	}

	e4clientid = *clientid
	e4clientkey = *clientkey

	if clientfilepath == nil {
		e4storagepath, err = generateClientFileName()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Client id and/or key not specified\n")
			exitCode = 1
			return
		}
	} else {
		e4storagepath = *clientfilepath
	}

	errc := make(chan error)
	controlc := make(chan command)
	var e4cli E4ProtectedClient

	go func() {
		var signalc = make(chan os.Signal, 1)
		signal.Notify(signalc, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-signalc)
	}()

	e4cli.E4 = e4c.NewClientPretty(e4clientid, e4clientkey, e4storagepath)
	e4cli.recv = make(chan [2]string)
	e4cli.Proto = MqttTransport{}

	// TODO: at some point we should be able to specify MQTT client
	// specific options as part of a client command. See cobra and
	// or viper.
	configmap := make(map[string]interface{})
	configmap["broker"] = *broker
	configmap["clientid"] = ""
	configmap["username"] = ""
	configmap["password"] = ""
	configmap["cleansession"] = true
	configmap["store"] = ":memory:"

	if err := e4cli.Proto.Initialize(e4cli.recv, e4cli.E4.ReceivingTopic, configmap); err != nil {
		return
	}

	go eventloop(errc, controlc, e4cli)
	go readStdin(errc, controlc)
	go ReceiveLoop(errc, e4cli)

	err = <-errc
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s", err)
		exitCode = 1
	}
}
