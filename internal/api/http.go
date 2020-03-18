package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	log "github.com/sirupsen/logrus"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/teserakt-io/c2/internal/config"
	"github.com/teserakt-io/c2/pkg/pb"
)

// HTTPServer defines methods available on a C2 HTTP server
type HTTPServer interface {
	ListenAndServe(ctx context.Context) error
}

type httpServer struct {
	logger       log.FieldLogger
	cfg          config.HTTPServerCfg
	grpcCertPath string
}

var _ HTTPServer = (*httpServer)(nil)

// NewHTTPServer creates a new http server for C2
func NewHTTPServer(scfg config.HTTPServerCfg, grpcCertPath string, logger log.FieldLogger) HTTPServer {
	return &httpServer{
		logger:       logger,
		cfg:          scfg,
		grpcCertPath: grpcCertPath,
	}
}

func (s *httpServer) ListenAndServe(ctx context.Context) error {
	tlsCert, err := tls.LoadX509KeyPair(s.cfg.Cert, s.cfg.Key)
	if err != nil {
		return err
	}

	tlsConfig := &tls.Config{Certificates: []tls.Certificate{tlsCert},
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	creds, err := credentials.NewClientTLSFromFile(s.grpcCertPath, "")
	if err != nil {
		return fmt.Errorf("failed to create TLS credentials from %v: %v", s.grpcCertPath, err)
	}

	httpMux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{OrigName: true, EmitDefaults: true}))
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithStatsHandler(&ocgrpc.ClientHandler{})}
	err = pb.RegisterC2HandlerFromEndpoint(ctx, httpMux, s.cfg.GRPCAddr, opts)
	if err != nil {
		return fmt.Errorf("failed to register http listener: %v", err)
	}

	och := &ochttp.Handler{Handler: httpMux}

	apiServer := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      och,
		TLSConfig:    tlsConfig,
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", s.cfg.Addr)
	if err != nil {
		s.logger.WithError(err).Error("failed to listen")

		return err
	}

	s.logger.WithField("addr", s.cfg.Addr).Info("starting http listener")

	return apiServer.ServeTLS(lis, s.cfg.Cert, s.cfg.Key)
}
