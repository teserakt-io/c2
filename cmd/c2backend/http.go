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

	e4 "gitlab.com/teserakt/e4common"
)

func (s *C2) createHTTPServer(scfg *startServerConfig) error {
	httpAddr := scfg.addr
	httpCert := scfg.certFile
	httpKey := scfg.keyFile

	var logger = log.With(s.logger, "protocol", "http")
	logger.Log("addr", httpAddr)

	tlsCert, err := tls.LoadX509KeyPair(httpCert, httpKey)
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

	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/key/{key:[0-9a-f]{128}}", s.handleNewClient).Methods("POST")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", s.handleRemoveClient).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topic/{topic}", s.handleNewTopicClient).Methods("PUT")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topic/{topic}", s.handleRemoveTopicClient).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topics/count", s.handleGetClientTopicCount).Methods("GET")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}/topics/{offset:[0-9]+}/{count:[0-9]+}", s.handleGetClientTopics).Methods("GET")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", s.handleResetClient).Methods("PUT")
	route.HandleFunc("/e4/topic/{topic}", s.handleNewTopic).Methods("POST")
	route.HandleFunc("/e4/topic/{topic}", s.handleRemoveTopic).Methods("DELETE")
	route.HandleFunc("/e4/client/{id:[0-9a-f]{64}}", s.handleNewClientKey).Methods("PATCH")
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
		Addr:         httpAddr,
		Handler:      och,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
	}

	return apiServer.ListenAndServeTLS(httpCert, httpKey)
}

func (s *C2) handleNewClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	key, err := hex.DecodeString(params["key"])
	if err != nil || !e4.IsValidKey(key) {
		resp.Text(http.StatusNotFound, "invalid key")
		return
	}

	ret := s.newClient(id, key)
	if ret != nil {
		resp.Text(http.StatusNotFound, "newClient failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	ret := s.removeClient(id)
	if ret != nil {
		resp.Text(http.StatusNotFound, "removeClient failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewTopicClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	topic := params["topic"]
	if err != nil || !e4.IsValidTopic(topic) {
		resp.Text(http.StatusNotFound, "invalid topic")
		return
	}

	ret := s.newTopicClient(id, topic)
	if ret != nil {
		resp.Text(http.StatusNotFound, "newTopicClient failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveTopicClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	topic := params["topic"]
	if err != nil || !e4.IsValidTopic(topic) {
		resp.Text(http.StatusNotFound, "invalid topic")
		return
	}

	ret := s.removeTopicClient(id, topic)
	if ret != nil {
		resp.Text(http.StatusNotFound, "remoteTopicClient failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleResetClient(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	ret := s.resetClient(id)
	if ret != nil {
		resp.Text(http.StatusNotFound, "resetClient failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if !e4.IsValidTopic(topic) {
		resp.Text(http.StatusNotFound, "invalid topic")
		return
	}

	ret := s.newTopic(topic)
	if ret != nil {
		resp.Text(http.StatusNotFound, "newTopic failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleRemoveTopic(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	topic := params["topic"]
	if !e4.IsValidTopic(topic) {
		resp.Text(http.StatusNotFound, "invalid topic")
		return
	}

	ret := s.removeTopic(topic)
	if ret != nil {
		resp.Text(http.StatusNotFound, "removeTopic failed")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *C2) handleNewClientKey(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	resp := Response{w}

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
		return
	}

	ret := s.newClientKey(id)
	if ret != nil {
		resp.Text(http.StatusNotFound, "newClientKey failed")
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

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
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

	id, err := hex.DecodeString(params["id"])
	if err != nil || !e4.IsValidID(id) {
		resp.Text(http.StatusNotFound, "invalid ID")
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

	if !e4.IsValidTopic(topic) {
		resp.Text(http.StatusNotFound, "invalid topic")
		return
	}

	err := s.sendMessage(topic, message)
	if err != nil {
		resp.Text(http.StatusNotFound, "message not sent")
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
