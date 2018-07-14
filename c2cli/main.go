package main

import (
	"encoding/hex"
	"errors"
	"strings"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/spf13/pflag"
	"github.com/abiosoft/ishell"

	pb "teserakt/c2proto"
	e4 "teserakt/e4common"
)

func main() {

	log.SetPrefix("c2cli\t\t")

	command := pflag.StringP("command", "c", "", "command type (required)")
	idalias := pflag.StringP("id", "i", "", "a client id alias, a UTF-8 string")
	keyhex := pflag.StringP("key", "k", "", "a 512-bit key, an hex string")
	pwd := pflag.StringP("pwd", "p", "", "a passphrase to derive a key from")
	top := pflag.StringP("topic", "t", "", "a topic, as UTF-8 string")
	c2 := pflag.StringP("c2", "h", "localhost:5555", "C2 host address")
	m := pflag.StringP("msg", "m", "", "message to send")
	inter := pflag.BoolP("shell", "s", false, "interactive shell")

	pflag.Parse()

	var id []byte
	var key []byte
	var err error
	var topic string
	var msg string

	conn, err := grpc.Dial(*c2, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	defer conn.Close()
	client := pb.NewC2Client(conn)

	if *inter {
		shell := ishell.New()
		shell.Println("Welcome to E4 C2CLI")

	    shell.AddCmd(&ishell.Cmd{
			Name: "nc",
			Help: "new client (nc client pwd)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 2 {
					c.Println("command failed: expecting 2 arguments")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				key = e4.HashPwd(c.Args[1])
				err := sendCommand(client, pb.C2Request_NEW_CLIENT, id, key, "", "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})	

		shell.AddCmd(&ishell.Cmd{
			Name: "rc",
			Help: "remove client (rc client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				err := sendCommand(client, pb.C2Request_NEW_CLIENT, id, nil, "", "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "ntc",
			Help: "new topic client (ntc client topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 2 {
					c.Println("command failed: expecting 2 arguments")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				topic = c.Args[1]
				err := sendCommand(client, pb.C2Request_NEW_TOPIC_CLIENT, id, nil, topic, "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rtc",
			Help: "remove topic client (rtc client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 2 {
					c.Println("command failed: expecting 2 arguments")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				topic = c.Args[1]
				err := sendCommand(client, pb.C2Request_REMOVE_TOPIC_CLIENT, id, nil, topic, "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rsc",
			Help: "reset client (rsc client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				err := sendCommand(client, pb.C2Request_RESET_CLIENT, id, nil, "", "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "nt",
			Help: "new topic (nt topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				topic = c.Args[0]
				err := sendCommand(client, pb.C2Request_NEW_TOPIC, nil, nil, topic, "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rt",
			Help: "remove topic (rt topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				topic = c.Args[0]
				err := sendCommand(client, pb.C2Request_REMOVE_TOPIC, nil, nil, topic, "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})
	
		shell.AddCmd(&ishell.Cmd{
			Name: "nck",
			Help: "new client key (nck client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				err := sendCommand(client, pb.C2Request_NEW_CLIENT_KEY, id, nil, "", "")
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "sm",
			Help: "send message (send client topic message)",
			Func: func(c *ishell.Context) {
				if len(c.Args) < 2 {
					c.Println("command failed: expecting 2+ arguments")
					return
				}
				topic = c.Args[0]
				msg = strings.Join(c.Args[1:], " ")
				err := sendCommand(client, pb.C2Request_SEND_MESSAGE, nil, nil, topic, msg)
				if err != nil {
					c.Println("command failed: ", err)
				} else {
					c.Println("command sent")
				}
			},
		})

		shell.Run()
	}

	if *command == "" {
		log.Fatal("missing command")
	}

	if *idalias != "" {
		id = e4.HashIDAlias(*idalias)
	}

	if *keyhex != "" {
		if *pwd != "" {
			log.Fatal("choose between key and password")
		}
		key, err = hex.DecodeString(*keyhex)
		if len(key) != e4.KeyLen {
			log.Fatalf("incorrect key size: %d bytes, expected %d", len(key), e4.KeyLen)
		}
		if err != nil {
			log.Fatalf("key decoding failed: %s", err)
		}
	}

	if *pwd != "" {
		key = e4.HashPwd(*pwd)
	}

	topic = *top
	msg = *m

	commandcode, err := commandToPbCode(*command)
	if err != nil {
		log.Fatalf(err.Error())
	}

	sendCommand(client, commandcode, id, key, topic, msg)

}


func commandToPbCode(command string) (pb.C2Request_Command, error) {

	switch command {
	case "nc":
		return pb.C2Request_NEW_CLIENT, nil
	case "rc":
		return pb.C2Request_REMOVE_CLIENT, nil
	case "ntc":
		return pb.C2Request_NEW_TOPIC_CLIENT, nil
	case "rtc":
		return pb.C2Request_REMOVE_TOPIC_CLIENT, nil
	case "rsc":
		return pb.C2Request_RESET_CLIENT, nil
	case "nt":
		return pb.C2Request_NEW_TOPIC, nil
	case "rt":
		return pb.C2Request_REMOVE_TOPIC, nil
	case "nck":
		return pb.C2Request_NEW_CLIENT_KEY, nil
	case "sm":
		return pb.C2Request_SEND_MESSAGE, nil
	default:
		return -1, errors.New("invalid command")
	}
}

// send command with given type, id, key, and topic
func sendCommand(client pb.C2Client, commandcode pb.C2Request_Command, id, key []byte, topic, msg string) error {

	req := &pb.C2Request{
		Command: commandcode,
		Id:      id,
		Key:     key,
		Topic:   topic,
		Msg: 	msg,
	}

	resp, err := client.C2Command(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Success {
		return nil
	} else {
		return errors.New(resp.Err)
	}
	return nil
}