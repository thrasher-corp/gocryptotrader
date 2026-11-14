// Package grpcclient connects to the TLS gRPC servers run by the engine and backtester
package grpcclient

import (
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// New returns a gRPC client connection which verifies the server against the certificate in certFile and sends the
// Basic auth credentials with every call
func New(target, certFile, username, password string) (*grpc.ClientConn, error) {
	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		return nil, fmt.Errorf("error loading gRPC TLS certificate: %w", err)
	}
	return grpc.NewClient(target,
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{Username: username, Password: password}),
	)
}
