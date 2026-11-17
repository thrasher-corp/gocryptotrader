package bingx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The three days of candles and funding rates the recorded responses hold, from the start of 8 November 2026 UTC
var (
	wrapperFixtureStart = time.UnixMilli(1794096000000).UTC()
	wrapperFixtureEnd   = time.UnixMilli(1794355200000).UTC()
)

// newWrapperTestExchange returns an exchange named after the test, so the ticker, order book, account, limit and
// currency state stores it writes are its own, with each asset's test pair enabled. Mock tests serve it the recorded
// responses with placeholder credentials; live tests load the tradable pairs and use the test credentials, if any
func newWrapperTestExchange(t *testing.T) *Exchange {
	t.Helper()
	ensureTradablePairs(t)
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	if mockTests {
		require.NoError(t, testexch.MockHTTPInstance(ex), "MockHTTPInstance must not error")
		// Mock responses are served locally, so throttling protects nothing
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
		ex.API.AuthenticatedSupport = true
		ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	} else {
		ex.CurrencyPairs.Load(&e.CurrencyPairs)
		if apiCredentials.Key != "" && apiCredentials.Secret != "" {
			ex.API.AuthenticatedSupport = true
			ex.SetCredentials(apiCredentials)
		}
	}
	for a, p := range assetToTradablePairMap {
		err := ex.CurrencyPairs.EnablePair(a, p)
		if errors.Is(err, currency.ErrPairAlreadyEnabled) {
			continue
		}
		require.NoErrorf(t, err, "EnablePair must not error for %s %s", a, p)
	}
	return ex
}

// wrapperSkipIfCannotManipulateOrders skips a test that would place, change or cancel orders, or change positions or
// account settings, live without credentials or permission
func wrapperSkipIfCannotManipulateOrders(t *testing.T, ex *Exchange) {
	t.Helper()
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	if !mockTests {
		sharedtestvalues.SkipTestIfCannotManipulateOrders(t, ex, canManipulateRealOrders)
	}
}

// wrapperRequest is a request a wrapper test server answered, with its parameters other than the receive window,
// timestamp and signature, and the API key it carried
type wrapperRequest struct {
	method string
	path   string
	params url.Values
	apiKey string
}

// wrapperNoReply is a reply wrapperTestServer gives by dropping the connection without answering, as a lost reply does
const wrapperNoReply = "no reply"

// wrapperCredentialsChange prefixes a reply wrapperTestServer gives once it has changed the exchange's default
// credentials to ones with the API key wrapperOtherAPIKey, as another caller may while a request is in flight
const (
	wrapperCredentialsChange = "credentials change: "
	wrapperOtherAPIKey       = "other-key"
)

// wrapperStatusPrefix prefixes a reply wrapperTestServer gives with an HTTP status other than 200
const wrapperStatusPrefix = "status "

// wrapperStatusReply returns a reply wrapperTestServer gives with an HTTP status other than 200, such as the error page
// of a firewall or gateway in front of BingX
func wrapperStatusReply(status int, body string) string {
	return wrapperStatusPrefix + strconv.Itoa(status) + " " + body
}

// wrapperTestServer returns an exchange whose REST endpoints answer each request with the next reply routes holds for
// its method and path, such as "GET /openApi/swap/v2/quote/ticker", or with none for wrapperNoReply, and a function
// returning the requests received so far. A request for a route without a reply left fails the test, as does a
// signature made with other than the test credentials
func wrapperTestServer(t *testing.T, routes map[string][]string) (ex *Exchange, requests func() []wrapperRequest) {
	t.Helper()
	var mu sync.Mutex
	var received []wrapperRequest
	ex = newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		if params.Has("signature") {
			verifyQuerySignature(t, r.URL.RawQuery)
		}
		for _, k := range []string{"recvWindow", "timestamp", "signature"} {
			params.Del(k)
		}
		route := r.Method + " " + r.URL.Path
		mu.Lock()
		received = append(received, wrapperRequest{method: r.Method, path: r.URL.Path, params: params, apiKey: r.Header.Get(keyHeader)})
		replies := routes[route]
		var reply string
		if len(replies) != 0 {
			reply, routes[route] = replies[0], replies[1:]
		}
		mu.Unlock()
		if len(replies) == 0 {
			assert.Failf(t, "unexpected request", "%s should not be requested", route)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if after, ok := strings.CutPrefix(reply, wrapperCredentialsChange); ok {
			ex.SetCredentials(&accounts.Credentials{Key: wrapperOtherAPIKey, Secret: "other-secret"})
			reply = after
		}
		if reply == wrapperNoReply {
			panic(http.ErrAbortHandler)
		}
		status := http.StatusOK
		if after, ok := strings.CutPrefix(reply, wrapperStatusPrefix); ok {
			var code string
			code, reply, _ = strings.Cut(after, " ")
			var err error
			status, err = strconv.Atoi(code)
			assert.NoErrorf(t, err, "the status of reply %q should be a number", after)
		}
		w.WriteHeader(status)
		_, err := w.Write([]byte(reply))
		assert.NoError(t, err, "Write should not error")
	})
	return ex, func() []wrapperRequest {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

// wrapperRecordsPage returns a JSON array of n records made by record from their index, starting at first
func wrapperRecordsPage(first, n int, record func(i int) string) string {
	records := make([]string, n)
	for i := range n {
		records[i] = record(first + i)
	}
	return "[" + strings.Join(records, ",") + "]"
}

// wrapperMilliseconds returns a time's Unix milliseconds as a request parameter
func wrapperMilliseconds(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "BingX", ex.Name, "SetDefaults should set the name")
	assert.Equal(t, protocol.TradingRequirements{SpotMarketBuyQuotation: true, SpotMarketSellBase: true}, ex.GetTradingRequirements(), "SetDefaults should take spot market buys in quote and sells in base")
	assert.ElementsMatch(t, asset.Items{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures}, ex.GetAssetTypes(false), "SetDefaults should support spot and both perpetual assets")
	for _, a := range ex.GetAssetTypes(false) {
		format, err := ex.GetPairFormat(a, true)
		require.NoErrorf(t, err, "GetPairFormat must not error for %s", a)
		assert.Equalf(t, currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}, format, "SetDefaults should format %s pairs as BASE-QUOTE", a)
	}
	assert.True(t, ex.Features.Supports.RESTCapabilities.TickerBatching, "SetDefaults should batch tickers")
	assert.True(t, ex.Features.Supports.RESTCapabilities.ModifyOrder, "SetDefaults should modify orders")
	assert.Equal(t, exchange.AutoWithdrawCrypto|exchange.NoFiatWithdrawals, ex.Features.Supports.WithdrawPermissions, "SetDefaults should withdraw crypto alone")
	assert.True(t, ex.Features.Supports.WebsocketCapabilities.Subscribe, "SetDefaults should keep the websocket capabilities")
	assert.Equal(t, map[kline.Interval]bool{kline.OneHour: true, kline.FourHour: true, kline.EightHour: true}, ex.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies, "SetDefaults should list every funding interval")
	assert.Equal(t, map[asset.Item]bool{asset.USDTMarginedFutures: true, asset.CoinMarginedFutures: true}, ex.Features.Supports.FuturesCapabilities.FundingRateBatching, "SetDefaults should batch funding rates of both perpetual assets")
	assert.Equal(t, uint64(coinMarginedCandleLimit), ex.Features.Enabled.Kline.GlobalResultLimit, "SetDefaults should limit a candle request to the smallest product limit")
	for _, interval := range []kline.Interval{kline.OneMin, kline.SixHour, kline.OneDay, kline.OneWeek, kline.OneMonth} {
		assert.Truef(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(interval), "SetDefaults should enable %s candles", interval)
	}
	assert.False(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(kline.ThreeDay), "SetDefaults should build three day candles from daily ones")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestUSDTMargined, exchange.RestCoinMargined} {
		endpoint, err := ex.API.Endpoints.GetURL(u)
		require.NoErrorf(t, err, "GetURL must not error for %s", u)
		assert.Equalf(t, apiURL, endpoint, "SetDefaults should set the %s endpoint", u)
	}
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	require.Error(t, ex.Setup(nil), "Setup must reject a nil configuration")
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should disable a disabled exchange")
	assert.False(t, ex.Websocket.IsInitialised(), "Setup should leave a disabled exchange's websocket unset")

	ex = newWebsocketTestExchange(t)
	assert.True(t, ex.IsEnabled(), "Setup should enable an enabled exchange")
	assert.True(t, ex.Websocket.IsInitialised(), "Setup should set up the websocket")

	for _, tc := range []struct {
		name      string
		configure func(*Exchange, *config.Exchange)
		errIs     error
		errText   string
	}{
		{
			name:      "default setup error",
			configure: func(_ *Exchange, cfg *config.Exchange) { cfg.ProxyAddress = string(rune(0x7f)) },
			errIs:     exchange.ErrSettingProxyAddress,
		},
		{
			name:      "websocket setup error",
			configure: func(_ *Exchange, cfg *config.Exchange) { cfg.WebsocketTrafficTimeout = time.Millisecond },
			errText:   "invalid traffic timeout",
		},
		{
			name:      "connection setups error",
			configure: func(ex *Exchange, _ *config.Exchange) { ex.API.Endpoints = ex.NewEndpoints() },
			errIs:     exchange.ErrEndpointPathNotFound,
		},
		{
			name:      "connection setup refused",
			configure: func(ex *Exchange, _ *config.Exchange) { ex.Websocket.TrafficAlert = nil },
			errText:   "traffic alert is nil",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			ex.Name = t.Name()
			cfg, err := ex.GetStandardConfig()
			require.NoError(t, err, "GetStandardConfig must not error")
			tc.configure(ex, cfg)
			err = ex.Setup(cfg)
			if tc.errIs != nil {
				assert.ErrorIsf(t, err, tc.errIs, "Setup should return the %s", tc.name)
				return
			}
			assert.ErrorContainsf(t, err, tc.errText, "Setup should return the %s", tc.name)
		})
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an unsupported asset")
	ensureTradablePairs(t)
	for _, tc := range []struct {
		a   asset.Item
		exp []string
	}{
		// The recorded spot symbols are all offline, and the recorded coin-M contracts leave out BTC-USD
		{asset.Spot, nil},
		{asset.USDTMarginedFutures, []string{"BTC-USDT", "ETH-USDT"}},
		{asset.CoinMarginedFutures, []string{"BNB-USD", "SUI-USD", "UNI-USD"}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			pairs, err := e.FetchTradablePairs(t.Context(), tc.a)
			require.NoError(t, err, "FetchTradablePairs must not error")
			if !mockTests {
				assert.Containsf(t, pairs.Strings(), assetToTradablePairMap[tc.a].String(), "FetchTradablePairs should list the %s test pair", tc.a)
				return
			}
			if tc.exp == nil {
				assert.Empty(t, pairs, "FetchTradablePairs should leave out offline symbols")
				return
			}
			assert.Equal(t, tc.exp, pairs.Strings(), "FetchTradablePairs should list the tradable symbols")
		})
	}

	t.Run("filters", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/common/symbols": {`{"code":0,"data":{"symbols":[
				{"symbol":"BTC-USDT","status":1,"apiStateBuy":true,"apiStateSell":false},
				{"symbol":"ETH-USDT","status":1,"apiStateBuy":false,"apiStateSell":true},
				{"symbol":"GPT-OLD-USDT","status":1,"apiStateBuy":true,"apiStateSell":true},
				{"symbol":"SHUT-USDT","status":1,"apiStateBuy":false,"apiStateSell":false},
				{"symbol":"OFF-USDT","status":25,"apiStateBuy":true,"apiStateSell":true},
				{"symbol":"NODASH","status":1,"apiStateBuy":true,"apiStateSell":true}]}}`},
			"GET /openApi/swap/v2/quote/contracts": {`{"code":0,"data":[
				{"symbol":"BTC-USDC","status":1,"apiStateOpen":"true"},
				{"symbol":"ETH-USDT","status":1,"apiStateOpen":"false"},
				{"symbol":"SOL-USDT","status":25,"apiStateOpen":"true"}]}`},
			"GET /openApi/cswap/v1/market/contracts": {`{"code":0,"data":[{"symbol":"BTC-USD","status":1},{"symbol":"ETH-USD","status":0}]}`},
		})
		pairs, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
		require.NoError(t, err, "FetchTradablePairs must not error")
		assert.Equal(t, []string{"BTC-USDT", "ETH-USDT", "GPT-OLD-USDT"}, pairs.Strings(), "FetchTradablePairs should list spot symbols online with buying or selling open")
		require.Len(t, pairs, 3, "FetchTradablePairs must return three spot pairs")
		assert.Equal(t, "GPT-OLD", pairs[2].Base.String(), "FetchTradablePairs should split a symbol at its last dash")
		pairs, err = ex.FetchTradablePairs(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "FetchTradablePairs must not error")
		assert.Equal(t, []string{"BTC-USDC"}, pairs.Strings(), "FetchTradablePairs should list USDT-M contracts online and open to the API, USDC-settled ones included")
		pairs, err = ex.FetchTradablePairs(t.Context(), asset.CoinMarginedFutures)
		require.NoError(t, err, "FetchTradablePairs must not error")
		assert.Equal(t, []string{"BTC-USD"}, pairs.Strings(), "FetchTradablePairs should list coin-M contracts online")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/common/symbols":    {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/quote/contracts":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/contracts": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.FetchTradablePairs(t.Context(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "FetchTradablePairs should return the %s request's error", a)
		}
	})

	t.Run("pair format missing", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/common/symbols": {`{"code":0,"data":{"symbols":[{"symbol":"BTC-USDT","status":1,"apiStateBuy":true,"apiStateSell":true}]}}`},
		})
		ex.CurrencyPairs.Delete(asset.Spot)
		_, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, currency.ErrAssetNotFound, "FetchTradablePairs should return the pair format lookup's error")
		assert.Len(t, requests(), 1, "FetchTradablePairs should fetch the symbols before formatting their pairs")
	})
}

func TestSymbolPair(t *testing.T) {
	t.Parallel()
	format := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	pair, err := symbolPair("btc-usdt", format)
	require.NoError(t, err, "symbolPair must not error")
	assert.Equal(t, "BTC-USDT", pair.String(), "symbolPair should return the pair in the format")
	pair, err = symbolPair("GPT-OLD-USDT", format)
	require.NoError(t, err, "symbolPair must not error")
	assert.Equal(t, "GPT-OLD", pair.Base.String(), "symbolPair should split a symbol at its last dash")
	assert.Equal(t, "USDT", pair.Quote.String(), "symbolPair should take the quote after the last dash")
	for _, symbol := range []string{"BTCUSDT", "-USDT", "BTC-", ""} {
		_, err = symbolPair(symbol, format)
		assert.ErrorIsf(t, err, errWebsocketInvalidSymbol, "symbolPair should reject %q", symbol)
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/common/symbols":    {`{"code":0,"data":{"symbols":[{"symbol":"BTC-USDT","status":1,"apiStateBuy":true,"apiStateSell":true}]}}`},
		"GET /openApi/swap/v2/quote/contracts":   {`{"code":100500,"msg":"busy"}`},
		"GET /openApi/cswap/v1/market/contracts": {`{"code":0,"data":[{"symbol":"ETH-USD","status":1}]}`},
	})
	require.ErrorIs(t, ex.UpdateTradablePairs(t.Context()), errAPIResponse, "UpdateTradablePairs must return the failing asset's error")
	pairs, err := ex.GetAvailablePairs(asset.CoinMarginedFutures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	assert.Equal(t, []string{"ETH-USD"}, pairs.Strings(), "UpdateTradablePairs should update the assets that did not fail")

	t.Run("no asset enabled", func(t *testing.T) {
		t.Parallel()
		// Each listing keeps its asset's enabled pair, as replacing enabled pairs would enable the asset again
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/common/symbols":    {`{"code":0,"data":{"symbols":[{"symbol":"BTC-USDT","status":1,"apiStateBuy":true,"apiStateSell":true}]}}`},
			"GET /openApi/swap/v2/quote/contracts":   {`{"code":0,"data":[{"symbol":"BTC-USDT","status":1,"apiStateOpen":"true"}]}`},
			"GET /openApi/cswap/v1/market/contracts": {`{"code":0,"data":[{"symbol":"BTC-USD","status":1}]}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			require.NoErrorf(t, ex.CurrencyPairs.SetAssetEnabled(a, false), "SetAssetEnabled must not error for %s", a)
		}
		require.ErrorIs(t, ex.UpdateTradablePairs(t.Context()), currency.ErrCurrencyPairsEmpty, "UpdateTradablePairs must report that no pair can be enabled")
		pairs, err := ex.GetAvailablePairs(asset.CoinMarginedFutures)
		require.NoError(t, err, "GetAvailablePairs must not error")
		assert.Equal(t, []string{"BTC-USD"}, pairs.Strings(), "UpdateTradablePairs should update every asset's pairs first")
	})

	t.Run("every asset", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		err := ex.UpdateTradablePairs(t.Context())
		if mockTests {
			// The recorded spot symbols are all offline, so spot keeps the pairs it had
			require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "UpdateTradablePairs must report an asset without tradable pairs")
		} else {
			require.NoError(t, err, "UpdateTradablePairs must not error")
		}
		for a, p := range assetToTradablePairMap {
			pairs, err := ex.GetAvailablePairs(a)
			require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
			if mockTests && a == asset.CoinMarginedFutures {
				assert.Equal(t, []string{"BNB-USD", "SUI-USD", "UNI-USD"}, pairs.Strings(), "UpdateTradablePairs should store the recorded coin-M pairs")
				continue
			}
			assert.Containsf(t, pairs.Strings(), p.String(), "UpdateTradablePairs should store the %s test pair", a)
		}
	})
}

func TestAvailablePair(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	pair, ok, err := ex.availablePair("BTC-USDT", asset.Spot)
	require.NoError(t, err, "availablePair must not error")
	assert.True(t, ok, "availablePair should find an available pair")
	assert.Equal(t, "BTC-USDT", pair.String(), "availablePair should return the available pair")
	for _, symbol := range []string{"NOPE-USDT", "NODASH", ""} {
		pair, ok, err = ex.availablePair(symbol, asset.Spot)
		require.NoErrorf(t, err, "availablePair must not error for %q", symbol)
		assert.Falsef(t, ok, "availablePair should report %q has no available pair", symbol)
		assert.Truef(t, pair.IsEmpty(), "availablePair should return no pair for %q", symbol)
	}
	_, _, err = new(Exchange).availablePair("BTC-USDT", asset.Spot)
	assert.Error(t, err, "availablePair should return the error of a pair manager without pairs")
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateTickers(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")

	t.Run("every asset", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		suiUSD := currency.NewPairWithDelimiter("SUI", "USD", currency.DashDelimiter)
		if mockTests {
			// The recorded coin-M tickers are SUI-USD's and BCH-USD's, of which SUI-USD is made available
			require.NoError(t, ex.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, currency.Pairs{coinMarginedTradablePair, suiUSD}, false), "StorePairs must not error")
		}
		for _, a := range ex.GetAssetTypes(false) {
			require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		}
		if !mockTests {
			for a, p := range assetToTradablePairMap {
				price, err := ticker.GetTicker(ex.Name, p, a)
				require.NoErrorf(t, err, "GetTicker must not error for %s %s", a, p)
				assert.Positivef(t, price.Last, "UpdateTickers should store the %s last price", a)
			}
			return
		}
		price, err := ticker.GetTicker(ex.Name, spotTradablePair, asset.Spot)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, &ticker.Price{
			Last:                73426.33,
			High:                74789.77,
			Low:                 73267.04,
			Bid:                 73426.32,
			BidSize:             0.003692,
			Ask:                 73426.34,
			AskSize:             0.005115,
			BaseVolume:          1481.23,
			QuoteVolume:         109617355.67,
			Open:                73735,
			PercentChange24Hour: -0.42,
			Pair:                spotTradablePair,
			ExchangeName:        ex.Name,
			AssetType:           asset.Spot,
			LastUpdated:         time.UnixMilli(1794478484303),
		}, price, "UpdateTickers should store every field of the spot ticker")
		_, err = ticker.GetTicker(ex.Name, currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), asset.Spot)
		assert.NoError(t, err, "UpdateTickers should store every available pair's ticker")
		price, err = ticker.GetTicker(ex.Name, usdtMarginedTradablePair, asset.USDTMarginedFutures)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, 0.0002, price.LastSize, "UpdateTickers should store the USDT-M ticker")
		_, err = ticker.GetTicker(ex.Name, currency.NewPairWithDelimiter("NCCOGOLD2USD", "USDT", currency.DashDelimiter), asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should skip a symbol without an available pair")
		price, err = ticker.GetTicker(ex.Name, suiUSD, asset.CoinMarginedFutures)
		require.NoError(t, err, "GetTicker must not error")
		assert.Equal(t, 2.72, price.PercentChange24Hour, "UpdateTickers should store the coin-M ticker")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/quote/ticker":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/ticker": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			assert.ErrorIsf(t, ex.UpdateTickers(t.Context(), a), errAPIResponse, "UpdateTickers should return the %s request's error", a)
		}
	})

	t.Run("ticker errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker": {`{"code":0,"data":[
				{"symbol":"BTC-USDT","lastPrice":1,"bidPrice":2,"askPrice":2},
				{"symbol":"ETH-USDT","lastPrice":3,"bidPrice":2,"askPrice":4}]}`},
			"GET /openApi/cswap/v1/market/ticker": {`{"code":0,"data":[
				{"symbol":"BTC-USD","priceChangePercent":"bad%","lastPrice":"1"},
				{"symbol":"ETH-USD","priceChangePercent":"1%","lastPrice":"2"}]}`},
		})
		require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Spot), ticker.ErrBidEqualsAsk, "UpdateTickers must report a ticker it cannot store")
		price, err := ticker.GetTicker(ex.Name, currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), asset.Spot)
		require.NoError(t, err, "UpdateTickers must store the tickers it can")
		assert.Equal(t, 3.0, price.Last, "UpdateTickers should store the valid ticker")
		require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.CoinMarginedFutures), errInvalidPercentage, "UpdateTickers must report a ticker reply it cannot decode")
	})

	t.Run("pairs never loaded", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker":   {`{"code":0,"data":[{"symbol":"BTC-USDT","lastPrice":1}]}`},
			"GET /openApi/swap/v2/quote/ticker":   {`{"code":0,"data":[{"symbol":"BTC-USDT","lastPrice":"1"}]}`},
			"GET /openApi/cswap/v1/market/ticker": {`{"code":0,"data":[{"symbol":"BTC-USD","lastPrice":"1"}]}`},
		})
		ex.CurrencyPairs = currency.PairsManager{}
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			assert.ErrorContainsf(t, ex.UpdateTickers(t.Context(), a), "pair matcher is nil", "UpdateTickers should return the %s pair lookup's error", a)
		}
	})
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must reject an empty pair")
	_, err = e.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")

	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a   asset.Item
		exp *ticker.Price
	}{
		{asset.Spot, &ticker.Price{Last: 74452.02, High: 74772.59, Low: 73221.5, Bid: 74452, BidSize: 0.005314, Ask: 74452.02, AskSize: 0.001442, BaseVolume: 1804.96, QuoteVolume: 133296362.38, Open: 73488.69, PercentChange24Hour: 1.31, LastUpdated: time.UnixMilli(1794423748498)}},
		{asset.USDTMarginedFutures, &ticker.Price{Last: 74466.8, LastSize: 0.0009, High: 74748, Low: 73187.5, Bid: 74467.2, BidSize: 15.9549, Ask: 74467.3, AskSize: 23.41, BaseVolume: 10092.81, QuoteVolume: 745120024.59, Open: 73531.9, PercentChange24Hour: 1.27, LastUpdated: time.UnixMilli(1794423585000)}},
		{asset.CoinMarginedFutures, &ticker.Price{Last: 74348.6, LastSize: 9, High: 74674.4, Low: 73100.4, Bid: 74348.6, BidSize: 3480, Ask: 74348.7, AskSize: 2670, BaseVolume: 819.6, Open: 73375.9, PercentChange24Hour: 1.33, LastUpdated: time.UnixMilli(1794423768967)}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			pair := assetToTradablePairMap[tc.a]
			result, err := ex.UpdateTicker(t.Context(), pair, tc.a)
			require.NoError(t, err, "UpdateTicker must not error")
			if !mockTests {
				assert.Positive(t, result.Last, "UpdateTicker should return the last price")
				return
			}
			tc.exp.Pair, tc.exp.ExchangeName, tc.exp.AssetType = pair, ex.Name, tc.a
			assert.Equal(t, tc.exp, result, "UpdateTicker should return every field of the ticker")
		})
	}

	t.Run("missing ticker", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker":   {`{"code":0,"data":[]}`},
			"GET /openApi/swap/v2/quote/ticker":   {`{"code":0,"data":[]}`},
			"GET /openApi/cswap/v1/market/ticker": {`{"code":0,"data":[]}`, `{"code":0,"data":[{"symbol":"BTC-USD","priceChangePercent":"x%"}]}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), a)
			assert.ErrorIsf(t, err, ticker.ErrTickerNotFound, "UpdateTicker should report a missing %s ticker", a)
		}
		_, err := ex.UpdateTicker(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), asset.CoinMarginedFutures)
		assert.ErrorIs(t, err, errInvalidPercentage, "UpdateTicker should report a ticker reply it cannot decode")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/quote/ticker":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/ticker": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "UpdateTicker should return the %s request's error", a)
		}
	})

	t.Run("invalid ticker", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/ticker": {`{"code":0,"data":[{"symbol":"BTC-USDT","bidPrice":2,"askPrice":2}]}`},
		})
		_, err := ex.UpdateTicker(t.Context(), currency.NewBTCUSDT(), asset.Spot)
		assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "UpdateTicker should return a locked ticker's processing error")
	})
}

func TestSpotTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	resp := &SpotTickerResponse{
		Symbol: "BTC-USDT", OpenPrice: 1, HighPrice: 2, LowPrice: 0.5, LastPrice: 1.5, PriceChange: 0.5, Volume: 10, QuoteVolume: 15,
		OpenTime: types.Time(time.UnixMilli(1)), CloseTime: types.Time(time.UnixMilli(2)), BidPrice: 1.4, BidQuantity: 3, AskPrice: 1.6,
		AskQuantity: 4, PriceChangePercent: 50,
	}
	assert.Equal(t, ticker.Price{
		Last: 1.5, High: 2, Low: 0.5, Bid: 1.4, BidSize: 3, Ask: 1.6, AskSize: 4, BaseVolume: 10, QuoteVolume: 15, Open: 1,
		PercentChange24Hour: 50, Pair: pair, ExchangeName: e.Name, AssetType: asset.Spot, LastUpdated: time.UnixMilli(2),
	}, e.spotTickerPrice(resp, pair), "spotTickerPrice should convert every field")
}

func TestSwapTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewBTCUSDT()
	resp := &Swap24HourTickerResponse{
		Symbol: "BTC-USDT", PriceChange: 0.5, PriceChangePercent: 50, LastPrice: 1.5, LastQuantity: 0.1, HighPrice: 2, LowPrice: 0.5,
		Volume: 10, QuoteVolume: 15, OpenPrice: 1, OpenTime: types.Time(time.UnixMilli(1)), CloseTime: types.Time(time.UnixMilli(2)),
		BidPrice: 1.4, BidQuantity: 3, AskPrice: 1.6, AskQuantity: 4,
	}
	assert.Equal(t, ticker.Price{
		Last: 1.5, LastSize: 0.1, High: 2, Low: 0.5, Bid: 1.4, BidSize: 3, Ask: 1.6, AskSize: 4, BaseVolume: 10, QuoteVolume: 15,
		Open: 1, PercentChange24Hour: 50, Pair: pair, ExchangeName: e.Name, AssetType: asset.USDTMarginedFutures,
		LastUpdated: time.UnixMilli(2),
	}, e.swapTickerPrice(resp, pair), "swapTickerPrice should convert every field")
}

func TestCoinMarginedTickerPrice(t *testing.T) {
	t.Parallel()
	pair := currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter)
	resp := &CoinMargined24HourPriceChangeResponse{
		Symbol: "BTC-USD", PriceChange: 0.5, PriceChangePercent: -1.03, LastPrice: 1.5, LastQuantity: 2, HighPrice: 2,
		LowPrice: 0.5, Volume: 1000, QuoteVolume: 12, OpenPrice: 1, CloseTime: types.Time(time.UnixMilli(2)), BidPrice: 1.4,
		BidQuantity: 3, AskPrice: 1.6, AskQuantity: 4,
	}
	assert.Equal(t, ticker.Price{
		Last: 1.5, LastSize: 2, High: 2, Low: 0.5, Bid: 1.4, BidSize: 3, Ask: 1.6, AskSize: 4, BaseVolume: 12, Open: 1,
		PercentChange24Hour: -1.03, Pair: pair, ExchangeName: e.Name, AssetType: asset.CoinMarginedFutures,
		LastUpdated: time.UnixMilli(2),
	}, e.coinMarginedTickerPrice(resp, pair), "coinMarginedTickerPrice should take the base volume from quoteVolume and leave the USD volume out")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")
	_, err = e.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an unsupported asset")

	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a            asset.Item
		bids, asks   orderbook.Levels
		lastUpdated  time.Time
		lastUpdateID int64
	}{
		{
			asset.Spot,
			orderbook.Levels{{Price: 73426.32, Amount: 0.003692}, {Price: 73423.15, Amount: 0.038286}, {Price: 73422.86, Amount: 0.000399}},
			orderbook.Levels{{Price: 74391.41, Amount: 0.000024}, {Price: 74392.54, Amount: 0.000107}, {Price: 74392.81, Amount: 0.000062}},
			time.UnixMilli(1794478485449), 16558843338,
		},
		{
			asset.USDTMarginedFutures,
			orderbook.Levels{{Price: 73391.2, Amount: 10.0368}, {Price: 73391, Amount: 0.403}, {Price: 73390.7, Amount: 0.3472}},
			orderbook.Levels{{Price: 73391.5, Amount: 3.2317}, {Price: 73391.6, Amount: 0.0134}, {Price: 73391.8, Amount: 1.0809}},
			time.UnixMilli(1794478486417), 0,
		},
		{
			asset.CoinMarginedFutures,
			orderbook.Levels{{Price: 73313.2, Amount: 1542}, {Price: 73313.1, Amount: 1}, {Price: 73308.9, Amount: 1155}},
			orderbook.Levels{{Price: 189000, Amount: 1}, {Price: 195000, Amount: 1}, {Price: 250000, Amount: 10}},
			time.UnixMilli(1794478487121), 0,
		},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			book, err := ex.UpdateOrderbook(t.Context(), assetToTradablePairMap[tc.a], tc.a)
			require.NoError(t, err, "UpdateOrderbook must not error")
			require.NotEmpty(t, book.Bids, "UpdateOrderbook must return bids")
			require.NotEmpty(t, book.Asks, "UpdateOrderbook must return asks")
			if !mockTests {
				assert.Less(t, book.Bids[0].Price, book.Asks[0].Price, "UpdateOrderbook should put the best bid below the best ask")
				return
			}
			assert.Equal(t, tc.bids, book.Bids, "UpdateOrderbook should return the bids highest first")
			assert.Equal(t, tc.asks, book.Asks, "UpdateOrderbook should return the asks lowest first")
			assert.Equal(t, tc.lastUpdated, book.LastUpdated, "UpdateOrderbook should date the book by its time")
			assert.Equal(t, tc.lastUpdateID, book.LastUpdateID, "UpdateOrderbook should keep the spot update ID")
		})
	}

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/market/depth":  {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/quote/depth":   {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/cswap/v1/market/depth": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "UpdateOrderbook should return the %s request's error", a)
		}
		got := requests()
		require.Len(t, got, 3, "UpdateOrderbook must request each asset's book")
		for _, r := range got {
			assert.Equalf(t, "1000", r.params.Get("limit"), "UpdateOrderbook should request the deepest book from %s", r.path)
		}
	})

	t.Run("invalid book", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v2/quote/depth": {`{"code":0,"data":{"T":1794478486417,"bids":[["0","1"]],"asks":[["1","1"]]}}`},
		})
		_, err := ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures)
		assert.ErrorIs(t, err, orderbook.ErrPriceZero, "UpdateOrderbook should return an invalid book's processing error")
	})
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset")

	ex := newWrapperTestExchange(t)
	// Added at run time, as the balance is, rather than exactly as constants are
	usdtFree, usdtLocked := 566773.193402631, 244.18616265388994
	for _, tc := range []struct {
		a   asset.Item
		exp map[string]accounts.Balance
	}{
		{asset.Spot, map[string]accounts.Balance{
			"USDT":   {Total: usdtFree + usdtLocked, Hold: usdtLocked, Free: usdtFree},
			"CHEEMS": {Total: 294854132046232 + 18350553840, Hold: 18350553840, Free: 294854132046232},
			"VST":    {},
		}},
		{asset.USDTMarginedFutures, map[string]accounts.Balance{
			"USDT": {Total: 194.8212, Hold: 1.0602, Free: 193.7609},
		}},
		{asset.CoinMarginedFutures, map[string]accounts.Balance{
			"BTC": {Total: 0.14438227, Hold: 0.00110845, Free: 0.14317271},
		}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
			started := time.Now()
			result, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			require.Len(t, result, 1, "UpdateAccountBalances must return one sub-account")
			assert.Equal(t, tc.a, result[0].AssetType, "UpdateAccountBalances should return the asset's sub-account")
			if !mockTests {
				return
			}
			balances := make(map[string]accounts.Balance, len(result[0].Balances))
			for c, b := range result[0].Balances {
				assert.Truef(t, c.Equal(b.Currency), "UpdateAccountBalances should name the %s balance's currency", c)
				assert.WithinRangef(t, b.UpdatedAt, started, time.Now(), "UpdateAccountBalances should stamp the %s balance when it is stored", c)
				b.Currency, b.UpdatedAt = currency.EMPTYCODE, time.Time{}
				balances[c.String()] = b
			}
			assert.Equal(t, tc.exp, balances, "UpdateAccountBalances should map free, held and total balances")
			stored, err := ex.GetCachedCurrencyBalances(t.Context(), tc.a)
			require.NoError(t, err, "GetCachedCurrencyBalances must not error")
			assert.NotEmpty(t, stored, "UpdateAccountBalances should store the balances")
		})
	}

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/account/balance": {`{"code":100001,"msg":"signature"}`},
			"GET /openApi/swap/v3/user/balance":    {`{"code":100001,"msg":"signature"}`},
			"GET /openApi/cswap/v1/user/balance":   {`{"code":100001,"msg":"signature"}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			_, err := ex.UpdateAccountBalances(t.Context(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "UpdateAccountBalances should return the %s request's error", a)
		}
	})

	for _, tc := range []struct {
		name      string
		overtaken int32
		requests  int32
		err       error
		free      float64
	}{
		{name: "stream write during the first request", overtaken: 1, requests: 2, free: 2},
		{name: "stream writes during every request", overtaken: balanceSnapshotAttempts, requests: balanceSnapshotAttempts, err: errBalanceSnapshotSuperseded, free: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ex *Exchange
			var requests atomic.Int32
			ex = newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				n := requests.Add(1)
				if n <= tc.overtaken {
					// A stream event stores spot balances while the request is under way
					_, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 100}})
					assert.NoError(t, err, "wsStoreSpotBalances should not error")
				}
				_, _ = io.WriteString(w, `{"code":0,"data":{"balances":[{"asset":"USDT","free":"`+strconv.Itoa(int(n))+`","locked":"0"}]}}`)
			})
			result, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
			require.ErrorIs(t, err, tc.err, "UpdateAccountBalances must return the expected error")
			assert.Equal(t, tc.requests, requests.Load(), "UpdateAccountBalances should request the balances again while stream writes overtake its requests")
			require.Len(t, result, 1, "UpdateAccountBalances must return the last reply's sub-account")
			assert.InDelta(t, float64(tc.requests), result[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the last reply")
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			stored, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
			require.NoError(t, err, "GetBalance must not error")
			assert.InDelta(t, tc.free, stored.Free, 1e-9, "UpdateAccountBalances should store only a reply no stream write overtook")
		})
	}

	t.Run("request started before the snapshot stored", func(t *testing.T) {
		t.Parallel()
		arrived, release := make(chan struct{}), make(chan struct{})
		var requests atomic.Int32
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			free := "1"
			if requests.Add(1) == 1 {
				close(arrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
			} else {
				free = "2"
			}
			_, _ = io.WriteString(w, `{"code":0,"data":[{"asset":"USDT","balance":"`+free+`","availableMargin":"`+free+`"}]}`)
		})
		type result struct {
			subAccounts accounts.SubAccounts
			err         error
		}
		older := make(chan result, 1)
		go func() {
			subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
			older <- result{subAccounts, err}
		}()
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first balance request must arrive")
		}
		_, err := ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the later request")
		close(release)
		var got result
		select {
		case got = <-older:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the earlier UpdateAccountBalances must return")
		}
		require.NoError(t, got.err, "UpdateAccountBalances must not error for a request started before the snapshot stored")
		require.Len(t, got.subAccounts, 1, "UpdateAccountBalances must return the stored sub-account")
		assert.InDelta(t, 2, got.subAccounts[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the stored balances, which are newer, for a request started before them")
		assert.Equal(t, int32(2), requests.Load(), "UpdateAccountBalances should not request superseded balances again")
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error")
		assert.InDelta(t, 2, stored.Free, 1e-9, "UpdateAccountBalances should keep the snapshot of the request started later")
	})

	t.Run("two credential sets", func(t *testing.T) {
		t.Parallel()
		arrived, release := make(chan struct{}), make(chan struct{})
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			free := "2"
			if r.Header.Get(keyHeader) == testAPIKey {
				close(arrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
				free = "1"
			}
			_, _ = io.WriteString(w, `{"code":0,"data":[{"asset":"USDT","balance":"`+free+`","availableMargin":"`+free+`"}]}`)
		})
		type result struct {
			subAccounts accounts.SubAccounts
			err         error
		}
		own := make(chan result, 1)
		go func() {
			subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
			own <- result{subAccounts, err}
		}()
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first account's balance request must arrive")
		}
		otherCreds := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
		_, err := ex.UpdateAccountBalances(accounts.DeployCredentialsToContext(t.Context(), otherCreds), asset.USDTMarginedFutures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the other account")
		close(release)
		var got result
		select {
		case got = <-own:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first account's UpdateAccountBalances must return")
		}
		require.NoError(t, got.err, "UpdateAccountBalances must not refuse a reply because another account stored a snapshot requested later")
		require.Len(t, got.subAccounts, 1, "UpdateAccountBalances must return one sub-account")
		assert.InDelta(t, 1, got.subAccounts[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the account's own balances")
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the first account")
		assert.InDelta(t, 1, stored.Free, 1e-9, "UpdateAccountBalances should store the first account's snapshot")
		stored, err = ex.Accounts.GetBalance("", otherCreds, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the other account")
		assert.InDelta(t, 2, stored.Free, 1e-9, "UpdateAccountBalances should store the other account's snapshot")
	})

	t.Run("default credentials change during the request", func(t *testing.T) {
		t.Parallel()
		arrived, release := make(chan struct{}), make(chan struct{})
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			free := "2"
			if r.Header.Get(keyHeader) == testAPIKey {
				close(arrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
				free = "1"
			}
			_, _ = io.WriteString(w, `{"code":0,"data":[{"asset":"USDT","balance":"`+free+`","availableMargin":"`+free+`"}]}`)
		})
		before, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		type result struct {
			subAccounts accounts.SubAccounts
			err         error
		}
		started := make(chan result, 1)
		go func() {
			subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
			started <- result{subAccounts, err}
		}()
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the balance request of the credentials it started with must arrive")
		}
		after := &accounts.Credentials{Key: "after-key", Secret: "after-secret"}
		ex.SetCredentials(after)
		_, err = ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the new default credentials")
		close(release)
		var got result
		select {
		case got = <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the UpdateAccountBalances started before the change must return")
		}
		require.NoError(t, got.err, "UpdateAccountBalances must not error when the default credentials change during its request")
		require.Len(t, got.subAccounts, 1, "UpdateAccountBalances must return one sub-account")
		assert.InDelta(t, 1, got.subAccounts[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the balances of the credentials it started with")
		stored, err := ex.Accounts.GetBalance("", before, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the credentials the request started with")
		assert.InDelta(t, 1, stored.Free, 1e-9, "UpdateAccountBalances should store the reply under the credentials it started with")
		stored, err = ex.Accounts.GetBalance("", after, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the new default credentials")
		assert.InDelta(t, 2, stored.Free, 1e-9, "UpdateAccountBalances should not store another account's reply over the new default credentials' snapshot")
	})

	t.Run("credentials differing only in their sub-account", func(t *testing.T) {
		t.Parallel()
		arrived, release := make(chan struct{}), make(chan struct{})
		var requests atomic.Int32
		ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			free := "2"
			if requests.Add(1) == 1 {
				close(arrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
				free = "1"
			}
			_, _ = io.WriteString(w, `{"code":0,"data":[{"asset":"USDT","balance":"`+free+`","availableMargin":"`+free+`"}]}`)
		})
		own, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		mainAccount := &accounts.Credentials{Key: own.Key, Secret: own.Secret, SubAccount: "main"}
		type result struct {
			subAccounts accounts.SubAccounts
			err         error
		}
		started := make(chan result, 1)
		go func() {
			subAccounts, err := ex.UpdateAccountBalances(t.Context(), asset.USDTMarginedFutures)
			started <- result{subAccounts, err}
		}()
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first account's balance request must arrive")
		}
		_, err = ex.UpdateAccountBalances(accounts.DeployCredentialsToContext(t.Context(), mainAccount), asset.USDTMarginedFutures)
		require.NoError(t, err, "UpdateAccountBalances must not error for the main sub-account's credentials")
		close(release)
		var got result
		select {
		case got = <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first account's UpdateAccountBalances must return")
		}
		require.NoError(t, got.err, "UpdateAccountBalances must not refuse a reply because credentials differing only in their sub-account stored a snapshot requested later")
		require.Len(t, got.subAccounts, 1, "UpdateAccountBalances must return one sub-account")
		assert.InDelta(t, 1, got.subAccounts[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the account's own balances")
		stored, err := ex.Accounts.GetBalance("", own, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the first account")
		assert.InDelta(t, 1, stored.Free, 1e-9, "UpdateAccountBalances should store the first account's snapshot")
		stored, err = ex.Accounts.GetBalance("", mainAccount, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the main sub-account's credentials")
		assert.InDelta(t, 2, stored.Free, 1e-9, "UpdateAccountBalances should store the main sub-account credentials' snapshot apart")
	})

	t.Run("default credentials change before a repeated request", func(t *testing.T) {
		t.Parallel()
		after := &accounts.Credentials{Key: "after-key", Secret: "after-secret"}
		var ex *Exchange
		var mu sync.Mutex
		var keys []string
		ex = newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			key := r.Header.Get(keyHeader)
			mu.Lock()
			keys = append(keys, key)
			first := len(keys) == 1
			mu.Unlock()
			free := "2"
			if first {
				// A stream write overtakes the first request, which is then sent again, after the default credentials change
				_, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 100}})
				assert.NoError(t, err, "wsStoreSpotBalances should not error")
				ex.SetCredentials(after)
				free = "1"
			} else if key == testAPIKey {
				free = "3"
			}
			_, _ = io.WriteString(w, `{"code":0,"data":{"balances":[{"asset":"USDT","free":"`+free+`","locked":"0"}]}}`)
		})
		before, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		result, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoError(t, err, "UpdateAccountBalances must not error")
		mu.Lock()
		got := slices.Clone(keys)
		mu.Unlock()
		assert.Equal(t, []string{testAPIKey, testAPIKey}, got, "UpdateAccountBalances should request again with the credentials it started with")
		require.Len(t, result, 1, "UpdateAccountBalances must return one sub-account")
		assert.InDelta(t, 3, result[0].Balances[currency.USDT].Free, 1e-9, "UpdateAccountBalances should return the repeated reply for the credentials it started with")
		stored, err := ex.Accounts.GetBalance("", before, asset.Spot, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the credentials the request started with")
		assert.InDelta(t, 3, stored.Free, 1e-9, "UpdateAccountBalances should store their own balances under the credentials it started with")
	})
}

func TestAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.accountBalances(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "accountBalances must reject an unsupported asset")
	refused := `{"code":100001,"msg":"signature"}`
	ex, _ := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v1/account/balance": {`{"code":0,"data":{"balances":[{"asset":"USDT","free":"2","locked":"1"}]}}`, refused},
		"GET /openApi/swap/v3/user/balance":    {testBalanceReplies["/openApi/swap/v3/user/balance"], refused},
		"GET /openApi/cswap/v1/user/balance":   {testBalanceReplies["/openApi/cswap/v1/user/balance"], refused},
	})
	for _, tc := range []struct {
		a   asset.Item
		c   currency.Code
		exp accounts.Balance
	}{
		{asset.Spot, currency.USDT, accounts.Balance{Total: 3, Hold: 1, Free: 2}},
		{asset.USDTMarginedFutures, currency.USDT, accounts.Balance{Total: 194.8212, Hold: 1.0602, Free: 193.7609}},
		{asset.CoinMarginedFutures, currency.BTC, accounts.Balance{Total: 0.14438227, Hold: 0.00110845, Free: 0.14317271}},
	} {
		subAccount, err := ex.accountBalances(t.Context(), tc.a)
		require.NoErrorf(t, err, "accountBalances must not error for %s", tc.a)
		assert.Equalf(t, tc.a, subAccount.AssetType, "accountBalances should return the %s sub-account", tc.a)
		tc.exp.Currency = tc.c
		assert.Equalf(t, tc.exp, subAccount.Balances[tc.c], "accountBalances should map the %s balance", tc.a)
		_, err = ex.accountBalances(t.Context(), tc.a)
		assert.ErrorIsf(t, err, errAPIResponse, "accountBalances should return the %s request's error", tc.a)
	}
	_, err = ex.GetCachedCurrencyBalances(t.Context(), asset.Spot)
	assert.Error(t, err, "accountBalances should not store the balances")
}

func TestAccountBalanceContext(t *testing.T) {
	t.Parallel()
	other := accounts.Credentials{Key: "other-key", Secret: "other-secret"}
	for _, tc := range []struct {
		name string
		// credentials and subAccount, when set, are deployed to the context as its credentials and sub-account override
		credentials *accounts.Credentials
		subAccount  string
		exp         accounts.Credentials
	}{
		{name: "default credentials", exp: accounts.Credentials{Key: testAPIKey, Secret: testAPISecret}},
		{name: "context credentials", credentials: &other, exp: other},
		{name: "sub-account override", subAccount: "sub", exp: accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "sub"}},
		{name: "context credentials naming the main sub-account", credentials: &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "main"}, exp: accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "main"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, _ := wrapperTestServer(t, map[string][]string{})
			requestCtx := t.Context()
			if tc.credentials != nil {
				requestCtx = accounts.DeployCredentialsToContext(requestCtx, tc.credentials)
			}
			if tc.subAccount != "" {
				requestCtx = accounts.DeploySubAccountOverrideToContext(requestCtx, tc.subAccount)
			}
			contexts := make([]context.Context, 0, 3)
			for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
				ctx, k, err := ex.accountBalanceContext(requestCtx, a)
				require.NoErrorf(t, err, "accountBalanceContext must not error for %s", a)
				assert.Equalf(t, balanceKey{credentials: tc.exp, asset: a}, k, "accountBalanceContext should key the %s balances by the whole of the credentials", a)
				contexts = append(contexts, ctx)
			}
			ex.SetCredentials(&accounts.Credentials{Key: "after-key", Secret: "after-secret"})
			for _, ctx := range contexts {
				got, err := ex.GetCredentials(ctx)
				require.NoError(t, err, "GetCredentials must not error for the context accountBalanceContext returns")
				assert.Equal(t, &tc.exp, got, "accountBalanceContext should carry the credentials it resolved, although the default credentials change")
			}
		})
	}

	t.Run("credentials differing only in their sub-account", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		_, own, err := ex.accountBalanceContext(t.Context(), asset.Spot)
		require.NoError(t, err, "accountBalanceContext must not error")
		_, main, err := ex.accountBalanceContext(accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: testAPIKey, Secret: testAPISecret, SubAccount: "main"}), asset.Spot)
		require.NoError(t, err, "accountBalanceContext must not error for context credentials")
		assert.NotEqual(t, own, main, "accountBalanceContext should key credentials differing only in their sub-account apart, as the accounts store keeps their balances apart")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		_, _, err := ex.accountBalanceContext(t.Context(), asset.Futures)
		assert.ErrorIs(t, err, asset.ErrNotSupported, "accountBalanceContext should reject an unsupported asset")
		ex.SetCredentials(&accounts.Credentials{})
		_, _, err = ex.accountBalanceContext(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "accountBalanceContext should return the credentials error")
		ex.SkipAuthCheck = true
		_, _, err = ex.accountBalanceContext(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "accountBalanceContext should refuse empty credentials the credentials check lets through")
	})
}

func TestStartBalanceSnapshot(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	spot := balanceKey{credentials: accounts.Credentials{Key: "a"}, asset: asset.Spot}
	ticket, generation := ex.startBalanceSnapshot(&spot)
	assert.Equal(t, uint64(1), ticket, "startBalanceSnapshot should number an asset's first request 1")
	assert.Zero(t, generation, "startBalanceSnapshot should start an asset's stream writes at zero")
	ex.balances.generations = map[balanceKey]uint64{spot: 3}
	ticket, generation = ex.startBalanceSnapshot(&spot)
	assert.Equal(t, uint64(2), ticket, "startBalanceSnapshot should number an asset's requests in the order they start")
	assert.Equal(t, uint64(3), generation, "startBalanceSnapshot should return the asset's stream write count")
	ticket, generation = ex.startBalanceSnapshot(&balanceKey{credentials: accounts.Credentials{Key: "a"}, asset: asset.USDTMarginedFutures})
	assert.Equal(t, uint64(1), ticket, "startBalanceSnapshot should number each asset's requests apart")
	assert.Zero(t, generation, "startBalanceSnapshot should keep each asset's stream write count apart")

	var wg sync.WaitGroup
	tickets := make(chan uint64, 100)
	for range 100 {
		wg.Go(func() {
			ticket, _ := ex.startBalanceSnapshot(&balanceKey{credentials: accounts.Credentials{Key: "a"}, asset: asset.CoinMarginedFutures})
			tickets <- ticket
		})
	}
	wg.Wait()
	close(tickets)
	issued := make(map[uint64]bool)
	for ticket := range tickets {
		issued[ticket] = true
	}
	assert.Len(t, issued, 100, "startBalanceSnapshot should issue every request its own ticket")

	t.Run("two accounts", func(t *testing.T) {
		t.Parallel()
		ex := new(Exchange)
		own, other := balanceKey{credentials: accounts.Credentials{Key: "a"}, asset: asset.Spot}, balanceKey{credentials: accounts.Credentials{Key: "b"}, asset: asset.Spot}
		ex.balances.generations = map[balanceKey]uint64{own: 3}
		ex.startBalanceSnapshot(&own)
		ticket, generation := ex.startBalanceSnapshot(&other)
		assert.Equal(t, uint64(1), ticket, "startBalanceSnapshot should number each account's requests apart")
		assert.Zero(t, generation, "startBalanceSnapshot should keep each account's stream write count apart")
	})
}

// testAccountBalanceKey returns the balance state key of ctx's account's asset a
func testAccountBalanceKey(ctx context.Context, t *testing.T, ex *Exchange, a asset.Item) balanceKey {
	t.Helper()
	_, k, err := ex.accountBalanceContext(ctx, a)
	require.NoErrorf(t, err, "accountBalanceContext must not error for %s", a)
	return k
}

func TestSaveBalanceSnapshot(t *testing.T) {
	t.Parallel()
	snapshot := func(a asset.Item, free float64) accounts.SubAccounts {
		subAccount := accounts.NewSubAccount(a, "")
		subAccount.Balances.Set(currency.USDT, accounts.Balance{Total: free, Free: free})
		return accounts.SubAccounts{subAccount}
	}
	storedFree := func(ctx context.Context, t *testing.T, ex *Exchange, a asset.Item) float64 {
		t.Helper()
		creds, err := ex.GetCredentials(ctx)
		require.NoError(t, err, "GetCredentials must not error")
		stored, err := ex.Accounts.GetBalance("", creds, a, currency.USDT)
		require.NoError(t, err, "GetBalance must not error")
		return stored.Free
	}
	ex, _ := wrapperTestServer(t, map[string][]string{})
	usdtMargined, spot := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures), testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
	earlier, _ := ex.startBalanceSnapshot(&usdtMargined)
	later, generation := ex.startBalanceSnapshot(&usdtMargined)
	stored := snapshot(asset.USDTMarginedFutures, 2)
	reported, done, err := ex.saveBalanceSnapshot(t.Context(), &usdtMargined, later, generation, stored)
	require.NoError(t, err, "saveBalanceSnapshot must not error")
	assert.True(t, done, "saveBalanceSnapshot should store a snapshot no stream write overtook")
	assert.Equal(t, stored, reported, "saveBalanceSnapshot should return the snapshot it stored")
	reported, done, err = ex.saveBalanceSnapshot(t.Context(), &usdtMargined, earlier, generation, snapshot(asset.USDTMarginedFutures, 1))
	require.NoError(t, err, "saveBalanceSnapshot must not error for a reply to a request started before the snapshot stored")
	assert.True(t, done, "saveBalanceSnapshot should finish a request started before the snapshot stored")
	require.Len(t, reported, 1, "saveBalanceSnapshot must return the stored sub-account")
	assert.InDelta(t, 2, reported[0].Balances[currency.USDT].Free, 1e-9, "saveBalanceSnapshot should return the stored balances, which are newer, for a request started before them")
	assert.InDelta(t, 2, storedFree(t.Context(), t, ex, asset.USDTMarginedFutures), 1e-9, "saveBalanceSnapshot should keep the snapshot of the request started later")

	ticket, generation := ex.startBalanceSnapshot(&spot)
	_, err = ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 5}})
	require.NoError(t, err, "wsStoreSpotBalances must not error")
	reported, done, err = ex.saveBalanceSnapshot(t.Context(), &spot, ticket, generation, snapshot(asset.Spot, 1))
	require.NoError(t, err, "saveBalanceSnapshot must not error for a snapshot a stream write overtook")
	assert.False(t, done, "saveBalanceSnapshot should leave a snapshot a stream write overtook to be requested again")
	assert.Nil(t, reported, "saveBalanceSnapshot should return no balances for a snapshot a stream write overtook")
	assert.InDelta(t, 5, storedFree(t.Context(), t, ex, asset.Spot), 1e-9, "saveBalanceSnapshot should keep the stream's balances")

	ticket, generation = ex.startBalanceSnapshot(&spot)
	_, done, err = ex.saveBalanceSnapshot(t.Context(), &spot, ticket, generation, accounts.SubAccounts{accounts.NewSubAccount(asset.Empty, "")})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "saveBalanceSnapshot should return the accounts store's error")
	assert.True(t, done, "saveBalanceSnapshot should count a snapshot it passed to the accounts store as stored, whatever the store returned")
	reported, done, err = ex.saveBalanceSnapshot(t.Context(), &spot, ticket-1, generation, snapshot(asset.Spot, 3))
	require.NoError(t, err, "saveBalanceSnapshot must not error for a reply to a request started before a save the accounts store returned an error for")
	assert.True(t, done, "saveBalanceSnapshot should finish a request started before a save the accounts store returned an error for")
	require.Len(t, reported, 1, "saveBalanceSnapshot must return the stored sub-account")
	assert.InDelta(t, 5, reported[0].Balances[currency.USDT].Free, 1e-9, "saveBalanceSnapshot should return the stored balances rather than store the reply")

	t.Run("save that stores balances and then fails", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		spot := testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
		earlier, _ := ex.startBalanceSnapshot(&spot)
		later, generation := ex.startBalanceSnapshot(&spot)
		_, done, err := ex.saveBalanceSnapshot(t.Context(), &spot, later, generation, append(snapshot(asset.Spot, 2), accounts.NewSubAccount(asset.Empty, "")))
		assert.ErrorIs(t, err, asset.ErrNotSupported, "saveBalanceSnapshot should return the accounts store's error")
		assert.True(t, done, "saveBalanceSnapshot should count a snapshot it passed to the accounts store as stored")
		require.InDelta(t, 2, storedFree(t.Context(), t, ex, asset.Spot), 1e-9, "the accounts store must keep the balances it stored before failing")
		reported, done, err := ex.saveBalanceSnapshot(t.Context(), &spot, earlier, generation, snapshot(asset.Spot, 1))
		require.NoError(t, err, "saveBalanceSnapshot must not error for a reply to a request started before a save that stored balances and then failed")
		assert.True(t, done, "saveBalanceSnapshot should finish a request started before a save that stored balances and then failed")
		require.Len(t, reported, 1, "saveBalanceSnapshot must return the stored sub-account")
		assert.InDelta(t, 2, reported[0].Balances[currency.USDT].Free, 1e-9, "saveBalanceSnapshot should return the balances the failed save stored rather than store the reply")
		assert.InDelta(t, 2, storedFree(t.Context(), t, ex, asset.Spot), 1e-9, "saveBalanceSnapshot should keep the balances of the request started later")
	})

	t.Run("context naming other credentials", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		before, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		usdtMargined := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures)
		earlier, _ := ex.startBalanceSnapshot(&usdtMargined)
		later, generation := ex.startBalanceSnapshot(&usdtMargined)
		after := &accounts.Credentials{Key: "after-key", Secret: "after-secret"}
		ex.SetCredentials(after)
		_, done, err := ex.saveBalanceSnapshot(t.Context(), &usdtMargined, later, generation, snapshot(asset.USDTMarginedFutures, 2))
		require.NoError(t, err, "saveBalanceSnapshot must not error")
		require.True(t, done, "saveBalanceSnapshot must store the snapshot of the request started later")
		stored, err := ex.Accounts.GetBalance("", before, asset.USDTMarginedFutures, currency.USDT)
		require.NoError(t, err, "GetBalance must not error for the key's credentials")
		assert.InDelta(t, 2, stored.Free, 1e-9, "saveBalanceSnapshot should store the snapshot under the key's credentials")
		_, err = ex.Accounts.GetBalance("", after, asset.USDTMarginedFutures, currency.USDT)
		assert.ErrorIs(t, err, accounts.ErrNoBalances, "saveBalanceSnapshot should store nothing under the context's credentials")
		reported, done, err := ex.saveBalanceSnapshot(t.Context(), &usdtMargined, earlier, generation, snapshot(asset.USDTMarginedFutures, 1))
		require.NoError(t, err, "saveBalanceSnapshot must not error for a reply to a request started before the snapshot stored")
		assert.True(t, done, "saveBalanceSnapshot should finish a request started before the snapshot stored")
		require.Len(t, reported, 1, "saveBalanceSnapshot must return the stored sub-account")
		assert.InDelta(t, 2, reported[0].Balances[currency.USDT].Free, 1e-9, "saveBalanceSnapshot should return the stored balances of the key's credentials")
	})

	t.Run("two credential sets", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		otherCtx := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-key", Secret: "other-secret"})
		own, other := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures), testAccountBalanceKey(otherCtx, t, ex, asset.USDTMarginedFutures)
		earlier, earlierGeneration := ex.startBalanceSnapshot(&own)
		later, laterGeneration := ex.startBalanceSnapshot(&other)
		_, done, err := ex.saveBalanceSnapshot(otherCtx, &other, later, laterGeneration, snapshot(asset.USDTMarginedFutures, 2))
		require.NoError(t, err, "saveBalanceSnapshot must not error")
		require.True(t, done, "saveBalanceSnapshot must store the other account's snapshot")
		own1 := snapshot(asset.USDTMarginedFutures, 1)
		reported, done, err := ex.saveBalanceSnapshot(t.Context(), &own, earlier, earlierGeneration, own1)
		require.NoError(t, err, "saveBalanceSnapshot must not error")
		assert.True(t, done, "saveBalanceSnapshot should store a reply no snapshot of its own account superseded")
		assert.Equal(t, own1, reported, "saveBalanceSnapshot should return the account's own reply rather than another account's balances")
		assert.InDelta(t, 1, storedFree(t.Context(), t, ex, asset.USDTMarginedFutures), 1e-9, "saveBalanceSnapshot should store the account's own snapshot")
		assert.InDelta(t, 2, storedFree(otherCtx, t, ex, asset.USDTMarginedFutures), 1e-9, "saveBalanceSnapshot should keep the other account's snapshot")

		spot := testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
		ticket, generation := ex.startBalanceSnapshot(&spot)
		_, err = ex.wsStoreSpotBalances(otherCtx, []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: 5}})
		require.NoError(t, err, "wsStoreSpotBalances must not error")
		_, done, err = ex.saveBalanceSnapshot(t.Context(), &spot, ticket, generation, snapshot(asset.Spot, 1))
		require.NoError(t, err, "saveBalanceSnapshot must not error")
		assert.True(t, done, "saveBalanceSnapshot should store a snapshot only another account's stream write followed")
		assert.InDelta(t, 1, storedFree(t.Context(), t, ex, asset.Spot), 1e-9, "saveBalanceSnapshot should store the account's own snapshot")
		assert.InDelta(t, 5, storedFree(otherCtx, t, ex, asset.Spot), 1e-9, "saveBalanceSnapshot should keep the other account's streamed balances")
	})

	t.Run("stream writes", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		spot := testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
		const rounds = 2000
		var overwritten int
		for i := range rounds {
			// The snapshot is requested before the stream event, so the stream's balances must survive whichever lands
			// first
			ticket, generation := ex.startBalanceSnapshot(&spot)
			streamed := float64(2*i + 1)
			var wg sync.WaitGroup
			wg.Go(func() {
				_, err := ex.wsStoreSpotBalances(t.Context(), []WebsocketSpotBalance{{Asset: currency.USDT, WalletBalance: types.Number(streamed)}})
				assert.NoError(t, err, "wsStoreSpotBalances should not error")
			})
			wg.Go(func() {
				_, _, err := ex.saveBalanceSnapshot(t.Context(), &spot, ticket, generation, snapshot(asset.Spot, float64(2*i)))
				assert.NoError(t, err, "saveBalanceSnapshot should not error")
			})
			wg.Wait()
			if storedFree(t.Context(), t, ex, asset.Spot) != streamed {
				overwritten++
			}
		}
		assert.Zerof(t, overwritten, "saveBalanceSnapshot should never store a snapshot over a later stream write, but did in %d of %d rounds", overwritten, rounds)
	})
}

func TestRefreshBalances(t *testing.T) {
	t.Parallel()
	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	ex.balances.interval = 10 * time.Millisecond
	ended := make(chan struct{})
	close(ended)
	ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, nil)
	ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, ended)
	testWaitBackground(t, ex)
	assert.Zero(t, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "refreshBalances should refresh nothing for an event without a live session")

	session := make(chan struct{})
	for range 3 {
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, session)
	}
	testWaitBackground(t, ex)
	assert.Equal(t, 2, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "refreshBalances should refresh once more for a burst of events during a refresh")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
	require.NoError(t, err, "GetBalance must not error")
	assert.InDelta(t, 193.7609, stored.Free, 1e-9, "refreshBalances should store REST's balances")

	ex.balances.interval = time.Hour
	ex.refreshBalances(t.Context(), asset.CoinMarginedFutures, session)
	require.Eventually(t, func() bool { return server.balanceRequestCount("/openApi/cswap/v1/user/balance") == 1 }, 5*time.Second, 10*time.Millisecond, "refreshBalances must refresh the balances")
	ex.refreshBalances(t.Context(), asset.CoinMarginedFutures, session)
	close(session)
	testWaitBackground(t, ex)
	assert.Equal(t, 1, server.balanceRequestCount("/openApi/cswap/v1/user/balance"), "refreshBalances should end its refreshes with the session")

	t.Run("session ends while a newer session's event is pending", func(t *testing.T) {
		t.Parallel()
		var requests atomic.Int32
		var abandoned atomic.Bool
		arrived, handled := make(chan struct{}), make(chan struct{})
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if requests.Add(1) == 1 {
				defer close(handled)
				close(arrived)
				// The server notices the client giving up, which ends the request's context, once the body is read
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				select {
				case <-r.Context().Done():
					abandoned.Store(true)
				case <-time.After(5 * time.Second):
				}
				return
			}
			_, _ = io.WriteString(w, testBalanceReplies["/openApi/swap/v3/user/balance"])
		})
		ex.balances.interval = time.Hour
		oldSession, newSession := make(chan struct{}), make(chan struct{})
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, oldSession)
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the old session's balance request must arrive")
		}
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, newSession)
		close(oldSession)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		require.Eventually(t, func() bool {
			_, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
			return err == nil
		}, 5*time.Second, 10*time.Millisecond, "refreshBalances must store the balances of a refresh for the newer session")
		close(newSession)
		testWaitBackground(t, ex)
		select {
		case <-handled:
		case <-time.After(10 * time.Second):
			require.FailNow(t, "the old session's balance request must be answered")
		}
		assert.True(t, abandoned.Load(), "refreshBalances should end the old session's request with its session")
		assert.Equal(t, int32(2), requests.Load(), "refreshBalances should refresh once for the newer session's event")
	})

	t.Run("two credential sets", func(t *testing.T) {
		t.Parallel()
		arrived, release := make(chan struct{}), make(chan struct{})
		var ownRequests atomic.Int32
		ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(keyHeader) == testAPIKey && ownRequests.Add(1) == 1 {
				close(arrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
			}
			_, _ = io.WriteString(w, testBalanceReplies["/openApi/swap/v3/user/balance"])
		})
		ex.balances.interval = time.Hour
		session := make(chan struct{})
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, session)
		select {
		case <-arrived:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first account's balance request must arrive")
		}
		otherCreds := &accounts.Credentials{Key: "other-key", Secret: "other-secret"}
		ex.refreshBalances(accounts.DeployCredentialsToContext(t.Context(), otherCreds), asset.USDTMarginedFutures, session)
		assert.Eventually(t, func() bool {
			_, err := ex.Accounts.GetBalance("", otherCreds, asset.USDTMarginedFutures, currency.USDT)
			return err == nil
		}, 5*time.Second, 10*time.Millisecond, "refreshBalances should refresh another account's balances while the first account's refresh runs")
		close(release)
		creds, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		assert.Eventually(t, func() bool {
			_, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
			return err == nil
		}, 5*time.Second, 10*time.Millisecond, "refreshBalances should store the first account's balances")
		close(session)
		testWaitBackground(t, ex)
		assert.Equal(t, int32(1), ownRequests.Load(), "refreshBalances should refresh the first account once")
	})

	t.Run("event during a refresh", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		creds := &accounts.Credentials{Key: testAPIKey, Secret: "event-secret"}
		ctx := accounts.DeployCredentialsToContext(t.Context(), creds)
		k := testAccountBalanceKey(ctx, t, ex, asset.USDTMarginedFutures)
		ex.balances.running = map[balanceKey]bool{k: true}
		ex.balances.pending = make(map[balanceKey]balanceRefresh)
		var session <-chan struct{} = make(chan struct{})
		ex.refreshBalances(ctx, asset.USDTMarginedFutures, session)
		ex.balances.mu.Lock()
		pending, ok := ex.balances.pending[k]
		ex.balances.mu.Unlock()
		require.True(t, ok, "refreshBalances must leave a refresh pending while the account's refresh runs")
		assert.Equal(t, session, pending.session, "refreshBalances should keep the event's session for the pending refresh")
		got, err := ex.GetCredentials(pending.ctx)
		require.NoError(t, err, "GetCredentials must not error for the pending refresh's context")
		assert.Equal(t, creds.Secret, got.Secret, "refreshBalances should keep the event's credentials for the pending refresh")
	})

	t.Run("default credentials change before a pending refresh runs", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		before, err := ex.GetCredentials(t.Context())
		require.NoError(t, err, "GetCredentials must not error")
		k := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures)
		ex.balances.running = map[balanceKey]bool{k: true}
		ex.balances.pending = make(map[balanceKey]balanceRefresh)
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, make(chan struct{}))
		ex.SetCredentials(&accounts.Credentials{Key: "after-key", Secret: "after-secret"})
		ex.balances.mu.Lock()
		pending, ok := ex.balances.pending[k]
		ex.balances.mu.Unlock()
		require.True(t, ok, "refreshBalances must leave a refresh pending while the account's refresh runs")
		got, err := ex.GetCredentials(pending.ctx)
		require.NoError(t, err, "GetCredentials must not error for the pending refresh's context")
		assert.Equal(t, before, got, "refreshBalances should keep the credentials the event named for the pending refresh, although the default credentials change")
	})

	t.Run("account without credentials", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, nil)
		ex.SetCredentials(&accounts.Credentials{})
		ex.refreshBalances(t.Context(), asset.USDTMarginedFutures, make(chan struct{}))
		testWaitBackground(t, ex)
		assert.Empty(t, ex.balances.running, "refreshBalances should start no refresh for an account its credentials cannot name")
		assert.Empty(t, requests(), "refreshBalances should request nothing for an account its credentials cannot name")
	})
}

func TestRunBalanceRefresh(t *testing.T) {
	t.Parallel()
	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	usdtMargined, spot := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures), testAccountBalanceKey(t.Context(), t, ex, asset.Spot)
	coinMargined := testAccountBalanceKey(t.Context(), t, ex, asset.CoinMarginedFutures)
	session := make(chan struct{})
	ex.balances.running = map[balanceKey]bool{usdtMargined: true}
	ex.balances.pending = map[balanceKey]balanceRefresh{usdtMargined: {ctx: t.Context(), session: session}}
	ex.runBalanceRefresh(&usdtMargined, balanceRefresh{ctx: t.Context(), session: session}, time.Millisecond)
	assert.Equal(t, 2, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "runBalanceRefresh should refresh once more for a change during its refresh")
	assert.False(t, ex.balances.running[usdtMargined], "runBalanceRefresh should mark the asset's refresh finished")

	ex.balances.running[spot] = true
	ex.runBalanceRefresh(&spot, balanceRefresh{ctx: t.Context(), session: session}, time.Millisecond)
	assert.False(t, ex.balances.running[spot], "runBalanceRefresh should finish after a refresh that failed, logging its error")

	newSession := make(chan struct{})
	ex.balances.mu.Lock()
	ex.balances.running[coinMargined], ex.balances.pending[coinMargined] = true, balanceRefresh{ctx: t.Context(), session: newSession}
	ex.balances.mu.Unlock()
	finished := make(chan struct{})
	go func() {
		ex.runBalanceRefresh(&coinMargined, balanceRefresh{ctx: t.Context(), session: session}, time.Hour)
		close(finished)
	}()
	require.Eventually(t, func() bool { return server.balanceRequestCount("/openApi/cswap/v1/user/balance") == 1 }, 5*time.Second, 10*time.Millisecond, "runBalanceRefresh must refresh the balances")
	close(session)
	require.Eventually(t, func() bool { return server.balanceRequestCount("/openApi/cswap/v1/user/balance") == 2 }, 5*time.Second, 10*time.Millisecond, "runBalanceRefresh must refresh for a newer session's pending change once its own session ends")
	close(newSession)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "runBalanceRefresh must finish once the newer session ends")
	}
	assert.False(t, ex.balances.running[coinMargined], "runBalanceRefresh should mark the asset's refresh finished")

	lastSession := make(chan struct{})
	ex.balances.mu.Lock()
	ex.balances.running[coinMargined], ex.balances.pending[coinMargined] = true, balanceRefresh{ctx: t.Context(), session: lastSession}
	ex.balances.mu.Unlock()
	finished = make(chan struct{})
	go func() {
		ex.runBalanceRefresh(&coinMargined, balanceRefresh{ctx: t.Context(), session: lastSession}, time.Hour)
		close(finished)
	}()
	require.Eventually(t, func() bool { return server.balanceRequestCount("/openApi/cswap/v1/user/balance") == 3 }, 5*time.Second, 10*time.Millisecond, "runBalanceRefresh must refresh the balances")
	close(lastSession)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "runBalanceRefresh must finish once its session ends")
	}
	assert.Equal(t, 3, server.balanceRequestCount("/openApi/cswap/v1/user/balance"), "runBalanceRefresh should not refresh again once its session ended")
	assert.Equal(t, uint64(3), ex.balances.tickets[coinMargined], "runBalanceRefresh should not start another refresh once its session ended")
	assert.Empty(t, ex.balances.pending, "runBalanceRefresh should drop a change an ended session left pending")
	assert.False(t, ex.balances.running[coinMargined], "runBalanceRefresh should mark the asset's refresh finished")

	t.Run("pending refresh with its event's context", func(t *testing.T) {
		t.Parallel()
		server := newTestListenKeyServer(t)
		ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
		testRESTEndpoints(t, ex, server.URL)
		usdtMargined := testAccountBalanceKey(t.Context(), t, ex, asset.USDTMarginedFutures)
		creds := &accounts.Credentials{Key: testAPIKey, Secret: "event-secret"}
		session := make(chan struct{})
		ex.balances.running = map[balanceKey]bool{usdtMargined: true}
		ex.balances.pending = map[balanceKey]balanceRefresh{usdtMargined: {ctx: accounts.DeployCredentialsToContext(t.Context(), creds), session: session}}
		ex.runBalanceRefresh(&usdtMargined, balanceRefresh{ctx: t.Context(), session: session}, time.Millisecond)
		require.Equal(t, 2, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "runBalanceRefresh must refresh once more for the pending refresh")
		_, err := ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
		assert.NoError(t, err, "runBalanceRefresh should run a pending refresh with the context of the event that asked for it")
	})
}

// TestUpdateSessionBalances runs serially, as it sets the process's log hook to see which failures are logged
func TestUpdateSessionBalances(t *testing.T) {
	var mu sync.Mutex
	var failures []string
	log.SetCustomLogHook(func(_, _ string, a ...any) bool {
		line := fmt.Sprint(a...)
		if !strings.Contains(line, "balance refresh failed") {
			return false
		}
		mu.Lock()
		defer mu.Unlock()
		failures = append(failures, line)
		return true
	})
	t.Cleanup(func() { log.SetCustomLogHook(nil) })
	// logged returns the balance refresh failures logged for the exchange named name
	logged := func(name string) []string {
		mu.Lock()
		defer mu.Unlock()
		var named []string
		for _, line := range failures {
			if strings.HasPrefix(line, name+" ") {
				named = append(named, line)
			}
		}
		return named
	}

	server := newTestListenKeyServer(t)
	ex := newWebsocketTestExchange(t, testWebsocketAuthenticated)
	testRESTEndpoints(t, ex, server.URL)
	session := make(chan struct{})
	ex.updateSessionBalances(t.Context(), asset.USDTMarginedFutures, session)
	assert.Equal(t, 1, server.balanceRequestCount("/openApi/swap/v3/user/balance"), "updateSessionBalances should request the balances")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	_, err = ex.Accounts.GetBalance("", creds, asset.USDTMarginedFutures, currency.USDT)
	assert.NoError(t, err, "updateSessionBalances should store the balances")
	testWaitBackground(t, ex)

	var abandoned atomic.Bool
	arrived, handled := make(chan struct{}), make(chan struct{})
	blocked := newTestServerExchange(t, func(_ http.ResponseWriter, r *http.Request) {
		defer close(handled)
		close(arrived)
		// The server notices the client giving up, which ends the request's context, once the body is read
		_, err := io.Copy(io.Discard, r.Body)
		assert.NoError(t, err, "reading the request body should not error")
		select {
		case <-r.Context().Done():
			abandoned.Store(true)
		case <-time.After(5 * time.Second):
		}
	})
	finished := make(chan struct{})
	go func() {
		blocked.updateSessionBalances(t.Context(), asset.USDTMarginedFutures, session)
		close(finished)
	}()
	select {
	case <-arrived:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the balance request must arrive")
	}
	close(session)
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "updateSessionBalances must return once its session ends")
	}
	testWaitBackground(t, blocked)
	select {
	case <-handled:
	case <-time.After(10 * time.Second):
		require.FailNow(t, "the balance request must be answered")
	}
	assert.True(t, abandoned.Load(), "updateSessionBalances should end its request with its session")
	assert.Empty(t, logged(blocked.Name), "updateSessionBalances should not log a refresh its session's end cut off, as the next connection refreshes the balances again")

	t.Run("failure while its session lasts", func(t *testing.T) {
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v3/user/balance": {`{"code":100001,"msg":"signature"}`},
		})
		ex.updateSessionBalances(t.Context(), asset.USDTMarginedFutures, make(chan struct{}))
		testWaitBackground(t, ex)
		got := logged(ex.Name)
		require.Len(t, got, 1, "updateSessionBalances must log a refresh that failed while its session lasts")
		assert.Contains(t, got[0], "usdtmarginedfutures balance refresh failed", "updateSessionBalances should log the asset whose refresh failed")
		assert.Contains(t, got[0], "code 100001", "updateSessionBalances should log the refresh's error")
	})
}

func TestPerpetualBalance(t *testing.T) {
	t.Parallel()
	assert.Equal(t, accounts.Balance{Total: 10, Hold: 3, Free: 7.5}, perpetualBalance(10, 2, 1, 7.5), "perpetualBalance should hold used and frozen margin and free the available margin")
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	ex := newWrapperTestExchange(t)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	for _, a := range ex.GetAssetTypes(false) {
		assert.NoErrorf(t, ex.ValidateAPICredentials(t.Context(), a), "ValidateAPICredentials should not error for %s", a)
	}
	assert.ErrorIs(t, ex.ValidateAPICredentials(t.Context(), asset.Futures), asset.ErrNotSupported, "ValidateAPICredentials should reject an unsupported asset")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		result, err := ex.GetAccountFundingHistory(t.Context())
		require.NoError(t, err, "GetAccountFundingHistory must not error")
		if !mockTests {
			return
		}
		require.Len(t, result, 2, "GetAccountFundingHistory must return the deposit and the withdrawal")
		applied := time.Date(2023, 12, 13, 20, 5, 2, 0, time.UTC)
		assert.Truef(t, applied.Equal(result[1].Timestamp), "GetAccountFundingHistory should date the withdrawal %s, not %s", applied, result[1].Timestamp)
		result[1].Timestamp = time.Time{}
		assert.Equal(t, []exchange.FundingHistory{
			{
				ExchangeName:    ex.Name,
				Status:          "Completed",
				TransferID:      "60*****1d",
				Timestamp:       time.UnixMilli(1701557778000),
				Currency:        "USDTTRC20",
				Amount:          49999,
				TransferType:    "deposit",
				CryptoToAddress: "TP******B4v",
				CryptoTxID:      "60*****1d",
				CryptoChain:     "TRC20",
			},
			{
				ExchangeName: ex.Name,
				// The recorded withdrawal reports no status
				Status:          "0",
				TransferID:      "125*****98",
				Currency:        "USDTTRC20",
				Amount:          3500,
				Fee:             1,
				TransferType:    "withdrawal",
				CryptoToAddress: "TR****zc",
				CryptoTxID:      "b9***********b67",
				CryptoChain:     "TRC20",
			},
		}, result, "GetAccountFundingHistory should convert every deposit and withdrawal")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/api/v3/capital/deposit/hisrec":   {`{"code":100001,"msg":"signature"}`, `[]`},
			"GET /openApi/api/v3/capital/withdraw/history": {`{"code":100001,"msg":"signature"}`},
		})
		_, err := ex.GetAccountFundingHistory(t.Context())
		assert.ErrorIs(t, err, errAPIResponse, "GetAccountFundingHistory should return the deposit request's error")
		_, err = ex.GetAccountFundingHistory(t.Context())
		assert.ErrorIs(t, err, errAPIResponse, "GetAccountFundingHistory should return the withdrawal request's error")
	})
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		result, err := ex.GetWithdrawalsHistory(t.Context(), currency.USDT, asset.Spot)
		require.NoError(t, err, "GetWithdrawalsHistory must not error")
		if !mockTests {
			return
		}
		require.Len(t, result, 1, "GetWithdrawalsHistory must return the withdrawal")
		assert.True(t, time.Date(2023, 12, 13, 20, 5, 2, 0, time.UTC).Equal(result[0].Timestamp), "GetWithdrawalsHistory should date the withdrawal by its application")
		result[0].Timestamp = time.Time{}
		assert.Equal(t, []exchange.WithdrawalHistory{{
			Status:          "0",
			TransferID:      "125*****98",
			Currency:        "USDTTRC20",
			Amount:          3500,
			Fee:             1,
			TransferType:    "withdrawal",
			CryptoToAddress: "TR****zc",
			CryptoTxID:      "b9***********b67",
			CryptoChain:     "TRC20",
		}}, result, "GetWithdrawalsHistory should convert every withdrawal")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/api/v3/capital/withdraw/history": {`{"code":100001,"msg":"signature"}`}})
		_, err := ex.GetWithdrawalsHistory(t.Context(), currency.USDT, asset.Spot)
		assert.ErrorIs(t, err, errAPIResponse, "GetWithdrawalsHistory should return the request's error")
	})
}

func TestAllDeposits(t *testing.T) {
	t.Parallel()
	record := func(i int) string { return `{"txId":"` + strconv.Itoa(i) + `","status":1}` }
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/api/v3/capital/deposit/hisrec": {wrapperRecordsPage(0, fundingRecordsPageLimit, record), wrapperRecordsPage(fundingRecordsPageLimit, 1, record)},
	})
	deposits, err := ex.allDeposits(t.Context())
	require.NoError(t, err, "allDeposits must not error")
	require.Len(t, deposits, fundingRecordsPageLimit+1, "allDeposits must return every page's records")
	assert.Equal(t, strconv.Itoa(fundingRecordsPageLimit), deposits[fundingRecordsPageLimit].TxID, "allDeposits should end with the last page's record")
	got := requests()
	require.Len(t, got, 2, "allDeposits must request pages until one is not full")
	assert.Equal(t, url.Values{"limit": {"1000"}}, got[0].params, "allDeposits should start at the first record")
	assert.Equal(t, url.Values{"limit": {"1000"}, "offset": {"1000"}}, got[1].params, "allDeposits should continue after the records fetched")

	ex, _ = wrapperTestServer(t, map[string][]string{"GET /openApi/api/v3/capital/deposit/hisrec": {wrapperRecordsPage(0, fundingRecordsPageLimit, record), `{"code":100500,"msg":"busy"}`}})
	_, err = ex.allDeposits(t.Context())
	assert.ErrorIs(t, err, errAPIResponse, "allDeposits should return a page's error")
}

func TestAllWithdrawals(t *testing.T) {
	t.Parallel()
	record := func(i int) string { return `{"id":"` + strconv.Itoa(i) + `"}` }
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/api/v3/capital/withdraw/history": {wrapperRecordsPage(0, fundingRecordsPageLimit, record), "[]"},
	})
	withdrawals, err := ex.allWithdrawals(t.Context(), currency.BTC)
	require.NoError(t, err, "allWithdrawals must not error")
	assert.Len(t, withdrawals, fundingRecordsPageLimit, "allWithdrawals should return every page's records")
	got := requests()
	require.Len(t, got, 2, "allWithdrawals must request pages until one is not full")
	assert.Equal(t, url.Values{"coin": {"BTC"}, "limit": {"1000"}}, got[0].params, "allWithdrawals should request the coin's withdrawals")
	assert.Equal(t, url.Values{"coin": {"BTC"}, "limit": {"1000"}, "offset": {"1000"}}, got[1].params, "allWithdrawals should continue after the records fetched")

	ex, _ = wrapperTestServer(t, map[string][]string{"GET /openApi/api/v3/capital/withdraw/history": {`{"code":100500,"msg":"busy"}`}})
	_, err = ex.allWithdrawals(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, errAPIResponse, "allWithdrawals should return a page's error")
}

func TestDepositStatusString(t *testing.T) {
	t.Parallel()
	for status, exp := range map[DepositStatus]string{
		DepositStatusInProgress:    "In progress",
		DepositStatusCompleted:     "Completed",
		DepositStatusChainUploaded: "Chain uploaded",
		9:                          "9",
	} {
		assert.Equalf(t, exp, depositStatusString(status), "depositStatusString should name status %d", status)
	}
}

func TestWithdrawStatusString(t *testing.T) {
	t.Parallel()
	for status, exp := range map[WithdrawStatus]string{
		WithdrawStatusUnderReview: "Under review",
		WithdrawStatusFailed:      "Failed",
		WithdrawStatusCompleted:   "Completed",
		0:                         "0",
	} {
		assert.Equalf(t, exp, withdrawStatusString(status), "withdrawStatusString should name status %d", status)
	}
}

func TestWithdrawalTransferTypeString(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "withdrawal", withdrawalTransferTypeString(TransferTypeWithdrawal), "withdrawalTransferTypeString should name an on-chain withdrawal")
	assert.Equal(t, "internal transfer", withdrawalTransferTypeString(TransferTypeInternalTransfer), "withdrawalTransferTypeString should name an internal transfer")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")
	_, err = e.GetRecentTrades(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), asset.CoinMarginedFutures)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetRecentTrades must report that BingX serves no coin-M trades")

	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a   asset.Item
		exp []trade.Data
	}{
		{asset.Spot, []trade.Data{
			{TID: "247268252", Side: order.Buy, Price: 73426.33, Amount: 0.0009, Timestamp: time.UnixMilli(1794478487590)},
			{TID: "247268253", Side: order.Buy, Price: 73426.33, Amount: 0.003899, Timestamp: time.UnixMilli(1794478488590)},
			{TID: "247268254", Side: order.Sell, Price: 73426.32, Amount: 0.003215, Timestamp: time.UnixMilli(1794478488594)},
		}},
		{asset.USDTMarginedFutures, []trade.Data{
			{TID: "759616927", Side: order.Sell, Price: 73403.3, Amount: 0.0014, Timestamp: time.UnixMilli(1794478306531)},
			{TID: "759616928", Side: order.Buy, Price: 73403.4, Amount: 0.0006, Timestamp: time.UnixMilli(1794478306533)},
			{TID: "759616929", Side: order.Buy, Price: 73403.4, Amount: 0.0014, Timestamp: time.UnixMilli(1794478306647)},
		}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			pair := assetToTradablePairMap[tc.a]
			result, err := ex.GetRecentTrades(t.Context(), pair, tc.a)
			require.NoError(t, err, "GetRecentTrades must not error")
			require.NotEmpty(t, result, "GetRecentTrades must return trades")
			if !mockTests {
				return
			}
			for i := range tc.exp {
				tc.exp[i].Exchange, tc.exp[i].CurrencyPair, tc.exp[i].AssetType = ex.Name, pair, tc.a
			}
			assert.Equal(t, tc.exp, result, "GetRecentTrades should return the trades oldest first with the taker's side")
		})
	}

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/trades": {`{"code":100500,"msg":"busy"}`},
			"GET /openApi/swap/v2/quote/trades": {`{"code":100500,"msg":"busy"}`},
		})
		for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures} {
			_, err := ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), a)
			assert.ErrorIsf(t, err, errAPIResponse, "GetRecentTrades should return the %s request's error", a)
		}
	})

	t.Run("trades that cannot be saved", func(t *testing.T) {
		t.Parallel()
		// Saving checks trades only while the database is enabled, as the test configuration enables it
		dbConfig := database.DB.GetConfig()
		require.True(t, dbConfig != nil && dbConfig.Enabled, "the test configuration must enable the database")
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/trades": {`{"code":0,"data":[{"id":1,"price":0,"qty":0.5,"time":1794478487590,"buyerMaker":true}]}`},
		})
		ex.SetSaveTradeDataStatus(true)
		result, err := ex.GetRecentTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot)
		assert.ErrorContains(t, err, "received invalid trade data", "GetRecentTrades should return the error of trades it cannot save")
		assert.Nil(t, result, "GetRecentTrades should return no trades it cannot save")
	})
}

func TestSpotRecentTradeSide(t *testing.T) {
	t.Parallel()
	assert.Equal(t, order.Buy, spotRecentTradeSide(true), "spotRecentTradeSide should read buyerMaker true as a buying taker")
	assert.Equal(t, order.Sell, spotRecentTradeSide(false), "spotRecentTradeSide should read buyerMaker false as a selling taker")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1794000000000)
	end := start.Add(time.Minute)
	pair := currency.NewBTCUSDT()
	_, err := e.GetHistoricTrades(t.Context(), pair, asset.Futures, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricTrades must reject an unsupported asset")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		_, err = e.GetHistoricTrades(t.Context(), pair, a, start, end)
		require.ErrorIsf(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades must report that BingX serves no %s trades by time", a)
	}
	_, err = e.GetHistoricTrades(t.Context(), currency.EMPTYPAIR, asset.Spot, start, end)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricTrades must reject an empty pair")
	_, err = e.GetHistoricTrades(t.Context(), pair, asset.Spot, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricTrades must reject a start after its end")

	// historicalTrade returns a historical trade at an offset in milliseconds from start
	historicalTrade := func(id string, offset int64, side SpotTradeSide) string {
		return `{"tid":"` + id + `","t":` + strconv.FormatInt(start.UnixMilli()+offset, 10) + `,"ms":` + strconv.FormatUint(uint64(side), 10) + `,"s":"BTC-USDT","p":1,"v":2}`
	}
	t.Run("pages back to the start", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/historicalTrades": {
				"[" + historicalTrade("t5", 70000, SpotTradeSideBuy) + "," + historicalTrade("t4", 50000, SpotTradeSideSell) + "]",
				// A page may repeat a trade the previous page held
				"[" + historicalTrade("t4", 50000, SpotTradeSideSell) + "," + historicalTrade("t3", 1000, SpotTradeSideBuy) + "," + historicalTrade("t2", -1, SpotTradeSideSell) + "]",
			},
		})
		result, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		require.NoError(t, err, "GetHistoricTrades must not error")
		assert.Equal(t, []trade.Data{
			{TID: "t3", Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.Spot, Side: order.Buy, Price: 1, Amount: 2, Timestamp: start.Add(time.Second)},
			{TID: "t4", Exchange: ex.Name, CurrencyPair: pair, AssetType: asset.Spot, Side: order.Sell, Price: 1, Amount: 2, Timestamp: start.Add(50 * time.Second)},
		}, result, "GetHistoricTrades should return the range's trades oldest first, once each")
		got := requests()
		require.Len(t, got, 2, "GetHistoricTrades must stop paging once a trade precedes the range")
		assert.Equal(t, url.Values{"limit": {"1000"}, "symbol": {"BTC-USDT"}}, got[0].params, "GetHistoricTrades should start from the latest trades")
		assert.Equal(t, url.Values{"fromId": {"t4"}, "limit": {"1000"}, "symbol": {"BTC-USDT"}}, got[1].params, "GetHistoricTrades should page back from the oldest trade fetched")
	})

	t.Run("range starts before the trades held", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/historicalTrades": {"[" + historicalTrade("t2", 2000, SpotTradeSideBuy) + "]", "[]"},
		})
		_, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		assert.ErrorIs(t, err, errTradesBeforeHistory, "GetHistoricTrades should report a range starting before the trades BingX holds")
	})

	t.Run("no trades", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v2/quote/historicalTrades": {"[]"}})
		result, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		require.NoError(t, err, "GetHistoricTrades must not error for a symbol without trades")
		assert.Empty(t, result, "GetHistoricTrades should return no trades for a symbol without any")
	})

	t.Run("stalled paging", func(t *testing.T) {
		t.Parallel()
		page := "[" + historicalTrade("t4", 50000, SpotTradeSideSell) + "]"
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v2/quote/historicalTrades": {page, page}})
		_, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		assert.ErrorIs(t, err, errTradePagingStalled, "GetHistoricTrades should stop paging that makes no progress")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v2/quote/historicalTrades": {`{"code":100500,"msg":"busy"}`}})
		_, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricTrades should return the request's error")
	})

	t.Run("trades that cannot be saved", func(t *testing.T) {
		t.Parallel()
		// Saving checks trades only while the database is enabled, as the test configuration enables it
		dbConfig := database.DB.GetConfig()
		require.True(t, dbConfig != nil && dbConfig.Enabled, "the test configuration must enable the database")
		withoutAmount := `{"tid":"t3","t":` + strconv.FormatInt(start.UnixMilli()+1000, 10) + `,"ms":` + strconv.FormatUint(uint64(SpotTradeSideBuy), 10) + `,"s":"BTC-USDT","p":1,"v":0}`
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v2/quote/historicalTrades": {"[" + withoutAmount + "," + historicalTrade("t2", -1, SpotTradeSideSell) + "]"},
		})
		ex.SetSaveTradeDataStatus(true)
		result, err := ex.GetHistoricTrades(t.Context(), pair, asset.Spot, start, end)
		assert.ErrorContains(t, err, "received invalid trade data", "GetHistoricTrades should return the error of trades it cannot save")
		assert.Nil(t, result, "GetHistoricTrades should return no trades it cannot save")
	})

	t.Run("live", func(t *testing.T) {
		t.Parallel()
		if mockTests {
			t.Skip("historical trades page back from the latest, which the recorded responses cannot follow")
		}
		ensureTradablePairs(t)
		_, err := e.GetHistoricTrades(t.Context(), spotTradablePair, asset.Spot, time.Now().Add(-time.Minute), time.Now())
		assert.NoError(t, err, "GetHistoricTrades should not error")
	})
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Futures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetServerTime must reject an unsupported asset")
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetServerTime(t.Context(), a)
			require.NoError(t, err, "GetServerTime must not error")
			if mockTests {
				assert.Equal(t, time.UnixMilli(1794406598194), result, "GetServerTime should return the server time")
				return
			}
			assert.WithinDuration(t, time.Now(), result, time.Minute, "GetServerTime should return the current time")
		})
	}

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v2/server/time": {`{"code":100500,"msg":"busy"}`}})
		_, err := ex.GetServerTime(t.Context(), asset.Spot)
		assert.ErrorIs(t, err, errAPIResponse, "GetServerTime should return the request's error")
	})
}

// wrapperCandleTestRange returns the days candle tests request: the recorded days in mock tests, and in live tests the
// three days before yesterday, which every product keeps daily candles for
func wrapperCandleTestRange() (start, end time.Time) {
	if mockTests {
		return wrapperFixtureStart, wrapperFixtureEnd
	}
	end = time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	return end.Add(-72 * time.Hour), end
}

// wrapperExpectedCandles returns the daily candles the recorded responses hold for an asset
func wrapperExpectedCandles(a asset.Item) []kline.Candle {
	day := func(d int) time.Time { return wrapperFixtureStart.Add(time.Duration(d) * 24 * time.Hour) }
	switch a {
	case asset.Spot:
		return []kline.Candle{
			{Time: day(0), Open: 75867.6, High: 76198.59, Low: 73650.61, Close: 74851.42, Volume: 928.381643, QuoteVolume: 69983685.91},
			{Time: day(1), Open: 74851.44, High: 75592.79, Low: 73748.91, Close: 75372.41, Volume: 2024.512451, QuoteVolume: 151241359.06},
			{Time: day(2), Open: 75362.07, High: 75430.94, Low: 72735.17, Close: 73796.05, Volume: 1980.566582, QuoteVolume: 146537856.26},
		}
	case asset.USDTMarginedFutures:
		return []kline.Candle{
			{Time: day(0), Open: 75836.2, High: 76166.5, Low: 73400, Close: 74817.6, Volume: 8960.1759},
			{Time: day(1), Open: 74817.6, High: 75570.2, Low: 73715, Close: 75322.2, Volume: 17808.1481},
			{Time: day(2), Open: 75322.2, High: 75400.7, Low: 72705.9, Close: 73763.5, Volume: 16904.9886},
		}
	default:
		return []kline.Candle{
			{Time: day(0), Open: 75741.5, High: 76069.6, Low: 73387.1, Close: 74727.5, Volume: 752371},
			{Time: day(1), Open: 74727.5, High: 75480.5, Low: 73628.2, Close: 75236.2, Volume: 677715},
			{Time: day(2), Open: 75236.2, High: 75310.2, Low: 72634.3, Close: 73683.9, Volume: 837231},
		}
	}
}

// wrapperHourCandles returns a perpetual candle reply holding n hourly candles from start
func wrapperHourCandles(start time.Time, n int) string {
	rows := make([]string, n)
	for i := range n {
		rows[i] = `{"open":"1","close":"1","high":"1","low":"1","volume":"1","time":` + wrapperMilliseconds(start.Add(time.Duration(i)*time.Hour)) + `}`
	}
	return `{"code":0,"data":[` + strings.Join(rows, ",") + `]}`
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start, end := wrapperCandleTestRange()
	_, err := e.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Futures, kline.OneDay, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject an unsupported asset")
	_, err = e.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneDay, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricCandles must reject a start after its end")

	ex := newWrapperTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := ex.GetHistoricCandles(t.Context(), assetToTradablePairMap[a], a, kline.OneDay, start, end)
			require.NoError(t, err, "GetHistoricCandles must not error")
			require.Len(t, result.Candles, 3, "GetHistoricCandles must return each day's candle")
			assert.Equal(t, start, result.Candles[0].Time, "GetHistoricCandles should start at the range's first UTC day")
			if mockTests {
				assert.Equal(t, wrapperExpectedCandles(a), result.Candles, "GetHistoricCandles should convert every candle in UTC periods")
			}
		})
	}

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v2/quote/klines": {`{"code":0,"data":[]}`}})
		_, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneHour, wrapperFixtureStart, wrapperFixtureStart.Add(3*time.Hour))
		require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandles must report a range without candles")
		got := requests()
		require.Len(t, got, 1, "GetHistoricCandles must make one request")
		assert.Equal(t, url.Values{
			"symbol":    {"BTC-USDT"},
			"interval":  {"1h"},
			"startTime": {"1794096000000"},
			"endTime":   {"1794106799999"},
			"timeZone":  {"0"},
			"limit":     {"1000"},
		}, got[0].params, "GetHistoricCandles should request UTC spot candles ending before the range's end")
	})

	t.Run("range starts before the candles held", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/swap/v3/quote/klines": {wrapperHourCandles(wrapperFixtureStart.Add(time.Hour), 2)}})
		_, err := ex.GetHistoricCandles(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures, kline.OneHour, wrapperFixtureStart, wrapperFixtureStart.Add(3*time.Hour))
		assert.ErrorIs(t, err, errCandlesBeforeHistory, "GetHistoricCandles should report a range starting before the candles BingX holds")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/cswap/v1/market/klines": {`{"code":100500,"msg":"busy"}`}})
		_, err := ex.GetHistoricCandles(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), asset.CoinMarginedFutures, kline.OneHour, wrapperFixtureStart, wrapperFixtureStart.Add(3*time.Hour))
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricCandles should return the request's error")
	})
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start, end := wrapperCandleTestRange()
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Futures, kline.OneDay, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandlesExtended must reject an unsupported asset")
	_, err = e.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneDay, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricCandlesExtended must reject a start after its end")

	ex := newWrapperTestExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		t.Run(a.String(), func(t *testing.T) {
			t.Parallel()
			result, err := ex.GetHistoricCandlesExtended(t.Context(), assetToTradablePairMap[a], a, kline.OneDay, start, end)
			require.NoError(t, err, "GetHistoricCandlesExtended must not error")
			require.Len(t, result.Candles, 3, "GetHistoricCandlesExtended must return each day's candle")
			if mockTests {
				assert.Equal(t, wrapperExpectedCandles(a), result.Candles, "GetHistoricCandlesExtended should convert every candle in UTC periods")
			}
		})
	}

	t.Run("pages by each product's limit", func(t *testing.T) {
		t.Parallel()
		// A range in the past, as candle requests end at the current time
		first := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		second := first.Add(swapCandleLimit * time.Hour)
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/swap/v3/quote/klines": {wrapperHourCandles(first, swapCandleLimit), wrapperHourCandles(second, 1)},
		})
		result, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.USDTMarginedFutures, kline.OneHour, first, second.Add(time.Hour))
		require.NoError(t, err, "GetHistoricCandlesExtended must not error")
		require.Len(t, result.Candles, swapCandleLimit+1, "GetHistoricCandlesExtended must return every page's candles")
		assert.Equal(t, second, result.Candles[swapCandleLimit].Time, "GetHistoricCandlesExtended should end with the last page's candle")
		got := requests()
		require.Len(t, got, 2, "GetHistoricCandlesExtended must page by the USDT-M limit")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1h"}, "startTime": {wrapperMilliseconds(first)}, "endTime": {wrapperMilliseconds(second)}, "limit": {"1440"}}, got[0].params, "GetHistoricCandlesExtended should request a full page first")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1h"}, "startTime": {wrapperMilliseconds(second)}, "endTime": {wrapperMilliseconds(second.Add(time.Hour))}, "limit": {"1440"}}, got[1].params, "GetHistoricCandlesExtended should request the rest after it")
	})

	t.Run("range starts before the candles held", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/cswap/v1/market/klines": {`{"code":0,"data":[{"open":"1","close":"1","high":"1","low":"1","volume":"1","time":1794103200000}]}`},
		})
		_, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewPairWithDelimiter("BTC", "USD", currency.DashDelimiter), asset.CoinMarginedFutures, kline.OneHour, wrapperFixtureStart, wrapperFixtureStart.Add(3*time.Hour))
		assert.ErrorIs(t, err, errCandlesBeforeHistory, "GetHistoricCandlesExtended should report a range starting before the candles BingX holds")
	})

	t.Run("request error", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/spot/v2/quote/klines": {`{"code":100204,"msg":"no data"}`}})
		_, err := ex.GetHistoricCandlesExtended(t.Context(), currency.NewBTCUSDT(), asset.Spot, kline.OneHour, wrapperFixtureStart, wrapperFixtureStart.Add(3*time.Hour))
		assert.ErrorIs(t, err, errAPIResponse, "GetHistoricCandlesExtended should return the request's error")
	})
}

func TestCandleLimit(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]uint64{asset.Spot: 1440, asset.USDTMarginedFutures: 1440, asset.CoinMarginedFutures: 1000} {
		limit, err := candleLimit(a)
		require.NoErrorf(t, err, "candleLimit must not error for %s", a)
		assert.Equalf(t, exp, limit, "candleLimit should return the %s candle endpoint's limit", a)
	}
	_, err := candleLimit(asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "candleLimit should reject an unsupported asset")
}

func TestCandles(t *testing.T) {
	t.Parallel()
	_, err := e.candles(t.Context(), asset.Futures, currency.NewBTCUSDT(), kline.OneDay, wrapperFixtureStart, wrapperFixtureEnd, 1000)
	require.ErrorIs(t, err, asset.ErrNotSupported, "candles must reject an unsupported asset")
	perpetualCandle := `{"code":0,"data":[{"open":"1","close":"1.5","high":"2","low":"0.5","volume":"10","time":1794096000000}]}`
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/spot/v2/quote/klines":   {`{"code":0,"data":[[1794096000000,1,2,0.5,1.5,10,1794182399999,15]]}`, `{"code":100500,"msg":"busy"}`},
		"GET /openApi/swap/v3/quote/klines":   {perpetualCandle, `{"code":100500,"msg":"busy"}`},
		"GET /openApi/cswap/v1/market/klines": {perpetualCandle, `{"code":100500,"msg":"busy"}`},
	})
	for _, tc := range []struct {
		a   asset.Item
		exp kline.Candle
	}{
		{asset.Spot, kline.Candle{Time: wrapperFixtureStart, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10, QuoteVolume: 15}},
		{asset.USDTMarginedFutures, kline.Candle{Time: wrapperFixtureStart, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}},
		{asset.CoinMarginedFutures, kline.Candle{Time: wrapperFixtureStart, Open: 1, High: 2, Low: 0.5, Close: 1.5, Volume: 10}},
	} {
		candles, err := ex.candles(t.Context(), tc.a, currency.NewBTCUSDT(), kline.OneDay, wrapperFixtureStart, wrapperFixtureEnd, 7)
		require.NoErrorf(t, err, "candles must not error for %s", tc.a)
		assert.Equalf(t, []kline.Candle{tc.exp}, candles, "candles should convert the %s candle in UTC", tc.a)
		_, err = ex.candles(t.Context(), tc.a, currency.NewBTCUSDT(), kline.OneDay, wrapperFixtureStart, wrapperFixtureEnd, 7)
		assert.ErrorIsf(t, err, errAPIResponse, "candles should return the %s request's error", tc.a)
	}
	got := requests()
	require.Len(t, got, 6, "candles must request each asset's candles twice")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1d"}, "startTime": {"1794096000000"}, "endTime": {"1794355199999"}, "timeZone": {"0"}, "limit": {"7"}}, got[0].params, "candles should request UTC spot candles ending before end")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1d"}, "startTime": {"1794096000000"}, "endTime": {"1794355200000"}, "limit": {"7"}}, got[2].params, "candles should request USDT-M candles ending at end, which is exclusive")
	assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "interval": {"1d"}, "startTime": {"1794096000000"}, "endTime": {"1794355200000"}, "limit": {"7"}}, got[4].params, "candles should request coin-M candles ending at end, which is exclusive")
}

func TestCheckCandleHistory(t *testing.T) {
	t.Parallel()
	start := wrapperFixtureStart
	assert.NoError(t, checkCandleHistory(nil, start), "checkCandleHistory should leave an empty series to the request processing")
	assert.NoError(t, checkCandleHistory([]kline.Candle{{Time: start.Add(time.Hour)}, {Time: start}}, start), "checkCandleHistory should accept candles from the range's start, in any order")
	assert.ErrorIs(t, checkCandleHistory([]kline.Candle{{Time: start.Add(2 * time.Hour)}, {Time: start.Add(time.Hour)}}, start), errCandlesBeforeHistory, "checkCandleHistory should report candles starting after the range")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")

	t.Run("offline and other types", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{})
		fee, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, PurchasePrice: 100, Amount: 2})
		require.NoError(t, err, "GetFeeByType must not error")
		assert.InDelta(t, 0.2, fee, 1e-12, "GetFeeByType should estimate an offline fee at BingX's standard spot rate")
		fee, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee})
		require.NoError(t, err, "GetFeeByType must not error")
		assert.Zero(t, fee, "GetFeeByType should charge no deposit fee")
		ex.SetCredentials(&accounts.Credentials{})
		builder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), PurchasePrice: 100, Amount: 2}
		fee, err = ex.GetFeeByType(t.Context(), builder)
		require.NoError(t, err, "GetFeeByType must not error")
		assert.InDelta(t, 0.2, fee, 1e-12, "GetFeeByType should estimate a trade fee offline without credentials")
		assert.Equal(t, exchange.CryptocurrencyTradeFee, builder.FeeType, "GetFeeByType should leave the fee builder as it was")
	})

	t.Run("account rates", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		builder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotTradablePair, PurchasePrice: 100, Amount: 2}
		taker, err := ex.GetFeeByType(t.Context(), builder)
		require.NoError(t, err, "GetFeeByType must not error")
		builder.IsMaker = true
		maker, err := ex.GetFeeByType(t.Context(), builder)
		require.NoError(t, err, "GetFeeByType must not error")
		withdrawal, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()})
		require.NoError(t, err, "GetFeeByType must not error")
		if !mockTests {
			return
		}
		assert.InDelta(t, 0.000325*200, taker, 1e-12, "GetFeeByType should charge the taker rate")
		assert.InDelta(t, 0.0001*200, maker, 1e-12, "GetFeeByType should charge the maker rate")
		assert.Equal(t, 0.0006, withdrawal, "GetFeeByType should charge the default network's withdrawal fee")
	})

	t.Run("request errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/user/commissionRate":      {`{"code":100001,"msg":"signature"}`},
			"GET /openApi/wallets/v1/capital/config/getall": {`{"code":100001,"msg":"signature"}`},
		})
		_, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT()})
		assert.ErrorIs(t, err, errAPIResponse, "GetFeeByType should return the commission rate request's error")
		_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT()})
		assert.ErrorIs(t, err, errAPIResponse, "GetFeeByType should return the coin configuration request's error")
	})
}

// wrapperCoinConfig is a coin configuration reply: USDT with a default network and another, a coin without networks and
// one without a default network
const wrapperCoinConfig = `{"code":0,"data":[
	{"coin":"USDT","networkList":[
		{"network":"ERC20","isDefault":false,"withdrawEnable":false,"depositEnable":true,"withdrawFee":"5"},
		{"network":"TRC20","isDefault":true,"withdrawEnable":true,"depositEnable":false,"withdrawFee":"1"}]},
	{"coin":"NONET","networkList":[]},
	{"coin":"NODEF","networkList":[{"network":"X","isDefault":false,"withdrawEnable":false,"depositEnable":false}]}]}`

func TestCoinNetworks(t *testing.T) {
	t.Parallel()
	_, err := e.coinNetworks(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "coinNetworks must reject an empty coin")
	ex, requests := wrapperTestServer(t, map[string][]string{
		"GET /openApi/wallets/v1/capital/config/getall": {wrapperCoinConfig, wrapperCoinConfig, `{"code":100001,"msg":"signature"}`},
	})
	networks, err := ex.coinNetworks(t.Context(), currency.USDT)
	require.NoError(t, err, "coinNetworks must not error")
	require.Len(t, networks, 2, "coinNetworks must return the coin's networks")
	assert.Equal(t, "TRC20", networks[1].Network, "coinNetworks should return the networks in order")
	assert.Equal(t, url.Values{"coin": {"USDT"}}, requests()[0].params, "coinNetworks should request the coin's configuration")
	_, err = ex.coinNetworks(t.Context(), currency.NewCode("NONET"))
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "coinNetworks should report a coin without networks")
	_, err = ex.coinNetworks(t.Context(), currency.USDT)
	assert.ErrorIs(t, err, errAPIResponse, "coinNetworks should return the request's error")
}

func TestDefaultNetwork(t *testing.T) {
	t.Parallel()
	ex, _ := wrapperTestServer(t, map[string][]string{
		"GET /openApi/wallets/v1/capital/config/getall": {wrapperCoinConfig, wrapperCoinConfig, wrapperCoinConfig},
	})
	network, err := ex.defaultNetwork(t.Context(), currency.USDT)
	require.NoError(t, err, "defaultNetwork must not error")
	assert.Equal(t, "TRC20", network.Network, "defaultNetwork should return the default network")
	_, err = ex.defaultNetwork(t.Context(), currency.NewCode("NODEF"))
	assert.ErrorIs(t, err, errDefaultNetworkAbsent, "defaultNetwork should report a coin without a default network")
	_, err = ex.defaultNetwork(t.Context(), currency.NewCode("MISSING"))
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "defaultNetwork should report a coin without a configuration")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty coin")
	ex := newWrapperTestExchange(t)
	sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
	chains, err := ex.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if mockTests {
		assert.Equal(t, []string{"BTC", "BEP20"}, chains, "GetAvailableTransferChains should list the coin's networks")
		return
	}
	assert.NotEmpty(t, chains, "GetAvailableTransferChains should list networks")
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty coin")

	t.Run("chain", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		address, err := ex.GetDepositAddress(t.Context(), currency.USDT, "", "trc20")
		require.NoError(t, err, "GetDepositAddress must not error")
		if mockTests {
			assert.Equal(t, &deposit.Address{Address: "40e*****95", Chain: "TRC20"}, address, "GetDepositAddress should return the chain's address")
		}
	})

	addresses := `{"code":0,"data":{"data":[{"coin":"USDT","network":"ERC20","address":"erc","tag":""},{"coin":"USDT","network":"TRC20","address":"trc","tag":"memo"}],"total":2}}`
	t.Run("default network", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{
			"GET /openApi/wallets/v1/capital/deposit/address": {addresses},
			"GET /openApi/wallets/v1/capital/config/getall":   {wrapperCoinConfig},
		})
		address, err := ex.GetDepositAddress(t.Context(), currency.USDT, "", "")
		require.NoError(t, err, "GetDepositAddress must not error")
		assert.Equal(t, &deposit.Address{Address: "trc", Tag: "memo", Chain: "TRC20"}, address, "GetDepositAddress should return the default network's address")
		assert.Equal(t, url.Values{"coin": {"USDT"}, "limit": {"1000"}}, requests()[0].params, "GetDepositAddress should request every address of the coin")
	})

	t.Run("only network", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/wallets/v1/capital/deposit/address": {`{"code":0,"data":{"data":[{"coin":"BTC","network":"BTC","address":"btc"}],"total":1}}`},
		})
		address, err := ex.GetDepositAddress(t.Context(), currency.BTC, "", "")
		require.NoError(t, err, "GetDepositAddress must not error")
		assert.Equal(t, &deposit.Address{Address: "btc", Chain: "BTC"}, address, "GetDepositAddress should return the only network's address")
	})

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/wallets/v1/capital/deposit/address": {addresses, addresses, `{"code":100437,"msg":"coin"}`},
			"GET /openApi/wallets/v1/capital/config/getall":   {`{"code":100001,"msg":"signature"}`},
		})
		_, err := ex.GetDepositAddress(t.Context(), currency.USDT, "", "SOL")
		assert.ErrorIs(t, err, deposit.ErrAddressNotFound, "GetDepositAddress should report a chain without an address")
		_, err = ex.GetDepositAddress(t.Context(), currency.USDT, "", "")
		assert.ErrorIs(t, err, errAPIResponse, "GetDepositAddress should return the default network request's error")
		_, err = ex.GetDepositAddress(t.Context(), currency.USDT, "", "")
		assert.ErrorIs(t, err, errAPIResponse, "GetDepositAddress should return the address request's error")
	})
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Type: withdraw.Crypto, Currency: currency.USDT, Amount: 1})
	require.ErrorIs(t, err, common.ErrExchangeNameNotSet, "WithdrawCryptocurrencyFunds must validate the request")

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		wrapperSkipIfCannotManipulateOrders(t, ex)
		resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange:      ex.Name,
			Type:          withdraw.Crypto,
			Currency:      currency.USDT,
			Amount:        4998,
			ClientOrderID: "abc123",
			Crypto:        withdraw.CryptoRequest{Address: "0x8****11", Chain: "BEP20"},
		})
		require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
		if mockTests {
			assert.Equal(t, &withdraw.ExchangeResponse{ID: "125*****4"}, resp, "WithdrawCryptocurrencyFunds should return the withdrawal's ID")
		}
	})

	t.Run("request", func(t *testing.T) {
		t.Parallel()
		ex, requests := wrapperTestServer(t, map[string][]string{"POST /openApi/wallets/v1/capital/withdraw/apply": {`{"code":100202,"msg":"balance"}`}})
		_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{Exchange: ex.Name, Type: withdraw.Crypto, Currency: currency.BTC, Amount: 1, Crypto: withdraw.CryptoRequest{Address: "addr", AddressTag: "tag"}})
		assert.ErrorIs(t, err, errAPIResponse, "WithdrawCryptocurrencyFunds should return the request's error")
		got := requests()
		require.Len(t, got, 1, "WithdrawCryptocurrencyFunds must make one request")
		assert.Equal(t, url.Values{"coin": {"BTC"}, "address": {"addr"}, "addressTag": {"tag"}, "amount": {"1"}, "walletType": {"1"}}, got[0].params, "WithdrawCryptocurrencyFunds should withdraw from the fund account")
	})
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should report that BingX withdraws no fiat")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should report that BingX withdraws no fiat")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an unsupported asset")
	require.ErrorIs(t, new(Exchange).UpdateOrderExecutionLimits(t.Context(), asset.Spot), currency.ErrPairManagerNotInitialised, "UpdateOrderExecutionLimits must return the pair format error")

	ex := newWrapperTestExchange(t)
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  limits.MinMaxLevel
	}{
		{asset.Spot, currency.NewPairWithDelimiter("GPT-OLD", "USDT", currency.DashDelimiter), limits.MinMaxLevel{
			PriceStepIncrementSize: 0.00001, AmountStepIncrementSize: 0.1, MinNotional: 5, MaximumQuoteAmount: 20000, Listed: time.UnixMilli(1678446300000),
		}},
		{asset.USDTMarginedFutures, currency.NewPairWithDelimiter("ETH", "USDT", currency.DashDelimiter), limits.MinMaxLevel{
			PriceStepIncrementSize: 0.01, AmountStepIncrementSize: 0.01, MinimumBaseAmount: 0.01, MinNotional: 2, Listed: time.UnixMilli(1586275200000),
		}},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("SUI", "USD", currency.DashDelimiter), limits.MinMaxLevel{
			PriceStepIncrementSize: 0.0001, MinimumBaseAmount: 1, Listed: time.UnixMilli(1741921200000),
		}},
	} {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), tc.a), "UpdateOrderExecutionLimits must not error")
			pair := tc.pair
			if !mockTests {
				pair = assetToTradablePairMap[tc.a]
			}
			l, err := ex.GetOrderExecutionLimits(tc.a, pair)
			require.NoError(t, err, "GetOrderExecutionLimits must not error")
			if !mockTests {
				assert.Positive(t, l.PriceStepIncrementSize, "UpdateOrderExecutionLimits should load the price step")
				return
			}
			tc.exp.Key = key.NewExchangeAssetPair(ex.Name, tc.a, pair)
			tc.exp.UpdatedAt = l.UpdatedAt
			assert.Equal(t, tc.exp, l, "UpdateOrderExecutionLimits should load the symbol's limits")
		})
	}

	t.Run("errors", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{
			"GET /openApi/spot/v1/common/symbols": {
				`{"code":100500,"msg":"busy"}`,
				`{"code":0,"data":{"symbols":[{"symbol":"NODASH","tickSize":1},{"symbol":"BTC-USDT","tickSize":0.1,"maxNotional":-1}]}}`,
			},
			"GET /openApi/swap/v2/quote/contracts":   {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":[{"symbol":"NODASH"}]}`},
			"GET /openApi/cswap/v1/market/contracts": {`{"code":100500,"msg":"busy"}`, `{"code":0,"data":[{"symbol":"NODASH"}]}`},
		})
		for _, a := range ex.GetAssetTypes(false) {
			assert.ErrorIsf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), errAPIResponse, "UpdateOrderExecutionLimits should return the %s request's error", a)
		}
		require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), "UpdateOrderExecutionLimits must skip a symbol it cannot read")
		l, err := ex.GetOrderExecutionLimits(asset.Spot, currency.NewBTCUSDT())
		require.NoError(t, err, "GetOrderExecutionLimits must not error")
		assert.Zero(t, l.MaximumQuoteAmount, "UpdateOrderExecutionLimits should read a negative maximum notional as no limit")
		for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
			assert.ErrorIsf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), limits.ErrEmptyLevels, "UpdateOrderExecutionLimits should report %s without limits to load", a)
		}
	})
}

func TestPrecisionStep(t *testing.T) {
	t.Parallel()
	for places, exp := range map[uint64]float64{0: 1, 1: 0.1, 4: 0.0001, 8: 0.00000001} {
		assert.Equalf(t, exp, precisionStep(places), "precisionStep should return the step of %d places", places)
	}
	assert.Equal(t, 1e-300, precisionStep(1<<63), "precisionStep should clamp a precision beyond float range")
}

func TestUpdateCurrencyStates(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateCurrencyStates(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateCurrencyStates must reject an unsupported asset")
	for _, a := range []asset.Item{asset.USDTMarginedFutures, asset.CoinMarginedFutures} {
		require.ErrorIsf(t, e.UpdateCurrencyStates(t.Context(), a), common.ErrFunctionNotSupported, "UpdateCurrencyStates must report that BingX keeps no %s currency states", a)
	}

	t.Run("states", func(t *testing.T) {
		t.Parallel()
		ex, _ := wrapperTestServer(t, map[string][]string{"GET /openApi/wallets/v1/capital/config/getall": {wrapperCoinConfig, `{"code":100001,"msg":"signature"}`}})
		require.NoError(t, ex.UpdateCurrencyStates(t.Context(), asset.Spot), "UpdateCurrencyStates must not error")
		assert.NoError(t, ex.States.CanDeposit(currency.USDT, asset.Spot), "UpdateCurrencyStates should allow deposits a network takes")
		assert.NoError(t, ex.States.CanWithdraw(currency.USDT, asset.Spot), "UpdateCurrencyStates should allow withdrawals a network takes")
		assert.Error(t, ex.States.CanDeposit(currency.NewCode("NODEF"), asset.Spot), "UpdateCurrencyStates should refuse deposits no network takes")
		assert.Error(t, ex.States.CanWithdraw(currency.NewCode("NONET"), asset.Spot), "UpdateCurrencyStates should refuse withdrawals of a coin without networks")
		assert.ErrorIs(t, ex.UpdateCurrencyStates(t.Context(), asset.Spot), errAPIResponse, "UpdateCurrencyStates should return the request's error")
	})

	t.Run("recorded", func(t *testing.T) {
		t.Parallel()
		ex := newWrapperTestExchange(t)
		sharedtestvalues.SkipTestIfCredentialsUnset(t, ex)
		require.NoError(t, ex.UpdateCurrencyStates(t.Context(), asset.Spot), "UpdateCurrencyStates must not error")
		if mockTests {
			assert.NoError(t, ex.States.CanWithdraw(currency.BTC, asset.Spot), "UpdateCurrencyStates should allow BTC withdrawals")
		}
	})
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Futures, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an unsupported asset")
	_, err = new(Exchange).GetCurrencyTradeURL(t.Context(), asset.USDTMarginedFutures, currency.NewBTCUSDT())
	require.ErrorIs(t, err, currency.ErrPairManagerNotInitialised, "GetCurrencyTradeURL must return the symbol formatting error")
	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  string
	}{
		{asset.Spot, currency.NewBTCUSDT(), "https://bingx.com/en/spot/BTCUSDT"},
		{asset.USDTMarginedFutures, currency.NewPair(currency.BTC, currency.USDC), "https://bingx.com/en/perpetual/BTC-USDC"},
		{asset.CoinMarginedFutures, currency.NewPairWithDelimiter("btc", "usd", "_"), "https://bingx.com/en/perpetual/BTC-USD"},
	} {
		resp, err := e.GetCurrencyTradeURL(t.Context(), tc.a, tc.pair)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", tc.a)
		assert.Equalf(t, tc.exp, resp, "GetCurrencyTradeURL should return the %s trade page", tc.a)
	}
	ensureTradablePairs(t)
	for a, p := range assetToTradablePairMap {
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, p)
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.NotEmptyf(t, resp, "GetCurrencyTradeURL should return the %s test pair's page", a)
	}
}
