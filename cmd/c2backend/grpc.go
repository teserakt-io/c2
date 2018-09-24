package main

import (
	"errors"

	pb "teserakt/e4go/pkg/c2proto"
	e4 "teserakt/e4go/pkg/e4common"
)

func (s *C2) gRPCnewClient(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, true, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClient(in.Id, in.Key)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveClient(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeClient(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopicClient(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopicClient(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopicClient(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCresetClient(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.resetClient(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopic(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopic(in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopic(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopic(in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewClientKey(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClientKey(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCgetClients(in *pb.C2Request) (*pb.C2Response, error) {
	ids, err := s.dbGetIDListHex()
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Ids: ids}, nil
}

func (s *C2) gRPCgetTopics(in *pb.C2Request) (*pb.C2Response, error) {
	topics, err := s.dbGetTopicsList()
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetClientTopicCount(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.countTopicsForID(in.Id)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetClientTopics(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	topics, err := s.getTopicsForID(in.Id, int(in.Offset), int(in.Count))
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetTopicClientCount(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.countIDsForTopic(in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetTopicClients(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	clients, err := s.getIdsforTopic(in.Topic, int(in.Offset), int(in.Count))
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: "", Ids: clients}, nil
}

func (s *C2) gRPClinkClientTopic(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.linkIDTopic(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCunlinkClientTopic(in *pb.C2Request) (*pb.C2Response, error) {
	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.unlinkIDTopic(in.Id, in.Topic)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCsendMessage(in *pb.C2Request) (*pb.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.sendMessage(in.Topic, in.Msg)
	if err != nil {
		return &pb.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &pb.C2Response{Success: true, Err: ""}, nil
}

// helper to check inputs' sanity
func checkRequest(in *pb.C2Request, needID, needKey, needTopic, needOffsetCount bool) error {
	if needID {
		if !e4.IsValidID(in.Id) {
			return errors.New("invalid id")
		}
	} else {
		if in.Id != nil {
			return errors.New("unexpected id")
		}
	}
	if needKey {
		if !e4.IsValidKey(in.Key) {
			return errors.New("invalid key")
		}
	} else {
		if in.Key != nil {
			return errors.New("unexpected key")
		}
	}
	if needTopic {
		if !e4.IsValidTopic(in.Topic) {
			return errors.New("invalid topic")
		}
	} else {
		if in.Topic != "" {
			return errors.New("unexpected topic")
		}
	}
	if needOffsetCount {
		if in.Count == 0 {
			return errors.New("No data to return with zero count")
		}
	}
	return nil
}
