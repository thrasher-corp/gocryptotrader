package engine

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbase"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deribit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestOrderManagerLookupResponseIdentity(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		newExchange func() exchange.IBotExchange
		responses   map[string]string
		wantPair    string
		wantDate    time.Time
		requested   []asset.Item
	}{
		{
			name: "coinbase", newExchange: func() exchange.IBotExchange { return new(coinbase.Exchange) },
			responses: map[string]string{
				"/orders/historical/1":     `{"order":{"order_id":"1","product_id":"SLP-20DEC30-CDE","product_type":"FUTURE","side":"BUY","status":"OPEN","created_time":"2026-10-22T17:00:00.123Z","order_configuration":{"limit_limit_gtc":{"base_size":"3","limit_price":"26"}}}}`,
				"/orders/historical/fills": `{"fills":[],"cursor":"0"}`,
			},
			wantPair: "SLP-20DEC30-CDE", wantDate: time.UnixMilli(1792688400123),
			requested: []asset.Item{asset.Spot, asset.Futures, asset.All, asset.Empty},
		},
		{
			name: "deribit", newExchange: func() exchange.IBotExchange { return new(deribit.Exchange) },
			responses: map[string]string{
				"/private/get_order_state": `{"result":{"order_id":"1","instrument_name":"BTC-PERPETUAL","direction":"buy","order_type":"limit","order_state":"open","time_in_force":"good_til_cancelled","price":26,"amount":3,"filled_amount":1,"creation_timestamp":1792648800123,"last_update_timestamp":1792648860456}}`,
			},
			wantPair: "BTC-PERPETUAL", wantDate: time.UnixMilli(1792648800123),
			requested: []asset.Item{asset.Spot, asset.Futures, asset.Options},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := tc.newExchange()
			require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
			b := ex.GetBase()
			b.SkipAuthCheck = true
			b.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: base64.StdEncoding.EncodeToString(make([]byte, 32))})
			require.NoError(t, b.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
			server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				for suffix, body := range tc.responses {
					if strings.HasSuffix(r.URL.Path, suffix) {
						_, err := w.Write([]byte(body))
						assert.NoError(t, err, "fixture response should write")
						return
					}
				}
				t.Errorf("lookup should not query unexpected path %s", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}))
			require.NoError(t, b.SetHTTPClient(server.Client()), "fixture client must be installed")
			require.NoError(t, b.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "fixture URL must be installed")
			em := NewExchangeManager()
			require.NoError(t, em.Add(ex), "real wrapper must be registered")
			var wg sync.WaitGroup
			manager, err := SetupOrderManager(em, &CommunicationManager{}, &wg, &config.OrderManager{})
			require.NoError(t, err, "order manager setup must succeed")
			manager.started.Store(true)
			for _, requested := range tc.requested {
				got, err := manager.GetOrderInfo(t.Context(), ex.GetName(), "1", currency.NewBTCUSD(), requested)
				require.NoError(t, err, "real wrapper lookup must survive the position tracker")
				assert.Equal(t, asset.Futures, got.AssetType, "stored order should use its returned market")
				assert.Equal(t, tc.wantPair, got.Pair.String(), "stored order should not inherit the requested spot pair")
				assert.True(t, tc.wantDate.Equal(got.Date), "stored order should retain its creation time")
				positions, err := manager.orderStore.futuresPositionController.GetPositionsForExchange(ex.GetName(), asset.Futures, got.Pair)
				require.NoError(t, err, "returned instrument must have a tracked futures position")
				assert.Len(t, positions, 1, "repeated lookups should retain one position for the returned instrument")
				_, err = manager.orderStore.futuresPositionController.GetPositionsForExchange(ex.GetName(), asset.Futures, currency.NewBTCUSD())
				assert.ErrorIs(t, err, futures.ErrPositionNotFound, "the requested spot pair should not acquire a futures position")
			}
		})
	}
}

func TestRPCServerBTSEActiveOrderAsset(t *testing.T) {
	t.Parallel()
	ex := new(btse.Exchange)
	require.NoError(t, testexch.Setup(ex), "exchange setup must succeed")
	ex.SkipAuthCheck = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: "mock-secret"})
	require.NoError(t, ex.Requester.DisableRateLimiter(), "mock requests must not wait for rate limits")
	sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Spot, currency.NewBTCUSD())
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload string
		switch r.URL.Path {
		case "/spot/api/v3.2/user/open_orders":
			payload = `[{"orderID":"1","symbol":"BTC-USD","side":"BUY","orderState":"open","orderType":76,"size":3,"filledSize":1,"price":26,"timestamp":1792648800123}]`
		case "/spot/api/v3.2/user/trade_history":
			payload = `[{"orderId":"1","tradeId":"fill-1","side":"BUY","price":25,"size":1,"feeAmount":0.02,"timestamp":1792648860456}]`
		default:
			t.Errorf("active order lookup should not query unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, err := w.Write([]byte(payload))
		assert.NoError(t, err, "fixture response should write")
	}))
	require.NoError(t, ex.SetHTTPClient(server.Client()), "fixture client must be installed")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "fixture URL must be installed")
	em := NewExchangeManager()
	require.NoError(t, em.Add(ex), "real BTSE wrapper must be registered")
	s := RPCServer{Engine: &Engine{ExchangeManager: em}}
	got, err := s.GetOrders(t.Context(), &gctrpc.GetOrdersRequest{
		Exchange: ex.Name, AssetType: "spot", Pair: &gctrpc.CurrencyPair{Base: "BTC", Quote: "USD"},
		StartDate: time.UnixMilli(1792648800123).Add(-time.Hour).Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   time.UnixMilli(1792648860456).Add(time.Hour).Format(common.SimpleTimeFormatWithTimezone),
	})
	require.NoError(t, err, "RPC active order lookup must succeed through the real wrapper")
	require.Len(t, got.Orders, 1, "RPC must return the active order")
	assert.Equal(t, "spot", got.Orders[0].AssetType, "RPC order should identify the spot market")
	require.Len(t, got.Orders[0].Trades, 1, "RPC must retain the fill")
	assert.Equal(t, "spot", got.Orders[0].Trades[0].AssetType, "RPC fill should agree with its order's market")
	assert.Equal(t, "fill-1", got.Orders[0].Trades[0].Id, "RPC should retain the actual fill ID")
}
