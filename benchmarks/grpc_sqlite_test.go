package benchmarks

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"gitlab.com/teserakt/c2/internal/config"
	"gitlab.com/teserakt/c2/pkg/c2"
	e4 "gitlab.com/teserakt/e4common"
	e4test "gitlab.com/teserakt/test-common"
)

// Generate dummy PEM key and cert and returns path to each file.
func generatePEM(b *testing.B) (keyFilename, certFilename string) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		b.Fatal("Private key cannot be created.", err.Error())
	}

	// Generate a pem block with the private key
	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	tml := x509.Certificate{
		// you can add any attr that you need
		NotBefore: time.Now(),
		NotAfter:  time.Now().AddDate(5, 0, 0),
		// you have to generate a different serial number each execution
		SerialNumber: big.NewInt(123123),
		Subject: pkix.Name{
			CommonName:   "localhost",
			Organization: []string{"Teserakt"},
		},
		BasicConstraintsValid: true,
	}
	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		b.Fatalf("Certificate cannot be created: %v", err)
	}

	// Generate a pem block with the certificate
	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	keyFilename = tempFileName("c2bench_key")
	certFilename = tempFileName("c2bench_cert")

	keyFile, err := os.Create(keyFilename)
	if err != nil {
		b.Fatalf("Failed to create key file at %s: %v", keyFilename, err)
	}
	defer keyFile.Close()

	if _, err := keyFile.Write(keyPem); err != nil {
		b.Fatalf("Failed to write key file %s: %v", keyFilename, err)
	}

	certFile, err := os.Create(certFilename)
	if err != nil {
		b.Fatalf("Failed to create cert file at %s: %v", certFilename, err)
	}
	defer certFile.Close()

	if _, err := certFile.Write(certPem); err != nil {
		b.Fatalf("Failed to write cert file %s: %v", certFilename, err)
	}

	return keyFilename, certFilename
}

func tempFileName(prefix string) string {
	randBytes := make([]byte, 16)
	rand.Read(randBytes)
	return filepath.Join(os.TempDir(), prefix+hex.EncodeToString(randBytes))
}

func setup(dbFilename string, b *testing.B) (e4.C2Client, func(*testing.B)) {
	keyPath, certPath := generatePEM(b)

	cfg := config.Config{
		IsProd:  false,
		Monitor: false,
		GRPC: config.ServerCfg{
			Addr: "localhost:5555",
			Key:  keyPath,
			Cert: certPath,
		},
		HTTP: config.ServerCfg{},
		MQTT: config.MQTTCfg{
			Enabled:  true,
			ID:       "c2bench",
			Broker:   "tcp://localhost:1883",
			QoSPub:   2,
			QoSSub:   2,
			Username: "",
			Password: "",
		},
		DB: config.DBCfg{
			Type: config.DBTypeSQLite,
			File: dbFilename,
		},
	}

	c2server, err := c2.New(log.NewNopLogger(), cfg)
	if err != nil {
		b.Fatalf("cannot create C2 instance: %v", err)
	}

	c2server.EnableGRPCEndpoint()
	errChan := make(chan error)

	go func(errChan chan<- error) {
		errChan <- c2server.ListenAndServe()
	}(errChan)

	select {
	case err := <-errChan:
		b.Fatalf("Failed to start C2 server: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	creds, err := credentials.NewClientTLSFromFile(cfg.GRPC.Cert, "")
	if err != nil {
		b.Fatalf("failed to create TLS credentials from %v: %v", cfg.GRPC.Cert, err)
	}

	conn, err := grpc.Dial(cfg.GRPC.Addr, grpc.WithTransportCredentials(creds))
	if err != nil {
		b.Fatalf("failed to connect to gRPC server: %v", err)
	}

	return e4.NewC2Client(conn), func(b *testing.B) {
		conn.Close()
		c2server.Close()
		os.Remove(dbFilename)
		os.Remove(keyPath)
		os.Remove(certPath)
	}
}

func BenchmarkC2NewTopicKeys(b *testing.B) {
	clientCounts := []int{1, 10, 100, 1000, 10000}
	for _, clientCount := range clientCounts {
		b.Run(fmt.Sprintf("NewTopicKey with %d clients", clientCount), func(b *testing.B) {
			grpcClient, tearDown := setup(tempFileName("c2benchdb-"), b)

			// Create a dummy topic and register clientCount clients on it.
			topic := "benchTopic"
			if _, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC, nil, nil, topic, "", 0, 0); err != nil {
				b.Fatalf("Failed to create topic: %v", err)
			}

			for i := 0; i < clientCount; i++ {
				id := e4.RandomID()
				key := e4.RandomKey()

				_, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_CLIENT, id, key, "", "", 0, 0)
				if err != nil {
					b.Fatalf("Failed to create new client: %v", err)
				}

				_, err = e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC_CLIENT, id, nil, topic, "", 0, 0)
				if err != nil {
					b.Fatalf("Failed to register client on topic: %v", err)
				}
			}

			b.ResetTimer()

			for n := 0; n < b.N; n++ {
				_, err := e4test.GrpcC2SendCommand(grpcClient, e4.C2Request_NEW_TOPIC, nil, nil, topic, "", 0, 0)
				if err != nil {
					b.Errorf("newTopic failed: %v", err)
				}
			}

			b.StopTimer()

			tearDown(b)
		})
	}
}
