// Package grpcserver starts the TLS gRPC servers and grpc-gateway REST proxies shared by the engine and backtester
package grpcserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	goruntime "runtime"
	"runtime/debug"
	"time"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	gracefulStopTimeout  = 5 * time.Second
	proxyShutdownTimeout = 5 * time.Second
	proxyReadTimeout     = time.Minute
)

var (
	errNilConfig          = errors.New("gRPC server config is nil")
	errRegisterUnset      = errors.New("gRPC service register function is unset")
	errProxyRegisterUnset = errors.New("gRPC proxy register function is unset")
)

// ProxyRegisterFunc registers a service's REST handlers on a grpc-gateway mux
// Generated Register<Service>HandlerFromEndpoint functions satisfy it
type ProxyRegisterFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

// Config configures a gRPC server and its optional REST proxy
type Config struct {
	ListenAddress string
	// ProxyListenAddress enables the REST proxy when set
	ProxyListenAddress string
	TLSDir             string
	Username           string
	Password           string
	// Authenticate defaults to requiring the Basic auth Username and Password
	Authenticate grpcauth.AuthFunc
	// StatusCode maps a handler error to the gRPC code clients should see; codes.Unknown leaves the error as it is
	StatusCode      func(error) codes.Code
	Register        func(grpc.ServiceRegistrar)
	RegisterProxy   ProxyRegisterFunc
	ProxyMuxOptions []runtime.ServeMuxOption
}

// Start serves gRPC over TLS, and the REST proxy when configured, until ctx is done
// Nothing is left serving when it returns an error
func Start(ctx context.Context, cfg *Config) error {
	if cfg == nil {
		return errNilConfig
	}
	if cfg.Register == nil {
		return errRegisterUnset
	}
	if cfg.ProxyListenAddress != "" && cfg.RegisterProxy == nil {
		return errProxyRegisterUnset
	}
	if err := CheckCerts(cfg.TLSDir); err != nil {
		return err
	}
	certFile, keyFile := filepath.Join(cfg.TLSDir, "cert.pem"), filepath.Join(cfg.TLSDir, "key.pem")
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("unable to load gRPC TLS key pair: %w", err)
	}

	listenConfig := net.ListenConfig{}
	lis, err := listenConfig.Listen(ctx, "tcp", cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("gRPC listener failed to bind: %w", err)
	}

	authenticate := cfg.Authenticate
	if authenticate == nil {
		authenticate = auth.BasicAuthFunc(cfg.Username, cfg.Password)
	}
	s := grpc.NewServer(serverOptions(creds, authenticate, cfg.StatusCode)...)
	cfg.Register(s)

	if cfg.ProxyListenAddress != "" {
		if err := startProxy(ctx, cfg, certFile, keyFile); err != nil {
			_ = lis.Close()
			return err
		}
	}

	context.AfterFunc(ctx, func() {
		done := make(chan struct{})
		go func() {
			s.GracefulStop()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(gracefulStopTimeout):
			s.Stop()
			<-done
		}
		_ = lis.Close()
	})

	go func() {
		if err := s.Serve(lis); err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf(log.GRPCSys, "gRPC server failed to serve: %s", err)
		}
	}()

	log.Debugf(log.GRPCSys, "gRPC server started on https://%s", cfg.ListenAddress)
	return nil
}

func startProxy(ctx context.Context, cfg *Config, certFile, keyFile string) error {
	listenConfig := net.ListenConfig{}
	lis, err := listenConfig.Listen(ctx, "tcp", cfg.ProxyListenAddress)
	if err != nil {
		return fmt.Errorf("gRPC proxy listener failed to bind: %w", err)
	}

	creds, err := credentials.NewClientTLSFromFile(certFile, "")
	if err != nil {
		_ = lis.Close()
		return fmt.Errorf("unable to load gRPC proxy TLS certificate: %w", err)
	}

	// The gateway's connection to the gRPC server is closed only after the HTTP server has drained, so it can't use ctx
	connCtx, closeConn := context.WithCancel(context.Background())
	mux := runtime.NewServeMux(cfg.ProxyMuxOptions...)
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{Username: cfg.Username, Password: cfg.Password}),
	}
	if err := cfg.RegisterProxy(connCtx, mux, cfg.ListenAddress, opts); err != nil {
		closeConn()
		_ = lis.Close()
		return fmt.Errorf("unable to register gRPC proxy handlers: %w", err)
	}

	server := &http.Server{
		Addr:              cfg.ProxyListenAddress,
		ReadHeaderTimeout: proxyReadTimeout,
		ReadTimeout:       proxyReadTimeout,
		Handler:           auth.RequireBasicAuth(mux, cfg.Username, cfg.Password),
	}

	context.AfterFunc(ctx, func() {
		defer closeConn()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), proxyShutdownTimeout)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf(log.GRPCSys, "gRPC proxy server shutdown failed: %s", err)
		}
	})

	go func() {
		if err := server.ServeTLS(lis, certFile, keyFile); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Errorf(log.GRPCSys, "gRPC proxy server failed to serve: %s", err)
		}
	}()

	log.Debugf(log.GRPCSys, "gRPC proxy server started on https://%s", cfg.ProxyListenAddress)
	return nil
}

// serverOptions puts panic recovery ahead of authentication so that it also covers the auth interceptor
func serverOptions(creds credentials.TransportCredentials, authenticate grpcauth.AuthFunc, statusCode func(error) codes.Code) []grpc.ServerOption {
	recoveryOpt := grpcrecovery.WithRecoveryHandlerContext(recoverPanic)
	return []grpc.ServerOption{
		grpc.Creds(creds),
		grpc.ChainUnaryInterceptor(
			grpcrecovery.UnaryServerInterceptor(recoveryOpt),
			grpcauth.UnaryServerInterceptor(authenticate),
			func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
				resp, err := handler(ctx, req)
				return resp, withStatusCode(err, statusCode)
			},
		),
		grpc.ChainStreamInterceptor(
			grpcrecovery.StreamServerInterceptor(recoveryOpt),
			grpcauth.StreamServerInterceptor(authenticate),
			func(srv any, stream grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
				return withStatusCode(handler(srv, stream), statusCode)
			},
		),
	}
}

// withStatusCode gives err the gRPC code statusCode maps it to, leaving errors which already carry a status alone
func withStatusCode(err error, statusCode func(error) codes.Code) error {
	if err == nil || statusCode == nil {
		return err
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	if code := statusCode(err); code != codes.Unknown && code != codes.OK {
		return status.Error(code, err.Error())
	}
	return err
}

// recoverPanic logs a recovered panic and returns a generic Internal error
// Only the Go runtime's own error text is logged, as other panic values may carry credentials
func recoverPanic(ctx context.Context, p any) error {
	method, _ := grpc.Method(ctx)
	reason := fmt.Sprintf("%T", p)
	if runtimeErr, ok := p.(goruntime.Error); ok {
		reason = runtimeErr.Error()
	}
	log.Errorf(log.GRPCSys, "gRPC %s panic: %s\n%s", method, reason, debug.Stack())
	return status.Error(codes.Internal, "internal server error")
}
