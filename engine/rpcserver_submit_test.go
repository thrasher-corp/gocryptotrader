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
}
