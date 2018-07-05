package main

import (
	"fmt"
	"io"
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

// C2 is the C2's state, consisting of ID keys, topic keys, and an MQTT connection.
type C2 struct {
	db       *badger.DB
	mqClient mqtt.Client
}

func main() {

	log.SetPrefix("c2backend\t")

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
			resp.Text(http.StatusNotFound, "Not found")
		})

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

// Response ...
type Response struct {
	http.ResponseWriter
}

// Text is a helper to write raw text as an HTTP response
func (r *Response) Text(code int, body string) {
	r.Header().Set("Content-Type", "text/plain")
	r.WriteHeader(code)

	_, err := io.WriteString(r, fmt.Sprintf("%s\n", body))
	if err != nil {
		log.Printf(err.Error())
	}
}

/*
// filter handlers by allowed method, as in https://www.codementor.io/codehakase/building-a-restful-api-with-golang-a6yivzqdo
func handleNewClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id := params["id"]
	key := params["key"]
}

func handleRemoveClient(w http.ResponseWriter, r *http.Request) {

}

func handleNewTopicClient(w http.ResponseWriter, r *http.Request) {

}
*/
