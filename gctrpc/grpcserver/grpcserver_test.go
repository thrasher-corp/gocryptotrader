package grpcserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
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
		_, err := client.GetInfo(rpcContext(t), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err), "a call without credentials should be unauthenticated")
		_, err = client.GetInfo(withBasicAuth(rpcContext(t), "wrong"), &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Unauthenticated, status.Code(err), "a call with the wrong password should be unauthenticated")
		info, err := client.GetInfo(withBasicAuth(rpcContext(t), testPassword), &gctrpc.GetInfoRequest{})
		require.NoError(t, err, "GetInfo must not error with valid credentials")
		assert.Equal(t, "1s", info.Uptime, "GetInfo should return the handler's response")
	})

	t.Run("handler panics are recovered", func(t *testing.T) {
		t.Parallel()
		client := newBufconnClient(t, auth.BasicAuthFunc(testUsername, testPassword))
		ctx := withBasicAuth(rpcContext(t), testPassword)

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
		ctx := withBasicAuth(rpcContext(t), testPassword)

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
		ctx := rpcContext(t)

		_, err := client.GetInfo(ctx, &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), "a unary auth panic should return codes.Internal")

		stream, err := client.GetTickerStream(ctx, &gctrpc.GetTickerStreamRequest{})
		require.NoError(t, err, "GetTickerStream must not error")
		_, err = stream.Recv()
		assert.Equal(t, codes.Internal, status.Code(err), "a stream auth panic should return codes.Internal")

		_, err = client.GetInfo(ctx, &gctrpc.GetInfoRequest{})
		assert.Equal(t, codes.Internal, status.Code(err), "the server should keep serving after auth panics")
	})
}

// reservedPorts holds the ports reserveAddress has handed to running tests
var reservedPorts sync.Map

// reserveAddress returns a localhost address with a free port; localhost also exercises the certificate's DNS name
// The port is below the ephemeral range, so a listener on port 0 or an outgoing connection cannot take it before use
func reserveAddress(t *testing.T) string {
	t.Helper()
	listenConfig := net.ListenConfig{}
	for range 100 {
		port := 20000 + rand.IntN(10000) //nolint:gosec // a test port needs no cryptographic randomness
		if _, taken := reservedPorts.LoadOrStore(port, struct{}{}); taken {
			continue
		}
		address := net.JoinHostPort("localhost", strconv.Itoa(port))
		lis, err := listenConfig.Listen(t.Context(), "tcp", address)
		if err != nil {
			reservedPorts.Delete(port)
			continue
		}
		require.NoError(t, lis.Close(), "Close must not error")
		t.Cleanup(func() { reservedPorts.Delete(port) })
		return address
	}
	require.FailNow(t, "reserveAddress must find a free port")
	return ""
}

// Serial, so the release can be checked once the subtest has ended
func TestReserveAddress(t *testing.T) {
	var port int
	t.Run("holds a free port below the ephemeral range while its test runs", func(t *testing.T) {
		address := reserveAddress(t)
		host, portText, err := net.SplitHostPort(address)
		require.NoError(t, err, "SplitHostPort must not error")
		assert.Equal(t, "localhost", host, "reserveAddress should use localhost")
		port, err = strconv.Atoi(portText)
		require.NoError(t, err, "Atoi must not error")
		assert.GreaterOrEqual(t, port, 20000, "reserveAddress should pick a port from 20000")
		assert.Less(t, port, 30000, "reserveAddress should pick a port below 30000")
		_, held := reservedPorts.Load(port)
		assert.True(t, held, "reserveAddress should hold the port while its test runs")
		listenConfig := net.ListenConfig{}
		lis, err := listenConfig.Listen(t.Context(), "tcp", address)
		require.NoError(t, err, "the reserved port must be free to listen on")
		assert.NoError(t, lis.Close(), "Close should not error")
	})
	_, held := reservedPorts.Load(port)
	assert.False(t, held, "reserveAddress should release the port once its test has ended")
}

// rpcContext bounds a test call, so a server that stops answering fails the test rather than hanging it
func rpcContext(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
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
		info, err := gctrpc.NewGoCryptoTraderServiceClient(conn).GetInfo(rpcContext(t), &gctrpc.GetInfoRequest{})
		require.NoError(t, err, "GetInfo must not error over gRPC with valid credentials")
		assert.Equal(t, "1s", info.Uptime, "GetInfo over gRPC should return the handler's response")

		badConn, err := grpcclient.New(cfg.ListenAddress, certFile, testUsername, "wrong")
		require.NoError(t, err, "New must not error")
		t.Cleanup(func() { assert.NoError(t, badConn.Close(), "Close should not error") })
		_, err = gctrpc.NewGoCryptoTraderServiceClient(badConn).GetInfo(rpcContext(t), &gctrpc.GetInfoRequest{})
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

	t.Run("proxy and gateway use the key pair Start loaded", func(t *testing.T) {
		t.Parallel()
		tlsDir := t.TempDir()
		require.NoError(t, genCert(tlsDir), "genCert must not error")
		caCert, err := os.ReadFile(filepath.Join(tlsDir, "cert.pem"))
		require.NoError(t, err, "ReadFile must not error")
		var gatewayDone <-chan struct{}
		cfg := &Config{
			ListenAddress:      reserveAddress(t),
			ProxyListenAddress: reserveAddress(t),
			TLSDir:             tlsDir,
			Username:           testUsername,
			Password:           testPassword,
			Register: func(r grpc.ServiceRegistrar) {
				registerTestService(r)
				// A new pair replaces the files after Start has loaded them
				require.NoError(t, genCert(tlsDir), "genCert must not error")
			},
			RegisterProxy: func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
				gatewayDone = ctx.Done()
				return gctrpc.RegisterGoCryptoTraderServiceHandlerFromEndpoint(ctx, mux, endpoint, opts)
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		require.NoError(t, Start(ctx, cfg), "Start must not error")

		roots := x509.NewCertPool()
		require.True(t, roots.AppendCertsFromPEM(caCert), "AppendCertsFromPEM must accept the generated certificate")
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}, Timeout: 10 * time.Second}
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+cfg.ProxyListenAddress+"/v1/getinfo", http.NoBody)
		require.NoError(t, err, "NewRequestWithContext must not error")
		req.SetBasicAuth(testUsername, testPassword)
		resp, err := client.Do(req)
		require.NoError(t, err, "proxy must serve the key pair Start loaded")
		assert.Equal(t, http.StatusOK, resp.StatusCode, "gateway should reach gRPC with the key pair Start loaded")
		require.NoError(t, resp.Body.Close(), "Close must not error")

		cancel()
		assert.Eventually(t, func() bool { return !canDial(t, cfg.ProxyListenAddress) }, 10*time.Second, 50*time.Millisecond, "proxy listener should close once the context is done")
		select {
		case <-gatewayDone:
		case <-time.After(10 * time.Second):
			assert.Fail(t, "gateway connection should close once the proxy has drained")
		}
	})

	t.Run("proxy registration failure leaves nothing serving", func(t *testing.T) {
		t.Parallel()
		cfg := &Config{
			ListenAddress:      reserveAddress(t),
			ProxyListenAddress: reserveAddress(t),
			TLSDir:             t.TempDir(),
			Register:           registerTestService,
			RegisterProxy: func(context.Context, *runtime.ServeMux, string, []grpc.DialOption) error {
				return errTestUnmapped
			},
		}
		assert.ErrorIs(t, Start(t.Context(), cfg), errTestUnmapped, "Start should return the registration error")
		assert.False(t, canDial(t, cfg.ListenAddress), "gRPC listener should be closed after a proxy failure")
		assert.False(t, canDial(t, cfg.ProxyListenAddress), "proxy listener should be closed after a registration failure")
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

	t.Run("invalid certificate", func(t *testing.T) {
		t.Parallel()
		tlsDir := t.TempDir()
		require.NoError(t, genCert(tlsDir), "genCert must not error")
		require.NoError(t, os.WriteFile(filepath.Join(tlsDir, "cert.pem"), []byte("not a certificate"), 0o600), "WriteFile must not error")
		assert.ErrorIs(t, Start(t.Context(), &Config{Register: registerTestService, TLSDir: tlsDir}), errCertDataIsNil, "Start should return the certificate check error")
	})

	t.Run("key pair load failure", func(t *testing.T) {
		t.Parallel()
		tlsDir := t.TempDir()
		require.NoError(t, genCert(tlsDir), "genCert must not error")
		require.NoError(t, os.WriteFile(filepath.Join(tlsDir, "key.pem"), []byte("not a key"), 0o600), "WriteFile must not error")
		assert.ErrorContains(t, Start(t.Context(), &Config{Register: registerTestService, TLSDir: tlsDir}), "unable to load gRPC TLS key pair", "Start should report the key pair load failure")
	})
}

func TestStartProxy(t *testing.T) {
	t.Parallel()
	tlsDir := t.TempDir()
	require.NoError(t, genCert(tlsDir), "genCert must not error")
	keyPair, err := tls.LoadX509KeyPair(filepath.Join(tlsDir, "cert.pem"), filepath.Join(tlsDir, "key.pem"))
	require.NoError(t, err, "LoadX509KeyPair must not error")
	registerPing := func(_ context.Context, mux *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
		return mux.HandlePath(http.MethodGet, "/ping", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			w.WriteHeader(http.StatusNoContent)
		})
	}

	t.Run("bind failure", func(t *testing.T) {
		t.Parallel()
		listenConfig := net.ListenConfig{}
		occupied, err := listenConfig.Listen(t.Context(), "tcp", "localhost:0")
		require.NoError(t, err, "Listen must not error")
		t.Cleanup(func() { assert.NoError(t, occupied.Close(), "Close should not error") })
		cfg := &Config{ProxyListenAddress: occupied.Addr().String(), RegisterProxy: registerPing}
		assert.ErrorContains(t, startProxy(t.Context(), cfg, &keyPair), "gRPC proxy listener failed to bind", "startProxy should report the bind failure")
	})

	t.Run("registration failure closes the listener and the gateway context", func(t *testing.T) {
		t.Parallel()
		var gatewayDone <-chan struct{}
		cfg := &Config{
			ProxyListenAddress: reserveAddress(t),
			RegisterProxy: func(ctx context.Context, _ *runtime.ServeMux, _ string, _ []grpc.DialOption) error {
				gatewayDone = ctx.Done()
				return errTestUnmapped
			},
		}
		assert.ErrorIs(t, startProxy(t.Context(), cfg, &keyPair), errTestUnmapped, "startProxy should return the registration error")
		assert.False(t, canDial(t, cfg.ProxyListenAddress), "proxy listener should be closed after a registration failure")
		select {
		case <-gatewayDone:
		default:
			assert.Fail(t, "gateway context should be cancelled after a registration failure")
		}
	})

	t.Run("serves the given key pair with Basic auth until the context is done", func(t *testing.T) {
		t.Parallel()
		var gatewayDone <-chan struct{}
		cfg := &Config{
			ProxyListenAddress: reserveAddress(t),
			Username:           testUsername,
			Password:           testPassword,
			RegisterProxy: func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
				gatewayDone = ctx.Done()
				return registerPing(ctx, mux, endpoint, opts)
			},
		}
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		require.NoError(t, startProxy(ctx, cfg, &keyPair), "startProxy must not error")

		roots := x509.NewCertPool()
		roots.AddCert(keyPair.Leaf)
		client := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}}, Timeout: 10 * time.Second}
		// Serial, as the proxy is shut down once these cases have run
		for _, tc := range []struct {
			name            string
			withCredentials bool
			wantStatus      int
		}{
			{name: "no credentials", wantStatus: http.StatusUnauthorized},
			{name: "valid credentials", withCredentials: true, wantStatus: http.StatusNoContent},
		} {
			t.Run(tc.name, func(t *testing.T) {
				req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://"+cfg.ProxyListenAddress+"/ping", http.NoBody)
				require.NoError(t, err, "NewRequestWithContext must not error")
				if tc.withCredentials {
					req.SetBasicAuth(testUsername, testPassword)
				}
				resp, err := client.Do(req)
				require.NoError(t, err, "Do must not error")
				assert.Equal(t, tc.wantStatus, resp.StatusCode, "HTTP status code should match")
				require.NotNil(t, resp.TLS, "response must be served over TLS")
				require.NotEmpty(t, resp.TLS.PeerCertificates, "response must carry the server certificate")
				assert.True(t, resp.TLS.PeerCertificates[0].Equal(keyPair.Leaf), "proxy should serve the given key pair")
				assert.NoError(t, resp.Body.Close(), "Close should not error")
			})
		}

		cancel()
		assert.Eventually(t, func() bool { return !canDial(t, cfg.ProxyListenAddress) }, 10*time.Second, 50*time.Millisecond, "proxy listener should close once the context is done")
		select {
		case <-gatewayDone:
		case <-time.After(10 * time.Second):
			assert.Fail(t, "gateway context should be cancelled once the proxy has drained")
		}
	})
}
