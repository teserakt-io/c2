package c2test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	e4 "gitlab.com/teserakt/e4common"
)

// NewTestingHTTPClient Create a new http client for testing
func NewTestingHTTPClient() *http.Client {
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
		InsecureSkipVerify: true,
	}

	httpTransport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: httpTransport,
	}
}

// NewTestingGRPCClient returns a new GRPC client to use for testing
func NewTestingGRPCClient(relativeCertPath string, serverAddr string) (e4.C2Client, func() error, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get current working directory: %v", err)
	}

	cert := path.Join(wd, relativeCertPath)
	creds, err := credentials.NewClientTLSFromFile(cert, "")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create TLS credentials from %v: %v", cert, err)
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to gRPC server: %v", err)
	}

	return e4.NewC2Client(conn), conn.Close, nil
}
