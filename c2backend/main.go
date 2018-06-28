package main

import (
	"log"
	"net"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/dgraph-io/badger"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	pb "teserakt/c2proto"
)

type C2 struct {
	dbi      *badger.DB // id:key
	dbt      *badger.DB // topic:key
	mqClient mqtt.Client
}

func (s *C2) C2Command(ctx context.Context, in *pb.C2Request) (*pb.C2Response, error) {

	log.Printf("command received: %s", pb.C2Request_Command_name[int32(in.Command)])

	switch in.Command {
	case pb.C2Request_NEW_CLIENT:
		return s.newClient(in)
	case pb.C2Request_REMOVE_CLIENT:
		return s.removeClient(in)
	case pb.C2Request_NEW_TOPIC_CLIENT:
		return s.newTopicClient(in)
	case pb.C2Request_REMOVE_TOPIC_CLIENT:
		return s.removeTopicClient(in)
	case pb.C2Request_RESET_CLIENT:
		return s.resetClient(in)
	case pb.C2Request_NEW_TOPIC:
		return s.newTopic(in)
	case pb.C2Request_REMOVE_TOPIC:
		return s.removeTopicClient(in)
	case pb.C2Request_NEW_CLIENT_KEY:
		return s.newClientKey(in)
	}
	return &pb.C2Response{Success: false, Err: "unknown command"}, nil
}

func main() {

	log.SetPrefix("c2backend\t")

	// open id keys db
	dbiOpts := badger.DefaultOptions
	dbiOpts.Dir = dbiDir
	dbiOpts.ValueDir = dbiDir
	dbi, err := badger.Open(dbiOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer dbi.Close()
	log.Print("id database open")

	// open topic keys db
	dbtOpts := badger.DefaultOptions
	dbtOpts.Dir = dbtDir
	dbtOpts.ValueDir = dbtDir
	dbt, err := badger.Open(dbtOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer dbt.Close()
	log.Print("topic database open")

	// start mqtt client
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(mqttBroker)
	mqOpts.SetClientID(mqttId)
	// mqOpts.SetUsername()
	// mqOpts.SetPassword()
	mqttClient := mqtt.NewClient(mqOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("MQTT connection failed")
	}

	log.Printf("connected to MQTT broker")

	// create server
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	c2 := C2{dbi, dbt, mqttClient}
	pb.RegisterC2Server(s, &c2)

	count, err := c2.countIDKeys()
	if err != nil {
		log.Fatal("failed to iterated over the id db")
	}
	log.Printf("%d ids in the db", count)
	count, err = c2.countTopicKeys()
	if err != nil {
		log.Fatal("failed to iterated over the topic db")
	}
	log.Printf("%d topics in the db", count)

	log.Print("starting server")
	s.Serve(lis)
}
