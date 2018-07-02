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

// C2 is the C2's state, consisting of ID keys, topic keys, and an MQTT connection.
type C2 struct {
	db      *badger.DB 
	mqClient mqtt.Client
}

// C2Command processes a command received over gRPC by the CLI tool.
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

func config () *viper.Viper {
	var v = viper.New()
	v.SetDefault("mqtt-broker", "test.mosquitto.org:1803")	
	v.SetDefault("mqtt-QoS, 2")
	v.SetDefault("mqtt-ID", "E4C2")
	v.SetDefault("db-dir", "/tmp/E4/db")
	v.SetDefault("grpc-host-port", "0.0.0.0:5555")
	v.SetDefault("http-host-port", "0.0.0.0:8888")
	var keys = v.AllKeys()
	sort.Strings(keys)
	for _, k := range keys {
		log.Print(k, v.Get(k))
	}
}

func main() {

	log.SetPrefix("c2backend\t")

	// load config
	v := config()
	var (

	)
	log.Print("config loaded")

	// open id keys db
	dbOpts := badger.DefaultOptions
	dbOpts.Dir = dbDir
	dbOpts.ValueDir = dbDir
	db, err := badger.Open(dbOpts)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	log.Print("database open")

	// start mqtt client
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(mqttBroker)
	mqOpts.SetClientID(mqttID)
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
	c2 := C2{db, mqttClient}
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
