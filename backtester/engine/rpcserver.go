package engine

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"strings"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	gctengine "github.com/thrasher-corp/gocryptotrader/engine"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// RPCServer struct
type RPCServer struct {
	btrpc.UnimplementedBacktesterServer
	*config.BacktesterConfig
}

func SetupRPCServer(cfg *config.BacktesterConfig) *RPCServer {
	return &RPCServer{
		BacktesterConfig: cfg,
	}

}

// StartRPCServer starts a gRPC server with TLS auth
func StartRPCServer(server *RPCServer) error {
	targetDir := utils.GetTLSDir(server.GRPC.TLSDir)
	if err := gctengine.CheckCerts(targetDir); err != nil {
		return err
	}
	log.Debugf(log.GRPCSys, "gRPC server support enabled. Starting gRPC server on https://%v.\n", server.GRPC.ListenAddress)
	lis, err := net.Listen("tcp", server.GRPC.ListenAddress)
	if err != nil {
		return err

	}

	creds, err := credentials.NewServerTLSFromFile(filepath.Join(targetDir, "cert.pem"), filepath.Join(targetDir, "key.pem"))
	if err != nil {
		return err
	}

	opts := []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.UnaryInterceptor(grpcauth.UnaryServerInterceptor(server.authenticateClient)),
	}
	s := grpc.NewServer(opts...)
	btrpc.RegisterBacktesterServer(s, server)

	go func() {
		if err = s.Serve(lis); err != nil {
			log.Error(log.GRPCSys, err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC server started!")

	if server.GRPC.GRPCProxyEnabled {
		server.StartRPCRESTProxy()
	}
	return nil
}

// StartRPCRESTProxy starts a gRPC proxy
func (s *RPCServer) StartRPCRESTProxy() {
	log.Debugf(log.GRPCSys, "gRPC proxy server support enabled. Starting gRPC proxy server on http://%v.\n", s.GRPC.GRPCProxyListenAddress)
	targetDir := utils.GetTLSDir(s.GRPC.TLSDir)
	creds, err := credentials.NewClientTLSFromFile(filepath.Join(targetDir, "cert.pem"), "")
	if err != nil {
		log.Errorf(log.GRPCSys, "Unabled to start gRPC proxy. Err: %s\n", err)
		return
	}

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: s.GRPC.Username,
			Password: s.GRPC.Password,
		}),
	}
	err = gctrpc.RegisterGoCryptoTraderHandlerFromEndpoint(context.Background(),
		mux, s.GRPC.ListenAddress, opts)
	if err != nil {
		log.Errorf(log.GRPCSys, "Failed to register gRPC proxy. Err: %s\n", err)
		return
	}

	go func() {
		if err := http.ListenAndServe(s.GRPC.GRPCProxyListenAddress, mux); err != nil {
			log.Errorf(log.GRPCSys, "gRPC proxy failed to server: %s\n", err)
			return
		}
	}()

	log.Debugln(log.GRPCSys, "gRPC proxy server started!")
}

func (s *RPCServer) authenticateClient(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx, fmt.Errorf("unable to extract metadata")
	}

	authStr, ok := md["authorization"]
	if !ok {
		return ctx, fmt.Errorf("authorization header missing")
	}

	if !strings.Contains(authStr[0], "Basic") {
		return ctx, fmt.Errorf("basic not found in authorization header")
	}

	decoded, err := crypto.Base64Decode(strings.Split(authStr[0], " ")[1])
	if err != nil {
		return ctx, fmt.Errorf("unable to base64 decode authorization header")
	}

	creds := strings.Split(string(decoded), ":")
	username := creds[0]
	password := creds[1]

	if username != s.GRPC.Username ||
		password != s.GRPC.Password {
		return ctx, fmt.Errorf("username/password mismatch")
	}
	return exchange.ParseCredentialsMetadata(ctx, md)
}

// ExecuteStrategyFromFile will backtest a strategy from the filepath provided
func (s *RPCServer) ExecuteStrategyFromFile(_ context.Context, request *btrpc.ExecuteStrategyFromFileRequest) (*btrpc.ExecuteStrategyFromFileResponse, error) {
	dir := request.StrategyFilePath
	cfg, err := config.ReadStrategyConfigFromFile(dir)
	if err != nil {
		return nil, err
	}
	err = ExecuteStrategy(cfg, s.BacktesterConfig)
	if err != nil {
		return nil, err
	}
	return &btrpc.ExecuteStrategyFromFileResponse{
		Success: true,
	}, nil
}
