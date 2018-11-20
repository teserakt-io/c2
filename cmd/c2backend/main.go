package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	stdlog "log"

	e4 "teserakt/e4/common/pkg/"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"github.com/spf13/viper"

	pb "teserakt/e4/api/pkg/c2proto"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// variables set at build time
var gitCommit string
var buildDate string

// C2 is the C2's state, consisting of ID keys, topic keys, and an MQTT connection.
type C2 struct {
	keyenckey [e4.KeyLen]byte
	db        *gorm.DB

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

	defer os.Exit(1)

	// our server
	var c2 C2

	// show banner
	fmt.Printf("E4: C2 back-end - version %s-%s\n", buildDate, gitCommit[:4])
	fmt.Println("Copyright (c) Teserakt AG, 2018")

	// init logger
	logFileName := fmt.Sprintf("/var/log/e4_c2backend.log")
	logFile, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0660)
	if err != nil {
		fmt.Printf("[ERROR] logs: unable to open file '%v' to write logs: %v\n", logFileName, err)
		fmt.Print("[WARN] logs: falling back to standard output only\n")
		logFile = os.Stdout
	}

	defer logFile.Close()

	c2.logger = log.NewJSONLogger(logFile)
	{
		c2.logger = log.With(c2.logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}
	defer c2.logger.Log("msg", "goodbye")

	// compatibility for packages that do not understand go-kit logger:
	stdloglogger := stdlog.New(log.NewStdlibAdapter(c2.logger), "", 0)

	// load config
	c := config(log.With(c2.logger, "unit", "config"))
	var (
		grpcAddr        = c.GetString("grpc-host-port")
		httpAddr        = c.GetString("http-host-port")
		mqttBroker      = c.GetString("mqtt-broker")
		mqttPassword    = c.GetString("mqtt-password")
		mqttUsername    = c.GetString("mqtt-username")
		mqttID          = c.GetString("mqtt-ID")
		dbLogging       = c.GetBool("db-logging")
		dbType          = c.GetString("db-type")
		dbPassphrase    = c.GetString("db-encryption-passphrase")
		grpcCert        = c.GetString("grpc-cert")
		grpcKey         = c.GetString("grpc-key")
		httpTLSCertPath = c.GetString("http-cert")
		httpTLSKeyPath  = c.GetString("http-key")
	)

	if dbPassphrase == "" {
		c2.logger.Log("msg", "no passphrase supplied")
		fmt.Fprintf(os.Stderr, "ERROR: No passphrase supplied. Refusing to start with an empty passphrase.\n")
		return
	}

	keyenckey := e4.HashPwd(dbPassphrase)
	copy(c2.keyenckey[:], keyenckey)

	var dbConnectionString string

	if dbType == "postgres" {
		var (
			dbUsername         = c.GetString("db-username")
			dbPassword         = c.GetString("db-password")
			dbHost             = c.GetString("db-host")
			dbDatabase         = c.GetString("db-database")
			dbSecureConnection = c.GetString("db-secure-connection")
		)
		var sslstring string
		fmt.Println("db-secure-connection", dbSecureConnection)
		if dbSecureConnection == "enable" {
			sslstring = "sslmode=verify-full"
		} else if dbSecureConnection == "selfsigned" {
			sslstring = "sslmode=require"
			c2.logger.Log("msg", "Self signed certificate used. We do not recommend this setup.")
			fmt.Fprintf(os.Stderr, "WARNING: Self-signed connection to database. We do not recommend this setup.\n")
		} else if dbSecureConnection == "insecure" {
			sslstring = "sslmode=disable"
			c2.logger.Log("msg", "Unencrypted database connection.")
			fmt.Fprintf(os.Stderr, "WARNING: Unencrypted database connection. We do not recommend this setup.\n")
		} else {
			c2.logger.Log("msg", "Invalid option for db-secure-connection")
			return
		}

		dbConnectionString = fmt.Sprintf("host=%s dbname=%s user=%s password=%s %s",
			dbHost, dbDatabase, dbUsername, dbPassword, sslstring)
	} else if dbType == "sqlite3" {
		var (
			dbPath = c.GetString("db-file")
		)
		dbConnectionString = fmt.Sprintf("%s", dbPath)

		c2.logger.Log("msg", "SQLite3 selected as database. SQLite3 is not supported for production environments")
		fmt.Fprintf(os.Stderr, "WARNING: SQLite3 database selected. NOT supported in production environments\n")
	} else {
		// defensive coding:
		c2.logger.Log("msg", "unknown or unsupported database type", "db-type", dbType)
		return
	}

	c2.logger.Log("msg", "config loaded")

	// open db.
	db, err := gorm.Open(dbType, dbConnectionString)

	if err != nil {
		c2.logger.Log("msg", "database opening failed", "error", err)
		return
	}

	defer db.Close()

	c2.logger.Log("msg", "database open")
	db.LogMode(dbLogging)
	db.SetLogger(stdloglogger)
	c2.db = db

	if dbType == "postgres" {
		c2.db.Exec("SET search_path TO e4_c2_test;")
	}
	// Ensure the database schema is ready to use:
	err = c2.dbInitialize()
	if err != nil {
		c2.logger.Log("msg", "database setup failed", "error", err)
		return
	}

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
		mqOpts.SetPassword(mqttPassword)
		mqOpts.SetUsername(mqttUsername)
		mqttClient := mqtt.NewClient(mqOpts)
		logger.Log("msg", "mqtt parameters", "broker", mqttBroker, "id", mqttID, "username", mqttUsername)
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
			close(errc)
			runtime.Goexit()
		}
		creds, err := credentials.NewServerTLSFromFile(grpcCert, grpcKey)
		if err != nil {
			logger.Log("msg", "failed to get credentials", "cert", grpcCert, "key", grpcKey, "error", err)
			close(errc)
			runtime.Goexit()
		}
		logger.Log("msg", "using TLS for gRPC", "cert", grpcCert, "key", grpcKey, "error", err)

		s := grpc.NewServer(grpc.Creds(creds))
		pb.RegisterC2Server(s, &c2)

		count, err := c2.countIDKeys()
		if err != nil {
			logger.Log("msg", "failed to count id keys", "error", err)
			close(errc)
			runtime.Goexit()
		}
		logger.Log("nbidkeys", count)
		count, err = c2.countTopicKeys()
		if err != nil {
			logger.Log("msg", "failed to count topic keys", "error", err)
			close(errc)
			runtime.Goexit()
		}
		logger.Log("nbtopickeys", count)

		logger.Log("msg", "starting grpc server")

		errc <- s.Serve(lis)
	}()

	// create http server
	go func() {
		var logger = log.With(c2.logger, "protocol", "http")
		logger.Log("addr", httpAddr)

		tlsCert, err := tls.LoadX509KeyPair(httpTLSCertPath, httpTLSKeyPath)
		if err != nil {
			errc <- err
			return
		}

		// TODO: maybe we could make some of these options configurable.
		tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCert},
			MinVersion:               tls.VersionTLS12,
			CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			PreferServerCipherSuites: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		}

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
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topics/count", c2.handleGetClientTopicCount).Methods("GET")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topics/{offset:[0-9]+}/{count:[0-9]+}", c2.handleGetClientTopics).Methods("GET")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", c2.handleResetClient).Methods("PUT")
		route.HandleFunc("/e4/topic/{topic}", c2.handleNewTopic).Methods("POST")
		route.HandleFunc("/e4/topic/{topic}", c2.handleRemoveTopic).Methods("DELETE")
		route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", c2.handleNewClientKey).Methods("PATCH")
		route.HandleFunc("/e4/topic/{topic}/message/{message}", c2.handleSendMessage).Methods("POST")
		route.HandleFunc("/e4/topic/{topic}/clients/count", c2.handleGetTopicClientCount).Methods("GET")
		route.HandleFunc("/e4/topic/{topic}/clients/{offset:[0-9]+}/{count:[0-9]+}", c2.handleGetTopicClients).Methods("GET")

		route.HandleFunc("/e4/topic", c2.handleGetTopics).Methods("GET")
		route.HandleFunc("/e4/client", c2.handleGetClients).Methods("GET")

		logger.Log("msg", "starting https server")

		apiServer := &http.Server{
			Addr:         httpAddr,
			Handler:      route,
			TLSConfig:    tlsConfig,
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		}

		errc <- apiServer.ListenAndServeTLS(httpTLSCertPath, httpTLSKeyPath)
	}()

	c2.logger.Log("error", <-errc)
}

// C2Command processes a command received over gRPC by the CLI tool.
func (s *C2) C2Command(ctx context.Context, in *pb.C2Request) (*pb.C2Response, error) {

	//log.Printf("command received: %s", pb.C2Request_Command_name[int32(in.Command)])
	s.logger.Log("msg", "received gRPC request", "request", pb.C2Request_Command_name[int32(in.Command)])

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
	case pb.C2Request_GET_CLIENT_TOPIC_COUNT:
		return s.gRPCgetClientTopicCount(in)
	case pb.C2Request_GET_CLIENT_TOPICS:
		return s.gRPCgetClientTopics(in)
	case pb.C2Request_GET_TOPIC_CLIENT_COUNT:
		return s.gRPCgetTopicClientCount(in)
	case pb.C2Request_GET_TOPIC_CLIENTS:
		return s.gRPCgetTopicClients(in)
	}
	return &pb.C2Response{Success: false, Err: "unknown command"}, nil
}

func binarydir() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir
}

func config(logger log.Logger) *viper.Viper {

	logger.Log("msg", "load configuration and command args")

	var v = viper.New()

	// Con
	v.SetConfigName("config")
	confdir, _ := filepath.Abs(filepath.Join(filepath.Join(binarydir(), ".."), "configs"))
	v.AddConfigPath(confdir)
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("../configs")
	v.SetDefault("mqtt-broker", "tcp://localhost:1883")
	v.SetDefault("mqtt-ID", "e4c2")
	v.SetDefault("mqtt-username", "")
	v.SetDefault("mqtt-password", "")
	v.SetDefault("db-logging", false)
	v.SetDefault("db-host", "localhost")
	v.SetDefault("db-database", "e4")
	v.SetDefault("db-secure-connection", "enable")
	v.SetDefault("grpc-host-port", "0.0.0.0:5555")
	v.SetDefault("http-host	-port", "0.0.0.0:8888")

	// Allow the whole environment to be configured by
	// env variables for testing.
	v.BindEnv("mqtt-broker", "E4C2_MQTT_BROKER")
	v.BindEnv("mqtt-ID", "E4C2_MQTT_ID")
	v.BindEnv("mqtt-QoS", "E4C2_MQTT_QOS")
	v.BindEnv("db-type", "E4C2_DB_TYPE")
	v.BindEnv("db-file", "E4C2_DB_FILE")
	v.BindEnv("db-username", "E4C2_DB_USERNAME")
	v.BindEnv("db-password", "E4C2_DB_PASSWORD")

	// This one in particular should survive even if others are subsequently
	// removed:
	v.BindEnv("db-encryption-passphrase", "E4C2_DB_ENCRYPTION_PASSPHRASE")

	v.BindEnv("db-secure-connection", "E4C2_DB_SECURE_CONNECTION")
	v.BindEnv("grpc-host-port", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("grpc-cert", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("grpc-key", "E4C2_GRPC_HOST_PORT")
	v.BindEnv("http-host-port", "E4C2_HTTP_HOST_PORT")
	v.BindEnv("http-cert", "E4C2_HTTP_HOST_PORT")
	v.BindEnv("http-key", "E4C2_HTTP_HOST_PORT")

	// Now
	//pflag.Parse()
	//viper.BindPFlags(pflag.CommandLine)

	err := v.ReadInConfig()
	if err != nil {
		logger.Log("error", err)
	}

	return v
}
