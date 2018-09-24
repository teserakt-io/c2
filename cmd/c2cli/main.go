package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"go/build"
	"log"
	"os"
<<<<<<< HEAD
	"path/filepath"
=======
	"strconv"
>>>>>>> 4fdf81e... Add commands to the shell client.
	"strings"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

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

	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = build.Default.GOPATH
	}
	// use os.Path

	certFile := "src/teserakt/e4go/cmd/c2backend/c2-cert.pem"
	certFile = filepath.FromSlash(certFile)
	//toJoin := []string{gopath, certFile}
	certPath := filepath.Join(gopath, certFile)
	fmt.Println(certPath)

	fs := pflag.NewFlagSet(os.Args[0], pflag.ExitOnError)

	c2 := fs.String("c2", "localhost:5555", "C2 host address")
	command := fs.StringP("command", "c", "", "a command type (if not in shell)")
	idalias := fs.StringP("id", "i", "", "client id alias as a UTF-8 string")
	keyhex := fs.StringP("key", "k", "", "512-bit key as an hex string")
	pwd := fs.StringP("pwd", "p", "", "password to derive a key from")
	top := fs.StringP("topic", "t", "", "topic as a UTF-8 string")
	cert := fs.StringP("cert", "e", certPath, "C2 certificate file")
	m := fs.StringP("msg", "m", "", "message to send")
	help := fs.BoolP("help", "h", false, "shows this")

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
	var offset uint64
	var count uint64

	creds, err := credentials.NewClientTLSFromFile(*cert, "")
	if err != nil {
		log.Fatalf("failed to create TLS credentials from %v: %v", *cert, err)
	}

	conn, err := grpc.Dial(*c2, grpc.WithTransportCredentials(creds))
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
			Help: "new client in C2 (nc client pwd|key)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 2 {
					c.Println("command failed: expecting 2 arguments")
					return
				}
				var err error
				id = e4.HashIDAlias(c.Args[0])
				if len(c.Args[1]) != 128 {
					key = e4.HashPwd(c.Args[1])
				} else {
					key, err = hex.DecodeString(c.Args[1])
					if err != nil {
						c.Println("command failed: invalid key")
					}
				}
				err = sendCommand(client, pb.C2Request_NEW_CLIENT, id, key, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rc",
			Help: "remove client from C2 (rc client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				err := sendCommand(client, pb.C2Request_REMOVE_CLIENT, id, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
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
				err := sendCommand(client, pb.C2Request_NEW_TOPIC_CLIENT, id, nil, topic, "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rtc",
			Help: "remove topic client (rtc client topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 2 {
					c.Println("command failed: expecting 2 arguments")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				topic = c.Args[1]
				err := sendCommand(client, pb.C2Request_REMOVE_TOPIC_CLIENT, id, nil, topic, "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
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
				err := sendCommand(client, pb.C2Request_RESET_CLIENT, id, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "nt",
			Help: "new topic in C2 (nt topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				topic = c.Args[0]
				err := sendCommand(client, pb.C2Request_NEW_TOPIC, nil, nil, topic, "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "rt",
			Help: "remove topic from C2 (rt topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: expecting 1 argument")
					return
				}
				topic = c.Args[0]
				err := sendCommand(client, pb.C2Request_REMOVE_TOPIC, nil, nil, topic, "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
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
				err := sendCommand(client, pb.C2Request_NEW_CLIENT_KEY, id, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "sm",
			Help: "send message (sm topic message)",
			Func: func(c *ishell.Context) {
				if len(c.Args) < 2 {
					c.Println("command failed: expecting 2+ arguments")
					return
				}
				topic = c.Args[0]
				msg = strings.Join(c.Args[1:], " ")
				err := sendCommand(client, pb.C2Request_SEND_MESSAGE, nil, nil, topic, msg, 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "lsc",
			Help: "list clients in C2",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 0 {
					c.Println("command failed: no argument expected")
					return
				}
				err := sendCommand(client, pb.C2Request_GET_CLIENTS, nil, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "lst",
			Help: "list topics in C2",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 0 {
					c.Println("command failed: no argument expected")
					return
				}
				err := sendCommand(client, pb.C2Request_GET_TOPICS, nil, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "hi",
			Help: "hashes client id alias to id",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: argument expected")
					return
				}
				hash := e4.HashIDAlias(c.Args[0])
				c.Println(hex.EncodeToString(hash))
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "hp",
			Help: "hashes password to key",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: argument expected")
					return
				}
				hash := e4.HashPwd(c.Args[0])
				c.Println(hex.EncodeToString(hash))
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "lctc",
			Help: "list client topic count",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: argument expected")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				err := sendCommand(client, pb.C2Request_GET_CLIENT_TOPIC_COUNT, id, nil, "", "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "lcts",
			Help: "list client topics",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 3 {
					c.Println("command failed: 3 arguments expected")
					return
				}
				id = e4.HashIDAlias(c.Args[0])
				offset, err := strconv.ParseUint(c.Args[1], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				count, err := strconv.ParseUint(c.Args[2], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				err = sendCommand(client, pb.C2Request_GET_CLIENT_TOPIC_COUNT, id, nil, "", "", offset, count)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "ltcc",
			Help: "list topic client count (topic)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 1 {
					c.Println("command failed: argument expected")
					return
				}
				topic = c.Args[0]
				err := sendCommand(client, pb.C2Request_GET_CLIENT_TOPIC_COUNT, nil, nil, topic, "", 0, 0)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "ltcs",
			Help: "list topic clients (client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 3 {
					c.Println("command failed: 3 arguments expected")
					return
				}
				topic = c.Args[0]
				offset, err := strconv.ParseUint(c.Args[1], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				count, err := strconv.ParseUint(c.Args[2], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				err = sendCommand(client, pb.C2Request_GET_CLIENT_TOPIC_COUNT, nil, nil, topic, "", offset, count)
				if err != nil {
					c.Println("command failed: ", err)
				}
			},
		})

		shell.AddCmd(&ishell.Cmd{
			Name: "lcts",
			Help: "list client topics (client)",
			Func: func(c *ishell.Context) {
				if len(c.Args) != 3 {
					c.Println("command failed: 3 arguments expected")
					return
				}
				topic = c.Args[0]
				offset, err := strconv.ParseUint(c.Args[1], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				count, err := strconv.ParseUint(c.Args[2], 10, 64)
				if err != nil {
					c.Println("Cannot convert argument to integer: ", err)
					return
				}
				err = sendCommand(client, pb.C2Request_GET_CLIENT_TOPIC_COUNT, nil, nil, topic, "", offset, count)
				if err != nil {
					c.Println("command failed: ", err)
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

	err = sendCommand(client, commandcode, id, key, topic, msg, offset, count)
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
	case "lsc":
		return pb.C2Request_GET_CLIENTS, nil
	case "lst":
		return pb.C2Request_GET_TOPICS, nil
	default:
		return -1, errors.New("invalid command")
	}
}

// send command with given type, id, key, and topic
func sendCommand(client pb.C2Client, commandcode pb.C2Request_Command, id, key []byte, topic, msg string, offset uint64, count uint64) error {

	req := &pb.C2Request{
		Command: commandcode,
		Id:      id,
		Key:     key,
		Topic:   topic,
		Msg:     msg,
		Offset:  offset,
		Count:   count,
	}

	resp, err := client.C2Command(context.Background(), req)
	if err != nil {
		return err
	}
	if resp.Success {
		if commandcode == pb.C2Request_GET_CLIENTS {
			for _, i := range resp.Ids {
				fmt.Println(i)
			}
		}
		if commandcode == pb.C2Request_GET_TOPICS {
			for _, t := range resp.Topics {
				fmt.Println(t)
			}
		}
		return nil
	}
	return errors.New(resp.Err)
}
