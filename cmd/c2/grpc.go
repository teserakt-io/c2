package main

import (
	"errors"
	"net"

	"github.com/go-kit/kit/log"
	e4 "gitlab.com/teserakt/e4common"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"gitlab.com/teserakt/c2/internal/config"
)

func (s *C2) createGRPCServer(scfg config.ServerCfg) error {

	var logger = log.With(s.logger, "protocol", "grpc")
	logger.Log("addr", scfg.Addr)

	lis, err := net.Listen("tcp", scfg.Addr)
	if err != nil {
		logger.Log("msg", "failed to listen", "error", err)
		return err
	}

	creds, err := credentials.NewServerTLSFromFile(scfg.Cert, scfg.Key)
	if err != nil {
		logger.Log("msg", "failed to get credentials", "cert", scfg.Cert, "key", scfg.Key, "error", err)
		return err
	}
	logger.Log("msg", "using TLS for gRPC", "cert", scfg.Cert, "key", scfg.Key, "error", err)

	if err = view.Register(ocgrpc.DefaultServerViews...); err != nil {
		logger.Log("msg", "failed to register ocgrpc server views", "error", err)
		return err
	}

	srv := grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(&ocgrpc.ServerHandler{}))

	e4.RegisterC2Server(srv, s)

	count, err := s.dbCountIDKeys()
	if err != nil {
		logger.Log("msg", "failed to count id keys", "error", err)
		return err
	}
	logger.Log("nbidkeys", count)
	count, err = s.dbCountTopicKeys()
	if err != nil {
		logger.Log("msg", "failed to count topic keys", "error", err)
		return err
	}

	logger.Log("nbtopickeys", count)
	logger.Log("msg", "starting grpc server")
	return srv.Serve(lis)
}

// C2Command processes a command received over gRPC by the CLI tool.
func (s *C2) C2Command(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {
	//log.Printf("command received: %s", e4.C2Request_Command_name[int32(in.Command)])
	s.logger.Log("msg", "received gRPC request", "request", e4.C2Request_Command_name[int32(in.Command)])

	switch in.Command {
	case e4.C2Request_NEW_CLIENT:
		return s.gRPCnewClient(ctx, in)
	case e4.C2Request_REMOVE_CLIENT:
		return s.gRPCremoveClient(ctx, in)
	case e4.C2Request_NEW_TOPIC_CLIENT:
		return s.gRPCnewTopicClient(ctx, in)
	case e4.C2Request_REMOVE_TOPIC_CLIENT:
		return s.gRPCremoveTopicClient(ctx, in)
	case e4.C2Request_RESET_CLIENT:
		return s.gRPCresetClient(ctx, in)
	case e4.C2Request_NEW_TOPIC:
		return s.gRPCnewTopic(ctx, in)
	case e4.C2Request_REMOVE_TOPIC:
		return s.gRPCremoveTopic(ctx, in)
	case e4.C2Request_NEW_CLIENT_KEY:
		return s.gRPCnewClientKey(ctx, in)
	case e4.C2Request_SEND_MESSAGE:
		return s.gRPCsendMessage(ctx, in)
	case e4.C2Request_GET_CLIENTS:
		return s.gRPCgetClients(ctx, in)
	case e4.C2Request_GET_TOPICS:
		return s.gRPCgetTopics(ctx, in)
	case e4.C2Request_GET_CLIENT_TOPIC_COUNT:
		return s.gRPCgetClientTopicCount(ctx, in)
	case e4.C2Request_GET_CLIENT_TOPICS:
		return s.gRPCgetClientTopics(ctx, in)
	case e4.C2Request_GET_TOPIC_CLIENT_COUNT:
		return s.gRPCgetTopicClientCount(ctx, in)
	case e4.C2Request_GET_TOPIC_CLIENTS:
		return s.gRPCgetTopicClients(ctx, in)
	}
	return &e4.C2Response{Success: false, Err: "unknown command"}, nil
}

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
		if err := e4.IsValidID(in.Id); err != nil {
			return err
		}
	} else {
		if in.Id != nil {
			return errors.New("unexpected id")
		}
	}
	if needKey {
		if err := e4.IsValidKey(in.Key); err != nil {
			return err
		}
	} else {
		if in.Key != nil {
			return errors.New("unexpected key")
		}
	}
	if needTopic {
		if err := e4.IsValidTopic(in.Topic); err != nil {
			return err
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
