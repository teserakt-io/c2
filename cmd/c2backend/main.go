package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/net/context"
	"google.golang.org/grpc"

	"github.com/dgraph-io/badger"
	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	pb "teserakt/e4go/pkg/c2proto"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// variables set at build time
var gitCommit string
var buildDate string

// C2 is the C2's state, consisting of ID keys, topic keys, and an MQTT connection.
type C2 struct {
	db         *badger.DB
	mqttClient mqtt.Client
	logger     log.Logger
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Access-Control-Allow-Origin", "*")
        w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
        next.ServeHTTP(w, r)
    })
}

func main() {

	// our server
	var c2 C2

	// init logger
	c2.logger = log.NewJSONLogger(os.Stdout)
	{
		c2.logger = log.With(c2.logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}
	defer c2.logger.Log("msg", "goodbye")

	// show banner
	fmt.Println("    /---------------------------------/")
	fmt.Println("   /  E4: C2 back-end                /")
	fmt.Printf("  /  version %s-%s          /\n", buildDate, gitCommit[:4])
	fmt.Println(" /  Teserakt AG, 2018              /")
	fmt.Println("/---------------------------------/")
	fmt.Println("")

	// load config
	c := config(log.With(c2.logger, "unit", "config"))
	var (
		grpcAddr   = c.GetString("grpc-host-port")
		httpAddr   = c.GetString("http-host-port")
		dbDir      = c.GetString("db-dir")
		mqttBroker = c.GetString("mqtt-broker")
		mqttID     = c.GetString("mqtt-ID")
	)
	c2.logger.Log("msg", "config loaded")

	// open db
	dbOpts := badger.DefaultOptions
	dbOpts.Dir = dbDir
	dbOpts.ValueDir = dbDir
	db, err := badger.Open(dbOpts)
	if err != nil {
		c2.logger.Log("msg", "database opening failed", "error", err)
		return
	}
	defer db.Close()
	c2.logger.Log("msg", "database open")
	c2.db = db

	// create critical error channel
	var errc = make(chan error)
	go func() {
		var c = make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errc <- fmt.Errorf("%s", <-c)
	}()

	// start mqtt client
	{
		logger := log.With(c2.logger, "protocol", "mqtt")
		logger.Log("addr", mqttBroker)
		mqOpts := mqtt.NewClientOptions()
		mqOpts.AddBroker(mqttBroker)
		mqOpts.SetClientID(mqttID)
		mqttClient := mqtt.NewClient(mqOpts)
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			logger.Log("msg", "connection failed", "error", token.Error())
			return
		}
		logger.Log("msg", "connected to broker")
		// instantiate C2
		c2.mqttClient = mqttClient
	}

	// create grpc server
	go func() {
		var logger = log.With(c2.logger, "protocol", "grpc")
		logger.Log("addr", grpcAddr)

		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Log("msg", "failed to listen", "error", err)
			return
		}
		s := grpc.NewServer()
		pb.RegisterC2Server(s, &c2)

		count, err := c2.countIDKeys()
		if err != nil {
			logger.Log("msg", "failed to count id keys", "error", err)
			return
		}
		logger.Log("nbidkeys", count)
		count, err = c2.countTopicKeys()
		if err != nil {
			logger.Log("msg", "failed to count topic keys", "error", err)
			return
		}
		logger.Log("nbtopickeys", count)

		logger.Log("msg", "starting grpc server")

		errc <- s.Serve(lis)
	}()

	// create http server
	go func() {
		var logger = log.With(c2.logger, "protocol", "http")
		logger.Log("addr", httpAddr)

		route := mux.NewRouter()
		route.Use(corsMiddleware)
		route.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusNoContent)
            return
        })

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

		logger.Log("msg", "starting http server")
		errc <- http.ListenAndServe(httpAddr, route)
	}()

	c2.logger.Log("error", <-errc)
}

// C2Command processes a command received over gRPC by the CLI tool.
func (s *C2) C2Command(ctx context.Context, in *pb.C2Request) (*pb.C2Response, error) {

	//log.Printf("command received: %s", pb.C2Request_Command_name[int32(in.Command)])

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
	case pb.C2Request_GET_CLIENTS:
		return s.gRPCgetClients(in)
	case pb.C2Request_GET_TOPICS:
		return s.gRPCgetTopics(in)
	}
	return &pb.C2Response{Success: false, Err: "unknown command"}, nil
}

func config(logger log.Logger) *viper.Viper {

	logger.Log("msg", "load configuration and command args")

	var v = viper.New()
	v.SetConfigName("config")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("../configs")
	v.SetDefault("mqtt-broker", "tcp://localhost:1883")
	v.SetDefault("mqtt-ID", "e4c2")
	v.SetDefault("db-dir", "/tmp/E4/db")
	v.SetDefault("grpc-host-port", "0.0.0.0:5555")
	v.SetDefault("http-host	-port", "0.0.0.0:8888")

	err := v.ReadInConfig()
	if err != nil {
		logger.Log("error", err)
	}

	return v
}
