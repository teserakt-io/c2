package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/dgraph-io/badger"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	pb "teserakt/c2proto"
)

var GitCommit string
var BuildDate string

// C2 is the C2's state, consisting of ID keys, topic keys, and an MQTT connection.
type C2 struct {
	db       *badger.DB
	mqClient mqtt.Client
}

func main() {

	log.SetPrefix("c2backend\t")

	fmt.Println("    /---------------------------------/")
	fmt.Println("   /  E4: C2 back-end                /")
	fmt.Printf("  /  version %s-%s          /\n", BuildDate, GitCommit[:4])
	fmt.Println(" /  Teserakt AG, 2018              /")
	fmt.Println("/---------------------------------/\n")

	// load config
	c := config()
	var (
		grpcAddr   = c.GetString("grpc-host-port")
		httpAddr   = c.GetString("http-host-port")
		dbDir      = c.GetString("db-dir")
		mqttBroker = c.GetString("mqtt-broker")
		mqttID     = c.GetString("mqtt-ID")
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

	// critical error channel
	var errc = make(chan error)
	go func() {
		var c = make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// start mqtt client
	mqOpts := mqtt.NewClientOptions()
	mqOpts.AddBroker(mqttBroker)
	mqOpts.SetClientID(mqttID)
	log.Printf("connecting to %s", mqttBroker)
	mqttClient := mqtt.NewClient(mqOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal("mqtt connection failed")
	}

	log.Printf("connected to mqtt broker")

	// instantiate C2
	c2 := C2{db, mqttClient}

	// create grpc server
	go func() {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}
		s := grpc.NewServer()
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

		log.Print("starting grpc server")

		errc <- s.Serve(lis)
	}()

	// create http server
	go func() {
		route := mux.NewRouter()

		route.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			resp := Response{w}
			resp.Text(http.StatusNotFound, "Nothing here")
		})

		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/key/{key:[0-9a-f]{128}}", c2.handleNewClient).Methods("POST")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", c2.handleRemoveClient).Methods("DELETE")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topic/{topic}", c2.handleNewTopicClient).Methods("PUT")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topic/{topic}", c2.handleRemoveTopicClient).Methods("DELETE")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", c2.handleResetClient).Methods("PUT")
		route.HandleFunc("/e4/topic/{topic}", c2.handleNewTopic).Methods("POST")
		route.HandleFunc("/e4/topic/{topic}", c2.handleRemoveTopic).Methods("DELETE")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", c2.handleNewClientKey).Methods("PATCH")
		route.HandleFunc("/e4/topic/{topic}/message/{message}", c2.handleSendMessage).Methods("POST")

		route.HandleFunc("/e4/topic", c2.handleGetTopics).Methods("GET")
		route.HandleFunc("/e4/client", c2.handleGetClients).Methods("GET")
		//route.HandleFunc("/e4/client/{}/topic", c2.handleGetClientsTopics).Methods("GET")
		//route.HandleFunc("/e4/topic/{topic}/client", c2.handleGetTopicsClients).Methods("GET")

		log.Print("starting http server")
		errc <- http.ListenAndServe(httpAddr, route)
	}()

	log.Print("error", <-errc)
}

// C2Command processes a command received over gRPC by the CLI tool.
func (s *C2) C2Command(ctx context.Context, in *pb.C2Request) (*pb.C2Response, error) {

	log.Printf("command received: %s", pb.C2Request_Command_name[int32(in.Command)])

	switch in.Command {
	case pb.C2Request_NEW_CLIENT:
		return s.gRPCnewClient(in)
	case pb.C2Request_REMOVE_CLIENT:
		return s.gRPCremoveClient(in)
	case pb.C2Request_NEW_TOPIC_CLIENT:
		return s.gRPCnewTopicClient(in)
	case pb.C2Request_REMOVE_TOPIC_CLIENT:
		return s.gRPCremoveTopicClient(in)
	case pb.C2Request_RESET_CLIENT:
		return s.gRPCresetClient(in)
	case pb.C2Request_NEW_TOPIC:
		return s.gRPCnewTopic(in)
	case pb.C2Request_REMOVE_TOPIC:
		return s.gRPCremoveTopic(in)
	case pb.C2Request_NEW_CLIENT_KEY:
		return s.gRPCnewClientKey(in)
	case pb.C2Request_SEND_MESSAGE:
		return s.gRPCsendMessage(in)
	}
	return &pb.C2Response{Success: false, Err: "unknown command"}, nil
}

func config() *viper.Viper {
	var v = viper.New()
	v.SetConfigName("config")
	v.AddConfigPath("./configs")
	v.SetDefault("mqtt-broker", "test.mosquitto.org:1883")
	v.SetDefault("mqtt-ID", "e4c2")
	v.SetDefault("db-dir", "/tmp/E4/db")
	v.SetDefault("grpc-host-port", "0.0.0.0:5555")
	v.SetDefault("http-host	-port", "0.0.0.0:8888")

	err := v.ReadInConfig()
	if err != nil {
		log.Print("failed to read config:", err)
	}

	return v
}
