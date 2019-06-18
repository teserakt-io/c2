package api

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/services"
	e4 "gitlab.com/teserakt/e4common"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"go.opencensus.io/plugin/ochttp"
)

// HTTPServer defines methods available on a C2 HTTP server
type HTTPServer interface {
	ListenAndServe(ctx context.Context) error
}

type httpServer struct {
	e4Service services.E4
	logger    log.Logger
	cfg       config.ServerCfg
	isProd    bool
}

var _ HTTPServer = &httpServer{}

// NewHTTPServer creates a new http server for C2
func NewHTTPServer(scfg config.ServerCfg, isProd bool, e4Service services.E4, logger log.Logger) HTTPServer {
	return &httpServer{
		e4Service: e4Service,
		logger:    logger,
		cfg:       scfg,
		isProd:    isProd,
	}
}

func (s *httpServer) ListenAndServe(ctx context.Context) error {
	s.logger.Log("addr", s.cfg.Addr)

	tlsCert, err := tls.LoadX509KeyPair(s.cfg.Cert, s.cfg.Key)
	if err != nil {
		return err
	}

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
		s.newResponse(w).Success(http.StatusNoContent, nil)
		return
	})

	route.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		s.newResponse(w).Text(http.StatusNotFound, "E4 C2 Server")
		return
	})

	// Clients can *ONLY* be created by name. All other APIs allow client
	// manipulation by name or by topic.
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}/key/{key:[0-9a-f]{64}}", s.handleNewClient).Methods("POST")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}", s.handleNewClientKey).Methods("PATCH")

	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}", s.handleRemoveClientByID).Methods("DELETE")
	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}/topic/{topic}", s.handleNewTopicClientByID).Methods("PUT")
	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}/topic/{topic}", s.handleRemoveTopicClientByID).Methods("DELETE")
	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}/topics/count", s.handleGetClientTopicCountByID).Methods("GET")
	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}/topics/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetClientTopicsByID).Methods("GET")
	route.HandleFunc("/e4/client/id/{id:[0-9a-f]{32}}", s.handleResetClientByID).Methods("PUT")

	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}", s.handleRemoveClientByName).Methods("DELETE")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}/topic/{topic}", s.handleNewTopicClientByName).Methods("PUT")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}/topic/{topic}", s.handleRemoveTopicClientByName).Methods("DELETE")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}/topics/count", s.handleGetClientTopicCountByName).Methods("GET")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}/topics/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetClientTopicsByName).Methods("GET")
	route.HandleFunc("/e4/client/name/{name:[^\\s\\/]{1,256}}", s.handleResetClientByName).Methods("PUT")

	route.HandleFunc("/e4/topic/{topic}", s.handleNewTopic).Methods("POST")
	route.HandleFunc("/e4/topic/{topic}", s.handleRemoveTopic).Methods("DELETE")
	route.HandleFunc("/e4/topic/{topic}/message/{message}", s.handleSendMessage).Methods("POST")
	route.HandleFunc("/e4/topic/{topic}/clients/count", s.handleGetTopicClientCount).Methods("GET")
	route.HandleFunc("/e4/topic/{topic}/clients/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetTopicClients).Methods("GET")

	route.HandleFunc("/e4/topics/all", s.handleGetTopics).Methods("GET")
	route.HandleFunc("/e4/clients/all", s.handleGetClients).Methods("GET")

	route.HandleFunc("/e4/topics/count", s.handleGetTopicCount).Methods("GET")
	route.HandleFunc("/e4/topics/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetTopicsPaginated).Methods("GET")
	route.HandleFunc("/e4/clients/count", s.handleGetClientsCount).Methods("GET")
	route.HandleFunc("/e4/clients/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetClientsPaginated).Methods("GET")

	s.logger.Log("msg", "starting https server")

	och := &ochttp.Handler{
		Handler: route,
	}

	apiServer := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      och,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		s.logger.Log("msg", "failed to listen", "error", err)

		return err
	}

	return apiServer.ServeTLS(lis, s.cfg.Cert, s.cfg.Key)
}

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, PATCH, DELETE")
		next.ServeHTTP(w, r)
	})
}

func decodeAndValidateID(idstr string) ([]byte, error) {
	id, err := hex.DecodeString(idstr)
	if err != nil {
		return nil, err
	}

	if err := e4.IsValidID(id); err != nil {
		return nil, err
	}

	return id, nil
}

func decodeAndValidateName(namestr string) (string, error) {
	name, err := url.QueryUnescape(namestr)
	if err != nil {
		return "", err
	}

	if err := e4.IsValidName(name); err != nil {
		return "", err
	}

	return name, nil
}

func decodeAndValidateKey(keystr string) ([]byte, error) {
	key, err := hex.DecodeString(keystr)
	if err != nil {
		return nil, err
	}

	if err := e4.IsValidKey(key); err != nil {
		return nil, err
	}

	return key, nil
}

func (s *httpServer) handleNewClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	key, err := decodeAndValidateKey(params["key"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid key: %s", err))
		return
	}

	if err := s.e4Service.NewClient(ctx, name, nil, key); err != nil {
		s.logger.Log("msg", "NewClient error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleRemoveClientByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.e4Service.RemoveClientByID(ctx, id); err != nil {
		s.logger.Log("msg", "RemoveClientByID error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleRemoveClientByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid Name: %s", err))
		return
	}

	if err := s.e4Service.RemoveClientByName(ctx, name); err != nil {
		s.logger.Log("msg", "RemoveClientByName error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleNewTopicClientByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopicClient(ctx, "", id, topic); err != nil {
		s.logger.Log("msg", "NewTopicClient error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleNewTopicClientByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopicClient(ctx, name, nil, topic); err != nil {
		s.logger.Log("msg", "NewTopicClient error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleRemoveTopicClientByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopicClientByID(ctx, id, topic); err != nil {
		s.logger.Log("msg", "RemoveTopicClientByID error", "errror", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleRemoveTopicClientByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopicClientByName(ctx, name, topic); err != nil {
		s.logger.Log("msg", "RemoveTopicClientByName error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleResetClientByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.e4Service.ResetClientByID(ctx, id); err != nil {
		s.logger.Log("msg", "ResetClientByID error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleResetClientByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	if err := s.e4Service.ResetClientByName(ctx, name); err != nil {
		s.logger.Log("msg", "ResetClientByName error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleNewTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopic(ctx, topic); err != nil {
		s.logger.Log("msg", "NewTopic error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleRemoveTopic(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopic(ctx, topic); err != nil {
		s.logger.Log("msg", "removeTopic error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleNewClientKey(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	if err := s.e4Service.NewClientKey(ctx, name, nil); err != nil {
		resp.Success(http.StatusInternalServerError, fmt.Sprintf("newClientKey failed: %s", err))
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)

	ids, err := s.e4Service.GetAllClientsAsNames(ctx)
	if err != nil {
		s.logger.Log("msg", "GetAllClientsAsNames error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, &ids)
}

func (s *httpServer) handleGetTopics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)

	topics, err := s.e4Service.GetAllTopics(ctx)
	if err != nil {
		s.logger.Log("msg", "GetAllTopics error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, topics)
}

func (s *httpServer) handleGetClientTopicCountByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	count, err := s.e4Service.CountTopicsForClientByID(ctx, id)
	if err != nil {
		s.logger.Log("msg", "CountTopicsForClientByID error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, count)
}

func (s *httpServer) handleGetClientTopicCountByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	count, err := s.e4Service.CountTopicsForClientByName(ctx, name)
	if err != nil {
		s.logger.Log("msg", "CountTopicsForClientByName error", "error", err)
		resp.Error(err)
		return
	}
	resp.Success(http.StatusOK, count)
}

func (s *httpServer) handleGetClientTopicsByID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsForClientByID(ctx, id, int(offset), int(count))
	if err != nil {
		s.logger.Log("msg", "GetTopicsForClientByID error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, topics)
}

func (s *httpServer) handleGetClientTopicsByName(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := s.newResponse(w)

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Success(http.StatusBadRequest, fmt.Sprintf("invalid name: %s", err))
		return
	}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsForClientByName(ctx, name, int(offset), int(count))
	if err != nil {
		s.logger.Log("msg", "GetTopicsForClientByName error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, topics)
}

func (s *httpServer) handleGetTopicClientCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)
	params := mux.Vars(r)

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, "invalid topic")
		return
	}

	count, err := s.e4Service.CountClientsForTopic(ctx, topic)
	if err != nil {
		s.logger.Log("msg", "GetClientsByNameForTopic error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, count)
}

func (s *httpServer) handleGetTopicClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)
	params := mux.Vars(r)

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, "invalid topic")
		return
	}
	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid count")
		return
	}

	clients, err := s.e4Service.GetClientsByNameForTopic(ctx, topic, int(offset), int(count))
	if err != nil {
		s.logger.Log("msg", "GetClientsByNameForTopic error", "error", err)
		resp.Error(err)
		return
	}
	resp.Success(http.StatusOK, clients)
}

func (s *httpServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	topic := params["topic"]
	message := params["message"]

	if err := e4.IsValidTopic(topic); err != nil {
		resp.Success(http.StatusBadRequest, "invalid topic")
		return
	}

	if err := s.e4Service.SendMessage(ctx, topic, message); err != nil {
		s.logger.Log("msg", "SendMessage error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, nil)
}

func (s *httpServer) handleGetTopicCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)

	count, err := s.e4Service.CountTopics(ctx)
	if err != nil {
		s.logger.Log("msg", "CountTopics error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, count)
}

func (s *httpServer) handleGetTopicsPaginated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsRange(ctx, int(offset), int(count))
	if err != nil {
		s.logger.Log("msg", "GetTopicsRange error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, topics)
}

func (s *httpServer) handleGetClientsCount(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	resp := s.newResponse(w)

	count, err := s.e4Service.CountClients(ctx)
	if err != nil {
		s.logger.Log("msg", "CountClients error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, count)
}

func (s *httpServer) handleGetClientsPaginated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	resp := s.newResponse(w)

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Success(http.StatusBadRequest, "invalid count")
		return
	}

	clients, err := s.e4Service.GetClientsAsNamesRange(ctx, int(offset), int(count))
	if err != nil {
		s.logger.Log("msg", "GetClientsAsNamesRange error", "error", err)
		resp.Error(err)
		return
	}

	resp.Success(http.StatusOK, clients)
}

func (s *httpServer) newResponse(w http.ResponseWriter) *response {
	return &response{
		ResponseWriter: w,
		isProd:         s.isProd,
	}
}

// response is a helper struct to create an http response
type response struct {
	http.ResponseWriter
	isProd bool
}

// Text is a helper to write raw text as an HTTP response
// this should not be used except in special circumstances
func (r *response) Text(code int, body string) {
	r.Header().Set("Content-Type", "text/plain")
	r.WriteHeader(code)
	io.WriteString(r, fmt.Sprintf("%s\n", body))
}

// Success returns a http success status code as specified by code
// and the json-encoded value of data, if specified.
// if nil data is specified, the response body is empty.
// this makes more sense than writing null, or [], to the body.
func (r *response) Success(code int, data interface{}) {
	r.Header().Set("Content-Type", "application/json")
	r.WriteHeader(code)

	if data != nil {
		json.NewEncoder(r).Encode(data)
	}
}

// Error set the proper response status code given an error
func (r *response) Error(err error) {
	r.Header().Set("Content-Type", "application/json")

	var message string
	switch {
	case services.IsErrRecordNotFound(err):
		r.WriteHeader(http.StatusNotFound)
		message = "record not found"
	default:
		r.WriteHeader(http.StatusInternalServerError)
		message = "an error occured, check the logs for details."
	}

	if !r.isProd {
		message = err.Error()
	}

	json.NewEncoder(r).Encode(struct {
		Error string `json:"error"`
	}{
		Error: message,
	})
}
