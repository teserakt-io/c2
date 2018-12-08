package main

import (
	"errors"

	e4 "gitlab.com/teserakt/e4common"
)

func (s *C2) gRPCnewClient(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, true, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClient(in.Id, in.Key)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveClient(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopicClient(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopicClient(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCresetClient(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.resetClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopic(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopic(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewClientKey(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClientKey(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCgetClients(in *e4.C2Request) (*e4.C2Response, error) {
	ids, err := s.dbGetIDListHex()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: ids}, nil
}

func (s *C2) gRPCgetTopics(in *e4.C2Request) (*e4.C2Response, error) {
	topics, err := s.dbGetTopicsList()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetClientTopicCount(in *e4.C2Request) (*e4.C2Response, error) {
	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.dbCountTopicsForID(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetClientTopics(in *e4.C2Request) (*e4.C2Response, error) {
	err := checkRequest(in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	topics, err := s.dbGetTopicsForID(in.Id, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetTopicClientCount(in *e4.C2Request) (*e4.C2Response, error) {
	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.dbCountIDsForTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetTopicClients(in *e4.C2Request) (*e4.C2Response, error) {
	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	clients, err := s.dbGetIdsforTopic(in.Topic, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: clients}, nil
}

func (s *C2) gRPCsendMessage(in *e4.C2Request) (*e4.C2Response, error) {

	err := checkRequest(in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.sendMessage(in.Topic, in.Msg)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

// helper to check inputs' sanity
func checkRequest(in *e4.C2Request, needID, needKey, needTopic, needOffsetCount bool) error {
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
