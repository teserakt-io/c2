// Copyright 2020 Teserakt AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/stats/view"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/internal/events"
	"github.com/teserakt-io/c2/internal/services"
	"github.com/teserakt-io/c2/pkg/pb"
)

// Request parameters validation errors
var (
	ErrClientRequired     = status.Errorf(codes.InvalidArgument, "a client is required.")
	ErrClientNameRequired = status.Errorf(codes.InvalidArgument, "a client name is required.")
)

// GRPCServer defines available endpoints on a GRPC server
type GRPCServer interface {
	pb.C2Server
	ListenAndServe(ctx context.Context) error
}

type grpcServer struct {
	logger          log.FieldLogger
	cfg             config.ServerCfg
	e4Service       services.E4
	eventDispatcher events.Dispatcher
}

var _ GRPCServer = (*grpcServer)(nil)

// NewGRPCServer creates a new server over GRPC
func NewGRPCServer(scfg config.ServerCfg, e4Service services.E4, eventDispatcher events.Dispatcher, logger log.FieldLogger) GRPCServer {
	return &grpcServer{
		cfg:             scfg,
		logger:          logger,
		e4Service:       e4Service,
		eventDispatcher: eventDispatcher,
	}
}

func (s *grpcServer) ListenAndServe(ctx context.Context) error {
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		s.logger.WithError(err).Error("failed to listen")

		return err
	}

	logFields := log.Fields{
		"cert": s.cfg.Cert,
		"key":  s.cfg.Key,
	}

	creds, err := credentials.NewServerTLSFromFile(s.cfg.Cert, s.cfg.Key)
	if err != nil {
		s.logger.WithError(err).WithFields(logFields).Error("failed to get credentials")
		return err
	}

	s.logger.WithFields(logFields).Info("using TLS for gRPC")

	if err = view.Register(ocgrpc.DefaultServerViews...); err != nil {
		s.logger.WithError(err).Error("failed to register ocgrpc server views")

		return err
	}

	srv := grpc.NewServer(grpc.Creds(creds), grpc.StatsHandler(&ocgrpc.ServerHandler{}))
	pb.RegisterC2Server(srv, s)

	s.logger.WithField("addr", s.cfg.Addr).Info("starting api grpc server")

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
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	if len(req.Client.Name) == 0 {
		return nil, ErrClientNameRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.NewClient(ctx, req.Client.Name, id, req.Key)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.NewClientResponse{}, nil
}

func (s *grpcServer) RemoveClient(ctx context.Context, req *pb.RemoveClientRequest) (*pb.RemoveClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.RemoveClient(ctx, id)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.RemoveClientResponse{}, nil
}

func (s *grpcServer) ResetClient(ctx context.Context, req *pb.ResetClientRequest) (*pb.ResetClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.ResetClient(ctx, id)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.ResetClientResponse{}, nil
}

func (s *grpcServer) NewClientKey(ctx context.Context, req *pb.NewClientKeyRequest) (*pb.NewClientKeyResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.NewClientKey(ctx, id)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.NewClientKeyResponse{}, nil
}

func (s *grpcServer) NewTopic(ctx context.Context, req *pb.NewTopicRequest) (*pb.NewTopicResponse, error) {
	err := s.e4Service.NewTopic(ctx, req.Topic)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.NewTopicResponse{}, nil
}

func (s *grpcServer) RemoveTopic(ctx context.Context, req *pb.RemoveTopicRequest) (*pb.RemoveTopicResponse, error) {
	err := s.e4Service.RemoveTopic(ctx, req.Topic)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.RemoveTopicResponse{}, nil
}

func (s *grpcServer) NewTopicClient(ctx context.Context, req *pb.NewTopicClientRequest) (*pb.NewTopicClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.NewTopicClient(ctx, id, req.Topic)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.NewTopicClientResponse{}, nil
}

func (s *grpcServer) RemoveTopicClient(ctx context.Context, req *pb.RemoveTopicClientRequest) (*pb.RemoveTopicClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	err = s.e4Service.RemoveTopicClient(ctx, id, req.Topic)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.RemoveTopicClientResponse{}, nil
}

func (s *grpcServer) CountTopicsForClient(ctx context.Context, req *pb.CountTopicsForClientRequest) (*pb.CountTopicsForClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	count, err := s.e4Service.CountTopicsForClient(ctx, id)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.CountTopicsForClientResponse{Count: int64(count)}, nil
}

func (s *grpcServer) GetTopicsForClient(ctx context.Context, req *pb.GetTopicsForClientRequest) (*pb.GetTopicsForClientResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	topics, err := s.e4Service.GetTopicsRangeByClient(ctx, id, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.GetTopicsForClientResponse{Topics: topics}, nil
}

func (s *grpcServer) CountClientsForTopic(ctx context.Context, req *pb.CountClientsForTopicRequest) (*pb.CountClientsForTopicResponse, error) {
	count, err := s.e4Service.CountClientsForTopic(ctx, req.Topic)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.CountClientsForTopicResponse{Count: int64(count)}, nil
}

func (s *grpcServer) GetClientsForTopic(ctx context.Context, req *pb.GetClientsForTopicRequest) (*pb.GetClientsForTopicResponse, error) {
	clients, err := s.e4Service.GetClientsRangeByTopic(ctx, req.Topic, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, grpcError(err)
	}

	pbClients := make([]*pb.Client, 0, len(clients))
	for _, client := range clients {
		pbClients = append(pbClients, &pb.Client{Name: client.Name})
	}

	return &pb.GetClientsForTopicResponse{Clients: pbClients}, nil
}

func (s *grpcServer) GetClients(ctx context.Context, req *pb.GetClientsRequest) (*pb.GetClientsResponse, error) {
	clients, err := s.e4Service.GetClientsRange(ctx, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, grpcError(err)
	}

	pbClients := make([]*pb.Client, 0, len(clients))
	for _, client := range clients {
		pbClients = append(pbClients, &pb.Client{Name: client.Name})
	}

	return &pb.GetClientsResponse{Clients: pbClients}, nil
}
func (s *grpcServer) GetTopics(ctx context.Context, req *pb.GetTopicsRequest) (*pb.GetTopicsResponse, error) {
	topics, err := s.e4Service.GetTopicsRange(ctx, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.GetTopicsResponse{Topics: topics}, nil
}

func (s *grpcServer) CountClients(ctx context.Context, req *pb.CountClientsRequest) (*pb.CountClientsResponse, error) {
	count, err := s.e4Service.CountClients(ctx)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.CountClientsResponse{Count: int64(count)}, nil
}

func (s *grpcServer) CountTopics(ctx context.Context, req *pb.CountTopicsRequest) (*pb.CountTopicsResponse, error) {
	count, err := s.e4Service.CountTopics(ctx)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.CountTopicsResponse{Count: int64(count)}, nil
}

func (s *grpcServer) LinkClient(ctx context.Context, req *pb.LinkClientRequest) (*pb.LinkClientResponse, error) {
	if req.SourceClient == nil {
		return nil, errors.New("source client name is required")
	}
	if req.TargetClient == nil {
		return nil, errors.New("target client name is required")
	}

	sourceClientID, err := validateE4NameOrIDPair(req.SourceClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	targetClientID, err := validateE4NameOrIDPair(req.TargetClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	if err := s.e4Service.LinkClient(ctx, sourceClientID, targetClientID); err != nil {
		return nil, grpcError(err)
	}

	return &pb.LinkClientResponse{}, nil
}

func (s *grpcServer) UnlinkClient(ctx context.Context, req *pb.UnlinkClientRequest) (*pb.UnlinkClientResponse, error) {
	if req.SourceClient == nil {
		return nil, errors.New("source client name is required")
	}
	if req.TargetClient == nil {
		return nil, errors.New("target client name is required")
	}

	sourceClientID, err := validateE4NameOrIDPair(req.SourceClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	targetClientID, err := validateE4NameOrIDPair(req.TargetClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	if err := s.e4Service.UnlinkClient(ctx, sourceClientID, targetClientID); err != nil {
		return nil, grpcError(err)
	}

	return &pb.UnlinkClientResponse{}, nil
}

func (s *grpcServer) CountLinkedClients(ctx context.Context, req *pb.CountLinkedClientsRequest) (*pb.CountLinkedClientsResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	count, err := s.e4Service.CountLinkedClients(ctx, id)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.CountLinkedClientsResponse{
		Count: int64(count),
	}, nil
}

func (s *grpcServer) GetLinkedClients(ctx context.Context, req *pb.GetLinkedClientsRequest) (*pb.GetLinkedClientsResponse, error) {
	if req.Client == nil {
		return nil, ErrClientRequired
	}

	id, err := validateE4NameOrIDPair(req.Client.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	linkedPairs, err := s.e4Service.GetLinkedClients(ctx, id, int(req.Offset), int(req.Count))
	if err != nil {
		return nil, grpcError(err)
	}

	pbClients := make([]*pb.Client, 0, len(linkedPairs))
	for _, pair := range linkedPairs {
		pbClients = append(pbClients, &pb.Client{Name: pair.Name})
	}

	return &pb.GetLinkedClientsResponse{
		Clients: pbClients,
	}, nil
}

func (s *grpcServer) SendClientPubKey(ctx context.Context, req *pb.SendClientPubKeyRequest) (*pb.SendClientPubKeyResponse, error) {
	if req.SourceClient == nil {
		return nil, errors.New("source client name is required")
	}
	if req.TargetClient == nil {
		return nil, errors.New("target client name is required")
	}

	sourceClientID, err := validateE4NameOrIDPair(req.SourceClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	targetClientID, err := validateE4NameOrIDPair(req.TargetClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	if err := s.e4Service.SendClientPubKey(ctx, sourceClientID, targetClientID); err != nil {
		return nil, grpcError(err)
	}

	return &pb.SendClientPubKeyResponse{}, nil
}

func (s *grpcServer) RemoveClientPubKey(ctx context.Context, req *pb.RemoveClientPubKeyRequest) (*pb.RemoveClientPubKeyResponse, error) {
	if req.SourceClient == nil {
		return nil, errors.New("source client name is required")
	}
	if req.TargetClient == nil {
		return nil, errors.New("target client name is required")
	}

	sourceClientID, err := validateE4NameOrIDPair(req.SourceClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	targetClientID, err := validateE4NameOrIDPair(req.TargetClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	if err := s.e4Service.RemoveClientPubKey(ctx, sourceClientID, targetClientID); err != nil {
		return nil, grpcError(err)
	}

	return &pb.RemoveClientPubKeyResponse{}, nil
}

func (s *grpcServer) ResetClientPubKeys(ctx context.Context, req *pb.ResetClientPubKeysRequest) (*pb.ResetClientPubKeysResponse, error) {
	if req.TargetClient == nil {
		return nil, errors.New("target client name is required")
	}

	targetClientID, err := validateE4NameOrIDPair(req.TargetClient.Name, nil)
	if err != nil {
		return nil, grpcError(err)
	}

	if err := s.e4Service.ResetClientPubKeys(ctx, targetClientID); err != nil {
		return nil, grpcError(err)
	}

	return &pb.ResetClientPubKeysResponse{}, nil
}

func (s *grpcServer) NewC2Key(ctx context.Context, req *pb.NewC2KeyRequest) (*pb.NewC2KeyResponse, error) {
	if !req.Force {
		return nil, grpcError(errors.New("force is required to true to prevent accidental executions"))
	}

	if err := s.e4Service.NewC2Key(ctx); err != nil {
		return nil, grpcError(err)
	}

	return &pb.NewC2KeyResponse{}, nil
}

func (s *grpcServer) ProtectMessage(ctx context.Context, req *pb.ProtectMessageRequest) (*pb.ProtectMessageResponse, error) {
	if len(req.BinaryData) == 0 {
		return nil, errors.New("binary data cannot be empty")
	}

	protected, err := s.e4Service.ProtectMessage(ctx, req.Topic, req.BinaryData)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.ProtectMessageResponse{
		Topic:               req.Topic,
		ProtectedBinaryData: protected,
	}, nil
}

func (s *grpcServer) UnprotectMessage(ctx context.Context, req *pb.UnprotectMessageRequest) (*pb.UnprotectMessageResponse, error) {
	if len(req.ProtectedBinaryData) == 0 {
		return nil, errors.New("protected binary data cannot be empty")
	}

	binaryData, err := s.e4Service.UnprotectMessage(ctx, req.Topic, req.ProtectedBinaryData)
	if err != nil {
		return nil, grpcError(err)
	}

	return &pb.UnprotectMessageResponse{
		Topic:      req.Topic,
		BinaryData: binaryData,
	}, nil
}

func (s *grpcServer) SubscribeToEventStream(req *pb.SubscribeToEventStreamRequest, srv pb.C2_SubscribeToEventStreamServer) error {
	listener := events.NewListener(s.eventDispatcher)
	defer listener.Close()

	ctx := srv.Context()
	grpcPeer := peerFromContext(ctx)
	logger := s.logger.WithField("client", grpcPeer.Addr)

	logger.Info("started new event stream")
	defer func() {
		logger.Warn("event stream closed")
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case evt := <-listener.C():
			ts, err := ptypes.TimestampProto(evt.Timestamp)
			if err != nil {
				return fmt.Errorf("failed to convert time.Time to proto.Timestamp: %v", err)
			}

			pbEvt := &pb.Event{
				Type:      pb.EventType(evt.Type),
				Source:    evt.Source,
				Target:    evt.Target,
				Timestamp: ts,
			}

			if err := srv.Send(pbEvt); err != nil {
				logger.WithError(err).Error("failed to send event")
				return err
			}

			logger.WithField("eventType", pbEvt.Type.String()).Info("successfully sent event")
		}
	}
}

func (s *grpcServer) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Code:   0,
		Status: "OK",
	}, nil
}

// validateE4NamedOrIDPair wrap around services.ValidateE4NameOrIDPair but will
// convert the error to a suitable GRPC error
func validateE4NameOrIDPair(name string, id []byte) ([]byte, error) {
	id, err := services.ValidateE4NameOrIDPair(name, id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	return id, nil
}

// grpcError will convert the given error to a GRPC error with appropriate code
func grpcError(err error) error {
	var code codes.Code
	switch err.(type) {
	case services.ErrClientNotFound, services.ErrTopicNotFound:
		code = codes.NotFound
	case services.ErrValidation:
		code = codes.InvalidArgument
	default:
		code = codes.Internal
	}

	return status.Errorf(code, "%s", err.Error())
}

func peerFromContext(ctx context.Context) *peer.Peer {
	p, ok := peer.FromContext(ctx)
	if !ok {
		p = &peer.Peer{}
	}

	return p
}
