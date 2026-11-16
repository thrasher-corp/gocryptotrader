package engine

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// rpcSubmitExchange captures submissions after RPC and order-manager validation.
type rpcSubmitExchange struct {
	exchange.IBotExchange
	submission *order.Submit
}

func (e *rpcSubmitExchange) SubmitOrder(_ context.Context, submission *order.Submit) (*order.SubmitResponse, error) {
	e.submission = submission
	return nil, errExpectedTestError
}

func (*rpcSubmitExchange) CheckOrderExecutionLimits(asset.Item, currency.Pair, float64, float64, order.Type) error {
	return nil
}

func (*rpcSubmitExchange) CanTradePair(currency.Pair, asset.Item) error {
	return nil
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	type testCase struct {
		name          string
		clientID      string
		clientOrderID string
	}
	for _, exchangeName := range []string{"HTX", "Binance"} {
		for _, tc := range []testCase{
			{name: "both omitted"},
			{name: "account only", clientID: "123"},
			{name: "order only", clientOrderID: "order-456"},
			{name: "distinct identifiers", clientID: "123", clientOrderID: "order-456"},
			{name: "equal identifiers", clientID: "123", clientOrderID: "123"},
		} {
			for _, transport := range []string{"protobuf", "JSON"} {
				t.Run(exchangeName+"/"+tc.name+"/"+transport, func(t *testing.T) {
					t.Parallel()
					em := NewExchangeManager()
					realExchange, err := em.NewExchangeByName(exchangeName)
					require.NoError(t, err, "test exchange must be available")
					realExchange.SetDefaults()
					base := realExchange.GetBase()
					base.Enabled = true
					pair := currency.NewBTCUSDT()
					base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{asset.Spot: {
						Available:     currency.Pairs{pair},
						Enabled:       currency.Pairs{pair},
						AssetEnabled:  true,
						ConfigFormat:  &currency.PairFormat{Uppercase: true},
						RequestFormat: &currency.PairFormat{Uppercase: true},
					}}
					exch := &rpcSubmitExchange{IBotExchange: realExchange}
					require.NoError(t, em.Add(exch), "test exchange must be registered")
					var wg sync.WaitGroup
					manager, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
					require.NoError(t, err, "order manager must be configured")
					manager.started.Store(true)
					server := &RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: manager}}
					req := &gctrpc.SubmitOrderRequest{
						Exchange:      exchangeName,
						Pair:          &gctrpc.CurrencyPair{Base: "BTC", Quote: "USDT", Delimiter: "-"},
						Side:          "buy",
						OrderType:     "limit",
						Price:         5,
						Amount:        1,
						ClientId:      tc.clientID,
						AssetType:     asset.Spot.String(),
						ClientOrderId: tc.clientOrderID,
					}
					decoded := new(gctrpc.SubmitOrderRequest)
					if transport == "protobuf" {
						wire, err := proto.Marshal(req)
						require.NoError(t, err, "protobuf request must encode")
						require.NoError(t, proto.Unmarshal(wire, decoded), "protobuf request must decode")
					} else {
						wire, err := protojson.Marshal(req)
						require.NoError(t, err, "JSON request must encode")
						require.NoError(t, protojson.Unmarshal(wire, decoded), "JSON request must decode")
					}
					_, err = server.SubmitOrder(t.Context(), decoded)
					require.ErrorIs(t, err, errExpectedTestError, "request must reach the captured exchange submission")
					require.NotNil(t, exch.submission, "exchange must receive the submission")
					assert.Equal(t, tc.clientID, exch.submission.ClientID, "client ID should retain its own input")
					assert.Equal(t, tc.clientOrderID, exch.submission.ClientOrderID, "client order ID should retain its own input")
					assert.True(t, proto.Equal(req, decoded), "RPC should leave the request unchanged")
				})
			}
		}
	}
	t.Run("rejected requests and margin type", func(t *testing.T) {
		t.Parallel()
		s, exch := setupFeesExchange(t)
		pair := &gctrpc.CurrencyPair{Delimiter: "-", Base: currency.BTC.String(), Quote: currency.USDT.String()}
		ethUSDT := &gctrpc.CurrencyPair{Delimiter: "-", Base: currency.ETH.String(), Quote: currency.USDT.String()}
		ctx := t.Context()
		// Cases share the RPC server and exchange, and SubmitOrder is called again after them, so they run in order
		for _, tc := range []struct {
			name string
			req  *gctrpc.SubmitOrderRequest
			err  error
		}{
			{name: "an unknown asset", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: pair, Side: "BUY", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "bogus"}, err: asset.ErrNotSupported},
			{name: "an unknown margin type", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: pair, Side: "BUY", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "spot", MarginType: "bogus"}, err: margin.ErrInvalidMarginType},
			{name: "no pair", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Side: "BUY", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "spot"}, err: errCurrencyPairUnset},
			{name: "an exchange that is not loaded", req: &gctrpc.SubmitOrderRequest{Exchange: "unknown", Pair: pair, Side: "BUY", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "spot"}, err: ErrExchangeNotFound},
			{name: "a pair the exchange does not list", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: ethUSDT, Side: "BUY", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "spot"}, err: errCurrencyPairInvalid},
			{name: "an unknown side", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: pair, Side: "SIDEWAYS", OrderType: "LIMIT", Amount: 1, Price: 1000, AssetType: "spot"}, err: order.ErrSideIsInvalid},
			{name: "an unknown order type", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: pair, Side: "BUY", OrderType: "BOGUS", Amount: 1, Price: 1000, AssetType: "spot"}, err: order.ErrUnrecognisedOrderType},
			{name: "an order the order manager rejects", req: &gctrpc.SubmitOrderRequest{Exchange: exch.GetName(), Pair: pair, Side: "BUY", OrderType: "LIMIT", Price: 1000, AssetType: "spot"}, err: order.ErrAmountIsInvalid},
		} {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := s.SubmitOrder(ctx, tc.req)
				assert.ErrorIsf(t, err, tc.err, "SubmitOrder should reject %s", tc.name)
				assert.Emptyf(t, resp.GetOrderId(), "SubmitOrder should not return an order ID for %s", tc.name)
			})
		}
		assert.Nil(t, exch.submission, "SubmitOrder should not send a rejected order to the exchange")

		exch.submitted = &order.SubmitResponse{
			Exchange:      exch.GetName(),
			OrderID:       "1",
			ClientOrderID: "2",
			Pair:          currency.NewBTCUSDT(),
			AssetType:     asset.USDTMarginedFutures,
			Side:          order.Long,
			Type:          order.Limit,
			Price:         1000,
			Amount:        1,
			Status:        order.New,
			MarginType:    margin.Isolated,
			Date:          time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		}
		resp, err := s.SubmitOrder(t.Context(), &gctrpc.SubmitOrderRequest{
			Exchange:   exch.GetName(),
			Pair:       pair,
			Side:       "LONG",
			OrderType:  "LIMIT",
			Amount:     1,
			Price:      1000,
			ClientId:   "2",
			AssetType:  "usdtmarginedfutures",
			MarginType: "isolated",
		})
		require.NoError(t, err, "SubmitOrder must not error")
		assert.Equal(t, &gctrpc.SubmitOrderResponse{OrderId: "1", OrderPlaced: true, Trades: []*gctrpc.Trades{}}, resp, "SubmitOrder should return the placed order")
		exp := &order.Submit{
			Exchange:   exch.GetName(),
			Type:       order.Limit,
			Side:       order.Long,
			Pair:       currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), "-"),
			AssetType:  asset.USDTMarginedFutures,
			Price:      1000,
			Amount:     1,
			ClientID:   "2",
			MarginType: margin.Isolated,
		}
		assert.Equal(t, exp, exch.submission, "SubmitOrder should send the requested order and margin type to the exchange")
	})
}
