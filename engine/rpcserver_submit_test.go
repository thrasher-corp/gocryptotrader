package engine

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
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

	// Guards the quote amount an order sized in the quote currency needs, such as OKX's margin market buys and spot
	// trailing stop buys: the RPC carries it to the exchange's submission beside or instead of the base amount
	t.Run("carries the quote amount", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name      string
			orderType string
			amount    float64
			price     float64
			quote     float64
		}{
			{"quote amount alone", "market", 0, 0, 100},
			{"base and quote amounts", "limit", 1, 5, 5},
			{"base amount alone", "limit", 1, 5, 0},
		} {
			for _, transport := range []string{"protobuf", "JSON"} {
				t.Run(tc.name+"/"+transport, func(t *testing.T) {
					t.Parallel()
					em := NewExchangeManager()
					realExchange, err := em.NewExchangeByName("Binance")
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
						Exchange:    "Binance",
						Pair:        &gctrpc.CurrencyPair{Base: "BTC", Quote: "USDT", Delimiter: "-"},
						Side:        "buy",
						OrderType:   tc.orderType,
						Amount:      tc.amount,
						Price:       tc.price,
						QuoteAmount: tc.quote,
						AssetType:   asset.Spot.String(),
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
					oType, err := order.StringToOrderType(tc.orderType)
					require.NoError(t, err, "order type must parse")
					exp := &order.Submit{
						Exchange: "Binance", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "-"), AssetType: asset.Spot, Side: order.Buy, Type: oType,
						Amount: tc.amount, Price: tc.price, QuoteAmount: tc.quote,
					}
					assert.Equal(t, exp, exch.submission, "the exchange should receive the requested amounts")
				})
			}
		}
	})

	t.Run("limit prices by asset", func(t *testing.T) {
		t.Parallel()
		spreadPair := currency.NewPairWithDelimiter("BTC", "USDT_BTC-USDT-SWAP", currency.DashDelimiter)
		spotPair := currency.NewPairWithDelimiter("BTC", "USDT", currency.DashDelimiter)
		for _, tc := range []struct {
			name  string
			a     asset.Item
			pair  currency.Pair
			price float64
			err   error
		}{
			{name: "spread at zero", a: asset.Spread, pair: spreadPair},
			{name: "spread below zero", a: asset.Spread, pair: spreadPair, price: -2},
			{name: "spot at zero", a: asset.Spot, pair: spotPair, err: order.ErrPriceMustBeSetIfLimitOrder},
			{name: "spot below zero", a: asset.Spot, pair: spotPair, price: -2, err: order.ErrPriceMustBeSetIfLimitOrder},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				em := NewExchangeManager()
				realExchange, err := em.NewExchangeByName("Okx")
				require.NoError(t, err, "test exchange must be available")
				realExchange.SetDefaults()
				base := realExchange.GetBase()
				base.Enabled = true
				base.CurrencyPairs.Pairs = map[asset.Item]*currency.PairStore{tc.a: {
					Available:     currency.Pairs{tc.pair},
					Enabled:       currency.Pairs{tc.pair},
					AssetEnabled:  true,
					ConfigFormat:  &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
					RequestFormat: &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter},
				}}
				exch := &rpcSubmitExchange{IBotExchange: realExchange}
				require.NoError(t, em.Add(exch), "test exchange must be registered")
				var wg sync.WaitGroup
				manager, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
				require.NoError(t, err, "order manager must be configured")
				manager.started.Store(true)
				server := &RPCServer{Engine: &Engine{ExchangeManager: em, OrderManager: manager}}
				_, err = server.SubmitOrder(t.Context(), &gctrpc.SubmitOrderRequest{
					Exchange:  "Okx",
					Pair:      &gctrpc.CurrencyPair{Base: tc.pair.Base.String(), Quote: tc.pair.Quote.String(), Delimiter: currency.DashDelimiter},
					Side:      "buy",
					OrderType: "limit",
					Price:     tc.price,
					Amount:    1,
					AssetType: tc.a.String(),
				})
				if tc.err != nil {
					require.ErrorIs(t, err, tc.err, "SubmitOrder must refuse the limit price")
					assert.Nil(t, exch.submission, "a refused order should not reach the exchange")
					return
				}
				require.ErrorIs(t, err, errExpectedTestError, "request must reach the captured exchange submission")
				exp := &order.Submit{Exchange: "Okx", Pair: tc.pair, AssetType: tc.a, Side: order.Buy, Type: order.Limit, Amount: 1, Price: tc.price}
				assert.Equal(t, exp, exch.submission, "the exchange should receive the spread's limit price")
			})
		}
	})
}
