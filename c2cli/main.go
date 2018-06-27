package main

import (
	"log"
    "flag"
    "encoding/hex"

	"golang.org/x/net/context"
    "google.golang.org/grpc"
    
    "golang.org/x/crypto/argon2"

	pb "teserakt/c2proto"
	e4 "teserakt/e4common"
)

const (
	address = "localhost:50051"
)

func sendCommand(client pb.C2Client, req *pb.C2Request) {
	resp, err := client.C2Command(context.Background(), req)
	if err != nil {
		log.Fatalf("command error: %v", err)
	}
	if resp.Success {
		log.Printf("command succeeded")
	} else {
		log.Printf("command failed: %s", resp.Err)
	}

}


func main() {

	log.SetPrefix("c2cli\t\t")

    command := flag.String("c", "", "command type (required)")
    idalias := flag.String("id", "", "a client id alias, a UTF-8 string")
    keyhex := flag.String("key", "", "a 512-bit key, an hex string")
    pwd := flag.String("pwd", "", "a passphrase to derive a key from")
    topic := flag.String("topic", "", "a topic, as UTF-8 string")

    flag.Parse()

    var id []byte = nil
    var key []byte = nil
    var err error

    if *command == "" {
        log.Fatal("missing command")
    }

    if *idalias != "" {
        id = e4.HashIdAlias(*idalias)
    }

    if *keyhex != "" {
        if *pwd != "" {
            log.Fatal("choose between key and password")
        }
        key, err = hex.DecodeString(*keyhex)
        log.Print(len(key))
        if err != nil {
            log.Fatalf("key decoding failed: %s", err)
        }
    }

    if *pwd != "" {
        key = argon2.Key([]byte(*pwd), nil, 1, 64*1024, 4, 64)
    }

    req := &pb.C2Request {
        Command: pb.C2Request_NEW_CLIENT,
        Id:  id,
        Key: key,
        Topic: *topic,
    }  

    switch *command {
    case "nc":
        req.Command = pb.C2Request_NEW_CLIENT
    case "rc":
        req.Command = pb.C2Request_REMOVE_CLIENT
    case "ntc":
        req.Command = pb.C2Request_NEW_TOPIC_CLIENT
    case "rtc":
        req.Command = pb.C2Request_REMOVE_TOPIC_CLIENT
    case "rsc":
        req.Command = pb.C2Request_RESET_CLIENT
    case "nt":
        req.Command = pb.C2Request_NEW_TOPIC
    case "rt":
        req.Command = pb.C2Request_REMOVE_TOPIC
    case "nck":
        req.Command = pb.C2Request_NEW_CLIENT_KEY
       default:
        log.Fatal("unknown command")
    }

	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
    client := pb.NewC2Client(conn)
    

    sendCommand(client, req)
    
}
