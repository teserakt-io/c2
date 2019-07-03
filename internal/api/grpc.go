package api

import (
	"context"
	"net"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/internal/services"
	"gitlab.com/teserakt/c2/pkg/pb"

	"github.com/go-kit/kit/log"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GRPCServer defines available endpoints on a GRPC server
type GRPCServer interface {
	pb.C2Server
	ListenAndServe(ctx context.Context) error
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

func (s *grpcServer) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
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
	pb.RegisterC2Server(srv, s)

	s.logger.Log("msg", "Starting api grpc server", "addr", s.cfg.Addr)

	errc := make(chan error)
	go func() {
		errc <- srv.Serve(lis)
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *grpcServer) NewClient(ctx context.Context, req *pb.NewClientRequest) (*pb.NewClientResponse, error) {
	err := s.e4Service.NewClient(ctx, req.Name, req.Id, req.Key)
	if err != nil {
		return nil, err
	}

	return &pb.NewClientResponse{}, nil
}

func (s *grpcServer) RemoveClient(ctx context.Context, req *pb.RemoveClientRequest) (*pb.RemoveClientResponse, error) {
	err := s.e4Service.RemoveClientByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveClientResponse{}, nil
}

func (s *grpcServer) ResetClient(ctx context.Context, req *pb.ResetClientRequest) (*pb.ResetClientResponse, error) {
	err := s.e4Service.ResetClientByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &pb.ResetClientResponse{}, nil
}

func (s *grpcServer) NewClientKey(ctx context.Context, req *pb.NewClientKeyRequest) (*pb.NewClientKeyResponse, error) {
	err := s.e4Service.NewClientKey(ctx, req.Name, req.Id)
	if err != nil {
		return nil, err
	}

	return &pb.NewClientKeyResponse{}, nil
}

func (s *grpcServer) NewTopic(ctx context.Context, req *pb.NewTopicRequest) (*pb.NewTopicResponse, error) {
	err := s.e4Service.NewTopic(ctx, req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.NewTopicResponse{}, nil
}

func (s *grpcServer) RemoveTopic(ctx context.Context, req *pb.RemoveTopicRequest) (*pb.RemoveTopicResponse, error) {
	err := s.e4Service.RemoveTopic(ctx, req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveTopicResponse{}, nil
}

func (s *grpcServer) NewTopicClient(ctx context.Context, req *pb.NewTopicClientRequest) (*pb.NewTopicClientResponse, error) {
	err := s.e4Service.NewTopicClient(ctx, req.Name, req.Id, req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.NewTopicClientResponse{}, nil
}

func (s *grpcServer) RemoveTopicClient(ctx context.Context, req *pb.RemoveTopicClientRequest) (*pb.RemoveTopicClientResponse, error) {
	err := s.e4Service.RemoveTopicClientByName(ctx, req.Name, req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.RemoveTopicClientResponse{}, nil
}

func (s *grpcServer) CountTopicsForClient(ctx context.Context, req *pb.CountTopicsForClientRequest) (*pb.CountTopicsForClientResponse, error) {
	count, err := s.e4Service.CountTopicsForClientByName(ctx, req.Name)
	if err != nil {
		return nil, err
	}

	return &pb.CountTopicsForClientResponse{Count: int64(count)}, nil
}

func (s *grpcServer) GetTopicsForClient(ctx context.Context, req *pb.GetTopicsForClientRequest) (*pb.GetTopicsForClientResponse, error) {
	topics, err := s.e4Service.GetTopicsForClientByName(ctx, req.Name, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, err
	}

	return &pb.GetTopicsForClientResponse{Topics: topics}, nil
}

func (s *grpcServer) CountClientsForTopic(ctx context.Context, req *pb.CountClientsForTopicRequest) (*pb.CountClientsForTopicResponse, error) {
	count, err := s.e4Service.CountClientsForTopic(ctx, req.Topic)
	if err != nil {
		return nil, err
	}

	return &pb.CountClientsForTopicResponse{Count: int64(count)}, nil
}

func (s *grpcServer) GetClientsForTopic(ctx context.Context, req *pb.GetClientsForTopicRequest) (*pb.GetClientsForTopicResponse, error) {
	names, err := s.e4Service.GetClientsByNameForTopic(ctx, req.Topic, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, err
	}

	return &pb.GetClientsForTopicResponse{Names: names}, nil
}

func (s *grpcServer) GetClients(ctx context.Context, req *pb.GetClientsRequest) (*pb.GetClientsResponse, error) {
	names, err := s.e4Service.GetClientsAsNamesRange(ctx, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, err
	}

	return &pb.GetClientsResponse{Names: names}, nil
}
func (s *grpcServer) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	topics, err := s.e4Service.GetTopicsRange(ctx, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, err
	}

	return &pb.GetTopicsResponse{Topics: topics}, nil
}

func (s *grpcServer) SendMessage(ctx context.Context, req *pb.SendMessageRequest) (*pb.SendMessageResponse, error) {
	err := s.e4Service.SendMessage(ctx, req.Topic, req.Message)
	if err != nil {
		return nil, err
	}

	return &pb.SendMessageResponse{}, nil
}
