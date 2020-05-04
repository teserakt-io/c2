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

package c2test

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/teserakt-io/c2/pkg/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
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
func NewTestingGRPCClient(relativeCertPath string, serverAddr string) (pb.C2Client, func() error, error) {
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

	return pb.NewC2Client(conn), conn.Close, nil
}
