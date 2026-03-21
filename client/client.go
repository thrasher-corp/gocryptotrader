package client

import (
	"context"
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

const defaultCallTimeout = 30 * time.Second

// Client wraps the GoCryptoTrader gRPC service client and the underlying
// connection so callers can call Close when done.
type Client struct {
	conn        *grpc.ClientConn
	svc         gctrpc.GoCryptoTraderServiceClient
	callTimeout time.Duration
}

// ConnectViaSocket connects to the GoCryptoTrader daemon over a Unix Domain
// Socket.  This is the preferred transport for processes on the same server:
// the kernel enforces isolation via socket file permissions so no TLS is
// required, and throughput is ~2-5× higher than TCP loopback.
func ConnectViaSocket(socketPath, username, password string) (*Client, error) {
	if socketPath == "" {
		socketPath = "/tmp/gocryptotrader.sock"
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}
	// "unix://" prefix tells gRPC to use a Unix domain socket.
	conn, err := grpc.NewClient("unix://"+socketPath, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: dial unix socket %s: %w", socketPath, err)
	}
	return newClient(conn, 0), nil
}

// ConnectViaTCP connects to the GoCryptoTrader daemon over TCP with TLS.
// Use this when the client runs on a different host, or when the Unix socket
// is not available.
func ConnectViaTCP(host, certPath, username, password string) (*Client, error) {
	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		return nil, fmt.Errorf("client: load TLS cert %s: %w", certPath, err)
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}
	conn, err := grpc.NewClient(host, opts...)
	if err != nil {
		return nil, fmt.Errorf("client: dial tcp %s: %w", host, err)
	}
	return newClient(conn, 0), nil
}

// ConnectFromConfig creates a client from a Config struct, choosing the
// transport based on whether SocketPath is set.
func ConnectFromConfig(cfg Config) (*Client, error) {
	if cfg.SocketPath != "" {
		c, err := ConnectViaSocket(cfg.SocketPath, cfg.Username, cfg.Password)
		if err != nil {
			return nil, err
		}
		if cfg.CallTimeout > 0 {
			c.callTimeout = cfg.CallTimeout
		}
		return c, nil
	}
	c, err := ConnectViaTCP(cfg.Host, cfg.CertPath, cfg.Username, cfg.Password)
	if err != nil {
		return nil, err
	}
	if cfg.CallTimeout > 0 {
		c.callTimeout = cfg.CallTimeout
	}
	return c, nil
}

func newClient(conn *grpc.ClientConn, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = defaultCallTimeout
	}
	return &Client{
		conn:        conn,
		svc:         gctrpc.NewGoCryptoTraderServiceClient(conn),
		callTimeout: timeout,
	}
}

// Close tears down the underlying gRPC connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// ctx returns a context with the configured per-call timeout applied on top
// of whatever context the caller provides.
func (c *Client) ctx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, c.callTimeout)
}

// GetInfo returns high-level information about the running GoCryptoTrader
// instance (uptime, enabled exchanges, subsystem status, etc.).
func (c *Client) GetInfo(ctx context.Context) (*gctrpc.GetInfoResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetInfo(ctx, &gctrpc.GetInfoRequest{})
}

// GetExchanges returns a list of configured exchanges.  When enabledOnly is
// true only enabled exchanges are included.
func (c *Client) GetExchanges(ctx context.Context, enabledOnly bool) (*gctrpc.GetExchangesResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetExchanges(ctx, &gctrpc.GetExchangesRequest{Enabled: enabledOnly})
}

// GetTicker returns the current ticker for a given exchange / pair / asset.
func (c *Client) GetTicker(ctx context.Context, exchange, base, quote, asset string) (*gctrpc.TickerResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetTicker(ctx, &gctrpc.GetTickerRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:      base,
			Quote:     quote,
			Delimiter: "-",
		},
		AssetType: asset,
	})
}

// GetOrderbook returns the current order book for a given exchange / pair /
// asset.
func (c *Client) GetOrderbook(ctx context.Context, exchange, base, quote, asset string) (*gctrpc.OrderbookResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetOrderbook(ctx, &gctrpc.GetOrderbookRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:      base,
			Quote:     quote,
			Delimiter: "-",
		},
		AssetType: asset,
	})
}

// GetAccountBalances returns account balance information from the given exchange.
func (c *Client) GetAccountBalances(ctx context.Context, exchange, asset string) (*gctrpc.GetAccountBalancesResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetAccountBalances(ctx, &gctrpc.GetAccountBalancesRequest{
		Exchange:  exchange,
		AssetType: asset,
	})
}

// SubmitOrder places an order on the given exchange.
func (c *Client) SubmitOrder(ctx context.Context, req *gctrpc.SubmitOrderRequest) (*gctrpc.SubmitOrderResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.SubmitOrder(ctx, req)
}

// CancelOrder cancels an open order.
func (c *Client) CancelOrder(ctx context.Context, req *gctrpc.CancelOrderRequest) (*gctrpc.GenericResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.CancelOrder(ctx, req)
}

// GetOrders returns open orders for the given exchange.
func (c *Client) GetOrders(ctx context.Context, exchange, asset, base, quote string) (*gctrpc.GetOrdersResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetOrders(ctx, &gctrpc.GetOrdersRequest{
		Exchange:  exchange,
		AssetType: asset,
		Pair: &gctrpc.CurrencyPair{
			Base:      base,
			Quote:     quote,
			Delimiter: "-",
		},
	})
}

// GetSubsystems returns the status of all engine subsystems.
func (c *Client) GetSubsystems(ctx context.Context) (*gctrpc.GetSubsystemsResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetSubsystems(ctx, &gctrpc.GetSubsystemsRequest{})
}

// GetRecentTrades returns recent trades for the given exchange / pair / asset.
func (c *Client) GetRecentTrades(ctx context.Context, exchange, base, quote, asset string) (*gctrpc.SavedTradesResponse, error) {
	ctx, cancel := c.ctx(ctx)
	defer cancel()
	return c.svc.GetRecentTrades(ctx, &gctrpc.GetSavedTradesRequest{
		Exchange: exchange,
		Pair: &gctrpc.CurrencyPair{
			Base:      base,
			Quote:     quote,
			Delimiter: "-",
		},
		AssetType: asset,
	})
}
