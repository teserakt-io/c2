package cli

//go:generate mockgen -destination=apiClient_mocks.go -package cli -self_package github.com/teserakt-io/c2/internal/cli github.com/teserakt-io/c2/internal/cli APIClientFactory,C2Client

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/teserakt-io/c2/pkg/pb"
)

const (
	// MaxPageSize defines the maximum number of clients to fetch per api query
	MaxPageSize int64 = 100
)

const (
	// EndpointFlag is the global flag name used to store the api endpoint url
	EndpointFlag = "endpoint"
	// CertFlag is the global flag name used to store the api certificate path
	CertFlag = "cert"
)

var (
	// ErrEndpointFlagUndefined is returned when the endpoint flag cannot be found on given command
	ErrEndpointFlagUndefined = errors.New("cannot retrieve endpoint flag on given cobra command")
	// ErrCertFlagUndefined is returned when the cert flag cannot be found on given command
	ErrCertFlagUndefined = errors.New("cannot retrieve cert flag on given cobra command")
)

// C2Client override the protobuf client definition to offer a Close method
// for the grpc connection
type C2Client interface {
	pb.C2Client
	Close() error
}

// APIClientFactory allows to create pb.C2Client instances
type APIClientFactory interface {
	NewClient(cmd *cobra.Command) (C2Client, error)
}

type apiClientFactory struct {
}

var _ APIClientFactory = (*apiClientFactory)(nil)

// NewAPIClientFactory creates a new C2AutomationEngineClient factory
func NewAPIClientFactory() APIClientFactory {
	return &apiClientFactory{}
}

type c2Client struct {
	pb.C2Client
	cnx *grpc.ClientConn
}

var _ C2Client = (*c2Client)(nil)
var _ pb.C2Client = (*c2Client)(nil)

// NewClient creates a new C2Client instance connecting to given api endpoint
func (c *apiClientFactory) NewClient(cmd *cobra.Command) (C2Client, error) {
	endpointFlag := cmd.Flag(EndpointFlag)
	if endpointFlag == nil || len(endpointFlag.Value.String()) == 0 {
		return nil, ErrEndpointFlagUndefined
	}

	certFlag := cmd.Flag(CertFlag)
	if certFlag == nil || len(certFlag.Value.String()) == 0 {
		return nil, ErrCertFlagUndefined
	}

	creds, err := credentials.NewClientTLSFromFile(certFlag.Value.String(), "")
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS credentials from certificate %v: %v", certFlag.Value.String(), err)
	}

	cnx, err := grpc.Dial(endpointFlag.Value.String(), grpc.WithTransportCredentials(creds))
	if err != nil {
		return nil, err
	}

	return &c2Client{
		C2Client: pb.NewC2Client(cnx),
		cnx:      cnx,
	}, nil
}

func (c *c2Client) Close() error {
	return c.cnx.Close()
}
