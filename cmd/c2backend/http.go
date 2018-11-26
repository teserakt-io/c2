package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	e4 "teserakt/e4/common"
)

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

	count, err := s.countTopicsForID(id)
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

	topics, err := s.getTopicsForID(id, int(offset), int(count))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
	}

	json.NewEncoder(w).Encode(&topics)
}

func (s *C2) handleGetTopicClientCount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	params := mux.Vars(r)

	count, err := s.countIDsForTopic(params["topic"])
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

	clients, err := s.getIdsforTopic(topic, int(offset), int(count))
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
