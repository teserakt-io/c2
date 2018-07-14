package main

import (
	"encoding/hex"
	"log"

	pb "teserakt/c2proto"
	e4 "teserakt/e4common"
)

func (s *C2) gRPCnewClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, true, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.newClient(in.Id, in.Key)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.removeClient(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	log.Printf("removed client %s", hex.EncodeToString(in.Id))
	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.newTopicClient(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.removeTopicClient(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCresetClient(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.resetClient(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopic(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, false, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.newTopic(in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopic(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, false, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.removeTopic(in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewClientKey(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, true, false, false) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.newClientKey(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCsendMessage(in *pb.C2Request) (*pb.C2Response, error) {

	if !checkRequest(in, false, false, true) {
		return &pb.C2Response{Success: false, Err: "invalid request"}, nil
	}

	err := s.sendMessage(in.Topic, in.Msg)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

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
