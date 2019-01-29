package main

import (
	"errors"

	e4 "gitlab.com/teserakt/e4common"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
)

func (s *C2) gRPCnewClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewClient")
	defer span.End()

	err := checkRequest(ctx, in, true, true, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClient(in.Id, in.Key)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopicClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewTopicClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopicClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveTopicClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCresetClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCresetClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.resetClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewTopic(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewTopic")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCremoveTopic(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveTopic")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.removeTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCnewClientKey(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewClientKey")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.newClientKey(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *C2) gRPCgetClients(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClients")
	defer span.End()

	ids, err := s.dbGetIDListHex()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: ids}, nil
}

func (s *C2) gRPCgetTopics(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopics")
	defer span.End()

	topics, err := s.dbGetTopicsList()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetClientTopicCount(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClientTopicCount")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.dbCountTopicsForID(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetClientTopics(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClientTopics")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	topics, err := s.dbGetTopicsForID(in.Id, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *C2) gRPCgetTopicClientCount(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopicClientCount")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.dbCountIDsForTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *C2) gRPCgetTopicClients(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopicClients")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	clients, err := s.dbGetIdsforTopic(in.Topic, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: clients}, nil
}

func (s *C2) gRPCsendMessage(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCsendMessage")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
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
func checkRequest(ctx context.Context, in *e4.C2Request, needID, needKey, needTopic, needOffsetCount bool) error {

	ctx, span := trace.StartSpan(ctx, "checkRequest")
	defer span.End()

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
