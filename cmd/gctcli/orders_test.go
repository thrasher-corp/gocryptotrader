package main

import (
	"context"
	"encoding/pem"
	"flag"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSubmitOrder(t *testing.T) {
	// The CLI connection settings and pair delimiter are process-wide singletons.
	previousHost, previousCert, previousTimeout, previousDelimiter := host, certPath, timeout, pairDelimiter
	t.Cleanup(func() {
		host, certPath, timeout, pairDelimiter = previousHost, previousCert, previousTimeout, previousDelimiter
	})
	timeout, pairDelimiter = 5*time.Second, "-"
	for _, tc := range []struct {
		name, clientID, clientOrderID string
		rpcError                      bool
	}{
		{name: "distinct identifiers", clientID: "account-7", clientOrderID: "order-42"},
		{name: "account only", clientID: "account-7"},
		{name: "client order only", clientOrderID: "order-42"},
		{name: "identifiers omitted"},
		{name: "RPC rejection", clientID: "account-7", clientOrderID: "order-42", rpcError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			requests := make(chan *gctrpc.SubmitOrderRequest, 1)
			rpc := grpc.NewServer(grpc.UnknownServiceHandler(func(_ any, stream grpc.ServerStream) error {
				req := new(gctrpc.SubmitOrderRequest)
				if err := stream.RecvMsg(req); err != nil {
					return err
				}
				requests <- req
				if tc.rpcError {
					return status.Error(codes.InvalidArgument, "fabricated order rejected")
				}
				return stream.SendMsg(&gctrpc.SubmitOrderResponse{})
			}))
			t.Cleanup(rpc.Stop)
			server := httptest.NewUnstartedServer(rpc)
			server.EnableHTTP2 = true
			server.StartTLS()
			t.Cleanup(server.Close)
			cert, err := os.CreateTemp(".", "gctcli-test-cert-*.pem")
			require.NoError(t, err, "test certificate file must be created")
			t.Cleanup(func() { assert.NoError(t, os.Remove(cert.Name()), "test certificate should be removed") })
			require.NoError(t, pem.Encode(cert, &pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}), "server certificate must be written")
			require.NoError(t, cert.Close(), "test certificate must close")
			host, certPath = strings.TrimPrefix(server.URL, "https://"), cert.Name()
			command := *submitOrderCommand
			command.Action = submitOrder
			app := &cli.App{Writer: io.Discard, ErrWriter: io.Discard, Commands: []*cli.Command{&command}}
			err = app.RunContext(t.Context(), []string{"gctcli", "submitorder", "--exchange", "HTX", "--pair", "BTC-USD", "--side", "buy", "--type", "limit", "--amount", "2", "--price", "100", "--asset", "SPOT", "--client_id", tc.clientID, "--client_order_id", tc.clientOrderID})
			if tc.rpcError {
				require.Error(t, err, "RPC rejection must reach the command caller")
				assert.Equal(t, codes.InvalidArgument, status.Code(err), "RPC error code should be preserved")
			} else {
				require.NoError(t, err, "submitOrder must send a valid command request")
			}
			select {
			case req := <-requests:
				assert.Equal(t, tc.clientID, req.ClientId, "account identifier should reach its RPC field")
				assert.Equal(t, tc.clientOrderID, req.ClientOrderId, "client order identifier should reach its separate RPC field")
				assert.Equal(t, "BTC", req.Pair.Base, "base currency should reach RPC")
				assert.Equal(t, "USD", req.Pair.Quote, "quote currency should reach RPC")
				assert.Equal(t, "spot", req.AssetType, "asset should be normalised before RPC")
			case <-time.After(5 * time.Second):
				require.FailNow(t, "submitOrder must reach the RPC service")
			}
		})
	}
	t.Run("validation precedes client setup", func(t *testing.T) {
		certPath = "missing-test-certificate.pem"
		set := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
		set.String(pairFlagName, "X", "")
		require.NoError(t, set.Parse([]string{"--" + pairFlagName, "X"}), "test flags must parse")
		c := cli.NewContext(&cli.App{}, set, nil)
		c.Context = context.Background()
		require.ErrorIs(t, submitOrder(c), errInvalidPair, "invalid order must return validation before attempting TLS setup")
	})
}

func TestSubmitOrderRequest(t *testing.T) {
	// pairDelimiter is a CLI singleton.
	previous := pairDelimiter
	pairDelimiter = "-"
	t.Cleanup(func() { pairDelimiter = previous })
	for _, tc := range []struct{ name, clientID, clientOrderID string }{
		{name: "separate identifiers", clientID: "account", clientOrderID: "order-42"},
		{name: "account only", clientID: "account"},
		{name: "order only", clientOrderID: "order-42"},
		{name: "neither"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := *submitOrderCommand
			command.Action = func(c *cli.Context) error {
				req, err := submitOrderRequest(c)
				require.NoError(t, err, "valid order flags must build a request")
				assert.Equal(t, tc.clientID, req.ClientId, "account identifier should remain separate")
				assert.Equal(t, tc.clientOrderID, req.ClientOrderId, "client order identifier should reach its own RPC field")
				assert.Equal(t, "BTC", req.Pair.Base, "base should decode")
				assert.Equal(t, "USD", req.Pair.Quote, "quote should decode")
				assert.Equal(t, "spot", req.AssetType, "asset should be normalised")
				assert.Equal(t, 2.0, req.Amount, "amount should reach RPC")
				assert.Equal(t, 100.0, req.Price, "price should reach RPC")
				return nil
			}
			app := &cli.App{Commands: []*cli.Command{&command}}
			require.NoError(t, app.Run([]string{"gctcli", "submitorder", "--exchange", "HTX", "--pair", "BTC-USD", "--side", "buy", "--type", "limit", "--amount", "2", "--price", "100", "--asset", "SPOT", "--client_id", tc.clientID, "--client_order_id", tc.clientOrderID}), "command flags must parse")
		})
	}
	for _, tc := range []struct {
		name, flag, value string
		want              error
	}{
		{name: "missing side", flag: sideFlagName, want: errSubmitOrderSideRequired},
		{name: "missing type", flag: "type", want: errSubmitOrderTypeRequired},
		{name: "zero amount", flag: amountFlagName, value: "0", want: errSubmitOrderAmountRequired},
		{name: "invalid pair", flag: pairFlagName, value: "X", want: errInvalidPair},
		{name: "invalid asset", flag: assetFlagName, value: "bogus", want: errInvalidAsset},
		{name: "invalid margin", flag: "margintype", value: "bogus", want: margin.ErrInvalidMarginType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			command := *submitOrderCommand
			command.Action = func(c *cli.Context) error {
				_, err := submitOrderRequest(c)
				require.ErrorIs(t, err, tc.want, "invalid order flags must return the expected sentinel")
				return nil
			}
			app := &cli.App{Commands: []*cli.Command{&command}}
			require.NoError(t, app.Run([]string{"gctcli", "submitorder", "--exchange", "HTX", "--pair", "BTC-USD", "--side", "buy", "--type", "limit", "--amount", "2", "--asset", "spot", "--" + tc.flag, tc.value}), "invalid value must reach request validation")
		})
	}
}
