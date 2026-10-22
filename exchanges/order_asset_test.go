package exchange_test

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitfinex"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bithumb"
	"github.com/thrasher-corp/gocryptotrader/exchanges/bitstamp"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btcmarkets"
	"github.com/thrasher-corp/gocryptotrader/exchanges/btse"
	"github.com/thrasher-corp/gocryptotrader/exchanges/coinbase"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deribit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/gemini"
	"github.com/thrasher-corp/gocryptotrader/exchanges/hitbtc"
	"github.com/thrasher-corp/gocryptotrader/exchanges/huobi"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/yobit"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

// These wrappers query a fixed market or look up an order by ID. The requested asset
// is not evidence of the returned order's market, even when the caller supplies it.
func TestGetOrderInfoReportedAsset(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name            string
		newExchange     func() exchange.IBotExchange
		responses       map[string]string
		requestedAssets []asset.Item
		wantAsset       asset.Item
		wantTrades      int
		wantErr         error
	}{
		{
			name: "btse", newExchange: func() exchange.IBotExchange { return new(btse.Exchange) }, wantAsset: asset.Spot, wantTrades: 1,
			responses: map[string]string{
				"/spot/api/v3.2/user/open_orders":   `[{"orderID":"1","symbol":"BTC-USD","side":"BUY","orderState":"open","orderType":1}]`,
				"/spot/api/v3.2/user/trade_history": `[{"orderId":"1","tradeId":"fill-1","side":"BUY","timestamp":1789618131546}]`,
			},
		},
		{
			name: "bithumb", newExchange: func() exchange.IBotExchange { return new(bithumb.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{"/info/orders": `{"status":"0000","data":[{"order_id":"1","type":"bid"}]}`},
		},
		{
			name: "bitstamp", newExchange: func() exchange.IBotExchange { return new(bitstamp.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{"/order_status/": `{"id":"1","status":"Open","transactions":[]}`},
		},
		{
			name: "btcmarkets", newExchange: func() exchange.IBotExchange { return new(btcmarkets.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{"/v3/orders/1": `{"orderId":"1","marketId":"BTC-AUD","side":"Bid","type":"Limit","status":"Accepted"}`},
		},
		{
			name: "gemini", newExchange: func() exchange.IBotExchange { return new(gemini.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{"/v1/order/status": `{"order_id":"1","symbol":"BTC-USD","type":"exchange limit","side":"buy"}`},
		},
		{
			name: "hitbtc", newExchange: func() exchange.IBotExchange { return new(hitbtc.Exchange) }, wantAsset: asset.Spot,
			requestedAssets: []asset.Item{asset.Spot},
			responses:       map[string]string{"/api/2/order/1": `{"id":"1","symbol":"BTCUSD","side":"buy"}`},
		},
		{
			name: "yobit", newExchange: func() exchange.IBotExchange { return new(yobit.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{"": `{"1":{"pair":"btc_usd","type":"buy"}}`},
		},
		{
			name: "bitfinex_open_spot", newExchange: func() exchange.IBotExchange { return new(bitfinex.Exchange) }, wantAsset: asset.Spot,
			requestedAssets: []asset.Item{asset.Spot, asset.Margin},
			responses: map[string]string{
				"/hist":      `[]`,
				"/v1/orders": `[{"id":1,"order_id":1,"symbol":"BTC-USD","side":"buy","type":"exchange limit"}]`,
			},
		},
		{
			name: "bitfinex_open_margin", newExchange: func() exchange.IBotExchange { return new(bitfinex.Exchange) }, wantAsset: asset.Margin,
			requestedAssets: []asset.Item{asset.Spot, asset.Margin},
			responses: map[string]string{
				"/hist":      `[]`,
				"/v1/orders": `[{"id":1,"order_id":1,"symbol":"BTC-USD","side":"buy","type":"limit"}]`,
			},
		},
		{
			name: "bitfinex_inactive_spot", newExchange: func() exchange.IBotExchange { return new(bitfinex.Exchange) }, wantAsset: asset.Spot,
			requestedAssets: []asset.Item{asset.Spot, asset.Margin},
			responses:       map[string]string{"/hist": `[{"id":1,"order_id":1,"symbol":"BTC-USD","side":"buy","type":"exchange limit"}]`},
		},
		{
			name: "bitfinex_inactive_margin", newExchange: func() exchange.IBotExchange { return new(bitfinex.Exchange) }, wantAsset: asset.Margin,
			requestedAssets: []asset.Item{asset.Spot, asset.Margin},
			responses:       map[string]string{"/hist": `[{"id":1,"order_id":1,"symbol":"BTC-USD","side":"buy","type":"limit"}]`},
		},
		{
			name: "bitfinex_unknown_type", newExchange: func() exchange.IBotExchange { return new(bitfinex.Exchange) }, wantAsset: asset.Empty,
			requestedAssets: []asset.Item{asset.Spot, asset.Margin},
			responses:       map[string]string{"/hist": `[{"id":1,"order_id":1,"symbol":"BTC-USD","side":"buy","type":"unrecognised"}]`},
		},
		{
			name: "coinbase_spot", newExchange: func() exchange.IBotExchange { return new(coinbase.Exchange) }, wantAsset: asset.Spot,
			responses: map[string]string{
				"/orders/historical/1":     `{"order":{"order_id":"1","product_id":"BTC-USD","product_type":"SPOT"}}`,
				"/orders/historical/fills": `{"fills":[],"cursor":"0"}`,
			},
		},
		{
			name: "coinbase_futures", newExchange: func() exchange.IBotExchange { return new(coinbase.Exchange) }, wantAsset: asset.Futures,
			responses: map[string]string{
				"/orders/historical/1":     `{"order":{"order_id":"1","product_id":"BTC-USD","product_type":"FUTURE"}}`,
				"/orders/historical/fills": `{"fills":[],"cursor":"0"}`,
			},
		},
		{
			name: "coinbase_unknown_product", newExchange: func() exchange.IBotExchange { return new(coinbase.Exchange) }, wantErr: asset.ErrNotSupported,
			responses: map[string]string{"/orders/historical/1": `{"order":{"order_id":"1","product_id":"BTC-USD","product_type":"UNKNOWN_PRODUCT_TYPE"}}`},
		},
		{
			name: "coinbase_missing_product", newExchange: func() exchange.IBotExchange { return new(coinbase.Exchange) }, wantErr: asset.ErrNotSupported,
			responses: map[string]string{"/orders/historical/1": `{"order":{"order_id":"1","product_id":"BTC-USD"}}`},
		},
		{
			name: "deribit_spot", newExchange: func() exchange.IBotExchange { return new(deribit.Exchange) }, wantAsset: asset.Spot,
			requestedAssets: []asset.Item{asset.Spot, asset.Futures, asset.Options},
			responses:       map[string]string{"/private/get_order_state": `{"result":{"order_id":"1","instrument_name":"BTC_USDC","order_type":"limit","order_state":"open","time_in_force":"good_til_cancelled"}}`},
		},
		{
			name: "deribit_futures", newExchange: func() exchange.IBotExchange { return new(deribit.Exchange) }, wantAsset: asset.Futures,
			requestedAssets: []asset.Item{asset.Spot, asset.Futures, asset.Options},
			responses:       map[string]string{"/private/get_order_state": `{"result":{"order_id":"1","instrument_name":"BTC-PERPETUAL","order_type":"limit","order_state":"open","time_in_force":"good_til_cancelled"}}`},
		},
		{
			name: "deribit_options", newExchange: func() exchange.IBotExchange { return new(deribit.Exchange) }, wantAsset: asset.Options,
			requestedAssets: []asset.Item{asset.Spot, asset.Futures, asset.Options},
			responses:       map[string]string{"/private/get_order_state": `{"result":{"order_id":"1","instrument_name":"BTC-26SEP25-30000-C","order_type":"limit","order_state":"open","time_in_force":"good_til_cancelled"}}`},
		},
		{
			name: "huobi_futures", newExchange: func() exchange.IBotExchange { return new(huobi.Exchange) }, wantAsset: asset.Futures, wantTrades: 1,
			requestedAssets: []asset.Item{asset.Futures},
			responses:       map[string]string{"/api/v1/contract_order_info": `{"status":"ok","data":[{"direction":"buy","order_price_type":"limit","status":3,"order_id_str":"1"}]}`},
		},
		{
			name: "huobi_coin_margined_futures", newExchange: func() exchange.IBotExchange { return new(huobi.Exchange) }, wantAsset: asset.CoinMarginedFutures, wantTrades: 1,
			requestedAssets: []asset.Item{asset.CoinMarginedFutures},
			responses:       map[string]string{"/swap-api/v1/swap_order_info": `{"status":"ok","data":[{"direction":"buy","order_price_type":"limit","status":3,"order_id_str":"1"}]}`},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := tc.newExchange()
			require.NoError(t, testexch.Setup(ex), "Setup must not error")
			b := ex.GetBase()
			b.SkipAuthCheck = true
			b.SetCredentials(&accounts.Credentials{Key: "mock-key", Secret: base64.StdEncoding.EncodeToString(make([]byte, 32))})
			pair := currency.NewBTCUSD()
			sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Spot, pair)
			if b.CurrencyPairs.IsAssetSupported(asset.Margin) {
				sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, asset.Margin, pair)
			}
			for _, a := range tc.requestedAssets {
				sharedtestvalues.SetupCurrencyPairsForExchangeAsset(t, ex, a, pair)
			}
			require.NoError(t, b.SetHTTPClient(&http.Client{Transport: orderAssetTransport{t: t, responses: tc.responses}}), "SetHTTPClient must not error")
			requestedAssets := tc.requestedAssets
			if len(requestedAssets) == 0 {
				requestedAssets = []asset.Item{asset.Spot, asset.Futures, asset.All, asset.Empty}
			}
			for _, requested := range requestedAssets {
				detail, err := ex.GetOrderInfo(t.Context(), "1", pair, requested)
				if tc.wantErr != nil {
					assert.ErrorIs(t, err, tc.wantErr, "GetOrderInfo should reject an unknown returned market")
					assert.Nil(t, detail, "GetOrderInfo should not fabricate an order's market")
					continue
				}
				require.NoErrorf(t, err, "GetOrderInfo must not error for requested asset %s", requested)
				require.NotNil(t, detail, "GetOrderInfo must return an order")
				assert.Equalf(t, tc.wantAsset, detail.AssetType, "GetOrderInfo should report the queried or returned market, not requested asset %s", requested)
				assert.Len(t, detail.Trades, tc.wantTrades, "GetOrderInfo should retain the order's fills")
			}
		})
	}
}

// All HTTP calls stay in memory, including wrappers that still hard-code their API URL.
type orderAssetTransport struct {
	t         *testing.T
	responses map[string]string
}

func (m orderAssetTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	for suffix, body := range m.responses {
		if strings.HasSuffix(r.URL.Path, suffix) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    r,
			}, nil
		}
	}
	m.t.Errorf("GetOrderInfo should only query the expected endpoints, got %s", r.URL.Path)
	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader(`{}`)), Request: r}, nil
}
