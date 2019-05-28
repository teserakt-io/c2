package api

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	ListenAndServe() error
}

type httpServer struct {
	e4Service services.E4
	logger    log.Logger
	cfg       config.ServerCfg
}

var _ HTTPServer = &httpServer{}

// NewHTTPServer creates a new http server for C2
func NewHTTPServer(scfg config.ServerCfg, e4Service services.E4, logger log.Logger) HTTPServer {
	return &httpServer{
		e4Service: e4Service,
		logger:    logger,
		cfg:       scfg,
	}
}

func (s *httpServer) ListenAndServe() error {
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
		w.WriteHeader(http.StatusNoContent)
		return
	})

	route.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		resp := Response{w}
		resp.Text(http.StatusNotFound, "Nothing here")
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

	return apiServer.ListenAndServeTLS(s.cfg.Cert, s.cfg.Key)
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
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	key, err := decodeAndValidateKey(params["key"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid key: %s", err))
		return
	}

	if err := s.e4Service.NewClient(name, nil, key); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleRemoveClientByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.e4Service.RemoveClientByID(id); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("removeClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleRemoveClientByName(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid Name: %s", err))
		return
	}

	if err := s.e4Service.RemoveClientByName(name); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("removeClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleNewTopicClientByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopicClient("", id, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newTopicClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleNewTopicClientByName(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopicClient(name, nil, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newTopicClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleRemoveTopicClientByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopicClientByID(id, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("remoteTopicClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleRemoveTopicClientByName(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopicClientByName(name, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("remoteTopicClient failed: %s", err))
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleResetClientByID(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.e4Service.ResetClientByID(id); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("resetClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleResetClientByName(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid name: %s", err))
		return
	}

	if err := s.e4Service.ResetClientByName(name); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("resetClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleNewTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.NewTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newTopic failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleRemoveTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.RemoveTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("removeTopic failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleNewClientKey(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.e4Service.NewClientKey(name, nil); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newClientKey failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ids, err := s.e4Service.GetAllClientsAsNames()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&ids)
}

func (s *httpServer) handleGetTopics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	topics, err := s.e4Service.GetAllTopics()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *httpServer) handleGetClientTopicCountByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	count, err := s.e4Service.CountTopicsForClientByID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)
}

func (s *httpServer) handleGetClientTopicCountByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid name: %s", err))
		return
	}

	count, err := s.e4Service.CountTopicsForClientByName(name)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)
}

func (s *httpServer) handleGetClientTopicsByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsForClientByID(id, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *httpServer) handleGetClientTopicsByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := Response{w}

	name, err := decodeAndValidateName(params["name"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsForClientByName(name, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *httpServer) handleGetTopicClientCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	count, err := s.e4Service.CountClientsForTopic(params["topic"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)

}

func (s *httpServer) handleGetTopicClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := Response{w}
	topic := params["topic"]

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid count")
		return
	}

	clients, err := s.e4Service.GetClientsByNameForTopic(topic, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&clients)
}

func (s *httpServer) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	message := params["message"]

	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.e4Service.SendMessage(topic, message); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("message not sent: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *httpServer) handleGetTopicCount(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")

	count, err := s.e4Service.CountTopics()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)
}

func (s *httpServer) handleGetTopicsPaginated(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := Response{w}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid count")
		return
	}

	topics, err := s.e4Service.GetTopicsRange(int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	json.NewEncoder(w).Encode(&topics)

}

func (s *httpServer) handleGetClientsCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	count, err := s.e4Service.CountClients()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)
}

func (s *httpServer) handleGetClientsPaginated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)
	resp := Response{w}

	offset, err := strconv.ParseUint(params["offset"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid offset")
		return
	}
	count, err := strconv.ParseUint(params["count"], 10, 64)
	if err != nil {
		resp.Text(http.StatusNotFound, "invalid count")
		return
	}

	clients, err := s.e4Service.GetClientsAsNamesRange(int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}
	json.NewEncoder(w).Encode(&clients)

}

// Response is a helper struct to create an http response
type Response struct {
	http.ResponseWriter
}

// Text is a helper to write raw text as an HTTP response
func (r *Response) Text(code int, body string) {
	r.Header().Set("Content-Type", "text/plain")
	r.WriteHeader(code)
	io.WriteString(r, fmt.Sprintf("%s\n", body))
}
