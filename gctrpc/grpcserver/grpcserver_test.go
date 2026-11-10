package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/grpcclient"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	testUsername = "test-user"
	testPassword = "test-password"
)

var (
	errTestInvalid  = errors.New("test invalid request")
	errTestUnmapped = errors.New("test unmapped failure")
)

type testService struct {
	gctrpc.UnimplementedGoCryptoTraderServiceServer
}

func (testService) GetInfo(context.Context, *gctrpc.GetInfoRequest) (*gctrpc.GetInfoResponse, error) {
	return &gctrpc.GetInfoResponse{Uptime: "1s"}, nil
}

func (testService) GetTicker(context.Context, *gctrpc.GetTickerRequest) (*gctrpc.TickerResponse, error) {
	panic("unary handler failure")
}

func (testService) GetTickerStream(*gctrpc.GetTickerStreamRequest, grpc.ServerStreamingServer[gctrpc.TickerResponse]) error {
	panic("stream handler failure")
}

func (testService) GetExchanges(context.Context, *gctrpc.GetExchangesRequest) (*gctrpc.GetExchangesResponse, error) {
	return nil, fmt.Errorf("handler: %w", errTestInvalid)
}

func (testService) GetRPCEndpoints(context.Context, *gctrpc.GetRPCEndpointsRequest) (*gctrpc.GetRPCEndpointsResponse, error) {
	return nil, errTestUnmapped
}

func (testService) GetOrderbookStream(*gctrpc.GetOrderbookStreamRequest, grpc.ServerStreamingServer[gctrpc.OrderbookResponse]) error {
	return errTestInvalid
}

func testStatusCode(err error) codes.Code {
	if errors.Is(err, errTestInvalid) {
		return codes.InvalidArgument
	}
	return codes.Unknown
}

func registerTestService(r grpc.ServiceRegistrar) {
	gctrpc.RegisterGoCryptoTraderServiceServer(r, testService{})
}

func withBasicAuth(ctx context.Context, password string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(testUsername+":"+password)))
}

func newBufconnClient(t *testing.T, authenticate func(context.Context) (context.Context, error)) gctrpc.GoCryptoTraderServiceClient {
	t.Helper()
	lis := bufconn.Listen(1 << 20)
	s := grpc.NewServer(serverOptions(insecure.NewCredentials(), authenticate, testStatusCode)...)
	registerTestService(s)
	go func() {
		assert.NoError(t, s.Serve(lis), "Serve should not error")
	}()
	t.Cleanup(s.Stop)
	conn, err := grpc.NewClient("passthrough:///bufconn",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return lis.DialContext(ctx) }),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err, "New must not error")
	t.Cleanup(func() { assert.NoError(t, conn.Close(), "Close should not error") })
	return gctrpc.NewGoCryptoTraderServiceClient(conn)
}

func TestRecoverPanic(t *testing.T) {
	t.Parallel()
	runtimeErr := func() (p any) {
		defer func() { p = recover() }()
		values := make([]int, 1)
		index := len(values)
		_ = values[index]
		return nil
	}()
	for _, tc := range []struct {
		name  string
		value any
	}{
		{name: "runtime error", value: runtimeErr},
		{name: "other value", value: "panic value text"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := recoverPanic(t.Context(), tc.value)
			assert.Equal(t, codes.Internal, status.Code(err), "recoverPanic should return codes.Internal")
			assert.Equal(t, "internal server error", status.Convert(err).Message(), "recoverPanic should not return the panic value to the client")
		})
	}
}

func TestWithStatusCode(t *testing.T) {
	t.Parallel()
	assert.NoError(t, withStatusCode(nil, testStatusCode), "withStatusCode should pass a nil error through")
	assert.Equal(t, errTestInvalid, withStatusCode(errTestInvalid, nil), "withStatusCode should leave errors alone without a mapping")

	mapped := withStatusCode(fmt.Errorf("wrapped: %w", errTestInvalid), testStatusCode)
	assert.Equal(t, codes.InvalidArgument, status.Code(mapped), "withStatusCode should give a mapped error its code")
	assert.Equal(t, "wrapped: test invalid request", status.Convert(mapped).Message(), "withStatusCode should keep the error message")

	assert.Equal(t, errTestUnmapped, withStatusCode(errTestUnmapped, testStatusCode), "withStatusCode should leave unmapped errors alone")

	existing := status.Error(codes.NotFound, "already has a status")
	assert.Equal(t, existing, withStatusCode(existing, func(error) codes.Code { return codes.InvalidArgument }), "withStatusCode should keep an existing status")
}

func TestServerOptions(t *testing.T) {
	t.Parallel()

	t.Run("authentication", func(t *testing.T) {
		t.Parallel()
		client := newBufconnClient(t, auth.BasicAuthFunc(testUsername, testPassword))
		_, err := client.GetInfo(t.Context(), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err), "a call without credentials should be unauthenticated")
		_, err = client.GetInfo(withBasicAuth(t.Context(), "wrong"), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err), "a call with the wrong password should be unauthenticated")
		info, err := client.GetInfo(withBasicAuth(t.Context(), testPassword), &gctrpc.GetInfoRequest{})
		require.NoError(t, err, "GetInfo must not error with valid credentials")
		assert.Equal(t, "1s", info.Uptime, "GetInfo should return the handler's response")
	})

	t.Run("handler panics are recovered", func(t *testing.T) {
		t.Parallel()
		client := newBufconnClient(t, auth.BasicAuthFunc(testUsername, testPassword))
		ctx := withBasicAuth(t.Context(), testPassword)

		_, err := client.GetTicker(ctx, &gctrpc.GetTickerRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), "a unary handler panic should return codes.Internal")

		stream, err := client.GetTickerStream(ctx, &gctrpc.GetTickerStreamRequest{})
		require.NoError(t, err, "GetTickerStream must not error")
		_, err = stream.Recv()
		assert.Equal(t, codes.Internal, status.Code(err), "a stream handler panic should return codes.Internal")

		_, err = client.GetInfo(ctx, &gctrpc.GetInfoRequest{})
		assert.NoError(t, err, "the server should keep serving after handler panics")
	})

	t.Run("handler errors carry their mapped status", func(t *testing.T) {
		t.Parallel()
		client := newBufconnClient(t, auth.BasicAuthFunc(testUsername, testPassword))
		ctx := withBasicAuth(t.Context(), testPassword)

		_, err := client.GetExchanges(ctx, &gctrpc.GetExchangesRequest{})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "a mapped unary error should carry its code")
		assert.Equal(t, "handler: test invalid request", status.Convert(err).Message(), "a mapped error should keep its message")

		_, err = client.GetRPCEndpoints(ctx, &gctrpc.GetRPCEndpointsRequest{})
		assert.Equal(t, codes.Unknown, status.Code(err), "an unmapped error should stay unknown")

		stream, err := client.GetOrderbookStream(ctx, &gctrpc.GetOrderbookStreamRequest{})
		require.NoError(t, err, "GetOrderbookStream must not error")
		_, err = stream.Recv()
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "a mapped stream error should carry its code")
	})

	t.Run("auth panics are recovered", func(t *testing.T) {
		t.Parallel()
		client := newBufconnClient(t, func(context.Context) (context.Context, error) { panic("auth failure") })

		_, err := client.GetInfo(t.Context(), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), "a unary auth panic should return codes.Internal")

		stream, err := client.GetTickerStream(t.Context(), &gctrpc.GetTickerStreamRequest{})
		require.NoError(t, err, "GetTickerStream must not error")
		_, err = stream.Recv()
		assert.Equal(t, codes.Internal, status.Code(err), "a stream auth panic should return codes.Internal")

		_, err = client.GetInfo(t.Context(), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), "the server should keep serving after auth panics")
	})
}

// reserveAddress returns a localhost address with a free port; localhost also exercises the certificate's DNS name
func reserveAddress(t *testing.T) string {
	t.Helper()
	listenConfig := net.ListenConfig{}
	lis, err := listenConfig.Listen(t.Context(), "tcp", "localhost:0")
	require.NoError(t, err, "Listen must not error")
	_, port, err := net.SplitHostPort(lis.Addr().String())
	require.NoError(t, err, "SplitHostPort must not error")
	require.NoError(t, lis.Close(), "Close must not error")
	return "localhost:" + port
}

func canDial(t *testing.T, address string) bool {
	t.Helper()
	dialer := net.Dialer{Timeout: 100 * time.Millisecond}
	conn, err := dialer.DialContext(t.Context(), "tcp", address)
	if err != nil {
		return false
	}
	assert.NoError(t, conn.Close(), "Close should not error")
	return true
}

func TestStart(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name    string
		cfg     *Config
		wantErr error
	}{
		{name: "nil config", wantErr: errNilConfig},
		{name: "register unset", cfg: &Config{}, wantErr: errRegisterUnset},
		{name: "proxy register unset", cfg: &Config{Register: registerTestService, ProxyListenAddress: "localhost:0"}, wantErr: errProxyRegisterUnset},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, Start(t.Context(), tc.cfg), tc.wantErr, "Start should return the expected error")
		})
	}

	t.Run("serves gRPC and the REST proxy until the context is done", func(t *testing.T) {
		t.Parallel()
		tlsDir := t.TempDir()
		cfg := &Config{
			ListenAddress:      reserveAddress(t),
			ProxyListenAddress: reserveAddress(t),
			TLSDir:             tlsDir,
			Username:           testUsername,
			Password:           testPassword,
			Register:           registerTestService,
			RegisterProxy:      gctrpc.RegisterGoCryptoTraderServiceHandlerFromEndpoint,
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		require.NoError(t, Start(ctx, cfg), "Start must not error")
		require.Eventually(t, func() bool { return canDial(t, cfg.ProxyListenAddress) }, 5*time.Second, 10*time.Millisecond, "proxy must accept connections")

		certFile := filepath.Join(tlsDir, "cert.pem")
		conn, err := grpcclient.New(cfg.ListenAddress, certFile, testUsername, testPassword)
		require.NoError(t, err, "New must not error")
		t.Cleanup(func() { assert.NoError(t, conn.Close(), "Close should not error") })
		info, err := gctrpc.NewGoCryptoTraderServiceClient(conn).GetInfo(t.Context(), &gctrpc.GetInfoRequest{})
		require.NoError(t, err, "GetInfo must not error over gRPC with valid credentials")
		assert.Equal(t, "1s", info.Uptime, "GetInfo over gRPC should return the handler's response")

		badConn, err := grpcclient.New(cfg.ListenAddress, certFile, testUsername, "wrong")
		require.NoError(t, err, "New must not error")
		t.Cleanup(func() { assert.NoError(t, badConn.Close(), "Close should not error") })
		_, err = gctrpc.NewGoCryptoTraderServiceClient(badConn).GetInfo(t.Context(), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err), "gRPC with the wrong password should be unauthenticated")

		caCert, err := os.ReadFile(certFile)
		require.NoError(t, err, "ReadFile must not error")
		roots := x509.NewCertPool()
		require.True(t, roots.AppendCertsFromPEM(caCert), "AppendCertsFromPEM must accept the generated certificate")
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}, Timeout: 10 * time.Second}
		get := func(url string, withCredentials bool) (int, []byte, http.Header) {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, url, http.NoBody)
			require.NoError(t, err, "NewRequestWithContext must not error")
			if withCredentials {
				req.SetBasicAuth(testUsername, testPassword)
			}
			resp, err := client.Do(req)
			require.NoError(t, err, "Do must not error")
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "ReadAll must not error")
			return resp.StatusCode, body, resp.Header
		}

		code, _, header := get("https://"+cfg.ProxyListenAddress+"/v1/getinfo", false)
		assert.Equal(t, http.StatusUnauthorized, code, "the proxy should reject requests without credentials")
		assert.Equal(t, `Basic realm="restricted"`, header.Get("WWW-Authenticate"), "the proxy should ask for Basic credentials")

		code, body, _ := get("https://"+cfg.ProxyListenAddress+"/v1/getinfo", true)
		require.Equalf(t, http.StatusOK, code, "proxy must serve authorised requests, body: %s", body)
		var proxied gctrpc.GetInfoResponse
		require.NoError(t, protojson.Unmarshal(body, &proxied), "protojson.Unmarshal must not error")
		assert.Equal(t, "1s", proxied.Uptime, "the proxy should relay the gRPC response")

		code, _, _ = get("http://"+cfg.ProxyListenAddress+"/v1/getinfo", true)
		assert.Equal(t, http.StatusBadRequest, code, "the proxy should refuse plain HTTP")

		cancel()
		assert.Eventually(t, func() bool { return !canDial(t, cfg.ListenAddress) }, 10*time.Second, 50*time.Millisecond, "gRPC listener should close once the context is done")
		assert.Eventually(t, func() bool { return !canDial(t, cfg.ProxyListenAddress) }, 10*time.Second, 50*time.Millisecond, "proxy listener should close once the context is done")
	})

	t.Run("proxy bind failure leaves nothing serving", func(t *testing.T) {
		t.Parallel()
		listenConfig := net.ListenConfig{}
		occupied, err := listenConfig.Listen(t.Context(), "tcp", "localhost:0")
		require.NoError(t, err, "Listen must not error")
		t.Cleanup(func() { assert.NoError(t, occupied.Close(), "Close should not error") })
		cfg := &Config{
			ListenAddress:      reserveAddress(t),
			ProxyListenAddress: occupied.Addr().String(),
			TLSDir:             t.TempDir(),
			Register:           registerTestService,
			RegisterProxy:      gctrpc.RegisterGoCryptoTraderServiceHandlerFromEndpoint,
		}
		assert.ErrorContains(t, Start(t.Context(), cfg), "gRPC proxy listener failed to bind", "Start should report the proxy bind failure")
		assert.False(t, canDial(t, cfg.ListenAddress), "gRPC listener should be closed after a proxy failure")
	})

	t.Run("gRPC bind failure", func(t *testing.T) {
		t.Parallel()
		listenConfig := net.ListenConfig{}
		occupied, err := listenConfig.Listen(t.Context(), "tcp", "localhost:0")
		require.NoError(t, err, "Listen must not error")
		t.Cleanup(func() { assert.NoError(t, occupied.Close(), "Close should not error") })
		cfg := &Config{ListenAddress: occupied.Addr().String(), TLSDir: t.TempDir(), Register: registerTestService}
		assert.ErrorContains(t, Start(t.Context(), cfg), "gRPC listener failed to bind", "Start should report the gRPC bind failure")
	})
}
