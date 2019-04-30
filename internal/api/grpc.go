package api

import (
	"context"
	"errors"
	"net"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/services"
	e4 "gitlab.com/teserakt/e4common"

	"github.com/go-kit/kit/log"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer defines available endpoints on a GRPC server
type GRPCServer interface {
	e4.C2Server
	ListenAndServe() error
}

type grpcServer struct {
	logger    log.Logger
	cfg       config.ServerCfg
	e4Service services.E4
}

var _ GRPCServer = &grpcServer{}

// NewGRPCServer creates a new server over GRPC
func NewGRPCServer(scfg config.ServerCfg, e4Service services.E4, logger log.Logger) GRPCServer {
	return &grpcServer{
		cfg:       scfg,
		logger:    logger,
		e4Service: e4Service,
	}
}

func (s *grpcServer) ListenAndServe() error {
	lis, err := net.Listen("tcp", s.cfg.Addr)
	if err != nil {
		s.logger.Log("msg", "failed to listen", "error", err)

		return err
	}

	creds, err := credentials.NewServerTLSFromFile(s.cfg.Cert, s.cfg.Key)
	if err != nil {
		s.logger.Log("msg", "failed to get credentials", "cert", s.cfg.Cert, "key", s.cfg.Key, "error", err)
		return err
	}

	s.logger.Log("msg", "using TLS for gRPC", "cert", s.cfg.Cert, "key", s.cfg.Key)

	if err = view.Register(ocgrpc.DefaultServerViews...); err != nil {
		s.logger.Log("msg", "failed to register ocgrpc server views", "error", err)

		return err
	}

	srv := grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	e4.RegisterC2Server(srv, s)

	s.logger.Log("msg", "Starting api grpc server", "addr", s.cfg.Addr)

	return srv.Serve(lis)
}

// C2Command processes a command received over gRPC by the CLI tool.
func (s *grpcServer) C2Command(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {
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

func (s *grpcServer) gRPCnewClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewClient")
	defer span.End()

	err := checkRequest(ctx, in, true, true, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.NewClient(in.Id, in.Key)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCremoveClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.RemoveClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCnewTopicClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewTopicClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.NewTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCremoveTopicClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveTopicClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.RemoveTopicClient(in.Id, in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCresetClient(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCresetClient")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.ResetClient(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCnewTopic(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewTopic")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.NewTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCremoveTopic(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCremoveTopic")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.RemoveTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCnewClientKey(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCnewClientKey")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.NewClientKey(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: ""}, nil
}

func (s *grpcServer) gRPCgetClients(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClients")
	defer span.End()

	ids, err := s.e4Service.GetAllClientHexIds()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: ids}, nil
}

func (s *grpcServer) gRPCgetTopics(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopics")
	defer span.End()

	topics, err := s.e4Service.GetAllTopicIds()
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *grpcServer) gRPCgetClientTopicCount(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClientTopicCount")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.e4Service.CountTopicsForID(in.Id)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *grpcServer) gRPCgetClientTopics(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetClientTopics")
	defer span.End()

	err := checkRequest(ctx, in, true, false, false, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	topics, err := s.e4Service.GetTopicsForID(in.Id, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Topics: topics}, nil
}

func (s *grpcServer) gRPCgetTopicClientCount(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopicClientCount")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	count, err := s.e4Service.CountIDsForTopic(in.Topic)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Count: uint64(count)}, nil
}

func (s *grpcServer) gRPCgetTopicClients(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCgetTopicClients")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	clients, err := s.e4Service.GetIdsforTopic(in.Topic, int(in.Offset), int(in.Count))
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	return &e4.C2Response{Success: true, Err: "", Ids: clients}, nil
}

func (s *grpcServer) gRPCsendMessage(ctx context.Context, in *e4.C2Request) (*e4.C2Response, error) {

	ctx, span := trace.StartSpan(ctx, "gRPCsendMessage")
	defer span.End()

	err := checkRequest(ctx, in, false, false, true, false)
	if err != nil {
		return &e4.C2Response{Success: false, Err: err.Error()}, nil
	}

	err = s.e4Service.SendMessage(in.Topic, in.Msg)
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
