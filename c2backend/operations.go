package main

import (
	"encoding/hex"
	"log"

	pb "teserakt/c2proto"
	e4 "teserakt/e4common"
)

// helper to check inputs' sanity
func checkRequest(in *pb.C2Request, needID, needKey, needTopic bool) bool {
	if needID {
		if !e4.IsValidID(in.Id) {
			log.Print("invalid id: ", hex.EncodeToString(in.Key))
			return false
		}
	} else {
		if in.Id != nil {
			log.Print("unexpected id: ", hex.EncodeToString(in.Key))
			return false
		}
	}
	if needKey {
		if !e4.IsValidKey(in.Key) {
			log.Print("invalid key")
			return false
		}
	} else {
		if in.Key != nil {
			log.Print("unexpected key")
			return false
		}
	}
	if needTopic {
		if !e4.IsValidTopic(in.Topic) {
			log.Printf("invalid topic: %s", in.Topic)
			return false
		}
	} else {
		if in.Topic != "" {
			log.Printf("unexpected topic: %s", in.Topic)
			return false
		}
	}
	return true
}

// local
func (s *C2) newClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, true, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.insertIDKey(in.Id, in.Key)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "db update failed"}, nil
	}
	log.Printf("added client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

// local
func (s *C2) removeClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.deleteIDKey(in.Id)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "deletion error"}, nil
	}

	log.Printf("removed client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

// remote
func (s *C2) newTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	topichash := e4.HashTopic(in.Topic)
	key, err := s.getTopicKey(topichash)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "unknown topic"}, nil
	}

	payload, err := s.CreateAndProtectForID(e4.SetTopicKey, topichash, key, in.Id)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "payload creation failed"}, nil
	}
	err = s.sendToClient(in.Id, payload)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "mqtt publish fail"}, nil
	}

	log.Printf("added topic '%s' to client %s", in.Topic, hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

// remote
func (s *C2) removeTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	topichash := e4.HashTopic(in.Topic)

	payload, err := s.CreateAndProtectForID(e4.RemoveTopic, topichash, nil, in.Id)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "payload creation failed"}, nil
	}
	err = s.sendToClient(in.Id, payload)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "mqtt publish fail"}, nil
	}

	log.Printf("removed topic '%s' from client %s", in.Topic, hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) resetClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	payload, err := s.CreateAndProtectForID(e4.ResetTopics, nil, nil, in.Id)
	err = s.sendToClient(in.Id, payload)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "mqtt publish fail"}, nil
	}

	log.Printf("reset client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

// local
func (s *C2) newTopic(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, false, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}
	topichash := e4.HashTopic(in.Topic)
	key := e4.RandomKey()

	err := s.insertTopicKey(topichash, key)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "db update failed"}, nil
	}
	log.Printf("added topic %s", in.Topic)
	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) removeTopic(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}
	topichash := e4.HashTopic(in.Topic)

	err := s.deleteTopicKey(topichash)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "deletion error"}, nil
	}

	log.Printf("removed topic %s", in.Topic)
	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) newClientKey(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}
	key := e4.RandomKey()

	// first send to the client, and only update locally afterwards
	payload, err := s.CreateAndProtectForID(e4.SetIDKey, nil, key, in.Id)
	err = s.sendToClient(in.Id, payload)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "mqtt publish fail"}, nil
	}

	err = s.insertIDKey(in.Id, key)
	if err != nil {
		log.Print(err)
		return &pb.C2Response{Success: false, Err: "db update failed"}, nil
	}
	log.Printf("updated key for client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}
