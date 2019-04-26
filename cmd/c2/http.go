package main

import (
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-kit/kit/log"
	"github.com/gorilla/mux"
	"go.opencensus.io/plugin/ochttp"

	"gitlab.com/teserakt/c2/internal/config"
	e4 "gitlab.com/teserakt/e4common"
)

func (s *C2) createHTTPServer(scfg config.ServerCfg) error {
	var logger = log.With(s.logger, "protocol", "http")
	logger.Log("addr", scfg.Addr)

	tlsCert, err := tls.LoadX509KeyPair(scfg.Cert, scfg.Key)
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

	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}/key/{key:[0-9a-f]{64}}", s.handleNewClient).Methods("POST")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}", s.handleRemoveClient).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}/topic/{topic}", s.handleNewTopicClient).Methods("PUT")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}/topic/{topic}", s.handleRemoveTopicClient).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}/topics/count", s.handleGetClientTopicCount).Methods("GET")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}/topics/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetClientTopics).Methods("GET")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}", s.handleResetClient).Methods("PUT")
	route.HandleFunc("/e4/topic/{topic}", s.handleNewTopic).Methods("POST")
	route.HandleFunc("/e4/topic/{topic}", s.handleRemoveTopic).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{32}}", s.handleNewClientKey).Methods("PATCH")
	route.HandleFunc("/e4/topic/{topic}/message/{message}", s.handleSendMessage).Methods("POST")
	route.HandleFunc("/e4/topic/{topic}/clients/count", s.handleGetTopicClientCount).Methods("GET")
	route.HandleFunc("/e4/topic/{topic}/clients/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetTopicClients).Methods("GET")

	route.HandleFunc("/e4/topic", s.handleGetTopics).Methods("GET")
	route.HandleFunc("/e4/client", s.handleGetClients).Methods("GET")

	logger.Log("msg", "starting https server")

	och := &ochttp.Handler{
		Handler: route,
	}

	apiServer := &http.Server{
		Addr:         scfg.Addr,
		Handler:      och,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	return apiServer.ListenAndServeTLS(scfg.Cert, scfg.Key)
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

func (s *C2) handleNewClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	key, err := decodeAndValidateKey(params["key"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid key: %s", err))
		return
	}

	if err := s.newClient(id, key); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.removeClient(id); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("removeClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewTopicClient(w http.ResponseWriter, r *http.Request) {
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

	if err := s.newTopicClient(id, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newTopicClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveTopicClient(w http.ResponseWriter, r *http.Request) {
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

	if err := s.removeTopicClient(id, topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("remoteTopicClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleResetClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.resetClient(id); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("resetClient failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.newTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newTopic failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.removeTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("removeTopic failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewClientKey(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	if err := s.newClientKey(id); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("newClientKey failed: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleGetClients(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	ids, err := s.dbGetIDListHex()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&ids)
}

func (s *C2) handleGetTopics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	topics, err := s.dbGetTopicsList()
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *C2) handleGetClientTopicCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	resp := Response{w}

	id, err := decodeAndValidateID(params["id"])
	if err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid ID: %s", err))
		return
	}

	count, err := s.dbCountTopicsForID(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)
}

func (s *C2) handleGetClientTopics(w http.ResponseWriter, r *http.Request) {
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

	topics, err := s.dbGetTopicsForID(id, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *C2) handleGetTopicClientCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	count, err := s.dbCountIDsForTopic(params["topic"])
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&count)

}

func (s *C2) handleGetTopicClients(w http.ResponseWriter, r *http.Request) {
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

	clients, err := s.dbGetIdsforTopic(topic, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&clients)
}

func (s *C2) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	message := params["message"]

	if err := e4.IsValidTopic(topic); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("invalid topic: %s", err))
		return
	}

	if err := s.sendMessage(topic, message); err != nil {
		resp.Text(http.StatusNotFound, fmt.Sprintf("message not sent: %s", err))
		return
	}

	w.WriteHeader(http.StatusOK)
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
