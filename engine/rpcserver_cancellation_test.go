package engine

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
)

type cancellationExchange struct {
	exchange.IBotExchange
	response *order.CancelAllResponse
	err      error
	request  *order.Cancel
}

func (e *cancellationExchange) CancelAllOrders(_ context.Context, request *order.Cancel) (*order.CancelAllResponse, error) {
	e.request = request
	return e.response, e.err
}

func TestRPCServerCancelAllOrders(t *testing.T) {
	t.Parallel()
	pair := &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD", Delimiter: "-"}
	errBatch := errors.New("second batch failed")
	for _, tc := range []struct {
		name        string
		request     *gctrpc.CancelAllOrdersRequest
		response    *order.CancelAllResponse
		exchangeErr error
		wantErr     error
		called      bool
		partial     bool
	}{
		{name: "nil request", wantErr: errNilRequestData},
		{name: "pair without asset", request: &gctrpc.CancelAllOrdersRequest{Pair: pair}, wantErr: errAssetTypeUnset},
		{name: "empty populated pair", request: &gctrpc.CancelAllOrdersRequest{AssetType: "spot", Pair: &gctrpc.CurrencyPair{}}, wantErr: currency.ErrCurrencyPairEmpty},
		{name: "unknown asset", request: &gctrpc.CancelAllOrdersRequest{AssetType: "unknown"}, wantErr: asset.ErrNotSupported},
		{name: "unavailable asset", request: &gctrpc.CancelAllOrdersRequest{AssetType: "futures"}, wantErr: asset.ErrNotSupported},
		{name: "unavailable pair", request: &gctrpc.CancelAllOrdersRequest{AssetType: "spot", Pair: &gctrpc.CurrencyPair{Base: "AAA", Quote: "BBB", Delimiter: "-"}}, wantErr: currency.ErrPairNotFound},
		{name: "available disabled scope", request: &gctrpc.CancelAllOrdersRequest{AssetType: "spot", Pair: pair}, response: &order.CancelAllResponse{Status: map[string]string{"1": "Cancelled"}}, called: true},
		{name: "unscoped", request: &gctrpc.CancelAllOrdersRequest{}, response: &order.CancelAllResponse{}, called: true},
		{name: "asset only", request: &gctrpc.CancelAllOrdersRequest{AssetType: "spot"}, response: &order.CancelAllResponse{}, called: true},
		{name: "nil exchange response", request: &gctrpc.CancelAllOrdersRequest{}, wantErr: common.ErrInvalidResponse, called: true},
		{name: "complete failure", request: &gctrpc.CancelAllOrdersRequest{}, exchangeErr: errBatch, wantErr: errBatch, called: true},
		{name: "empty partial failure", request: &gctrpc.CancelAllOrdersRequest{}, response: &order.CancelAllResponse{}, exchangeErr: errBatch, wantErr: errBatch, called: true},
		{name: "partial failure", request: &gctrpc.CancelAllOrdersRequest{AssetType: "spot", Pair: pair}, response: &order.CancelAllResponse{Status: map[string]string{"1": "Cancelled", "2": "Cancelled"}}, exchangeErr: errBatch, called: true, partial: true},
		{name: "partial status failure", request: &gctrpc.CancelAllOrdersRequest{}, response: &order.CancelAllResponse{Status: map[string]string{"1": "Cancelled", "2": "Cancelled"}}, exchangeErr: status.Error(codes.Unavailable, "second batch failed"), called: true, partial: true},
		{name: "partial cancellation", request: &gctrpc.CancelAllOrdersRequest{}, response: &order.CancelAllResponse{Status: map[string]string{"1": "Cancelled", "2": "Cancelled"}}, exchangeErr: context.Canceled, called: true, partial: true},
		{name: "invalid partial details", request: &gctrpc.CancelAllOrdersRequest{}, response: &order.CancelAllResponse{Status: map[string]string{string([]byte{255}): "Cancelled"}}, exchangeErr: errBatch, wantErr: errBatch, called: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			manager := NewExchangeManager()
			base, err := manager.NewExchangeByName(testExchange)
			require.NoError(t, err, "test exchange must be created")
			base.SetDefaults()
			base.SetEnabled(true)
			require.NoError(t, base.GetBase().CurrencyPairs.Store(asset.Spot, &currency.PairStore{
				Available:     currency.Pairs{currency.NewBTCUSD()},
				ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: "-"},
				RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: "-"},
			}), "disabled available asset must be stored")
			base.GetBase().CurrencyPairs.UseGlobalFormat = false
			ex := &cancellationExchange{IBotExchange: base, response: tc.response, err: tc.exchangeErr}
			require.NoError(t, manager.Add(ex), "test exchange must be registered")
			server := &RPCServer{Engine: &Engine{ExchangeManager: manager}}
			if tc.request != nil {
				tc.request.Exchange = base.GetName()
			}
			response, err := server.CancelAllOrders(t.Context(), tc.request)
			assert.Equal(t, tc.called, ex.request != nil, "only valid requests should reach the exchange")
			if !tc.partial {
				require.ErrorIs(t, err, tc.wantErr, "RPC must return the expected validation or exchange error")
				if tc.wantErr == nil {
					require.NotNil(t, response, "successful RPC must return a response")
					assert.Equal(t, tc.response.Status, response.Orders[0].OrderStatus, "statuses should be retained")
					assert.Equal(t, int64(len(tc.response.Status)), response.Count, "count should match retained statuses")
					if tc.request.Pair != nil {
						assert.True(t, ex.request.Pair.Equal(currency.NewBTCUSD()), "requested pair should reach the exchange")
					}
				}
				return
			}
			require.Error(t, err, "partial failure must remain an error")
			require.Nil(t, response, "partial results must travel in error details")
			listener := bufconn.Listen(1024 * 1024)
			t.Cleanup(func() { assert.NoError(t, listener.Close(), "listener should close") })
			transport := grpc.NewServer()
			gctrpc.RegisterGoCryptoTraderServiceServer(transport, server)
			t.Cleanup(transport.Stop)
			go func() { assert.NoError(t, transport.Serve(listener), "gRPC server should serve") }()
			conn, err := grpc.NewClient("passthrough:///cancellation", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
			require.NoError(t, err, "gRPC client must initialise")
			t.Cleanup(func() { assert.NoError(t, conn.Close(), "connection should close") })
			_, err = gctrpc.NewGoCryptoTraderServiceClient(conn).CancelAllOrders(t.Context(), tc.request)
			require.Error(t, err, "partial failure must survive gRPC")
			wantStatus, ok := status.FromError(tc.exchangeErr)
			if !ok {
				wantStatus = status.FromContextError(tc.exchangeErr)
			}
			require.Equal(t, wantStatus.Message(), status.Convert(err).Message(), "failure message must survive gRPC")
			require.Equal(t, wantStatus.Code(), status.Code(err), "failure code must survive gRPC")
			details := status.Convert(err).Details()
			require.Len(t, details, 1, "partial response must survive gRPC")
			partial, ok := details[0].(*gctrpc.CancelAllOrdersResponse)
			require.True(t, ok, "detail must decode as a cancellation response")
			assert.Equal(t, tc.response.Status, partial.Orders[0].OrderStatus, "completed cancellations should survive transport")
			assert.Equal(t, int64(2), partial.Count, "partial count should survive transport")
		})
	}
}
