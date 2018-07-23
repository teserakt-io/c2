package main

import (
	"encoding/hex"
	"errors"
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/abiosoft/ishell"
	"github.com/spf13/pflag"

	pb "teserakt/e4go/pkg/c2proto"
	e4 "teserakt/e4go/pkg/e4common"
)

// variables set at build time
var gitCommit string
var buildDate string

func main() {

	log.SetFlags(0)

	fs := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	c2 := fs.String("c2", "localhost:5555", "C2 host address")
	command := fs.StringP("command", "c", "", "a command type (if not in shell)")
	idalias := fs.StringP("id", "i", "", "client id alias as a UTF-8 string")
	keyhex := fs.StringP("key", "k", "", "512-bit key as an hex string")
	pwd := fs.StringP("pwd", "p", "", "password to derive a key from")
	top := fs.StringP("topic", "t", "", "topic as a UTF-8 string")
	m := fs.StringP("msg", "m", "", "message to send")
	help := fs.BoolP("help", "h", false, "same")

	fs.Parse(os.Args[1:])
	fs.SortFlags = false

	if *help {
		fs.PrintDefaults()
		return
	}

	var id []byte
	var key []byte
	var err error
	var topic string
	var msg string

	conn, err := grpc.Dial(*c2, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to gRPC server: %v", err)
	}

	defer conn.Close()
	client := pb.NewC2Client(conn)

	if *command == "" {
		shell := ishell.New()
		shell.SetPrompt("➩ ")
		shell.Println("    /---------------------------------/")
		shell.Println("   /  E4: C2 command-line interface  /")
		shell.Printf("  /  version %s-%s          /\n", buildDate, gitCommit[:4])
		shell.Println(" /  Teserakt AG, 2018              /")
		shell.Println("/---------------------------------/\n")
		shell.Println("type 'help' for help (duh)\n")

		shell.AddCmd(&ishell.Cmd{
			Name: "c2",
			Help: "set C2 host",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
			},
		})

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
					c.Println("command succeeded")
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

		log.Println("bye!")
		return
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

	err = sendCommand(client, commandcode, id, key, topic, msg)
	if err != nil {
		log.Fatalf(err.Error())
	}
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
		Msg:     msg,
	}

	resp, err := client.C2Command(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Success {
		return nil
	}
	return errors.New(resp.Err)
}
