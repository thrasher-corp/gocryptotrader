package kucoin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/key"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/deposit"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// marketTestServerExchange returns an exchange named for the test whose spot and futures hosts are a test server
// answering each request path with its body in responses, so tests can drive replies the shared fixtures cannot
func marketTestServerExchange(t *testing.T, responses map[string]string) *Exchange {
	t.Helper()
	server := httptest.NewTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := responses[r.URL.Path]
		assert.Truef(t, ok, "request path %s should be one the test serves", r.URL.Path)
		_, err := fmt.Fprint(w, body)
		assert.NoError(t, err, "writing the response should not error")
	}))
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestFutures} {
		require.NoErrorf(t, ex.API.Endpoints.SetRunningURL(u.String(), server.URL+"/api"), "SetRunningURL must not error for %s", u)
	}
	return ex
}

// storeOptionsPair enables an options pair on ex, an asset KuCoin does not trade, so wrapper methods reach their
// unsupported asset branches past the pair store checks
func storeOptionsPair(t *testing.T, ex *Exchange) currency.Pair {
	t.Helper()
	pair := currency.NewPairWithDelimiter("BTC", "26DEC26-100000-C", "-")
	pairFormat := &currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	require.NoError(t, ex.CurrencyPairs.Store(asset.Options, &currency.PairStore{
		AssetEnabled:  true,
		Available:     currency.Pairs{pair},
		Enabled:       currency.Pairs{pair},
		RequestFormat: pairFormat,
		ConfigFormat:  pairFormat,
	}), "Store must not error")
	return pair
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	exp := protocol.Features{
		TickerFetching:        true,
		TickerBatching:        true,
		OrderbookFetching:     true,
		AutoPairUpdates:       true,
		AccountInfo:           true,
		CryptoDeposit:         true,
		CryptoWithdrawal:      true,
		SubmitOrder:           true,
		GetOrder:              true,
		GetOrders:             true,
		CancelOrder:           true,
		CancelOrders:          true,
		ModifyOrder:           true,
		TradeFetching:         true,
		UserTradeHistory:      true,
		TradeFee:              true,
		CryptoWithdrawalFee:   true,
		KlineFetching:         true,
		DepositHistory:        true,
		WithdrawalHistory:     true,
		MultiChainDeposits:    true,
		MultiChainWithdrawals: true,
	}
	assert.Equal(t, exp, ex.Features.Supports.RESTCapabilities, "SetDefaults should set the REST capabilities")
	assert.Equal(t, map[kline.Interval]bool{kline.OneHour: true, kline.FourHour: true, kline.EightHour: true}, ex.Features.Supports.FuturesCapabilities.SupportedFundingRateFrequencies, "SetDefaults should support KuCoin's 1, 4 and 8 hour funding intervals")
	assert.Equal(t, uint64(500), ex.Features.Enabled.Kline.GlobalResultLimit, "SetDefaults should limit candle requests to 500")
	assert.True(t, ex.Features.Enabled.Kline.Intervals.ExchangeSupported(kline.SixHour), "SetDefaults should support spot's 6 hour candles")
	assert.False(t, ex.Features.Supports.FuturesCapabilities.CollateralMode, "SetDefaults should not claim collateral mode support, which KuCoin offers on its website only")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	require.Error(t, ex.Setup(nil), "Setup must reject a missing configuration")
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "GetStandardConfig must not error")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should leave a disabled exchange disabled")

	withoutURL := func(missing exchange.URL) func(*testing.T, *Exchange, *config.Exchange) {
		return func(t *testing.T, ex *Exchange, _ *config.Exchange) {
			t.Helper()
			urls := ex.API.Endpoints.GetURLMap()
			ex.API.Endpoints = ex.NewEndpoints()
			for k, v := range urls {
				if k != missing.String() {
					require.NoError(t, ex.API.Endpoints.SetRunningURL(k, v), "SetRunningURL must not error")
				}
			}
		}
	}
	for _, tc := range []struct {
		name      string
		configure func(*testing.T, *Exchange, *config.Exchange)
		err       error
		// errText is matched in the error's message; the websocket manager does not export its errors
		errText string
	}{
		{name: "proxy address", configure: func(_ *testing.T, _ *Exchange, cfg *config.Exchange) { cfg.ProxyAddress = string(rune(0x7f)) }, err: exchange.ErrSettingProxyAddress},
		{name: "subscription migration", configure: func(_ *testing.T, _ *Exchange, cfg *config.Exchange) {
			// A legacy realtime book for an asset KuCoin does not trade cannot have its pair coverage worked out
			cfg.Features = &config.FeaturesConfig{Subscriptions: subscription.List{
				{Enabled: true, Channel: marketOrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{currency.NewBTCUSDT()}},
				{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.Options, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}},
			}}
		}, err: asset.ErrNotSupported},
		{name: "websocket traffic timeout", configure: func(_ *testing.T, _ *Exchange, cfg *config.Exchange) { cfg.WebsocketTrafficTimeout = time.Millisecond }, errText: "invalid traffic timeout"},
		{name: "no spot stream URL", configure: withoutURL(exchange.WebsocketSpot), err: exchange.ErrEndpointPathNotFound},
		{name: "no futures stream URL", configure: withoutURL(exchange.WebsocketFutures), err: exchange.ErrEndpointPathNotFound},
		{name: "spot connection refused", configure: func(_ *testing.T, ex *Exchange, _ *config.Exchange) { ex.Websocket.TrafficAlert = nil }, errText: "traffic alert is nil"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			cfg, err := ex.GetStandardConfig()
			require.NoError(t, err, "GetStandardConfig must not error")
			tc.configure(t, ex, cfg)
			err = ex.Setup(cfg)
			require.Error(t, err, "Setup must fail")
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "Setup should return the failing step's error")
			}
			if tc.errText != "" {
				assert.ErrorContains(t, err, tc.errText, "Setup should return the failing step's error")
			}
		})
	}

	// The order connections always dial and sign in with the account's keys, so only authenticated websocket use with
	// credentials sets them up, and the UTA one only alongside a configured UTA channel
	for _, tc := range []struct {
		name          string
		authenticated bool
		credentials   bool
		utaChannel    bool
		exp           []string
	}{
		{name: "public", exp: []string{kucoinWebsocketURL, kucoinFuturesWebsocketURL, kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL, kucoinUTAPrivateWebsocketURL}},
		{name: "credentials without authenticated websocket use", credentials: true, utaChannel: true, exp: []string{kucoinWebsocketURL, kucoinFuturesWebsocketURL, kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL, kucoinUTAPrivateWebsocketURL}},
		{name: "authenticated websocket use without credentials", authenticated: true, utaChannel: true, exp: []string{kucoinWebsocketURL, kucoinFuturesWebsocketURL, kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL, kucoinUTAPrivateWebsocketURL}},
		{name: "authenticated", authenticated: true, credentials: true, exp: []string{kucoinWebsocketURL, kucoinFuturesWebsocketURL, kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL, kucoinUTAPrivateWebsocketURL, kucoinOrderWebsocketURL}},
		{name: "authenticated with a UTA channel", authenticated: true, credentials: true, utaChannel: true, exp: []string{kucoinWebsocketURL, kucoinFuturesWebsocketURL, kucoinUTASpotPublicWebsocketURL, kucoinUTAFuturesPublicWebsocketURL, kucoinUTAPrivateWebsocketURL, kucoinOrderWebsocketURL, kucoinUTATradeWebsocketURL}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := new(Exchange)
			ex.SetDefaults()
			cfg, err := ex.GetStandardConfig()
			require.NoError(t, err, "GetStandardConfig must not error")
			cfg.API.AuthenticatedWebsocketSupport = tc.authenticated
			if tc.credentials {
				cfg.API.Credentials.Key, cfg.API.Credentials.Secret, cfg.API.Credentials.ClientID = "test-key", "test-secret", "test-passphrase"
			}
			if tc.utaChannel {
				cfg.Features = &config.FeaturesConfig{Subscriptions: subscription.List{{Enabled: true, Channel: UTABalanceChannel, Authenticated: true}}}
			}
			require.NoError(t, ex.Setup(cfg), "Setup must not error")
			registered, err := ex.Websocket.GetConfiguredWebsocketURLs()
			require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
			assert.ElementsMatch(t, tc.exp, registered, "Setup should register the expected connections")
		})
	}
}

func TestFuturesContractPair(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		symbol string
		base   currency.Code
		exp    currency.Pair
		err    error
	}{
		{symbol: "XBTUSDTM", base: currency.XBT, exp: currency.NewPair(currency.XBT, currency.USDTM)},
		{symbol: "XBTMZ26", base: currency.XBT, exp: currency.NewPair(currency.XBT, currency.NewCode("MZ26"))},
		{symbol: "xbtusdtm", base: currency.XBT, exp: currency.NewPair(currency.XBT, currency.NewCode("usdtm"))},
		{symbol: "XBTUSDTM", err: errContractSymbolMismatch},
		{symbol: "XBT", base: currency.XBT, err: errContractSymbolMismatch},
		{symbol: "ETHUSDTM", base: currency.XBT, err: errContractSymbolMismatch},
	} {
		pair, err := futuresContractPair(&Contract{Symbol: tc.symbol, BaseCurrency: tc.base})
		require.ErrorIsf(t, err, tc.err, "futuresContractPair must return the expected error for %q with base %q", tc.symbol, tc.base)
		assert.Equalf(t, tc.exp, pair, "futuresContractPair should split %q at its base currency %q", tc.symbol, tc.base)
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an asset KuCoin does not trade")
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		_, err = e.FetchTradablePairs(cancelledContext(t), a)
		require.ErrorIsf(t, err, context.Canceled, "FetchTradablePairs must return the failed %s request's error", a)
	}

	ex := marketTestServerExchange(t, map[string]string{
		"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"ETHUSDTM","baseCurrency":"ETH","status":"Paused"},{"symbol":"XBTUSDTM","baseCurrency":"ETH","status":"Open"}]}`,
		"/api/v2/symbols":          `{"code":"200000","data":[{"symbol":"ETH-USDT","enableTrading":false},{"symbol":"AB","enableTrading":true}]}`,
	})
	_, err = ex.FetchTradablePairs(t.Context(), asset.Futures)
	require.ErrorIs(t, err, errContractSymbolMismatch, "FetchTradablePairs must reject a contract whose symbol does not start with its base currency")
	_, err = ex.FetchTradablePairs(t.Context(), asset.Spot)
	require.ErrorIs(t, err, currency.ErrCreatingPair, "FetchTradablePairs must reject a symbol too short to be a pair")

	for _, tc := range []struct {
		asset asset.Item
		exp   currency.Pairs
	}{
		{
			asset: asset.Spot,
			exp: currency.Pairs{
				currency.NewPairWithDelimiter("BTC", "USDT", "-"),
				currency.NewPairWithDelimiter("DOLO", "USDT", "-"),
				currency.NewPairWithDelimiter("BTC", "EUR", "-"),
			},
		},
		{
			// Only symbols with margin trading enabled
			asset: asset.Margin,
			exp:   currency.Pairs{currency.NewPairWithDelimiter("BTC", "USDT", "-")},
		},
		{
			asset: asset.Futures,
			exp: currency.Pairs{
				currency.NewPair(currency.XBT, currency.USDTM),
				currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDTM),
				currency.NewPair(currency.XBT, currency.NewCode("MZ26")),
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			pairs, err := e.FetchTradablePairs(t.Context(), tc.asset)
			require.NoError(t, err, "FetchTradablePairs must not error")
			if mockTests {
				assert.Equal(t, tc.exp, pairs, "FetchTradablePairs should return every tradable symbol's pair")
				return
			}
			assert.NotEmpty(t, pairs, "FetchTradablePairs should return pairs")
		})
	}
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v2/symbols":          `{"code":"200000","data":[]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[]}`,
	})
	require.ErrorIs(t, ex.UpdateTradablePairs(t.Context()), currency.ErrCurrencyPairsEmpty, "UpdateTradablePairs must not store an empty pair list")

	ex = testInstance(t)
	require.ErrorIs(t, ex.UpdateTradablePairs(cancelledContext(t)), context.Canceled, "UpdateTradablePairs must return a failed request's error")
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	exp := map[asset.Item]currency.Pairs{
		asset.Spot: {
			currency.NewPairWithDelimiter("BTC", "USDT", "-"),
			currency.NewPairWithDelimiter("DOLO", "USDT", "-"),
			currency.NewPairWithDelimiter("BTC", "EUR", "-"),
		},
		asset.Margin: {currency.NewPairWithDelimiter("BTC", "USDT", "-")},
		asset.Futures: {
			currency.NewPairWithDelimiter("XBT", "USDTM", "_"),
			currency.NewPairWithDelimiter("ANTHROPIC", "USDTM", "_"),
			currency.NewPairWithDelimiter("XBT", "MZ26", "_"),
		},
	}
	for _, a := range ex.GetAssetTypes(true) {
		pairs, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		if mockTests {
			assert.Equalf(t, exp[a], pairs, "UpdateTradablePairs should store the %s pairs in the configured format", a)
			continue
		}
		assert.NotEmptyf(t, pairs, "UpdateTradablePairs should store %s pairs", a)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	ex.Name = t.Name()
	_, err := ex.UpdateOrderbook(t.Context(), currency.NewBTCUSDT(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an asset KuCoin does not trade")
	_, err = ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook must reject an empty pair")
	for a, p := range map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSDT(), asset.Futures: currency.NewPair(currency.XBT, currency.USDTM)} {
		_, err = ex.UpdateOrderbook(cancelledContext(t), p, a)
		require.ErrorIsf(t, err, context.Canceled, "UpdateOrderbook must return the failed %s request's error", a)
	}
	_, err = ex.UpdateOrderbook(t.Context(), storeOptionsPair(t, ex), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an enabled asset KuCoin does not trade")

	futuresPair := currency.NewPair(currency.XBT, currency.USDTM)
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/level2/snapshot": `{"code":"200000","data":null}`})
	_, err = ex.UpdateOrderbook(t.Context(), futuresPair, asset.Futures)
	require.ErrorIs(t, err, common.ErrNoResponse, "UpdateOrderbook must reject a futures reply without a book")
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/level2/snapshot": `{"code":"200000","data":{"symbol":"XBTUSDTM","bids":[[0,5]],"asks":[[83390.4,327]]}}`})
	_, err = ex.UpdateOrderbook(t.Context(), futuresPair, asset.Futures)
	require.ErrorIs(t, err, orderbook.ErrPriceZero, "UpdateOrderbook must reject a book that fails validation")

	spotBook := func(a asset.Item) *orderbook.Book {
		return &orderbook.Book{
			Bids:              orderbook.Levels{{Amount: 0.01, Price: 83475.5}, {Amount: 0.01, Price: 83475.4}},
			Asks:              orderbook.Levels{{Amount: 2.03945572, Price: 83475.6}, {Amount: 0.03136865, Price: 83476}},
			Exchange:          t.Name(),
			Pair:              currency.NewPairWithDelimiter("BTC", "USDT", "-"),
			Asset:             a,
			LastUpdated:       time.UnixMilli(1791898291420),
			ValidateOrderbook: true,
			RestSnapshot:      true,
		}
	}
	exp := map[asset.Item]*orderbook.Book{
		asset.Spot:   spotBook(asset.Spot),
		asset.Margin: spotBook(asset.Margin),
		asset.Futures: {
			Bids:              orderbook.Levels{{Amount: 128, Price: 83390.3}, {Amount: 109, Price: 83390.1}, {Amount: 5, Price: 83383.3}},
			Asks:              orderbook.Levels{{Amount: 327, Price: 83390.4}, {Amount: 19, Price: 83392.1}, {Amount: 46, Price: 83395.9}},
			Exchange:          t.Name(),
			Pair:              currency.NewPair(currency.XBT, currency.USDTM),
			Asset:             asset.Futures,
			LastUpdated:       time.Unix(0, 1791898178522000000),
			ValidateOrderbook: true,
			RestSnapshot:      true,
		},
	}
	ensureTradablePairs(t)
	ex = testInstance(t)
	ex.Name = t.Name()
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		result, err := ex.UpdateOrderbook(t.Context(), assertToTradablePairMap[a], a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", a)
		if mockTests {
			assert.WithinDurationf(t, time.Now(), result.InsertedAt, time.Minute, "UpdateOrderbook should insert the %s book now", a)
			exp[a].InsertedAt = result.InsertedAt
			assert.Equalf(t, exp[a], result, "UpdateOrderbook should store the %s book with KuCoin's time", a)
			continue
		}
		assert.NotEmptyf(t, result.Bids, "UpdateOrderbook should return %s bids", a)
		assert.NotEmptyf(t, result.Asks, "UpdateOrderbook should return %s asks", a)
	}
}

func TestMergeRoundedOrderbookLevels(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		levels []orderbook.Level
		exp    []orderbook.Level
	}{
		{
			name:   "empty",
			levels: nil,
			exp:    nil,
		},
		{
			name: "single",
			levels: []orderbook.Level{
				{Price: 9.99, Amount: 1},
			},
			exp: []orderbook.Level{
				{Price: 9.99, Amount: 1},
			},
		},
		{
			name: "adjacentDuplicatesMerge",
			levels: []orderbook.Level{
				{Price: 1e18, Amount: 0.00001405},
				{Price: 1e18, Amount: 0.00064},
				{Price: 9.99, Amount: 1},
			},
			exp: []orderbook.Level{
				{Price: 1e18, Amount: 0.00065405},
				{Price: 9.99, Amount: 1},
			},
		},
		{
			name: "nonAdjacentDuplicatesRemainSeparate",
			levels: []orderbook.Level{
				{Price: 1.0, Amount: 1.0},
				{Price: 2.0, Amount: 2.0},
				{Price: 1.0, Amount: 3.0},
			},
			exp: []orderbook.Level{
				{Price: 1.0, Amount: 1.0},
				{Price: 2.0, Amount: 2.0},
				{Price: 1.0, Amount: 3.0},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			merged := mergeRoundedOrderbookLevels(tc.levels)
			require.Len(t, merged, len(tc.exp), "Merged level count must match expected length")
			for i := range tc.exp {
				assert.Equal(t, tc.exp[i].Price, merged[i].Price, "Merged level price should match expected value")
				assert.InDelta(t, tc.exp[i].Amount, merged[i].Amount, 1e-12, "Merged level amount should match expected value")
			}
		})
	}
}

// fixtureTicker returns the ticker the fixtures give an asset's test pair, stored under exchangeName
func fixtureTicker(exchangeName string, a asset.Item) *ticker.Price {
	if a == asset.Futures {
		return &ticker.Price{
			Last:         83396.1,
			LastSize:     1,
			High:         84133.2,
			Low:          80985.9,
			Bid:          83396.1,
			BidSize:      3742,
			Ask:          83396.2,
			AskSize:      1,
			BaseVolume:   5016.614,
			QuoteVolume:  414990992.737,
			OpenInterest: 9510596,
			MarkPrice:    83403.06,
			IndexPrice:   83437.84,
			Pair:         currency.NewPairWithDelimiter("XBT", "USDTM", "_"),
			ExchangeName: exchangeName,
			AssetType:    a,
			LastUpdated:  time.UnixMilli(1791898156711),
		}
	}
	return &ticker.Price{
		Last:         83493.9,
		LastSize:     0.00001941,
		High:         84149,
		Low:          81035.5,
		Bid:          83493.8,
		BidSize:      0.33258019,
		Ask:          83493.9,
		AskSize:      0.00549083,
		BaseVolume:   6187.000634905228,
		QuoteVolume:  512747165.9412732,
		Pair:         currency.NewPairWithDelimiter("BTC", "USDT", "-"),
		ExchangeName: exchangeName,
		AssetType:    a,
		LastUpdated:  time.UnixMilli(1791898249918),
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must reject an asset KuCoin does not trade")
	require.ErrorIs(t, ex.UpdateTickers(cancelledContext(t), asset.Futures), context.Canceled, "UpdateTickers must return the failed futures request's error")

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/allTickers": `{"code":"200000","data":[
			{"symbol":"XBTUSDTM","price":"83396.1","bestBidPrice":"83396.1","bestAskPrice":"83396.1"},
			{"symbol":"ETHUSDTM","price":"2509.61","bestBidPrice":"2509.35","bestAskPrice":"2509.36"}
		]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"XBT"}]}`,
	})
	enabled := currency.Pairs{currency.NewPairWithDelimiter("XBT", "USDTM", "_"), currency.NewPairWithDelimiter("ETH", "USDTM", "_")}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, enabled, false), "StorePairs must not error for available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, enabled, true), "StorePairs must not error for enabled pairs")
	err := ex.UpdateTickers(t.Context(), asset.Futures)
	require.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "UpdateTickers must report a ticker the store rejects")
	require.ErrorIs(t, err, errFuturesContractNotFound, "UpdateTickers must report a pair whose contract is missing")
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/market/allTickers": `{"code":"200000","data":{"ticker":[{"symbol":"BTC-USDT","buy":"83493.9","sell":"83493.9"}]}}`})
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Spot), ticker.ErrBidEqualsAsk, "UpdateTickers must report a spot ticker the store rejects")
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/market/allTickers": `{"code":"200000","data":{"ticker":[{"symbol":""}]}}`})
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Spot), currency.ErrSymbolStringEmpty, "UpdateTickers must reject a spot ticker without a symbol")

	ensureTradablePairs(t)
	ex = testInstance(t)
	ex.Name = t.Name()
	exp := map[asset.Item][]*ticker.Price{
		asset.Spot: {
			fixtureTicker(ex.Name, asset.Spot),
			{
				Last:         2511.6,
				LastSize:     0.014,
				High:         2544.63,
				Low:          2409.42,
				Bid:          2512.16,
				BidSize:      1.3236729,
				Ask:          2512.17,
				AskSize:      0.0819805,
				BaseVolume:   37647.42377968,
				QuoteVolume:  93872638.19838758,
				Pair:         currency.NewPairWithDelimiter("ETH", "USDT", "-"),
				ExchangeName: ex.Name,
				AssetType:    asset.Spot,
				LastUpdated:  time.UnixMilli(1791898249918),
			},
		},
		asset.Margin:  {fixtureTicker(ex.Name, asset.Margin)},
		asset.Futures: {fixtureTicker(ex.Name, asset.Futures)},
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		if mockTests {
			for _, want := range exp[a] {
				tick, err := ticker.GetTicker(ex.Name, want.Pair, a)
				require.NoErrorf(t, err, "GetTicker must not error for %s %s", a, want.Pair)
				assert.Equalf(t, want, tick, "UpdateTickers should store the %s %s ticker", a, want.Pair)
			}
			continue
		}
		tick, err := ticker.GetTicker(ex.Name, assertToTradablePairMap[a], a)
		require.NoErrorf(t, err, "GetTicker must not error for %s", a)
		assert.Positivef(t, tick.Last, "UpdateTickers should store a %s last price", a)
		assert.Positivef(t, tick.High, "UpdateTickers should store a %s high", a)
		assert.Positivef(t, tick.Bid, "UpdateTickers should store a %s bid", a)
	}
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must reject an empty pair")
	_, err = ex.UpdateTicker(cancelledContext(t), currency.NewBTCUSDT(), asset.Spot)
	require.ErrorIs(t, err, context.Canceled, "UpdateTicker must return the failed request's error")

	ensureTradablePairs(t)
	ex.Name = t.Name()
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		result, err := ex.UpdateTicker(t.Context(), assertToTradablePairMap[a], a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", a)
		if mockTests {
			assert.Equalf(t, fixtureTicker(ex.Name, a), result, "UpdateTicker should return the stored %s ticker", a)
			continue
		}
		assert.Positivef(t, result.Last, "UpdateTicker should return a %s last price", a)
	}
}

// spotHourlyCandles are the fixtures' BTC-USDT hourly candles from 2026-10-01 00:00 to 03:00 UTC
var spotHourlyCandles = []kline.Candle{
	{Time: time.Unix(1790812800, 0), Open: 83609.9, High: 83639.1, Low: 83437.3, Close: 83514.4, Volume: 58.26389404},
	{Time: time.Unix(1790816400, 0), Open: 83514.3, High: 83680.6, Low: 83491.7, Close: 83520.4, Volume: 48.62890716},
	{Time: time.Unix(1790820000, 0), Open: 83520.5, High: 83653.8, Low: 83417.3, Close: 83498.3, Volume: 46.08686037},
}

// futuresHourlyCandles are the fixtures' XBTUSDTM hourly candles from 2026-10-01 00:00 to 04:00 UTC; volumes are in lots
var futuresHourlyCandles = []kline.Candle{
	{Time: time.Unix(1790812800, 0), Open: 83577.8, High: 83612.8, Low: 83382.8, Close: 83480, Volume: 41880},
	{Time: time.Unix(1790816400, 0), Open: 83460.1, High: 83656.2, Low: 83450.1, Close: 83471.9, Volume: 40994},
	{Time: time.Unix(1790820000, 0), Open: 83471.9, High: 83627.3, Low: 83374.6, Close: 83467.3, Volume: 39259},
	{Time: time.Unix(1790823600, 0), Open: 83450.6, High: 83747.8, Low: 83387.2, Close: 83718.3, Volume: 39235},
}

func TestFuturesCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	_, err := e.futuresCandles(t.Context(), "XBTUSDTM", kline.OneHour, end, start, 2)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresCandles must reject a window that ends before it starts")
	_, err = e.futuresCandles(cancelledContext(t), "XBTUSDTM", kline.OneHour, start, end, 2)
	require.ErrorIs(t, err, context.Canceled, "futuresCandles must return a failed request's error")

	result, err := e.futuresCandles(t.Context(), "XBTUSDTM", kline.OneHour, start, end, 2)
	require.NoError(t, err, "futuresCandles must not error")
	if mockTests {
		// Two requests of up to 2 candles each; KuCoin includes the candle at the end of a window, so both return 02:00
		exp := append(slices.Clone(futuresHourlyCandles[:3]), futuresHourlyCandles[2:]...)
		assert.Equal(t, exp, result, "futuresCandles should return each request's candles")
		return
	}
	assert.GreaterOrEqual(t, len(result), 3, "futuresCandles should return a candle for each hour")
}

func TestGetCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	_, err := e.getCandles(t.Context(), "BTC-USDT", asset.Options, kline.OneHour, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "getCandles must reject an asset KuCoin does not trade")
	_, err = e.getCandles(t.Context(), "BTC-USDT", asset.Spot, kline.TenMin, start, end)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "getCandles must reject an interval spot candles lack")
	_, err = e.getCandles(t.Context(), "XBTUSDTM", asset.Futures, kline.SixHour, start, end)
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "getCandles must reject the 6 hour interval futures candles lack")
	_, err = e.getCandles(cancelledContext(t), "BTC-USDT", asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, context.Canceled, "getCandles must return the failed spot request's error")

	for _, tc := range []struct {
		asset  asset.Item
		symbol string
		exp    []kline.Candle
	}{
		{asset: asset.Spot, symbol: "BTC-USDT", exp: []kline.Candle{spotHourlyCandles[2], spotHourlyCandles[1], spotHourlyCandles[0]}},
		{asset: asset.Futures, symbol: "XBTUSDTM", exp: futuresHourlyCandles},
	} {
		result, err := e.getCandles(t.Context(), tc.symbol, tc.asset, kline.OneHour, start, end)
		require.NoErrorf(t, err, "getCandles must not error for %s", tc.asset)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "getCandles should return the %s candles in KuCoin's order", tc.asset)
			continue
		}
		assert.NotEmptyf(t, result, "getCandles should return %s candles", tc.asset)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	_, err := e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricCandles must reject an empty pair")
	ensureTradablePairs(t)
	_, err = e.GetHistoricCandles(cancelledContext(t), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, context.Canceled, "GetHistoricCandles must return the failed request's error")
	_, err = e.GetHistoricCandles(t.Context(), futuresTradablePair, asset.Futures, kline.SixHour, start, start.Add(12*time.Hour))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetHistoricCandles must not request futures candles at the 6 hour granularity they lack")

	spotPair := currency.NewPairWithDelimiter("BTC", "USDT", "-")
	futuresPair := currency.NewPair(currency.XBT, currency.USDTM)
	for _, tc := range []struct {
		asset    asset.Item
		interval kline.Interval
		exp      *kline.Item
	}{
		{asset: asset.Spot, interval: kline.OneHour, exp: &kline.Item{Exchange: e.Name, Pair: spotPair, Asset: asset.Spot, Interval: kline.OneHour, Candles: spotHourlyCandles}},
		{asset: asset.Margin, interval: kline.OneHour, exp: &kline.Item{Exchange: e.Name, Pair: spotPair, Asset: asset.Margin, Interval: kline.OneHour, Candles: spotHourlyCandles}},
		{asset: asset.Futures, interval: kline.OneHour, exp: &kline.Item{Exchange: e.Name, Pair: futuresPair, Asset: asset.Futures, Interval: kline.OneHour, Candles: futuresHourlyCandles[:3]}},
		{
			// KuCoin has no 3 hour candles, so they are built from the hourly ones
			asset:    asset.Spot,
			interval: kline.ThreeHour,
			exp: &kline.Item{
				Exchange: e.Name, Pair: spotPair, Asset: asset.Spot, Interval: kline.ThreeHour,
				Candles: []kline.Candle{{Time: time.Unix(1790812800, 0), Open: 83609.9, High: 83680.6, Low: 83417.3, Close: 83498.3, Volume: 152.97966157}},
			},
		},
		{
			asset:    asset.Futures,
			interval: kline.ThreeHour,
			exp: &kline.Item{
				Exchange: e.Name, Pair: futuresPair, Asset: asset.Futures, Interval: kline.ThreeHour,
				Candles: []kline.Candle{{Time: time.Unix(1790812800, 0), Open: 83577.8, High: 83656.2, Low: 83374.6, Close: 83467.3, Volume: 122133}},
			},
		},
	} {
		result, err := e.GetHistoricCandles(t.Context(), assertToTradablePairMap[tc.asset], tc.asset, tc.interval, start, end)
		require.NoErrorf(t, err, "GetHistoricCandles must not error for %s %s", tc.asset, tc.interval)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetHistoricCandles should return the %s %s candles", tc.asset, tc.interval)
			continue
		}
		assert.NotEmptyf(t, result.Candles, "GetHistoricCandles should return %s %s candles", tc.asset, tc.interval)
	}
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(3 * time.Hour)
	_, err := e.GetHistoricCandlesExtended(t.Context(), currency.EMPTYPAIR, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricCandlesExtended must reject an empty pair")
	ensureTradablePairs(t)
	_, err = e.GetHistoricCandlesExtended(cancelledContext(t), spotTradablePair, asset.Spot, kline.OneHour, start, end)
	require.ErrorIs(t, err, context.Canceled, "GetHistoricCandlesExtended must return the failed request's error")
	_, err = e.GetHistoricCandlesExtended(t.Context(), futuresTradablePair, asset.Futures, kline.SixHour, start, start.Add(12*time.Hour))
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetHistoricCandlesExtended must not request futures candles at the 6 hour granularity they lack")

	exp := map[asset.Item]*kline.Item{
		asset.Spot:    {Exchange: e.Name, Pair: currency.NewPairWithDelimiter("BTC", "USDT", "-"), Asset: asset.Spot, Interval: kline.OneHour, Candles: spotHourlyCandles},
		asset.Margin:  {Exchange: e.Name, Pair: currency.NewPairWithDelimiter("BTC", "USDT", "-"), Asset: asset.Margin, Interval: kline.OneHour, Candles: spotHourlyCandles},
		asset.Futures: {Exchange: e.Name, Pair: currency.NewPair(currency.XBT, currency.USDTM), Asset: asset.Futures, Interval: kline.OneHour, Candles: futuresHourlyCandles[:3]},
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		result, err := e.GetHistoricCandlesExtended(t.Context(), assertToTradablePairMap[a], a, kline.OneHour, start, end)
		require.NoErrorf(t, err, "GetHistoricCandlesExtended must not error for %s", a)
		if mockTests {
			assert.Equalf(t, exp[a], result, "GetHistoricCandlesExtended should return the %s candles", a)
			continue
		}
		assert.NotEmptyf(t, result.Candles, "GetHistoricCandlesExtended should return %s candles", a)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetServerTime must reject an asset KuCoin does not trade")
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		result, err := e.GetServerTime(t.Context(), a)
		require.NoErrorf(t, err, "GetServerTime must not error for %s", a)
		if mockTests {
			assert.Equalf(t, time.UnixMilli(1791898075306), result, "GetServerTime should return the %s server time", a)
			continue
		}
		assert.WithinDurationf(t, time.Now(), result, time.Minute, "GetServerTime should return the current %s server time", a)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentTrades must reject an empty pair")
	requestPairs := map[asset.Item]currency.Pair{asset.Spot: currency.NewBTCUSDT(), asset.Futures: currency.NewPair(currency.XBT, currency.USDTM)}
	for a, p := range requestPairs {
		_, err = e.GetRecentTrades(cancelledContext(t), p, a)
		require.ErrorIsf(t, err, context.Canceled, "GetRecentTrades must return the failed %s request's error", a)
	}
	ex := testInstance(t)
	_, err = ex.GetRecentTrades(t.Context(), storeOptionsPair(t, ex), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an enabled asset KuCoin does not trade")
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/market/histories": `{"code":"200000","data":[{"side":"sideways"}]}`,
		"/api/v1/trade/history":    `{"code":"200000","data":[{"side":"sideways"}]}`,
	})
	for a, p := range requestPairs {
		_, err = ex.GetRecentTrades(t.Context(), p, a)
		require.ErrorIsf(t, err, order.ErrSideIsInvalid, "GetRecentTrades must reject a %s trade with an unknown side", a)
	}

	spotTrades := func(a asset.Item) []trade.Data {
		return []trade.Data{
			{TID: "24806547213402117", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "USDT", "-"), AssetType: a, Side: order.Buy, Price: 83490.5, Amount: 0.0015, Timestamp: time.UnixMilli(1791898248229)},
			{TID: "24806548332756992", Exchange: e.Name, CurrencyPair: currency.NewPairWithDelimiter("BTC", "USDT", "-"), AssetType: a, Side: order.Sell, Price: 83494.3, Amount: 0.00024787, Timestamp: time.UnixMilli(1791898253238)},
		}
	}
	futuresPair := currency.NewPair(currency.XBT, currency.USDTM)
	ensureTradablePairs(t)
	for _, tc := range []struct {
		asset asset.Item
		exp   []trade.Data
	}{
		{asset: asset.Spot, exp: spotTrades(asset.Spot)},
		{asset: asset.Margin, exp: spotTrades(asset.Margin)},
		{
			// Futures amounts are in lots
			asset: asset.Futures,
			exp: []trade.Data{
				{TID: "1947309219236", Exchange: e.Name, CurrencyPair: futuresPair, AssetType: asset.Futures, Side: order.Sell, Price: 83396.1, Amount: 1, Timestamp: time.UnixMilli(1791898156711)},
				{TID: "1947309222207", Exchange: e.Name, CurrencyPair: futuresPair, AssetType: asset.Futures, Side: order.Buy, Price: 83396.2, Amount: 1, Timestamp: time.UnixMilli(1791898168284)},
				{TID: "1947309222386", Exchange: e.Name, CurrencyPair: futuresPair, AssetType: asset.Futures, Side: order.Buy, Price: 83400, Amount: 113, Timestamp: time.UnixMilli(1791898168297)},
			},
		},
	} {
		t.Run(tc.asset.String(), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRecentTrades(t.Context(), assertToTradablePairMap[tc.asset], tc.asset)
			require.NoError(t, err, "GetRecentTrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRecentTrades should return each trade with its own side, oldest first")
				return
			}
			require.NotEmpty(t, result, "GetRecentTrades must return trades")
			assert.Positive(t, result[0].Price, "GetRecentTrades should return a price")
		})
	}
}

// expectedOrders returns, by name, the orders the order listing tests expect from the recorded fixtures
func expectedOrders() map[string]order.Detail {
	btcusdt := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter)
	ethusdt := currency.NewPairWithDelimiter(currency.ETH.String(), currency.USDT.String(), currency.DashDelimiter)
	xbtusdtm := currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter)
	// KuCoin's Get Open Orders example's filled funds over its filled size, divided as the wrapper does
	cost, executed := 0.5, 0.00001
	return map[string]order.Detail{
		"futures open": {
			Exchange: e.Name, OrderID: "235919172150824990", ClientOrderID: "gct-futures-open", Pair: xbtusdtm,
			AssetType: asset.Futures, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy, Status: order.Open,
			Price: 65000, Amount: 3, ContractAmount: 3, RemainingAmount: 3, CostAsset: currency.USDT,
			SettlementCurrency: currency.USDT, Leverage: 3, TimeInForce: order.GoodTillCancel,
			Date: time.UnixMilli(1729236185949), LastUpdated: time.UnixMilli(1729236190949),
		},
		"futures stop filled": {
			Exchange: e.Name, OrderID: "235919172150825000", ClientOrderID: "gct-futures-done", Pair: xbtusdtm,
			AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
			Status: order.Filled, TriggerPrice: 60000, Amount: 2, ContractAmount: 2, ExecutedAmount: 2,
			AverageExecutedPrice: 60000, Cost: 120, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
			Leverage: 3, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729236185949),
			LastUpdated: time.UnixMilli(1729236190949), CloseTime: time.UnixMilli(1729236200000),
		},
		"spot filled": {
			Exchange: e.Name, OrderID: "6717422bd51c29000775ea03", ClientOrderID: "gct-filled", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Status: order.Filled, Price: 70000, Amount: 0.5,
			QuoteAmount: 35000, ExecutedAmount: 0.5, AverageExecutedPrice: 70000, Cost: 35000, CostAsset: currency.USDT,
			Fee: 35, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729577515444),
			LastUpdated: time.UnixMilli(1729577515481), CloseTime: time.UnixMilli(1729577515481),
		},
		"spot partially filled cancelled": {
			Exchange: e.Name, OrderID: "6717422bd51c29000775ea04", ClientOrderID: "gct-partially-filled", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Status: order.PartiallyFilledCancelled,
			Amount: 0.25, ExecutedAmount: 0.125, AverageExecutedPrice: 70000, Cost: 8750, CostAsset: currency.USDT,
			Fee: 8.75, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729577516444),
			LastUpdated: time.UnixMilli(1729577516481), CloseTime: time.UnixMilli(1729577516481),
		},
		"margin cancelled": {
			Exchange: e.Name, OrderID: "671667306afcdb0007231092", ClientOrderID: "gct-margin-cancelled", Pair: btcusdt,
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Sell,
			Status: order.Cancelled, Price: 70000, Amount: 0.001, QuoteAmount: 70, CostAsset: currency.USDT,
			FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729521456248),
			LastUpdated: time.UnixMilli(1729651011877), CloseTime: time.UnixMilli(1729651011877),
		},
		"spot OCO done": {
			Exchange: e.Name, OrderID: "674c3cfa72cf2800072ee7d1", ClientOrderID: "gct-oco-done", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.OCO, Status: order.Closed, Date: time.UnixMilli(1733049600000),
			LastUpdated: time.UnixMilli(1733049600000),
		},
		"spot OCO cancelled": {
			Exchange: e.Name, OrderID: "674c3cfa72cf2800072ee7d2", ClientOrderID: "gct-oco-cancelled", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.OCO, Status: order.Cancelled, Date: time.UnixMilli(1733049700000),
			LastUpdated: time.UnixMilli(1733049700000),
		},
		"margin OCO triggered": {
			Exchange: e.Name, OrderID: "674c3cfa72cf2800072ee7d5", ClientOrderID: "gct-margin-oco-triggered",
			Pair: ethusdt, AssetType: asset.Margin, Type: order.OCO, Status: order.Closed,
			Date: time.UnixMilli(1733049800000), LastUpdated: time.UnixMilli(1733049800000),
		},
		"margin OCO cancelled": {
			Exchange: e.Name, OrderID: "674c3cfa72cf2800072ee7d6", ClientOrderID: "gct-margin-oco-cancelled",
			Pair: btcusdt, AssetType: asset.Margin, Type: order.OCO, Status: order.Cancelled,
			Date: time.UnixMilli(1733049900000), LastUpdated: time.UnixMilli(1733049900000),
		},
		"spot stop limit": {
			Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k61", ClientOrderID: "gct-spot-stop-limit", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.StopLimit, Side: order.Buy, Status: order.New, Price: 61000,
			TriggerPrice: 60000, Amount: 0.001, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1740626554884),
			LastUpdated: time.UnixMilli(1740626554883),
		},
		"spot stop market": {
			Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k63", ClientOrderID: "gct-spot-stop-market", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.StopMarket, Side: order.Sell, Status: order.New, TriggerPrice: 40000,
			Amount: 0.001, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1740626554884),
			LastUpdated: time.UnixMilli(1740626554883),
		},
		"spot OCO open": {
			Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k64", ClientOrderID: "gct-spot-oco", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.OCO, Side: order.Sell, Status: order.New, Price: 70000,
			TriggerPrice: 50000, Amount: 0.001, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1740626554884),
			LastUpdated: time.UnixMilli(1740626554883),
		},
		"margin stop market": {
			Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k65", ClientOrderID: "gct-margin-stop-market", Pair: btcusdt,
			AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
			Status: order.New, TriggerPrice: 40000, Amount: 0.001, TimeInForce: order.GoodTillCancel,
			Date: time.UnixMilli(1740626554884), LastUpdated: time.UnixMilli(1740626554883),
		},
		"margin OCO open": {
			Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k66", ClientOrderID: "gct-margin-oco", Pair: btcusdt,
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.OCO, Side: order.Buy, Status: order.New,
			Price: 50000, TriggerPrice: 70000, Amount: 0.001, TimeInForce: order.GoodTillCancel,
			Date: time.UnixMilli(1740626554884), LastUpdated: time.UnixMilli(1740626554883),
		},
		"spot ETH buy": {
			Exchange: e.Name, OrderID: "67120bbef094e200070976f7", ClientOrderID: "gct-eth-buy", Pair: ethusdt,
			AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Status: order.Open, Price: 3000, Amount: 0.1,
			QuoteAmount: 300, RemainingAmount: 0.1, CostAsset: currency.USDT, FeeAsset: currency.USDT,
			TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729235902748), LastUpdated: time.UnixMilli(1729235909862),
		},
		"spot ETH sell": {
			Exchange: e.Name, OrderID: "67120bbef094e200070976f8", ClientOrderID: "gct-eth-sell", Pair: ethusdt,
			AssetType: asset.Spot, Type: order.Limit, Side: order.Sell, Status: order.Open, Price: 4000, Amount: 0.1,
			QuoteAmount: 400, RemainingAmount: 0.1, CostAsset: currency.USDT, FeeAsset: currency.USDT,
			TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729235902748), LastUpdated: time.UnixMilli(1729235909862),
		},
		"spot BTC partially filled": {
			Exchange: e.Name, OrderID: "67120bbef094e200070976f6", ClientOrderID: "5c52e11203aa677f33e493fb", Pair: btcusdt,
			AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Price: 50000,
			Amount: 0.00003, QuoteAmount: 1.5, ExecutedAmount: 0.00001, RemainingAmount: 0.00001,
			AverageExecutedPrice: cost / executed, Cost: 0.5, CostAsset: currency.USDT, Fee: 0.0005, FeeAsset: currency.USDT,
			TimeInForce: order.GoodTillTime | order.PostOnly, HiddenOrder: true, Date: time.UnixMilli(1729235902748),
			LastUpdated: time.UnixMilli(1729235909862),
		},
		"margin BTC partially filled": {
			Exchange: e.Name, OrderID: "671667306afcdb000723107f", ClientOrderID: "5c52e11203aa677f33e493fb", Pair: btcusdt,
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy,
			Status: order.PartiallyFilled, Price: 50000, Amount: 0.00002, QuoteAmount: 1, ExecutedAmount: 0.00001,
			RemainingAmount: 0.00001, AverageExecutedPrice: cost / executed, Cost: 0.5, CostAsset: currency.USDT,
			Fee: 0.0005, FeeAsset: currency.USDT, TimeInForce: order.GoodTillTime | order.PostOnly,
			Date: time.UnixMilli(1729521456248), LastUpdated: time.UnixMilli(1729521460940),
		},
		"margin BTC stop": {
			Exchange: e.Name, OrderID: "671667306afcdb0007231080", ClientOrderID: "5c52e11203aa677f33e493fc", Pair: btcusdt,
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.StopLimit, Side: order.Sell, Status: order.Open,
			Price: 70000, TriggerPrice: 69000, Amount: 0.0001, QuoteAmount: 7, RemainingAmount: 0.00005,
			CostAsset: currency.USDT, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, HiddenOrder: true,
			Date: time.UnixMilli(1729521500000), LastUpdated: time.UnixMilli(1729521600000),
		},
		"margin ETH open": {
			Exchange: e.Name, OrderID: "671667306afcdb0007231090", ClientOrderID: "gct-margin-eth", Pair: ethusdt,
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy, Status: order.Open,
			Price: 3000, Amount: 0.1, QuoteAmount: 300, RemainingAmount: 0.1, CostAsset: currency.USDT,
			FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729235902748),
			LastUpdated: time.UnixMilli(1729235909862),
		},
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an asset KuCoin does not trade")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Side: order.ClosePosition, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetOrderHistory must reject a side KuCoin does not list")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.OCO})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetOrderHistory must reject a futures order type KuCoin does not list")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{btcusdt}, Side: order.AnySide, Type: order.StopLimit})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetOrderHistory must reject stop orders, which KuCoin keeps no history of")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetOrderHistory must require pairs for spot orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		for a, p := range assertToTradablePairMap {
			_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{p}, Side: order.AnySide, Type: order.AnyType})
			assert.NoErrorf(t, err, "GetOrderHistory should not error for %s", a)
		}
		return
	}

	orders := expectedOrders()
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{
			name: "futures",
			req:  &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{currency.NewPair(currency.XBT, currency.USDTM)}, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders["futures stop filled"]},
		},
		{
			// An OCO order that is still open is not history
			name: "spot OCO",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Side: order.AnySide, Type: order.OCO},
			exp:  order.FilteredOrders{orders["spot OCO done"], orders["spot OCO cancelled"]},
		},
		{
			name: "spot",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Side: order.AnySide, Type: order.AnyType},
			exp:  order.FilteredOrders{orders["spot filled"], orders["spot partially filled cancelled"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetOrderHistory must not error")
			assert.Equal(t, tc.exp, got, "GetOrderHistory should return the closed orders")
		})
	}
}

func TestClosedHFOrders(t *testing.T) {
	t.Parallel()
	spotFormat := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	btcusdt := currency.NewBTCUSDT()
	_, err := e.closedHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.AnyType}, spotFormat, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "closedHFOrders must require pairs")
	_, err = e.closedHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Type: order.TrailingStop}, spotFormat, "")
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "closedHFOrders must reject an order type KuCoin does not list")

	if !mockTests {
		t.Skip("closedHFOrders' replies come from fixtures; TestGetOrderHistory lists live orders")
	}

	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err = e.closedHFOrders(cancelledContext(t), &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{btcusdt}, Type: order.AnyType}, spotFormat, "")
		require.ErrorIsf(t, err, context.Canceled, "closedHFOrders must return the %s request error", a)
	}
	ethusdt := currency.NewPair(currency.ETH, currency.USDT)
	_, err = e.closedHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{ethusdt}, Type: order.AnyType}, spotFormat, "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "closedHFOrders must reject a spot order of an unknown side")
	_, err = e.closedHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{ethusdt}, Type: order.AnyType}, spotFormat, "")
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "closedHFOrders must reject a margin order of an unknown type")
	orders := expectedOrders()
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		side string
		exp  []order.Detail
	}{
		{
			name: "spot",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Type: order.AnyType},
			exp:  []order.Detail{orders["spot filled"], orders["spot partially filled cancelled"]},
		},
		{
			name: "cross margin sell limit",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{btcusdt}, Type: order.Limit},
			side: "sell",
			exp:  []order.Detail{orders["margin cancelled"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.closedHFOrders(t.Context(), tc.req, spotFormat, tc.side)
			require.NoError(t, err, "closedHFOrders must not error")
			assert.Equal(t, tc.exp, got, "closedHFOrders should return the closed orders")
		})
	}
}

func TestOCOOrderHistory(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("ocoOrderHistory's replies come from fixtures; TestGetOrderHistory lists live orders")
	}

	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err := e.ocoOrderHistory(cancelledContext(t), &order.MultiOrderRequest{AssetType: a, Type: order.OCO}, "")
		require.ErrorIsf(t, err, context.Canceled, "ocoOrderHistory must return the %s request error", a)
	}
	_, err := e.ocoOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.OCO}, "ETH-USDT")
	require.ErrorIs(t, err, order.ErrUnsupportedStatusType, "ocoOrderHistory must reject an OCO order of an unknown status")
	orders := expectedOrders()
	for _, tc := range []struct {
		name   string
		req    *order.MultiOrderRequest
		symbol string
		exp    []order.Detail
	}{
		{
			name:   "spot",
			req:    &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.OCO},
			symbol: "BTC-USDT",
			exp:    []order.Detail{orders["spot OCO done"], orders["spot OCO cancelled"]},
		},
		{
			name: "cross margin",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, Type: order.OCO},
			exp:  []order.Detail{orders["margin OCO triggered"], orders["margin OCO cancelled"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.ocoOrderHistory(t.Context(), tc.req, tc.symbol)
			require.NoError(t, err, "ocoOrderHistory must not error")
			assert.Equal(t, tc.exp, got, "ocoOrderHistory should return the OCO orders that are no longer open")
		})
	}
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an asset KuCoin does not trade")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.ClosePosition, Type: order.AnyType})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetActiveOrders must reject a side KuCoin does not list")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.OCO})
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "GetActiveOrders must reject a futures order type KuCoin does not list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		for a, p := range assertToTradablePairMap {
			_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Pairs: currency.Pairs{p}, Side: order.AnySide, Type: order.AnyType})
			assert.NoErrorf(t, err, "GetActiveOrders should not error for %s", a)
		}
		return
	}

	orders := expectedOrders()
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  order.FilteredOrders
	}{
		{
			// A Long request lists buy orders
			name: "futures long limit",
			req:  &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{currency.NewPair(currency.XBT, currency.USDTM)}, Side: order.Long, Type: order.Limit},
			exp:  order.FilteredOrders{orders["futures open"]},
		},
		{
			// KuCoin lists stop limit and stop market orders together, so the stop market order is filtered out
			name: "spot stop limit",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}, Side: order.AnySide, Type: order.StopLimit},
			exp:  order.FilteredOrders{orders["spot stop limit"]},
		},
		{
			// KuCoin lists a symbol's open orders of both sides, so the sell order is filtered out
			name: "spot buy",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT)}, Side: order.Bid, Type: order.AnyType},
			exp:  order.FilteredOrders{orders["spot ETH buy"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetActiveOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetActiveOrders must not error")
			assert.Equal(t, tc.exp, got, "GetActiveOrders should return the open orders the request matches")
		})
	}
}

func TestCollectPages(t *testing.T) {
	t.Parallel()
	errFetch := errors.New("fetch failed")
	_, err := collectPages(func(uint64) ([]int, uint64, error) { return nil, 0, errFetch })
	require.ErrorIs(t, err, errFetch, "collectPages must return the fetch error")

	var pages []uint64
	got, err := collectPages(func(page uint64) ([]uint64, uint64, error) {
		pages = append(pages, page)
		return []uint64{page * 10, page*10 + 1}, 3, nil
	})
	require.NoError(t, err, "collectPages must not error")
	assert.Equal(t, []uint64{10, 11, 20, 21, 30, 31}, got, "collectPages should gather every page's items")
	assert.Equal(t, []uint64{1, 2, 3}, pages, "collectPages should fetch each page once")

	got, err = collectPages(func(uint64) ([]uint64, uint64, error) { return nil, 0, nil })
	require.NoError(t, err, "collectPages must not error for a list without pages")
	assert.Empty(t, got, "collectPages should return no items for a list without pages")
}

func TestCollectByLastID(t *testing.T) {
	t.Parallel()
	errFetch := errors.New("fetch failed")
	_, err := collectByLastID(func(uint64) ([]int, uint64, error) { return nil, 0, errFetch })
	require.ErrorIs(t, err, errFetch, "collectByLastID must return the fetch error")

	var lastIDs []uint64
	got, err := collectByLastID(func(lastID uint64) ([]uint64, uint64, error) {
		lastIDs = append(lastIDs, lastID)
		if lastID == 200 {
			return []uint64{lastID}, 300, nil
		}
		return make([]uint64, closedOrdersLimit), lastID + 100, nil
	})
	require.NoError(t, err, "collectByLastID must not error")
	assert.Len(t, got, 2*closedOrdersLimit+1, "collectByLastID should gather full pages until a short one")
	assert.Equal(t, []uint64{0, 100, 200}, lastIDs, "collectByLastID should page back from each page's last ID")

	lastIDs = nil
	got, err = collectByLastID(func(lastID uint64) ([]uint64, uint64, error) {
		lastIDs = append(lastIDs, lastID)
		return make([]uint64, closedOrdersLimit), 0, nil
	})
	require.NoError(t, err, "collectByLastID must not error when the last ID stops changing")
	assert.Len(t, got, closedOrdersLimit, "collectByLastID should stop when the last ID stops changing")
	assert.Equal(t, []uint64{0}, lastIDs, "collectByLastID should not fetch the same page again")
}

func TestAllFuturesOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		_, err := e.allFuturesOrders(t.Context(), &GetFuturesOrdersRequest{Status: "active", Symbol: futuresTradablePair.String()})
		assert.NoError(t, err, "allFuturesOrders should not error")
		return
	}

	_, err := e.allFuturesOrders(cancelledContext(t), &GetFuturesOrdersRequest{Status: "done", Symbol: "XBTUSDM"})
	require.ErrorIs(t, err, context.Canceled, "allFuturesOrders must return the request error")
	got, err := e.allFuturesOrders(t.Context(), &GetFuturesOrdersRequest{Status: "active", Symbol: "XBTUSDM"})
	require.NoError(t, err, "allFuturesOrders must not error for a reply without data")
	assert.Empty(t, got, "allFuturesOrders should return no orders for a reply without data")
	got, err = e.allFuturesOrders(t.Context(), &GetFuturesOrdersRequest{Status: "done", Symbol: "XBTUSDM"})
	require.NoError(t, err, "allFuturesOrders must not error")
	exp := []FuturesOrder{
		{
			ID: "235919172150825002", Symbol: "XBTUSDM", Type: "limit", Side: "buy", Price: 65000, Size: 3, Value: 195,
			TimeInForce: "GTC", Leverage: 3, ClientOrderID: "gct-page-one", CreatedAt: types.Time(time.UnixMilli(1729236185949)),
			UpdatedAt: types.Time(time.UnixMilli(1729236190949)), OrderTime: types.Time(time.UnixMilli(1729236185949)),
			SettleCurrency: currency.USDT, MarginMode: "ISOLATED", Status: "done", PositionSide: "BOTH",
		},
		{
			ID: "235919172150825003", Symbol: "XBTUSDM", Type: "limit", Side: "sell", Price: 65000, Size: 3, Value: 195,
			TimeInForce: "GTC", Leverage: 3, ClientOrderID: "gct-page-two", CreatedAt: types.Time(time.UnixMilli(1729236185949)),
			UpdatedAt: types.Time(time.UnixMilli(1729236190949)), OrderTime: types.Time(time.UnixMilli(1729236185949)),
			SettleCurrency: currency.USDT, MarginMode: "ISOLATED", Status: "done", PositionSide: "BOTH",
		},
	}
	assert.Equal(t, exp, got, "allFuturesOrders should return the orders of every page")
}

func TestFuturesOrders(t *testing.T) {
	t.Parallel()
	_, err := e.futuresOrders(t.Context(), &order.MultiOrderRequest{Type: order.OCO}, "active", "", "")
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "futuresOrders must reject an order type KuCoin does not list")

	if !mockTests {
		t.Skip("futuresOrders' replies come from fixtures; TestGetActiveOrders lists live orders")
	}

	_, err = e.futuresOrders(cancelledContext(t), &order.MultiOrderRequest{Type: order.AnyType}, "active", "", "")
	require.ErrorIs(t, err, context.Canceled, "futuresOrders must return the request error")
	_, err = e.futuresOrders(t.Context(), &order.MultiOrderRequest{Type: order.AnyType}, "done", "ETHUSDTM", "")
	require.ErrorIs(t, err, currency.ErrPairNotFound, "futuresOrders must reject an order of an unlisted contract")
	got, err := e.futuresOrders(t.Context(), &order.MultiOrderRequest{Type: order.Limit}, "active", "XBTUSDTM", "buy")
	require.NoError(t, err, "futuresOrders must not error")
	assert.Equal(t, []order.Detail{expectedOrders()["futures open"]}, got, "futuresOrders should return the orders")
}

func TestOpenHFOrders(t *testing.T) {
	t.Parallel()
	spotFormat := currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	if !mockTests {
		t.Skip("openHFOrders' replies come from fixtures; TestGetActiveOrders lists live orders")
	}
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
	}{
		{name: "the symbols with open orders", req: &order.MultiOrderRequest{AssetType: asset.Spot}},
		{name: "the symbols with open orders for several pairs", req: &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}}},
		{name: "a spot pair's orders", req: &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT()}}},
		{name: "a margin pair's orders", req: &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{currency.NewBTCUSDT()}}},
	} {
		_, err := e.openHFOrders(cancelledContext(t), tc.req, spotFormat)
		require.ErrorIsf(t, err, context.Canceled, "openHFOrders must return the request error listing %s", tc.name)
	}
	xrpusdt := currency.NewPair(currency.XRP, currency.USDT)
	_, err := e.openHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{xrpusdt}}, spotFormat)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "openHFOrders must reject a spot order of an unknown side")
	_, err = e.openHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{xrpusdt}}, spotFormat)
	require.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "openHFOrders must reject a margin order of an unknown type")
	orders := expectedOrders()
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  []order.Detail
	}{
		{
			name: "every spot symbol",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot},
			exp:  []order.Detail{orders["spot ETH buy"], orders["spot ETH sell"], orders["spot BTC partially filled"]},
		},
		{
			name: "every cross margin symbol",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin},
			exp:  []order.Detail{orders["margin BTC partially filled"], orders["margin BTC stop"], orders["margin ETH open"]},
		},
		{
			// XRP-USDT has no open orders, so its orders, which would not convert, are not requested
			name: "the spot pairs with open orders",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.NewBTCUSDT(), xrpusdt}},
			exp:  []order.Detail{orders["spot BTC partially filled"]},
		},
		{
			name: "the cross margin pairs with open orders",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDT), xrpusdt}},
			exp:  []order.Detail{orders["margin ETH open"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.openHFOrders(t.Context(), tc.req, spotFormat)
			require.NoError(t, err, "openHFOrders must not error")
			assert.Equal(t, tc.exp, got, "openHFOrders should return the open orders of each symbol")
		})
	}
}

func TestUntriggeredStopOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("untriggeredStopOrders' replies come from fixtures; TestGetActiveOrders lists live orders")
	}

	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err := e.untriggeredStopOrders(cancelledContext(t), &order.MultiOrderRequest{AssetType: a, Type: order.Stop}, "", "")
		require.ErrorIsf(t, err, context.Canceled, "untriggeredStopOrders must return the %s request error", a)
	}
	_, err := e.untriggeredStopOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.Stop}, "ETH-USDT", "")
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "untriggeredStopOrders must reject a stop order of an unknown side")
	orders := expectedOrders()
	for _, tc := range []struct {
		name   string
		req    *order.MultiOrderRequest
		symbol string
		side   string
		exp    []order.Detail
	}{
		{
			// The triggered stop order is listed with the open orders, so it is left out
			name:   "a symbol's spot stop orders",
			req:    &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.StopLimit},
			symbol: "BTC-USDT",
			exp:    []order.Detail{orders["spot stop limit"], orders["spot stop market"]},
		},
		{
			name: "spot OCO orders",
			req:  &order.MultiOrderRequest{AssetType: asset.Spot, Type: order.OCO},
			exp:  []order.Detail{orders["spot OCO open"]},
		},
		{
			name:   "a symbol's isolated margin stop sell orders",
			req:    &order.MultiOrderRequest{AssetType: asset.Margin, Type: order.Stop, MarginType: margin.Isolated},
			symbol: "BTC-USDT",
			side:   "sell",
			exp:    []order.Detail{orders["margin stop market"]},
		},
		{
			name: "cross margin OCO orders",
			req:  &order.MultiOrderRequest{AssetType: asset.Margin, Type: order.OCO},
			exp:  []order.Detail{orders["margin OCO open"]},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.untriggeredStopOrders(t.Context(), tc.req, tc.symbol, tc.side)
			require.NoError(t, err, "untriggeredStopOrders must not error")
			assert.Equal(t, tc.exp, got, "untriggeredStopOrders should return the untriggered orders")
		})
	}
}

func TestFilterOrders(t *testing.T) {
	t.Parallel()
	orders := []order.Detail{
		{OrderID: "1", Side: order.Buy, Type: order.Limit},
		{OrderID: "2", Side: order.Sell, Type: order.Limit},
		{OrderID: "3", Side: order.Buy, Type: order.StopLimit},
		{OrderID: "4", Side: order.Sell, Type: order.StopMarket},
	}
	for _, tc := range []struct {
		name string
		req  *order.MultiOrderRequest
		exp  []string
	}{
		{name: "a Long request", req: &order.MultiOrderRequest{Side: order.Long, Type: order.AnyType}, exp: []string{"1", "3"}},
		{name: "an Ask request", req: &order.MultiOrderRequest{Side: order.Ask, Type: order.AnyType}, exp: []string{"2", "4"}},
		{name: "a Stop request", req: &order.MultiOrderRequest{Side: order.AnySide, Type: order.Stop}, exp: []string{"1", "2", "3", "4"}},
		{name: "a ConditionalStop sell request", req: &order.MultiOrderRequest{Side: order.Sell, Type: order.ConditionalStop}, exp: []string{"2", "4"}},
		{name: "a StopLimit request", req: &order.MultiOrderRequest{Side: order.AnySide, Type: order.StopLimit}, exp: []string{"3"}},
	} {
		got := e.filterOrders(tc.req, orders)
		orderIDs := make([]string, len(got))
		for i := range got {
			orderIDs[i] = got[i].OrderID
		}
		assert.Equalf(t, tc.exp, orderIDs, "filterOrders should keep the orders %s matches", tc.name)
	}
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFeeByType must reject an empty pair")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.InternationalBankDepositFee, Pair: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, errFeeTypeNotSupported, "GetFeeByType must reject a fee type it cannot estimate")
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v1/trade-fees":         `{"code":"200000","data":[]}`,
		"/api/v1/withdrawals/quotas": `{"code":"200000","data":null}`,
		"/api/v1/base-fee":           `{"code":"200000","data":null}`,
	})
	for _, feeType := range []exchange.FeeType{exchange.CryptocurrencyTradeFee, exchange.CryptocurrencyWithdrawalFee, exchange.InternationalBankWithdrawalFee} {
		_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSDT(), FiatCurrency: currency.USD})
		require.ErrorIsf(t, err, common.ErrNoResponse, "GetFeeByType must reject a reply without fees for fee type %d", feeType)
	}
	for _, feeType := range []exchange.FeeType{exchange.CryptocurrencyTradeFee, exchange.CryptocurrencyWithdrawalFee, exchange.InternationalBankWithdrawalFee} {
		_, err = ex.GetFeeByType(cancelledContext(t), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSDT(), FiatCurrency: currency.USD})
		require.ErrorIsf(t, err, context.Canceled, "GetFeeByType must return the failed request's error for fee type %d", feeType)
	}

	ex.SetCredentials(&accounts.Credentials{})
	fee, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error without credentials")
	assert.Equal(t, 0.002, fee, "GetFeeByType should estimate the trade fee offline without credentials")
	fee, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyDepositFee, Pair: currency.NewBTCUSDT(), Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error for a deposit")
	assert.Zero(t, fee, "GetFeeByType should return no deposit fee")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name    string
		builder *exchange.FeeBuilder
		exp     float64
	}{
		// The pair has no delimiter, so the request proves the symbol is formatted
		{name: "taker", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), Amount: 2}, exp: 0.002},
		{name: "maker", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: currency.NewBTCUSDT(), Amount: 2, IsMaker: true}, exp: 0.0016},
		{name: "withdrawal", builder: &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyWithdrawalFee, Pair: currency.NewBTCUSDT(), Amount: 2}, exp: 0.0005},
		{name: "fiat taker", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankWithdrawalFee, Pair: currency.NewBTCUSDT(), FiatCurrency: currency.USD, Amount: 2}, exp: 0.002},
		{name: "fiat maker", builder: &exchange.FeeBuilder{FeeType: exchange.InternationalBankWithdrawalFee, Pair: currency.NewBTCUSDT(), FiatCurrency: currency.USD, Amount: 2, IsMaker: true}, exp: 0.0016},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			fee, err := e.GetFeeByType(t.Context(), tc.builder)
			require.NoError(t, err, "GetFeeByType must not error")
			if mockTests {
				assert.Equal(t, tc.exp, fee, "GetFeeByType should return the fee from the account's rates")
				return
			}
			assert.GreaterOrEqual(t, fee, 0.0, "GetFeeByType should not return a negative fee")
		})
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ValidateAPICredentials(t.Context(), asset.Options), asset.ErrNotSupported, "ValidateAPICredentials must reject an asset KuCoin does not trade")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		assert.NoErrorf(t, testInstance(t).ValidateAPICredentials(t.Context(), a), "ValidateAPICredentials should not error for %s", a)
	}
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains must reject an empty currency")
	_, err = e.GetAvailableTransferChains(cancelledContext(t), currency.BTC)
	require.ErrorIs(t, err, context.Canceled, "GetAvailableTransferChains must return the failed request's error")
	ex := marketTestServerExchange(t, map[string]string{"/api/v3/currencies/BTC": `{"code":"200000","data":null}`})
	_, err = ex.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.ErrorIs(t, err, common.ErrNoResponse, "GetAvailableTransferChains must reject a reply without the currency")

	result, err := e.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	if mockTests {
		assert.Equal(t, []string{"btc", "bsc", "btcln"}, result, "GetAvailableTransferChains should return the currency's chain IDs")
		return
	}
	assert.Contains(t, result, "btc", "GetAvailableTransferChains should return the bitcoin chain's ID")
}

func TestDepositHistory(t *testing.T) {
	t.Parallel()
	ex := marketTestServerExchange(t, map[string]string{"/api/v1/deposits": `{"code":"200000","data":null}`})
	_, err := ex.depositHistory(cancelledContext(t), currency.BTC)
	require.ErrorIs(t, err, context.Canceled, "depositHistory must return the failed request's error")
	_, err = ex.depositHistory(t.Context(), currency.BTC)
	require.ErrorIs(t, err, common.ErrNoResponse, "depositHistory must reject a reply without a page")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.depositHistory(t.Context(), currency.USDT)
	require.NoError(t, err, "depositHistory must not error")
	if mockTests {
		exp := []DepositRecord{
			{
				ID:            "69425929fc86870007ecd8b2",
				Currency:      currency.USDT,
				Chain:         "TRC20",
				Status:        "SUCCESS",
				SubStatus:     "SUCCESS",
				URL:           "https://tronscan.org/#/transaction/7e1c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d81",
				Address:       "T***",
				Amount:        25,
				Fee:           0.5,
				WalletTxID:    "7e1c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d81",
				CreatedAt:     types.Time(time.Unix(1765955881, 0)),
				UpdatedAt:     types.Time(time.Unix(1765955941, 0)),
				Remark:        "Deposit",
				NeedSubmitWHT: "false",
			},
			{
				ID:            "69425929fc86870007ecd8b5",
				Currency:      currency.USDT,
				Chain:         "TON",
				Status:        "FAILURE",
				SubStatus:     "ROLLBACK",
				URL:           "https://tonviewer.com/transaction/4b2c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fc",
				Address:       "UQ***",
				Memo:          "2085123456",
				IsInner:       true,
				Amount:        5,
				WalletTxID:    "4b2c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fc",
				CreatedAt:     types.Time(time.Unix(1765755881, 0)),
				UpdatedAt:     types.Time(time.Unix(1765755941, 0)),
				Remark:        "Deposit",
				Arrears:       true,
				NeedSubmitWHT: "false",
			},
		}
		assert.Equal(t, exp, result, "depositHistory should return every page's deposits")
	}
}

func TestWithdrawalHistory(t *testing.T) {
	t.Parallel()
	ex := marketTestServerExchange(t, map[string]string{"/api/v1/withdrawals": `{"code":"200000","data":null}`})
	_, err := ex.withdrawalHistory(cancelledContext(t), currency.BTC)
	require.ErrorIs(t, err, context.Canceled, "withdrawalHistory must return the failed request's error")
	_, err = ex.withdrawalHistory(t.Context(), currency.BTC)
	require.ErrorIs(t, err, common.ErrNoResponse, "withdrawalHistory must reject a reply without a page")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.withdrawalHistory(t.Context(), currency.USDT)
	require.NoError(t, err, "withdrawalHistory must not error")
	if mockTests {
		exp := []Withdrawal{
			{
				ID:         "670deec84d64da0007d7c948",
				Currency:   currency.USDT,
				Chain:      "trx",
				Status:     "SUCCESS",
				Address:    "T***",
				Amount:     10,
				Fee:        1,
				WalletTxID: "8f2c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d82",
				CreatedAt:  types.Time(time.Unix(1765954390, 0)),
				UpdatedAt:  types.Time(time.Unix(1765954990, 0)),
				Remark:     "Withdrawal",
			},
			{
				ID:                   "670deec84d64da0007d7c94b",
				Currency:             currency.USDT,
				Chain:                "ton",
				Status:               "FAILURE",
				Address:              "UQ***",
				Memo:                 "2085123456",
				IsInner:              true,
				Amount:               3,
				CreatedAt:            types.Time(time.Unix(1765754390, 0)),
				UpdatedAt:            types.Time(time.Unix(1765754990, 0)),
				Remark:               "Withdrawal",
				FailureReason:        "WALLET_FAILURE",
				FailureReasonMessage: "The wallet failed to process the withdrawal",
				Arrears:              true,
			},
		}
		assert.Equal(t, exp, result, "withdrawalHistory should return every page's withdrawals")
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := marketTestServerExchange(t, nil).GetAccountFundingHistory(cancelledContext(t))
	require.ErrorIs(t, err, context.Canceled, "GetAccountFundingHistory must return the failed account list request's error")
	accountList := `{"code":"200000","data":[{"currency":"BTC","type":"main"}]}`
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v1/accounts": accountList,
		"/api/v1/deposits": `{"code":"400100","msg":"Request failed"}`,
	})
	_, err = ex.GetAccountFundingHistory(t.Context())
	require.ErrorContains(t, err, "400100", "GetAccountFundingHistory must return the failed deposit history request's error")
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/accounts":    accountList,
		"/api/v1/deposits":    `{"code":"200000","data":{"currentPage":1,"totalPage":1,"items":[]}}`,
		"/api/v1/withdrawals": `{"code":"400100","msg":"Request failed"}`,
	})
	_, err = ex.GetAccountFundingHistory(t.Context())
	require.ErrorContains(t, err, "400100", "GetAccountFundingHistory must return the failed withdrawal history request's error")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if mockTests {
		exp := []exchange.FundingHistory{
			{
				ExchangeName:    e.Name,
				Status:          "SUCCESS",
				TransferID:      "69425929fc86870007ecd8b2",
				Description:     "Deposit",
				Timestamp:       time.Unix(1765955881, 0),
				Currency:        "USDT",
				Amount:          25,
				Fee:             0.5,
				TransferType:    "deposit",
				CryptoToAddress: "T***",
				CryptoTxID:      "7e1c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d81",
				CryptoChain:     "TRC20",
			},
			{
				ExchangeName:    e.Name,
				Status:          "FAILURE",
				TransferID:      "69425929fc86870007ecd8b5",
				Description:     "Deposit",
				Timestamp:       time.Unix(1765755881, 0),
				Currency:        "USDT",
				Amount:          5,
				TransferType:    "deposit",
				CryptoToAddress: "UQ***",
				CryptoTxID:      "4b2c3c6c9000d3c8292781a152d8cf603accb5a4f2f86d123f0ea45a393ae7fc",
				CryptoChain:     "TON",
			},
			{
				ExchangeName:    e.Name,
				Status:          "SUCCESS",
				TransferID:      "670deec84d64da0007d7c948",
				Description:     "Withdrawal",
				Timestamp:       time.Unix(1765954390, 0),
				Currency:        "USDT",
				Amount:          10,
				Fee:             1,
				TransferType:    "withdrawal",
				CryptoToAddress: "T***",
				CryptoTxID:      "8f2c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d82",
				CryptoChain:     "trx",
			},
			{
				ExchangeName:    e.Name,
				Status:          "FAILURE",
				TransferID:      "670deec84d64da0007d7c94b",
				Description:     "Withdrawal",
				Timestamp:       time.Unix(1765754390, 0),
				Currency:        "USDT",
				Amount:          3,
				TransferType:    "withdrawal",
				CryptoToAddress: "UQ***",
				CryptoChain:     "ton",
			},
			{
				ExchangeName:    e.Name,
				Status:          "SUCCESS",
				TransferID:      "69425929fc86870007ecd8b3",
				Description:     "Deposit",
				Timestamp:       time.Unix(1765855881, 0),
				Currency:        "BTC",
				Amount:          0.01,
				TransferType:    "deposit",
				CryptoToAddress: "bc1q***",
				CryptoTxID:      "5d2c7e1f0b6a3c8e9f4d1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f7",
				CryptoChain:     "BTC",
			},
			{
				ExchangeName:    e.Name,
				Status:          "SUCCESS",
				TransferID:      "670deec84d64da0007d7c949",
				Description:     "Withdrawal",
				Timestamp:       time.Unix(1765854390, 0),
				Currency:        "BTC",
				Amount:          0.002,
				Fee:             0.0001,
				TransferType:    "withdrawal",
				CryptoToAddress: "bc1q***",
				CryptoTxID:      "9a3d7e1f0b6a3c8e9f4d1a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e6f8",
				CryptoChain:     "btc",
			},
		}
		assert.Equal(t, exp, result, "GetAccountFundingHistory should return each listed currency's deposits and withdrawals")
	}
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetWithdrawalsHistory must reject an asset KuCoin does not trade")
	_, err = marketTestServerExchange(t, nil).GetWithdrawalsHistory(cancelledContext(t), currency.BTC, asset.Spot)
	require.ErrorIs(t, err, context.Canceled, "GetWithdrawalsHistory must return the failed request's error")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetWithdrawalsHistory(t.Context(), currency.USDT, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if mockTests {
		exp := []exchange.WithdrawalHistory{
			{
				Status:          "SUCCESS",
				TransferID:      "670deec84d64da0007d7c948",
				Description:     "Withdrawal",
				Timestamp:       time.Unix(1765954390, 0),
				Currency:        "USDT",
				Amount:          10,
				Fee:             1,
				TransferType:    "withdrawal",
				CryptoToAddress: "T***",
				CryptoTxID:      "8f2c64f2a4c6b0e1d6fb04a7f86d1d3ff8bba5fa9b6b0b2f2b23a1cb5c6a7d82",
				CryptoChain:     "trx",
			},
			{
				Status:          "FAILURE",
				TransferID:      "670deec84d64da0007d7c94b",
				Description:     "Withdrawal",
				Timestamp:       time.Unix(1765754390, 0),
				Currency:        "USDT",
				Amount:          3,
				TransferType:    "withdrawal",
				CryptoToAddress: "UQ***",
				CryptoChain:     "ton",
			},
		}
		assert.Equal(t, exp, result, "GetWithdrawalsHistory should return every withdrawal of the currency")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.GetOrderInfo(t.Context(), "", btcusdt, asset.Spot)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")
	_, err = e.GetOrderInfo(t.Context(), "68a3e81d2a090d00070575f7", btcusdt, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an asset KuCoin does not trade")
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err = e.GetOrderInfo(t.Context(), "68a3e81d2a090d00070575f7", currency.EMPTYPAIR, a)
		require.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "GetOrderInfo must require the pair of a %s order", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		_, err = e.GetOrderInfo(t.Context(), "6717422bd51c29000775ea99", spotTradablePair, asset.Spot)
		assert.Error(t, err, "GetOrderInfo should report an order KuCoin does not have")
		return
	}

	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		_, err = e.GetOrderInfo(cancelledContext(t), "68a3e81d2a090d00070575f7", btcusdt, a)
		require.ErrorIsf(t, err, context.Canceled, "GetOrderInfo must return the %s request error", a)
	}
	_, err = e.GetOrderInfo(cancelledContext(t), "236655147005071361", xbtusdtm, asset.Futures)
	require.ErrorIs(t, err, context.Canceled, "GetOrderInfo must return the futures request error")

	for _, tc := range []struct {
		name    string
		orderID string
		pair    currency.Pair
		a       asset.Item
		err     error
	}{
		{name: "a futures order KuCoin does not have", orderID: "gct-null-futures-order", pair: xbtusdtm, a: asset.Futures, err: order.ErrOrderNotFound},
		{name: "a futures order of an unlisted contract", orderID: "gct-unlisted-futures-order", a: asset.Futures, err: currency.ErrPairNotFound},
		{name: "a futures order of another pair", orderID: "236655147005071361", pair: currency.NewPair(currency.ETH, currency.USDTM), a: asset.Futures, err: errOrderPairMismatch},
		{name: "a margin order of an unknown type", orderID: "gct-bad-margin-order", pair: btcusdt, a: asset.Margin, err: order.ErrUnrecognisedOrderType},
		{name: "a spot order of an unknown side", orderID: "gct-bad-spot-order", pair: btcusdt, a: asset.Spot, err: order.ErrSideIsInvalid},
	} {
		_, err = e.GetOrderInfo(t.Context(), tc.orderID, tc.pair, tc.a)
		require.ErrorIsf(t, err, tc.err, "GetOrderInfo must reject %s", tc.name)
	}
	// KuCoin's Get Order By OrderId examples' filled funds over their filled size, divided as the wrapper does
	cost, executed := 0.5, 0.00001
	for _, tc := range []struct {
		name    string
		orderID string
		pair    currency.Pair
		a       asset.Item
		exp     *order.Detail
	}{
		{
			// Futures amounts are in lots; a triggered stop order keeps its stop condition, so it is a stop order
			name: "futures", orderID: "236655147005071361", pair: xbtusdtm, a: asset.Futures,
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "236655147005071361", ClientOrderID: "5c52e11203aa677f33e493fb",
				Pair:      currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter),
				AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.StopLimit, Side: order.Buy,
				Status: order.PartiallyFilled, Price: 65000.1, TriggerPrice: 65500, Amount: 3, ContractAmount: 3,
				ExecutedAmount: 1, RemainingAmount: 2, AverageExecutedPrice: 65000.1, Cost: 65.0001, CostAsset: currency.USDT,
				SettlementCurrency: currency.USDT, Leverage: 3, TimeInForce: order.GoodTillCancel,
				Date: time.UnixMilli(1729236185949), LastUpdated: time.UnixMilli(1729236190949),
			},
		},
		{
			name: "margin", orderID: "671667306afcdb000723107f", pair: btcusdt, a: asset.Margin,
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "671667306afcdb000723107f", ClientOrderID: "5c52e11203aa677f33e493fb",
				Pair:      currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Margin, MarginType: margin.Multi, Type: order.StopLimit, Side: order.Buy,
				Status: order.PartiallyFilled, Price: 50000, TriggerPrice: 50500, Amount: 0.00003, QuoteAmount: 1.5,
				ExecutedAmount: 0.00001, RemainingAmount: 0.00001, AverageExecutedPrice: cost / executed, Cost: 0.5,
				CostAsset: currency.USDT, Fee: 0.0005, FeeAsset: currency.USDT, TimeInForce: order.GoodTillTime | order.PostOnly,
				HiddenOrder: true, Date: time.UnixMilli(1729521456248), LastUpdated: time.UnixMilli(1729651011877),
			},
		},
		{
			name: "spot", orderID: "68a3e81d2a090d00070575f7", pair: btcusdt, a: asset.Spot,
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "68a3e81d2a090d00070575f7", ClientOrderID: "5c52e11203aa677f33e493fb",
				Pair:      currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Price: 50000,
				Amount: 0.00003, QuoteAmount: 1.5, ExecutedAmount: 0.00001, RemainingAmount: 0.00001,
				AverageExecutedPrice: cost / executed, Cost: 0.5, CostAsset: currency.USDT, Fee: 0.0005,
				FeeAsset: currency.USDT, TimeInForce: order.GoodTillTime | order.PostOnly, HiddenOrder: true,
				Date: time.UnixMilli(1755572253338), LastUpdated: time.UnixMilli(1755572263485),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetOrderInfo(t.Context(), tc.orderID, tc.pair, tc.a)
			require.NoError(t, err, "GetOrderInfo must not error")
			assert.Equal(t, tc.exp, got, "GetOrderInfo should return the order")
		})
	}
}

func TestHFOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *HFOrder
		err  error
	}{
		{name: "a malformed symbol", o: &HFOrder{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &HFOrder{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &HFOrder{Symbol: "BTC-USDT", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.hfOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "hfOrderDetail should reject %s", tc.name)
	}

	btcusdt := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter)
	for _, tc := range []struct {
		name string
		o    *HFOrder
		exp  order.Detail
	}{
		{
			name: "partially filled open order",
			o: &HFOrder{
				ID: "67120bbef094e200070976f6", ClientOrderID: "gct-open", Symbol: "BTC-USDT", Type: "limit", Side: "buy",
				Price: 50000, Size: 0.5, Funds: 25000, DealSize: 0.25, DealFunds: 12500, RemainSize: 0.25, Fee: 12.5,
				FeeCurrency: currency.USDT, TimeInForce: "GTC", PostOnly: true, Active: true,
				CreatedAt: types.Time(time.UnixMilli(1729235902748)), LastUpdatedAt: types.Time(time.UnixMilli(1729235909862)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "67120bbef094e200070976f6", ClientOrderID: "gct-open", Pair: btcusdt,
				AssetType: asset.Spot, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Price: 50000,
				Amount: 0.5, QuoteAmount: 25000, ExecutedAmount: 0.25, RemainingAmount: 0.25, AverageExecutedPrice: 50000,
				Cost: 12500, CostAsset: currency.USDT, Fee: 12.5, FeeAsset: currency.USDT,
				TimeInForce: order.GoodTillCancel | order.PostOnly, Date: time.UnixMilli(1729235902748),
				LastUpdated: time.UnixMilli(1729235909862),
			},
		},
		{
			name: "cancelled market order",
			o: &HFOrder{
				ID: "6717422bd51c29000775ea03", ClientOrderID: "gct-cancelled", Symbol: "BTC-USDT", Type: "market",
				Side: "sell", Size: 0.001, FeeCurrency: currency.USDT, TimeInForce: "GTC", Hidden: true, CancelExist: true,
				CreatedAt: types.Time(time.UnixMilli(1729577515444)), LastUpdatedAt: types.Time(time.UnixMilli(1729577515481)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "6717422bd51c29000775ea03", ClientOrderID: "gct-cancelled", Pair: btcusdt,
				AssetType: asset.Spot, Type: order.Market, Side: order.Sell, Status: order.Cancelled, Amount: 0.001,
				CostAsset: currency.USDT, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, HiddenOrder: true,
				Date: time.UnixMilli(1729577515444), LastUpdated: time.UnixMilli(1729577515481),
				CloseTime: time.UnixMilli(1729577515481),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.hfOrderDetail(tc.o)
			require.NoError(t, err, "hfOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "hfOrderDetail should convert the order")
		})
	}
}

func TestMarginHFOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *MarginHFOrder
		err  error
	}{
		{name: "a malformed symbol", o: &MarginHFOrder{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &MarginHFOrder{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &MarginHFOrder{Symbol: "BTC-USDT", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.marginHFOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "marginHFOrderDetail should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name string
		o    *MarginHFOrder
		exp  order.Detail
	}{
		{
			name: "open cross margin order",
			o: &MarginHFOrder{
				ID: "671667306afcdb0007231090", ClientOrderID: "gct-margin-open", Symbol: "BTC-USDT", Type: "limit",
				Side: "buy", Price: 50000, Size: 0.001, Funds: 50, RemainSize: 0.001, FeeCurrency: currency.USDT,
				TimeInForce: "GTC", TradeType: CrossMarginTradeType, Active: true,
				CreatedAt: types.Time(time.UnixMilli(1729521456248)), LastUpdatedAt: types.Time(time.UnixMilli(1729521460940)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "671667306afcdb0007231090", ClientOrderID: "gct-margin-open",
				Pair:      currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy, Status: order.Open,
				Price: 50000, Amount: 0.001, QuoteAmount: 50, RemainingAmount: 0.001, CostAsset: currency.USDT,
				FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729521456248),
				LastUpdated: time.UnixMilli(1729521460940),
			},
		},
		{
			// A triggered stop order keeps its stop condition, so it is a stop order
			name: "filled isolated margin stop order",
			o: &MarginHFOrder{
				ID: "671667306afcdb0007231094", ClientOrderID: "gct-margin-filled", Symbol: "ETH-USDT", Type: "market",
				Side: "sell", Size: 0.5, DealSize: 0.5, DealFunds: 1500, Fee: 1.5, FeeCurrency: currency.USDT,
				Stop: "loss", StopTriggered: true, StopPrice: 3100, TimeInForce: "GTC", TradeType: IsolatedMarginTradeType,
				CreatedAt: types.Time(time.UnixMilli(1729521456248)), LastUpdatedAt: types.Time(time.UnixMilli(1729651011877)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "671667306afcdb0007231094", ClientOrderID: "gct-margin-filled",
				Pair:      currency.NewPairWithDelimiter(currency.ETH.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
				Status: order.Filled, TriggerPrice: 3100, Amount: 0.5, ExecutedAmount: 0.5, AverageExecutedPrice: 3000,
				Cost: 1500, CostAsset: currency.USDT, Fee: 1.5, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel,
				Date: time.UnixMilli(1729521456248), LastUpdated: time.UnixMilli(1729651011877),
				CloseTime: time.UnixMilli(1729651011877),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.marginHFOrderDetail(tc.o)
			require.NoError(t, err, "marginHFOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "marginHFOrderDetail should convert the order")
		})
	}
}

func TestFuturesOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *FuturesOrder
		err  error
	}{
		{name: "an unlisted contract", o: &FuturesOrder{Symbol: "UNLISTEDUSDTM", Type: "limit", Side: "buy"}, err: currency.ErrPairNotFound},
		{name: "an unknown type", o: &FuturesOrder{Symbol: "XBTUSDTM", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &FuturesOrder{Symbol: "XBTUSDTM", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.futuresOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "futuresOrderDetail should reject %s", tc.name)
	}

	pair := currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		o    *FuturesOrder
		exp  order.Detail
	}{
		{
			name: "open cross margin order",
			o: &FuturesOrder{
				ID: "235919172150824990", ClientOrderID: "gct-futures-open", Symbol: "XBTUSDTM", Type: "limit", Side: "buy",
				Price: 65000, Size: 3, TimeInForce: "GTC", PostOnly: true, Leverage: 3, IsActive: true,
				SettleCurrency: currency.USDT, MarginMode: "CROSS",
				CreatedAt: types.Time(time.UnixMilli(1729236185949)), UpdatedAt: types.Time(time.UnixMilli(1729236190949)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "235919172150824990", ClientOrderID: "gct-futures-open", Pair: pair,
				AssetType: asset.Futures, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy, Status: order.Open,
				Price: 65000, Amount: 3, ContractAmount: 3, RemainingAmount: 3, CostAsset: currency.USDT,
				SettlementCurrency: currency.USDT, Leverage: 3, TimeInForce: order.GoodTillCancel | order.PostOnly,
				Date: time.UnixMilli(1729236185949), LastUpdated: time.UnixMilli(1729236190949),
			},
		},
		{
			// A take profit or stop loss stop order is a stop order whatever its direction
			name: "filled isolated margin stop order",
			o: &FuturesOrder{
				ID: "235919172150825000", ClientOrderID: "gct-futures-done", Symbol: "XBTUSDTM", Type: "market", Side: "sell",
				Size: 2, DealSize: 2, DealValue: 120, AverageDealPrice: 60000, Stop: "up", StopPrice: 60000,
				TimeInForce: "GTC", Leverage: 5, ReduceOnly: true, SettleCurrency: currency.USDT, MarginMode: "ISOLATED",
				CreatedAt: types.Time(time.UnixMilli(1729236185949)), UpdatedAt: types.Time(time.UnixMilli(1729236200000)),
				EndAt: types.Time(time.UnixMilli(1729236200000)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "235919172150825000", ClientOrderID: "gct-futures-done", Pair: pair,
				AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
				Status: order.Filled, TriggerPrice: 60000, Amount: 2, ContractAmount: 2, ExecutedAmount: 2,
				AverageExecutedPrice: 60000, Cost: 120, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
				Leverage: 5, ReduceOnly: true, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729236185949),
				LastUpdated: time.UnixMilli(1729236200000), CloseTime: time.UnixMilli(1729236200000),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.futuresOrderDetail(tc.o)
			require.NoError(t, err, "futuresOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "futuresOrderDetail should convert the order")
		})
	}
}

func TestStopOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *StopOrder
		oco  bool
		err  error
	}{
		{name: "a malformed symbol", o: &StopOrder{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &StopOrder{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &StopOrder{Symbol: "BTC-USDT", Type: "limit"}, oco: true, err: order.ErrSideIsInvalid},
	} {
		_, err := e.stopOrderDetail(tc.o, asset.Spot, tc.oco)
		assert.ErrorIsf(t, err, tc.err, "stopOrderDetail should reject %s", tc.name)
	}

	btcusdt := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter)
	for _, tc := range []struct {
		name string
		o    *StopOrder
		a    asset.Item
		oco  bool
		exp  order.Detail
	}{
		{
			name: "spot stop limit order",
			o: &StopOrder{
				ID: "vs93gptvr9t2fsql003l8k61", ClientOrderID: "gct-spot-stop-limit", Symbol: "BTC-USDT", Status: "NEW",
				Type: "limit", Side: "buy", Price: 61000, Size: 0.001, TimeInForce: "GTC", TradeType: SpotTradeType,
				Stop: "entry", StopPrice: 60000, CreatedAt: types.Time(time.UnixMilli(1740626554884)),
				OrderTime: types.Time(time.Unix(0, 1740626554883000000)),
			},
			a: asset.Spot,
			exp: order.Detail{
				Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k61", ClientOrderID: "gct-spot-stop-limit", Pair: btcusdt,
				AssetType: asset.Spot, Type: order.StopLimit, Side: order.Buy, Status: order.New, Price: 61000,
				TriggerPrice: 60000, Amount: 0.001, TimeInForce: order.GoodTillCancel,
				Date: time.UnixMilli(1740626554884), LastUpdated: time.Unix(0, 1740626554883000000),
			},
		},
		{
			name: "isolated margin stop market order",
			o: &StopOrder{
				ID: "vs93gptvr9t2fsql003l8k65", ClientOrderID: "gct-margin-stop-market", Symbol: "BTC-USDT", Status: "NEW",
				Type: "market", Side: "sell", Funds: 25, TimeInForce: "GTC", Hidden: true,
				TradeType: IsolatedMarginTradeType, Stop: "loss", StopPrice: 40000,
				CreatedAt: types.Time(time.UnixMilli(1740626554884)), OrderTime: types.Time(time.Unix(0, 1740626554883000000)),
			},
			a: asset.Margin,
			exp: order.Detail{
				Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k65", ClientOrderID: "gct-margin-stop-market", Pair: btcusdt,
				AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
				Status: order.New, TriggerPrice: 40000, QuoteAmount: 25, TimeInForce: order.GoodTillCancel, HiddenOrder: true,
				Date: time.UnixMilli(1740626554884), LastUpdated: time.Unix(0, 1740626554883000000),
			},
		},
		{
			// An OCO order is listed by its stop order, whose type does not make it a stop order
			name: "cross margin OCO order",
			o: &StopOrder{
				ID: "vs93gptvr9t2fsql003l8k66", ClientOrderID: "gct-margin-oco", Symbol: "BTC-USDT", Status: "NEW",
				Type: "oco", Side: "buy", Price: 50000, Size: 0.001, TimeInForce: "GTC", TradeType: CrossMarginTradeType,
				Stop: "entry", StopPrice: 70000, CreatedAt: types.Time(time.UnixMilli(1740626554884)),
				OrderTime: types.Time(time.Unix(0, 1740626554883000000)),
			},
			a:   asset.Margin,
			oco: true,
			exp: order.Detail{
				Exchange: e.Name, OrderID: "vs93gptvr9t2fsql003l8k66", ClientOrderID: "gct-margin-oco", Pair: btcusdt,
				AssetType: asset.Margin, MarginType: margin.Multi, Type: order.OCO, Side: order.Buy, Status: order.New,
				Price: 50000, TriggerPrice: 70000, Amount: 0.001, TimeInForce: order.GoodTillCancel,
				Date: time.UnixMilli(1740626554884), LastUpdated: time.Unix(0, 1740626554883000000),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.stopOrderDetail(tc.o, tc.a, tc.oco)
			require.NoError(t, err, "stopOrderDetail must not error")
			assert.Equal(t, tc.exp, got, "stopOrderDetail should convert the order")
		})
	}
}

func TestOCOOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := e.ocoOrderDetail(&OCOOrder{Symbol: "BT", Status: "NEW"}, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCreatingPair, "ocoOrderDetail must reject a malformed symbol")
	_, err = e.ocoOrderDetail(&OCOOrder{Symbol: "BTC-USDT", Status: "PENDING"}, asset.Spot)
	require.ErrorIs(t, err, order.ErrUnsupportedStatusType, "ocoOrderDetail must reject an unknown status")

	for status, exp := range map[string]order.Status{
		"NEW":       order.New,
		"DONE":      order.Closed,
		"TRIGGERED": order.Closed,
		"CANCELED":  order.Cancelled,
		"CANCELLED": order.Cancelled,
	} {
		got, err := e.ocoOrderDetail(&OCOOrder{OrderID: "674c3cfa72cf2800072ee7ce", Symbol: "BTC-USDT", ClientOrderID: "gct-oco", OrderTime: types.Time(time.UnixMilli(1733049594803)), Status: status}, asset.Margin)
		require.NoErrorf(t, err, "ocoOrderDetail must not error for status %s", status)
		assert.Equalf(t, order.Detail{
			Exchange: e.Name, OrderID: "674c3cfa72cf2800072ee7ce", ClientOrderID: "gct-oco",
			Pair:      currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter),
			AssetType: asset.Margin, Type: order.OCO, Status: exp, Date: time.UnixMilli(1733049594803),
			LastUpdated: time.UnixMilli(1733049594803),
		}, got, "ocoOrderDetail should convert an OCO order of status %s", status)
	}
}

func TestOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		kind string
		stop bool
		exp  order.Type
	}{
		{kind: "limit", exp: order.Limit},
		{kind: "LIMIT", stop: true, exp: order.StopLimit},
		{kind: "market", exp: order.Market},
		{kind: "market", stop: true, exp: order.StopMarket},
	} {
		got, err := orderType(tc.kind, tc.stop)
		require.NoErrorf(t, err, "orderType must not error for %s", tc.kind)
		assert.Equalf(t, tc.exp, got, "orderType should return %s for %s with stop %t", tc.exp, tc.kind, tc.stop)
	}
	_, err := orderType("limit_stop", false)
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "orderType should reject an order type KuCoin's orders do not carry")
}

func TestOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		active    bool
		cancelled bool
		executed  float64
		exp       order.Status
	}{
		{name: "an open order", active: true, exp: order.Open},
		{name: "a partially filled open order", active: true, cancelled: true, executed: 1, exp: order.PartiallyFilled},
		{name: "a cancelled order", cancelled: true, exp: order.Cancelled},
		{name: "a partially filled cancelled order", cancelled: true, executed: 1, exp: order.PartiallyFilledCancelled},
		{name: "a filled order", executed: 1, exp: order.Filled},
	} {
		assert.Equalf(t, tc.exp, orderStatus(tc.active, tc.cancelled, tc.executed), "orderStatus should return %s for %s", tc.exp, tc.name)
	}
}

func TestTradeTypeMarginType(t *testing.T) {
	t.Parallel()
	for tradeType, exp := range map[string]margin.Type{
		IsolatedMarginTradeType: margin.Isolated,
		CrossMarginTradeType:    margin.Multi,
		SpotTradeType:           margin.Unset,
	} {
		assert.Equalf(t, exp, tradeTypeMarginType(tradeType), "tradeTypeMarginType should return %s for %s", exp, tradeType)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := e.GetDepositAddress(t.Context(), currency.EMPTYCODE, "", "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetDepositAddress must reject an empty currency")
	_, err = marketTestServerExchange(t, nil).GetDepositAddress(cancelledContext(t), currency.USDT, "", "")
	require.ErrorIs(t, err, context.Canceled, "GetDepositAddress must return the failed request's error")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		if _, err = e.GetDepositAddress(t.Context(), currency.BTC, "", "btc"); err != nil {
			assert.ErrorIs(t, err, errNoDepositAddress, "GetDepositAddress should only fail when the account has no address on the chain")
		}
		return
	}
	for _, tc := range []struct {
		name     string
		currency currency.Code
		chain    string
		exp      *deposit.Address
		err      error
	}{
		{name: "only address", currency: currency.BTC, exp: &deposit.Address{Address: "bc1q***", Chain: "btc"}},
		{name: "chain ID", currency: currency.USDT, chain: "trx", exp: &deposit.Address{Address: "***", Chain: "trx"}},
		{name: "chain name", currency: currency.USDT, chain: "TON", exp: &deposit.Address{Address: "***", Tag: "2085123456", Chain: "ton"}},
		{name: "several addresses", currency: currency.USDT, err: errMultipleDepositAddress},
		{name: "no address on the chain", currency: currency.USDT, chain: "eth", err: errNoDepositAddress},
		{name: "no address", currency: currency.ETH, err: errNoDepositAddress},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDepositAddress(t.Context(), tc.currency, "", tc.chain)
			require.ErrorIs(t, err, tc.err, "GetDepositAddress must return the expected error")
			assert.Equal(t, tc.exp, result, "GetDepositAddress should return the address on the chain, by ID or name")
		})
	}
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Amount:   0.00000000001,
		Crypto:   withdraw.CryptoRequest{Address: core.BitcoinDonationAddress},
	})
	require.ErrorContains(t, err, withdraw.ErrStrNoCurrencySet, "WithdrawCryptocurrencyFunds must reject a request without a currency")
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Amount:   0.00000000001,
		Currency: currency.BTC,
	})
	require.ErrorContains(t, err, withdraw.ErrStrAddressNotSet, "WithdrawCryptocurrencyFunds must reject a request without an address")
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Currency: currency.BTC,
		Crypto:   withdraw.CryptoRequest{Address: core.BitcoinDonationAddress},
	})
	require.ErrorContains(t, err, withdraw.ErrStrAmountMustBeGreaterThanZero, "WithdrawCryptocurrencyFunds must reject a request without an amount")
	request := &withdraw.Request{
		Exchange:         e.Name,
		Currency:         currency.BTC,
		Amount:           0.5,
		Description:      "GoCryptoTrader donation",
		InternalTransfer: true,
		Crypto: withdraw.CryptoRequest{
			Address:    core.BitcoinDonationAddress,
			AddressTag: "2085123456",
			Chain:      "btc",
		},
	}
	_, err = marketTestServerExchange(t, nil).WithdrawCryptocurrencyFunds(cancelledContext(t), request)
	require.ErrorIs(t, err, context.Canceled, "WithdrawCryptocurrencyFunds must return the failed request's error")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.WithdrawCryptocurrencyFunds(t.Context(), request)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error")
	if !mockTests {
		assert.NotEmpty(t, result.ID, "WithdrawCryptocurrencyFunds should return a withdrawal ID")
		return
	}
	assert.Equal(t, &withdraw.ExchangeResponse{ID: "670a973cf07b3800070e216e"}, result, "WithdrawCryptocurrencyFunds should return the withdrawal ID")
	_, err = e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: e.Name,
		Currency: currency.BTC,
		Amount:   0.7,
		Crypto:   withdraw.CryptoRequest{Address: core.BitcoinDonationAddress},
	})
	assert.ErrorIs(t, err, common.ErrNoResponse, "WithdrawCryptocurrencyFunds should reject a reply without a withdrawal")
}

func TestRequestFormat(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	_, err := ex.requestFormat(asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "requestFormat must reject an asset KuCoin does not trade")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Margin, false), "SetAssetEnabled must not error")
	_, err = ex.requestFormat(asset.Margin)
	require.ErrorIs(t, err, asset.ErrNotEnabled, "requestFormat must reject a disabled asset")
	ex.CurrencyPairs.Pairs[asset.Futures].RequestFormat = nil
	_, err = ex.requestFormat(asset.Futures)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "requestFormat must return a missing request format")

	got, err := ex.requestFormat(asset.Spot)
	require.NoError(t, err, "requestFormat must not error for spot")
	assert.Equal(t, currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}, got, "requestFormat should return the spot request format")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	btcusdt := currency.NewBTCUSDT()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 100000, Amount: 1})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "SubmitOrder must require a client order ID")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Options, Side: order.Buy, Type: order.Limit, Price: 1, Amount: 1, ClientOrderID: "gct-options"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder must reject an asset KuCoin does not trade")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1.5, ClientOrderID: "gct-futures-fraction"})
	require.ErrorIs(t, err, errFuturesSizeNotWholeLots, "SubmitOrder must reject a futures size that is not whole lots")
	// Margin and spot submissions are handed to their own submitters, whose validation shows they were reached
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.OCO, Price: 1, Amount: 1, ClientOrderID: "gct-margin-oco"})
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "SubmitOrder must submit margin orders through submitMarginOrder")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.OCO, Price: 1, Amount: 1, ClientOrderID: "gct-spot-oco"})
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "SubmitOrder must submit spot orders through submitSpotOrder")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		// A buy far below the market rests on the book until it is cancelled
		tick, err := e.UpdateTicker(t.Context(), spotTradablePair, asset.Spot)
		require.NoError(t, err, "UpdateTicker must not error")
		s := &order.Submit{Exchange: e.Name, Pair: spotTradablePair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: math.Floor(tick.Last * 0.8), Amount: 0.0001, ClientOrderID: "gct-live-" + strconv.FormatInt(time.Now().UnixMilli(), 10)}
		resp, err := e.SubmitOrder(t.Context(), s)
		require.NoError(t, err, "SubmitOrder must not error")
		require.NotEmpty(t, resp.OrderID, "SubmitOrder must return the order ID")
		require.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: resp.OrderID, Pair: s.Pair, AssetType: s.AssetType}), "CancelOrder must not error")
		return
	}

	_, err = e.SubmitOrder(cancelledContext(t), &order.Submit{Exchange: e.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1, ClientOrderID: "gct-futures-cancelled"})
	require.ErrorIs(t, err, context.Canceled, "SubmitOrder must return the futures request error")

	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Sell, Type: order.Market, Amount: 1, ClientOrderID: "gct-futures-null"})
	require.ErrorIs(t, err, common.ErrNoResponse, "SubmitOrder must reject a futures reply without data")
	s := &order.Submit{
		Exchange:      e.Name,
		Pair:          xbtusdtm,
		AssetType:     asset.Futures,
		Side:          order.Buy,
		Type:          order.Limit,
		Price:         100000,
		Amount:        2,
		Leverage:      3,
		MarginType:    margin.Isolated,
		TimeInForce:   order.ImmediateOrCancel,
		ClientOrderID: "gct-futures-limit",
	}
	resp, err := e.SubmitOrder(t.Context(), s)
	require.NoError(t, err, "SubmitOrder must not error")
	require.NotNil(t, resp, "SubmitOrder must return a response")
	assert.NotZero(t, resp.Date, "SubmitOrder should date the order")
	resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
	exp := &order.SubmitResponse{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          xbtusdtm,
		AssetType:     asset.Futures,
		TimeInForce:   order.ImmediateOrCancel,
		Leverage:      3,
		Price:         100000,
		Amount:        2,
		ClientOrderID: "gct-futures-limit",
		MarginType:    margin.Isolated,
		Status:        order.New,
		OrderID:       "250445104152670300",
	}
	assert.Equal(t, exp, resp, "SubmitOrder should return KuCoin's order ID with the order")
}

func TestFuturesOrderRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{name: "fractional size", submit: &order.Submit{Type: order.Market, Amount: 1.5}, err: errFuturesSizeNotWholeLots},
		{name: "negative leverage", submit: &order.Submit{Type: order.Market, Amount: 1, Leverage: -2}, err: errInvalidLeverage},
		{name: "fractional leverage", submit: &order.Submit{Type: order.Market, Amount: 1, Leverage: 2.5}, err: errInvalidLeverage},
		{name: "good till time", submit: &order.Submit{Type: order.Limit, Price: 1, Amount: 1, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
		{name: "no margin", submit: &order.Submit{Type: order.Limit, Price: 1, Amount: 1, MarginType: margin.NoMargin}, err: margin.ErrMarginTypeUnsupported},
		{name: "trailing stop", submit: &order.Submit{Type: order.TrailingStop, Amount: 1, TriggerPrice: 1}, err: order.ErrUnsupportedOrderType},
		{name: "stop without trigger price", submit: &order.Submit{Type: order.StopLimit, Price: 1, Amount: 1}, err: errStopPriceRequired},
	} {
		_, err := futuresOrderRequest(tc.submit, "XBTUSDTM", "buy")
		assert.ErrorIsf(t, err, tc.err, "futuresOrderRequest should reject a %s", tc.name)
	}

	for _, tc := range []struct {
		name   string
		submit *order.Submit
		side   string
		exp    *FuturesOrderRequest
	}{
		{
			name:   "isolated limit order",
			submit: &order.Submit{Side: order.Buy, Type: order.Limit, Price: 100000, Amount: 2, Leverage: 3, MarginType: margin.Isolated, TimeInForce: order.ImmediateOrCancel, ReduceOnly: true, ClientOrderID: "gct-futures-limit"},
			side:   "buy",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-limit", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit", ReduceOnly: true, MarginMode: "ISOLATED", Price: 100000, Size: 2, TimeInForce: "IOC"},
		},
		{
			name:   "cross margin market order by value",
			submit: &order.Submit{Side: order.Sell, Type: order.Market, QuoteAmount: 500, MarginType: margin.Multi, ClientOrderID: "gct-futures-value"},
			side:   "sell",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-value", Side: "sell", Symbol: "XBTUSDTM", Type: "market", MarginMode: "CROSS", ValueQuantity: 500},
		},
		{
			// A stop sell protects a long position, so it triggers as the price falls
			name:   "stop limit sell on the index price",
			submit: &order.Submit{Side: order.Sell, Type: order.StopLimit, Price: 90000, Amount: 1, TriggerPrice: 91000, TriggerPriceType: order.IndexPrice, TimeInForce: order.GoodTillCancel | order.PostOnly, ClientOrderID: "gct-futures-stop"},
			side:   "sell",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-stop", Side: "sell", Symbol: "XBTUSDTM", Type: "limit", Stop: "down", StopPriceType: "IP", StopPrice: 91000, Price: 90000, Size: 1, PostOnly: true},
		},
		{
			// A take profit sell closes a long position in profit, so it triggers as the price rises
			name:   "take profit sell on the mark price",
			submit: &order.Submit{Side: order.Sell, Type: order.TakeProfit, Price: 120000, Amount: 1, TriggerPrice: 119000, TriggerPriceType: order.MarkPrice, ClientOrderID: "gct-futures-tp"},
			side:   "sell",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-tp", Side: "sell", Symbol: "XBTUSDTM", Type: "limit", Stop: "up", StopPriceType: "MP", StopPrice: 119000, Price: 120000, Size: 1},
		},
		{
			name:   "stop market buy on the last price",
			submit: &order.Submit{Side: order.Buy, Type: order.StopMarket, Amount: 1, TriggerPrice: 110000, ClientOrderID: "gct-futures-stop-market"},
			side:   "buy",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-stop-market", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Stop: "up", StopPriceType: "TP", StopPrice: 110000, Size: 1},
		},
		{
			name:   "take profit market buy",
			submit: &order.Submit{Side: order.Buy, Type: order.TakeProfitMarket, Amount: 1, TriggerPrice: 90000, TriggerPriceType: order.LastPrice, ClientOrderID: "gct-futures-tp-market"},
			side:   "buy",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-tp-market", Side: "buy", Symbol: "XBTUSDTM", Type: "market", Stop: "down", StopPriceType: "TP", StopPrice: 90000, Size: 1},
		},
		{
			name:   "stop is a stop limit order",
			submit: &order.Submit{Side: order.Short, Type: order.Stop, Price: 94000, Amount: 3, TriggerPrice: 95000, ClientOrderID: "gct-futures-stop-short"},
			side:   "sell",
			exp:    &FuturesOrderRequest{ClientOrderID: "gct-futures-stop-short", Side: "sell", Symbol: "XBTUSDTM", Type: "limit", Stop: "down", StopPriceType: "TP", StopPrice: 95000, Price: 94000, Size: 3},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := futuresOrderRequest(tc.submit, "XBTUSDTM", tc.side)
			require.NoError(t, err, "futuresOrderRequest must not error")
			assert.Equal(t, tc.exp, got, "futuresOrderRequest should describe the submission")
		})
	}
}

func TestTimeInForceString(t *testing.T) {
	t.Parallel()
	for tif, exp := range map[order.TimeInForce]string{
		order.UnknownTIF:                      "",
		order.GoodTillCancel:                  "",
		order.PostOnly:                        "",
		order.GoodTillCancel | order.PostOnly: "",
		order.ImmediateOrCancel:               "IOC",
		order.FillOrKill:                      "FOK",
	} {
		got, err := timeInForceString(tif)
		require.NoErrorf(t, err, "timeInForceString must not error for %s", tif)
		assert.Equalf(t, exp, got, "timeInForceString should return KuCoin's time in force for %s", tif)
	}
	_, err := timeInForceString(order.GoodTillTime)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "timeInForceString should reject good till time")
}

func TestFuturesMarginMode(t *testing.T) {
	t.Parallel()
	got, err := futuresMarginMode(margin.Isolated)
	require.NoError(t, err, "futuresMarginMode must not error for isolated margin")
	assert.Equal(t, "ISOLATED", got, "futuresMarginMode should return ISOLATED for isolated margin")
	got, err = futuresMarginMode(margin.Multi)
	require.NoError(t, err, "futuresMarginMode must not error for cross margin")
	assert.Equal(t, "CROSS", got, "futuresMarginMode should return CROSS for cross margin")
	for _, m := range []margin.Type{margin.Unset, margin.NoMargin} {
		_, err = futuresMarginMode(m)
		assert.ErrorIsf(t, err, margin.ErrMarginTypeUnsupported, "futuresMarginMode should reject margin type %q", m)
	}
}

func TestSubmitSpotOrder(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	oco := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Price: 69000, Amount: 0.001, ClientOrderID: "gct-spot-oco"}
	_, err := e.submitSpotOrder(t.Context(), oco, "BTC-USDT", "sell")
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "submitSpotOrder must validate an OCO order's prices")
	oco.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, Price: 70000},
		StopLoss:   order.RiskManagement{Enabled: true, Price: 50000},
	}
	_, err = e.submitSpotOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.TrailingStop, Amount: 1, ClientOrderID: "gct-spot-trailing"}, "BTC-USDT", "buy")
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "submitSpotOrder must reject an order type KuCoin does not place")
	stop := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.StopLimit, Price: 61000, Amount: 0.001, ClientOrderID: "gct-spot-stop", RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 60000}}}
	limit := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, TimeInForce: order.FillOrKill, ClientOrderID: "gct-spot-limit"}

	if !mockTests {
		t.Skip("submitSpotOrder's replies come from fixtures; TestSubmitOrder places a live order")
	}

	_, err = e.submitSpotOrder(cancelledContext(t), oco, "BTC-USDT", "sell")
	require.ErrorIs(t, err, context.Canceled, "submitSpotOrder must return the OCO order's request error")
	_, err = e.submitSpotOrder(cancelledContext(t), stop, "BTC-USDT", "buy")
	require.ErrorIs(t, err, context.Canceled, "submitSpotOrder must return the stop order's request error")
	_, err = e.submitSpotOrder(cancelledContext(t), limit, "BTC-USDT", "buy")
	require.ErrorIs(t, err, context.Canceled, "submitSpotOrder must return the order's request error")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		side   string
	}{
		{name: "OCO", side: "sell", submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Price: 69000, Amount: 0.001, ClientOrderID: "gct-spot-oco-null", RiskManagementModes: oco.RiskManagementModes}},
		{name: "stop", side: "sell", submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.StopMarket, QuoteAmount: 25, ClientOrderID: "gct-spot-stop-null", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}}},
		{name: "market", side: "buy", submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientOrderID: "gct-spot-null"}},
	} {
		_, err = e.submitSpotOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
		require.ErrorIsf(t, err, common.ErrNoResponse, "submitSpotOrder must reject a %s order reply without data", tc.name)
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		side   string
		exp    *order.SubmitResponse
	}{
		{
			name:   "OCO",
			submit: oco,
			side:   "sell",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.OCO, Side: order.Sell, Pair: btcusdt, AssetType: asset.Spot, Price: 69000, Amount: 0.001, ClientOrderID: "gct-spot-oco", Status: order.New, OrderID: "674c316e688dea0007c7b990"},
		},
		{
			name:   "stop limit",
			submit: stop,
			side:   "buy",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.StopLimit, Side: order.Buy, Pair: btcusdt, AssetType: asset.Spot, Price: 61000, Amount: 0.001, ClientOrderID: "gct-spot-stop", Status: order.New, OrderID: "vs93gptvr9t2fsql003l8k60"},
		},
		{
			name:   "fill or kill limit",
			submit: limit,
			side:   "buy",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.Limit, Side: order.Buy, Pair: btcusdt, AssetType: asset.Spot, TimeInForce: order.FillOrKill, Price: 50000, Amount: 0.001, ClientOrderID: "gct-spot-limit", Status: order.New, OrderID: "6717422bd51c29000775ea10"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.submitSpotOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
			require.NoError(t, err, "submitSpotOrder must not error")
			require.NotNil(t, resp, "submitSpotOrder must return a response")
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equal(t, tc.exp, resp, "submitSpotOrder should return KuCoin's order ID with the order")
		})
	}
}

func TestSubmitMarginOrder(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	oco := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.OCO, Price: 51000, Amount: 0.001, MarginType: margin.Isolated, AutoBorrow: true, ClientOrderID: "gct-margin-oco"}
	_, err := e.submitMarginOrder(t.Context(), oco, "BTC-USDT", "buy")
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "submitMarginOrder must validate an OCO order's prices")
	oco.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, Price: 50000},
		StopLoss:   order.RiskManagement{Enabled: true, Price: 70000},
	}
	_, err = e.submitMarginOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.StopLimit, Price: 1, Amount: 1, ClientOrderID: "gct-margin-stop-no-trigger"}, "BTC-USDT", "buy")
	require.ErrorIs(t, err, errStopPriceRequired, "submitMarginOrder must reject a stop order without a trigger")
	stop := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Sell, Type: order.Limit, Price: 39000, Amount: 0.001, AutoRepay: true, ClientOrderID: "gct-margin-stop", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}}
	limit := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, MarginType: margin.Isolated, TimeInForce: order.PostOnly, AutoBorrow: true, AutoRepay: true, ClientOrderID: "gct-margin-limit"}

	if !mockTests {
		t.Skip("submitMarginOrder's replies come from fixtures, and KuCoin margin orders borrow funds")
	}

	_, err = e.submitMarginOrder(cancelledContext(t), oco, "BTC-USDT", "buy")
	require.ErrorIs(t, err, context.Canceled, "submitMarginOrder must return the OCO order's request error")
	_, err = e.submitMarginOrder(cancelledContext(t), stop, "BTC-USDT", "sell")
	require.ErrorIs(t, err, context.Canceled, "submitMarginOrder must return the stop order's request error")
	_, err = e.submitMarginOrder(cancelledContext(t), limit, "BTC-USDT", "buy")
	require.ErrorIs(t, err, context.Canceled, "submitMarginOrder must return the order's request error")
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		side   string
		err    error
	}{
		{name: "an OCO order reply without data", side: "buy", err: common.ErrNoResponse, submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.OCO, Price: 51000, Amount: 0.001, ClientOrderID: "gct-margin-oco-null", RiskManagementModes: oco.RiskManagementModes}},
		{name: "a stop order reply without data", side: "sell", err: common.ErrNoResponse, submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Sell, Type: order.Market, Amount: 0.001, ClientOrderID: "gct-margin-stop-null", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}}},
		{name: "an order reply without data", side: "buy", err: common.ErrNoResponse, submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientOrderID: "gct-margin-null"}},
		{name: "an order reply without an order ID", side: "buy", err: order.ErrOrderIDNotSet, submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, QuoteAmount: 25, ClientOrderID: "gct-margin-no-id"}},
	} {
		_, err = e.submitMarginOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
		require.ErrorIsf(t, err, tc.err, "submitMarginOrder must reject %s", tc.name)
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		side   string
		exp    *order.SubmitResponse
	}{
		{
			name:   "isolated OCO",
			submit: oco,
			side:   "buy",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.OCO, Side: order.Buy, Pair: btcusdt, AssetType: asset.Margin, Price: 51000, Amount: 0.001, ClientOrderID: "gct-margin-oco", MarginType: margin.Isolated, Status: order.New, OrderID: "674c316e688dea0007c7b991"},
		},
		{
			name:   "cross stop loss",
			submit: stop,
			side:   "sell",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.Limit, Side: order.Sell, Pair: btcusdt, AssetType: asset.Margin, Price: 39000, Amount: 0.001, ClientOrderID: "gct-margin-stop", Status: order.New, OrderID: "vs8hoq2h6t43dlku003vff30"},
		},
		{
			// KuCoin borrows the shortfall of an order placed with auto borrow and reports the loan
			name:   "isolated limit with auto borrow",
			submit: limit,
			side:   "buy",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.Limit, Side: order.Buy, Pair: btcusdt, AssetType: asset.Margin, TimeInForce: order.PostOnly, Price: 50000, Amount: 0.001, ClientOrderID: "gct-margin-limit", MarginType: margin.Isolated, Status: order.New, OrderID: "671bb90194422f00073ff4f2", BorrowSize: 0.001, LoanApplyID: "600656d9a33ac90009de4f6f"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.submitMarginOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
			require.NoError(t, err, "submitMarginOrder must not error")
			require.NotNil(t, resp, "submitMarginOrder must return a response")
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equal(t, tc.exp, resp, "submitMarginOrder should return KuCoin's order ID with the order")
		})
	}
}

func TestHFOrderRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{name: "an order type KuCoin does not place", submit: &order.Submit{Type: order.TrailingStop}, err: order.ErrUnsupportedOrderType},
		{name: "both stop conditions", submit: &order.Submit{Type: order.Limit, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true}, StopEntry: order.RiskManagement{Enabled: true}}}, err: errStopModesExclusive},
		{name: "a stop order without a stop condition", submit: &order.Submit{Type: order.StopMarket}, err: errStopPriceRequired},
		{name: "a good till time limit order", submit: &order.Submit{Type: order.Limit, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
	} {
		_, _, _, err := hfOrderRequest(tc.submit, "BTC-USDT", "buy")
		assert.ErrorIsf(t, err, tc.err, "hfOrderRequest should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name         string
		submit       *order.Submit
		exp          *PlaceHFOrderRequest
		expStop      string
		expStopPrice float64
	}{
		{
			name:   "fill or kill limit order",
			submit: &order.Submit{Type: order.Limit, Price: 50000, Amount: 0.001, TimeInForce: order.FillOrKill, ClientOrderID: "gct-spot-limit"},
			exp:    &PlaceHFOrderRequest{ClientOrderID: "gct-spot-limit", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Price: 50000, Size: 0.001, TimeInForce: "FOK"},
		},
		{
			name:   "post only limit order",
			submit: &order.Submit{Type: order.Limit, Price: 50000, Amount: 0.001, TimeInForce: order.GoodTillCancel | order.PostOnly, ClientOrderID: "gct-spot-post-only"},
			exp:    &PlaceHFOrderRequest{ClientOrderID: "gct-spot-post-only", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Price: 50000, Size: 0.001, PostOnly: true},
		},
		{
			// Market orders take no time in force
			name:   "market order by funds",
			submit: &order.Submit{Type: order.Market, QuoteAmount: 25, TimeInForce: order.GoodTillTime, ClientOrderID: "gct-spot-market"},
			exp:    &PlaceHFOrderRequest{ClientOrderID: "gct-spot-market", Side: "buy", Symbol: "BTC-USDT", Type: "market", Funds: 25},
		},
		{
			name:         "stop limit order triggered as the price rises",
			submit:       &order.Submit{Type: order.StopLimit, Price: 61000, Amount: 0.001, ClientOrderID: "gct-spot-stop", RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 60000}}},
			exp:          &PlaceHFOrderRequest{ClientOrderID: "gct-spot-stop", Side: "buy", Symbol: "BTC-USDT", Type: "limit", Price: 61000, Size: 0.001},
			expStop:      "entry",
			expStopPrice: 60000,
		},
		{
			name:         "market order triggered as the price falls",
			submit:       &order.Submit{Type: order.Market, Amount: 0.001, ClientOrderID: "gct-spot-stop-loss", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}},
			exp:          &PlaceHFOrderRequest{ClientOrderID: "gct-spot-stop-loss", Side: "buy", Symbol: "BTC-USDT", Type: "market", Size: 0.001},
			expStop:      "loss",
			expStopPrice: 40000,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, stop, stopPrice, err := hfOrderRequest(tc.submit, "BTC-USDT", "buy")
			require.NoError(t, err, "hfOrderRequest must not error")
			assert.Equal(t, tc.exp, got, "hfOrderRequest should describe the submission")
			assert.Equal(t, tc.expStop, stop, "hfOrderRequest should return the stop condition")
			assert.Equal(t, tc.expStopPrice, stopPrice, "hfOrderRequest should return the trigger price")
		})
	}
}

func TestOCOPrices(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		side  order.Side
		modes order.RiskManagementModes
		err   error
	}{
		{name: "no take profit", side: order.Sell, modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 50000}}, err: errTakeProfitPriceRequired},
		{name: "a take profit without a price", side: order.Sell, modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}}, err: errTakeProfitPriceRequired},
		{name: "no stop loss", side: order.Sell, modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}}, err: errStopLossPriceRequired},
		{name: "a sell stop loss above its take profit", side: order.Sell, modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000}}, err: errOCOPricesInverted},
		{name: "a buy stop loss below its take profit", side: order.Buy, modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000}}, err: errOCOPricesInverted},
	} {
		_, _, err := ocoPrices(tc.side, &tc.modes)
		assert.ErrorIsf(t, err, tc.err, "ocoPrices should reject %s", tc.name)
	}

	stopPrice, limitPrice, err := ocoPrices(order.Sell, &order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000}})
	require.NoError(t, err, "ocoPrices must not error for a sell order")
	assert.Equal(t, 50000.0, stopPrice, "ocoPrices should trigger a sell order's stop order at its stop loss price")
	assert.Equal(t, 70000.0, limitPrice, "ocoPrices should place a sell order's stop order at its take profit price")
	stopPrice, limitPrice, err = ocoPrices(order.Bid, &order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000}})
	require.NoError(t, err, "ocoPrices must not error for a buy order")
	assert.Equal(t, 70000.0, stopPrice, "ocoPrices should trigger a buy order's stop order at its stop loss price")
	assert.Equal(t, 50000.0, limitPrice, "ocoPrices should place a buy order's stop order at its take profit price")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancellation")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: btcusdt}), order.ErrOrderIDNotSet, "CancelOrder must require an order ID or client order ID")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Options, OrderID: "1"}), asset.ErrNotSupported, "CancelOrder must reject an asset KuCoin does not trade")
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
	}{
		{name: "a futures order by client order ID", cancel: &order.Cancel{AssetType: asset.Futures, ClientOrderID: "5bd6e9286d99522a52e458de"}},
		{name: "a margin order", cancel: &order.Cancel{AssetType: asset.Margin, OrderID: "671663e02188630007e21c9c"}},
		{name: "a spot order", cancel: &order.Cancel{AssetType: asset.Spot, ClientOrderID: "5c52e11203aa677f33e493fb"}},
	} {
		require.ErrorIsf(t, e.CancelOrder(t.Context(), tc.cancel), currency.ErrCurrencyPairEmpty, "CancelOrder must require the pair to cancel %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		err := e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Pair: spotTradablePair, OrderID: "6717422bd51c29000775ea99"})
		assert.Error(t, err, "CancelOrder should report an order KuCoin does not have")
		return
	}

	err := e.CancelOrder(cancelledContext(t), &order.Cancel{AssetType: asset.Spot, Type: order.StopLimit, ClientOrderID: "gct-spot-stop"})
	require.ErrorIs(t, err, context.Canceled, "CancelOrder must return the error of the stop order's lookup")

	err = e.CancelOrder(t.Context(), &order.Cancel{AssetType: asset.Spot, Type: order.Stop, ClientOrderID: "gct-missing-stop"})
	require.ErrorIs(t, err, order.ErrOrderNotFound, "CancelOrder must report a stop order KuCoin does not find")
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
	}{
		{name: "futures order", cancel: &order.Cancel{AssetType: asset.Futures, OrderID: "235303670076489728"}},
		{name: "futures order by client order ID", cancel: &order.Cancel{AssetType: asset.Futures, Pair: currency.NewPair(currency.XBT, currency.USDTM), ClientOrderID: "5bd6e9286d99522a52e458de"}},
		{name: "margin OCO order", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.OCO, OrderID: "674c316e688dea0007c7b986"}},
		{name: "margin OCO order by client order ID", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.OCO, ClientOrderID: "5c52e11203aa677f33e493fb"}},
		{name: "margin stop order", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.StopLimit, OrderID: "vs8hoq2h6t43dlku003vff2r"}},
		{name: "margin stop order by client order ID", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.Stop, ClientOrderID: "9e64g33505cc899h55g615hd"}},
		{name: "spot OCO order", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.OCO, OrderID: "674c3b6e688dea0007c7bab2"}},
		{name: "spot OCO order by client order ID", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.OCO, ClientOrderID: "5c52e1203aa6f3g7f1e493fb"}},
		{name: "spot stop order", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.StopMarket, OrderID: "vs8hoo8q2ceshiue003b67c0"}},
		{name: "spot stop order by client order ID", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.ConditionalStop, Pair: btcusdt, ClientOrderID: "gct-spot-stop"}},
		{name: "margin order", cancel: &order.Cancel{AssetType: asset.Margin, Pair: btcusdt, OrderID: "671663e02188630007e21c9c"}},
		{name: "margin order by client order ID", cancel: &order.Cancel{AssetType: asset.Margin, Pair: btcusdt, ClientOrderID: "5c52e11203aa677f33e493fb"}},
		{name: "spot order", cancel: &order.Cancel{AssetType: asset.Spot, Pair: btcusdt, OrderID: "671124f9365ccb00073debd4"}},
		{name: "spot order by client order ID", cancel: &order.Cancel{AssetType: asset.Spot, Pair: btcusdt, ClientOrderID: "5c52e11203aa677f33e493fb"}},
	} {
		assert.NoErrorf(t, e.CancelOrder(t.Context(), tc.cancel), "CancelOrder should cancel a %s", tc.name)
	}
}

func TestIsStopOrderType(t *testing.T) {
	t.Parallel()
	for oType, exp := range map[order.Type]bool{
		order.Stop:            true,
		order.StopLimit:       true,
		order.StopMarket:      true,
		order.ConditionalStop: true,
		order.Limit:           false,
		order.OCO:             false,
	} {
		assert.Equalf(t, exp, isStopOrderType(oType), "isStopOrderType should return %t for %s", exp, oType)
	}
}

func TestCancelSpotStopOrders(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.cancelSpotStopOrders(t.Context(), nil), order.ErrOrderIDNotSet, "cancelSpotStopOrders must not cancel every stop order")

	if !mockTests {
		t.Skip("cancelSpotStopOrders' replies come from fixtures; TestCancelOrder covers live cancellation")
	}

	require.ErrorIs(t, e.cancelSpotStopOrders(cancelledContext(t), []string{"vs8hoo8q2ceshiue003b67c0"}), context.Canceled, "cancelSpotStopOrders must return the request error")
	require.ErrorIs(t, e.cancelSpotStopOrders(t.Context(), []string{"gct-null-stop"}), errOrderNotCancelled, "cancelSpotStopOrders must report an order a reply without data does not list")
	require.ErrorIs(t, e.cancelSpotStopOrders(t.Context(), []string{"vs8hoo8q2ceshiue003b67c2", "vs8hoo8q2ceshiue003b67c3"}), errOrderNotCancelled, "cancelSpotStopOrders must report an order KuCoin did not cancel")
	assert.NoError(t, e.cancelSpotStopOrders(t.Context(), []string{"vs8hoo8q2ceshiue003b67c0"}), "cancelSpotStopOrders should cancel a stop order KuCoin reports cancelled")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancellation")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an asset KuCoin does not trade")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		for a, p := range map[asset.Item]currency.Pair{asset.Spot: spotTradablePair, asset.Margin: marginTradablePair, asset.Futures: futuresTradablePair} {
			_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: a, Pair: p})
			assert.NoErrorf(t, err, "CancelAllOrders should not error for %s", a)
		}
		return
	}

	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
	}{
		{name: "the futures orders of every symbol", cancel: &order.Cancel{AssetType: asset.Futures}},
		{name: "a symbol's futures orders", cancel: &order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm}},
		{name: "margin OCO orders", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.OCO}},
		{name: "margin stop orders", cancel: &order.Cancel{AssetType: asset.Margin, Type: order.Stop}},
		{name: "the margin orders of every symbol", cancel: &order.Cancel{AssetType: asset.Margin}},
		{name: "a symbol's margin orders", cancel: &order.Cancel{AssetType: asset.Margin, Pair: btcusdt}},
		{name: "spot OCO orders", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.OCO}},
		{name: "spot stop orders", cancel: &order.Cancel{AssetType: asset.Spot, Type: order.StopLimit}},
		{name: "a symbol's spot orders", cancel: &order.Cancel{AssetType: asset.Spot, Pair: btcusdt}},
		{name: "the spot orders of every symbol", cancel: &order.Cancel{AssetType: asset.Spot}},
	} {
		_, err = e.CancelAllOrders(cancelledContext(t), tc.cancel)
		require.ErrorIsf(t, err, context.Canceled, "CancelAllOrders must return the request error cancelling %s", tc.name)
	}

	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures, Pair: currency.NewPair(currency.SOL, currency.USDTM)})
	require.ErrorContains(t, err, "400100", "CancelAllOrders must return the error of the stop order cancellation")
	cancelled := order.Cancelled.String()
	for _, tc := range []struct {
		name   string
		cancel *order.Cancel
		exp    order.CancelAllResponse
	}{
		{
			name:   "a symbol's futures limit and stop orders",
			cancel: &order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm},
			exp:    order.CancelAllResponse{Status: map[string]string{"235919172150824960": cancelled, "235919172150824961": cancelled, "235919172150824962": cancelled}},
		},
		{
			// KuCoin cancels futures limit orders by symbol, so they are cancelled for each symbol with active orders
			name:   "the futures limit and stop orders of every symbol",
			cancel: &order.Cancel{AssetType: asset.Futures},
			exp:    order.CancelAllResponse{Status: map[string]string{"235919172150824960": cancelled, "235919172150824961": cancelled, "235919172150824970": cancelled, "235919172150824980": cancelled}},
		},
		{
			// An OCO order's two orders are both cancelled
			name:   "an isolated margin OCO order",
			cancel: &order.Cancel{AssetType: asset.Margin, Type: order.OCO, MarginType: margin.Isolated, Pair: btcusdt, OrderID: "674c316e688dea0007c7b986"},
			exp:    order.CancelAllResponse{Status: map[string]string{"674c316e688dea0007c7b986": cancelled, "674c316e688dea0007c7b987": cancelled}},
		},
		{
			name:   "a symbol's cross margin stop orders",
			cancel: &order.Cancel{AssetType: asset.Margin, Type: order.Stop, Pair: btcusdt},
			exp:    order.CancelAllResponse{Status: map[string]string{"vs8t6q2ig66rv9r3003i782s": cancelled, "vs8t6q2ig68nhgf9003vcd8c": cancelled}},
		},
		{
			name:   "a symbol's cross margin orders",
			cancel: &order.Cancel{AssetType: asset.Margin, Pair: btcusdt},
			exp:    order.CancelAllResponse{Status: map[string]string{"BTC-USDT": cancelled}},
		},
		{
			name:   "the isolated margin orders of every symbol",
			cancel: &order.Cancel{AssetType: asset.Margin, MarginType: margin.Isolated},
			exp:    order.CancelAllResponse{Status: map[string]string{"BTC-USDT": cancelled, "ETH-USDT": cancelled}},
		},
		{
			name:   "a symbol's spot OCO orders",
			cancel: &order.Cancel{AssetType: asset.Spot, Type: order.OCO, Pair: btcusdt},
			exp:    order.CancelAllResponse{Status: map[string]string{"674c388172cf2800072ee746": cancelled, "674c388172cf2800072ee747": cancelled}},
		},
		{
			// A client order ID is not an order ID, so it is not sent as one
			name:   "a spot stop order",
			cancel: &order.Cancel{AssetType: asset.Spot, Type: order.StopLimit, Pair: btcusdt, OrderID: "vs8hoo8q2ceshiue003b67c0", ClientOrderID: "gct-spot-stop"},
			exp:    order.CancelAllResponse{Status: map[string]string{"vs8hoo8q2ceshiue003b67c0": cancelled}},
		},
		{
			name:   "a symbol's spot orders",
			cancel: &order.Cancel{AssetType: asset.Spot, Pair: btcusdt},
			exp:    order.CancelAllResponse{Status: map[string]string{"BTC-USDT": cancelled}},
		},
		{
			// KuCoin reports the symbols whose orders it could not cancel, with the reason
			name:   "the spot orders of every symbol",
			cancel: &order.Cancel{AssetType: asset.Spot},
			exp:    order.CancelAllResponse{Status: map[string]string{"ETH-USDT": cancelled, "BTC-USDT": cancelled, "KCS-USDT": "Order cancellation failed, please try again later."}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.CancelAllOrders(t.Context(), tc.cancel)
			require.NoError(t, err, "CancelAllOrders must not error")
			assert.Equal(t, tc.exp, resp, "CancelAllOrders should report the cancelled orders")
		})
	}
}

func TestOpenOrderSymbols(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		_, err := e.openOrderSymbols(t.Context(), asset.Spot, SpotTradeType)
		assert.NoError(t, err, "openOrderSymbols should not error")
		return
	}

	_, err := e.openOrderSymbols(cancelledContext(t), asset.Margin, CrossMarginTradeType)
	require.ErrorIs(t, err, context.Canceled, "openOrderSymbols must return the margin request error")
	_, err = e.openOrderSymbols(cancelledContext(t), asset.Spot, SpotTradeType)
	require.ErrorIs(t, err, context.Canceled, "openOrderSymbols must return the spot request error")
	got, err := e.openOrderSymbols(t.Context(), asset.Margin, CrossMarginTradeType)
	require.NoError(t, err, "openOrderSymbols must not error for margin")
	assert.Equal(t, []string{"BTC-USDT", "ETH-USDT"}, got, "openOrderSymbols should return the symbols with open cross margin orders")
	got, err = e.openOrderSymbols(t.Context(), asset.Spot, SpotTradeType)
	require.NoError(t, err, "openOrderSymbols must not error for spot")
	assert.Equal(t, []string{"ETH-USDT", "BTC-USDT"}, got, "openOrderSymbols should return the symbols with open spot orders")
}

// clearSavedTimes checks each of a sub-account's balances was stamped when it was saved and clears the stamp, so the
// sub-account can be compared with an expected one
func clearSavedTimes(t *testing.T, s *accounts.SubAccount) {
	t.Helper()
	for c, b := range s.Balances {
		assert.WithinDurationf(t, time.Now(), b.UpdatedAt, time.Minute, "the %s balance should be stamped when saved", c)
		b.UpdatedAt = time.Time{}
		s.Balances[c] = b
	}
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an asset KuCoin does not trade")
	storeOptionsPair(t, ex)
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an enabled asset KuCoin does not trade")
	ex = marketTestServerExchange(t, nil)
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		_, err = ex.UpdateAccountBalances(cancelledContext(t), a)
		require.ErrorIsf(t, err, context.Canceled, "UpdateAccountBalances must return the failed %s request's error", a)
	}

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/account-overview": `{"code":"200000","data":null}`,
		"/api/v3/margin/accounts":  `{"code":"200000","data":null}`,
		"/api/v1/accounts":         `{"code":"200000","data":[{"currency":"","type":"trade","balance":"1"}]}`,
	})
	for _, a := range []asset.Item{asset.Margin, asset.Futures} {
		_, err = ex.UpdateAccountBalances(t.Context(), a)
		require.ErrorIsf(t, err, common.ErrNoResponse, "UpdateAccountBalances must reject a %s reply without an account", a)
	}
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "UpdateAccountBalances must reject a spot account without a currency")

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/accounts": `{"code":"200000","data":[
			{"id":"548674591753","currency":"BTC","type":"trade","balance":"0.5","available":"0.4","holds":"0.1"},
			{"id":"548674591754","currency":"BTC","type":"margin","balance":"7","available":"7","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"BTC","type":"main","balance":"0.25","available":"0.25","holds":"0"}
		]}`,
	})
	result, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	require.Len(t, result, 1, "UpdateAccountBalances must return one sub-account")
	clearSavedTimes(t, result[0])
	exp := &accounts.SubAccount{
		AssetType: asset.Spot,
		Balances: accounts.CurrencyBalances{
			currency.BTC: {Currency: currency.BTC, Total: 0.75, Hold: 0.1, Free: 0.65},
		},
	}
	assert.Equal(t, exp, result[0], "UpdateAccountBalances should add the main and trade accounts and leave out the abandoned margin type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ex = testInstance(t)
	for _, tc := range []struct {
		asset asset.Item
		exp   accounts.CurrencyBalances
	}{
		{
			asset: asset.Spot,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.01, Free: 0.01},
				currency.USDT: {Currency: currency.USDT, Total: 26.66759503, Hold: 6, Free: 20.66759503},
			},
		},
		{
			asset: asset.Margin,
			exp: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.0001, Hold: 0.00002, Free: 0.00008, Borrowed: 0.000001},
				currency.USDT: {Currency: currency.USDT, Total: 38.68855864, Hold: 18.66939173, Free: 20.01916691, Borrowed: 12.5},
			},
		},
		{
			// Hold is the position margin, order margin and frozen funds
			asset: asset.Futures,
			exp: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 198.733127406, Hold: 25.8, Free: 172.933127406},
				currency.USDC: {Currency: currency.USDC, Total: 50.5, Hold: 12.5, Free: 38},
				currency.XBT:  {Currency: currency.XBT, Total: 0.0025, Hold: 0.0007000000000000001, Free: 0.0018},
				currency.ETH:  {Currency: currency.ETH, Total: 0.15, Hold: 0.034999999999999996, Free: 0.115},
				currency.SOL:  {Currency: currency.SOL, Total: 3.5, Hold: 1.5, Free: 2},
				currency.XRP:  {Currency: currency.XRP, Total: 120, Hold: 45, Free: 75},
			},
		},
	} {
		result, err := ex.UpdateAccountBalances(t.Context(), tc.asset)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", tc.asset)
		require.Lenf(t, result, 1, "UpdateAccountBalances must return one %s sub-account", tc.asset)
		if mockTests {
			clearSavedTimes(t, result[0])
			assert.Equalf(t, &accounts.SubAccount{AssetType: tc.asset, Balances: tc.exp}, result[0], "UpdateAccountBalances should return the %s balances", tc.asset)
			continue
		}
		assert.Equalf(t, tc.asset, result[0].AssetType, "UpdateAccountBalances should return a %s sub-account", tc.asset)
	}
}

func TestContractFundingRate(t *testing.T) {
	t.Parallel()
	next := time.UnixMilli(1791907200000)
	for _, tc := range []struct {
		name     string
		contract *Contract
		exp      fundingrate.Rate
	}{
		{
			name:     "current interval",
			contract: &Contract{FundingFeeRate: 0.0001, NextFundingRateDateTime: types.Time(next), FundingRateGranularity: 28800000, CurrentFundingRateGranularity: 14400000},
			exp:      fundingrate.Rate{Time: next.Add(-4 * time.Hour), Rate: decimal.MustFromFloat(0.0001)},
		},
		{
			name:     "configured interval when the current one is null",
			contract: &Contract{FundingFeeRate: -0.0007, NextFundingRateDateTime: types.Time(next), FundingRateGranularity: 28800000},
			exp:      fundingrate.Rate{Time: next.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(-0.0007)},
		},
		{
			name:     "no funding",
			contract: &Contract{},
			exp:      fundingrate.Rate{Rate: decimal.MustFromFloat(0)},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, contractFundingRate(tc.contract), "contractFundingRate should time the rate at the start of its funding period")
		})
	}
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	require.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails must reject a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.USDTMarginedFutures)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails must reject a futures asset KuCoin does not trade")
	_, err = e.GetFuturesContractDetails(cancelledContext(t), asset.Futures)
	require.ErrorIs(t, err, context.Canceled, "GetFuturesContractDetails must return the failed request's error")
	ex := marketTestServerExchange(t, map[string]string{"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"ETH"}]}`})
	_, err = ex.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.ErrorIs(t, err, errContractSymbolMismatch, "GetFuturesContractDetails must reject a contract whose symbol does not start with its base currency")

	result, err := e.GetFuturesContractDetails(t.Context(), asset.Futures)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	if mockTests {
		exp := []futures.Contract{
			{
				Exchange:           e.Name,
				Name:               currency.NewPair(currency.XBT, currency.USDTM),
				Underlying:         currency.NewPair(currency.XBT, currency.USDT),
				Asset:              asset.Futures,
				StartDate:          time.UnixMilli(1585555200000),
				IsActive:           true,
				Status:             "Open",
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				MarginCurrency:     currency.USDT,
				Multiplier:         0.001,
				MaxLeverage:        125,
				LatestRate:         fundingrate.Rate{Time: time.UnixMilli(1791878400000), Rate: decimal.MustFromFloat(0.000051)},
				FundingRateFloor:   decimal.MustFromFloat(-0.003),
				FundingRateCeiling: decimal.MustFromFloat(0.003),
			},
			{
				Exchange:           e.Name,
				Name:               currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDTM),
				Underlying:         currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDT),
				Asset:              asset.Futures,
				StartDate:          time.UnixMilli(1780380000000),
				IsActive:           true,
				Status:             "Open",
				Type:               futures.Perpetual,
				SettlementType:     futures.Linear,
				SettlementCurrency: currency.USDT,
				MarginCurrency:     currency.USDT,
				Multiplier:         0.001,
				MaxLeverage:        20,
				LatestRate:         fundingrate.Rate{Time: time.UnixMilli(1791878400000), Rate: decimal.MustFromFloat(0.00005)},
				FundingRateFloor:   decimal.MustFromFloat(-0.02),
				FundingRateCeiling: decimal.MustFromFloat(0.02),
			},
			{
				// A dated contract has no funding
				Exchange:           e.Name,
				Name:               currency.NewPair(currency.XBT, currency.NewCode("MZ26")),
				Underlying:         currency.NewPair(currency.XBT, currency.USD),
				Asset:              asset.Futures,
				StartDate:          time.UnixMilli(1790071200000),
				EndDate:            time.UnixMilli(1798185600000),
				IsActive:           true,
				Status:             "Open",
				Type:               futures.Quarterly,
				SettlementType:     futures.Inverse,
				SettlementCurrency: currency.XBT,
				MarginCurrency:     currency.XBT,
				Multiplier:         -1,
				MaxLeverage:        20,
				LatestRate:         fundingrate.Rate{Rate: decimal.MustFromFloat(0)},
				FundingRateFloor:   decimal.MustFromFloat(0),
				FundingRateCeiling: decimal.MustFromFloat(0),
			},
		}
		assert.Equal(t, exp, result, "GetFuturesContractDetails should return every contract with its current funding rate")
		return
	}
	require.NotEmpty(t, result, "GetFuturesContractDetails must return contracts")
	assert.NotEmpty(t, result[0].Name, "GetFuturesContractDetails should return a contract name")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject a non-futures asset")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must reject a predicted rate request, which KuCoin cannot serve")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.XBT, currency.NewCode("MZ26"))})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates must reject a dated contract")
	perpetual := &fundingrate.LatestRateRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.XBT, currency.USDTM)}
	for _, r := range []*fundingrate.LatestRateRequest{perpetual, {Asset: asset.Futures}} {
		_, err = e.GetLatestFundingRates(cancelledContext(t), r)
		require.ErrorIsf(t, err, context.Canceled, "GetLatestFundingRates must return the failed request's error for pair %q", r.Pair)
	}
	ex := testInstance(t)
	ex.CurrencyPairs.UseGlobalFormat = true
	_, err = ex.GetLatestFundingRates(t.Context(), perpetual)
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetLatestFundingRates must return the pair formatting error")
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/funding-rate/XBTUSDTM/current": `{"code":"200000","data":null}`,
		"/api/v1/contracts/active":              `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"ETH","type":"FFWCSX"}]}`,
	})
	_, err = ex.GetLatestFundingRates(t.Context(), perpetual)
	require.ErrorIs(t, err, common.ErrNoResponse, "GetLatestFundingRates must reject a reply without a funding rate")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, errContractSymbolMismatch, "GetLatestFundingRates must reject a contract whose symbol does not start with its base currency")

	latestRate := func(pair currency.Pair, rate float64, start time.Time) fundingrate.LatestRateResponse {
		return fundingrate.LatestRateResponse{
			Exchange:       e.Name,
			Asset:          asset.Futures,
			Pair:           pair,
			LatestRate:     fundingrate.Rate{Time: start, Rate: decimal.MustFromFloat(rate)},
			TimeOfNextRate: time.UnixMilli(1791907200000),
		}
	}
	for _, tc := range []struct {
		request *fundingrate.LatestRateRequest
		exp     []fundingrate.LatestRateResponse
	}{
		{
			request: perpetual,
			exp:     []fundingrate.LatestRateResponse{latestRate(perpetual.Pair, 0.000052, time.UnixMilli(1791878400000))},
		},
		{
			// Every perpetual contract; the dated XBTMZ26 has no funding
			request: &fundingrate.LatestRateRequest{Asset: asset.Futures},
			exp: []fundingrate.LatestRateResponse{
				latestRate(currency.NewPair(currency.XBT, currency.USDTM), 0.000051, time.UnixMilli(1791878400000)),
				latestRate(currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDTM), 0.00005, time.UnixMilli(1791878400000)),
			},
		},
	} {
		result, err := e.GetLatestFundingRates(t.Context(), tc.request)
		require.NoErrorf(t, err, "GetLatestFundingRates must not error for pair %q", tc.request.Pair)
		require.NotEmptyf(t, result, "GetLatestFundingRates must return rates for pair %q", tc.request.Pair)
		for i := range result {
			assert.WithinDurationf(t, time.Now(), result[i].TimeChecked, time.Minute, "GetLatestFundingRates should stamp the %s rate with the time checked", result[i].Pair)
			result[i].TimeChecked = time.Time{}
		}
		if mockTests {
			assert.Equalf(t, tc.exp, result, "GetLatestFundingRates should return the current rates for pair %q", tc.request.Pair)
			continue
		}
		assert.NotZerof(t, result[0].TimeOfNextRate, "GetLatestFundingRates should return the next funding time for pair %q", tc.request.Pair)
	}
}

func TestIsPerpetualPair(t *testing.T) {
	t.Parallel()
	for pair, exp := range map[currency.Pair]bool{
		currency.NewPair(currency.XBT, currency.USDTM):           true,
		currency.NewPair(currency.XBT, currency.USDCM):           true,
		currency.NewPair(currency.XBT, currency.USDM):            true,
		currency.NewPair(currency.XBT, currency.NewCode("MZ26")): false,
		currency.EMPTYPAIR: false,
	} {
		assert.Equalf(t, exp, isPerpetualPair(pair), "isPerpetualPair should identify perpetual contracts for %q", pair)
	}
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		asset asset.Item
		pair  currency.Pair
		exp   bool
	}{
		{asset: asset.Futures, pair: currency.NewPair(currency.XBT, currency.USDTM), exp: true},
		{asset: asset.Futures, pair: currency.NewPair(currency.XBT, currency.USDCM), exp: true},
		{asset: asset.Futures, pair: currency.NewPair(currency.XBT, currency.USDM), exp: true},
		{asset: asset.Futures, pair: currency.NewPair(currency.XBT, currency.EOS)},
		{asset: asset.Futures, pair: currency.EMPTYPAIR},
		{asset: asset.Spot, pair: currency.NewPair(currency.XBT, currency.USDTM)},
	} {
		is, err := e.IsPerpetualFutureCurrency(tc.asset, tc.pair)
		require.NoError(t, err, "IsPerpetualFutureCurrency must not error")
		assert.Equalf(t, tc.exp, is, "IsPerpetualFutureCurrency should identify perpetual contracts for %s %q", tc.asset, tc.pair)
	}
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.GetLeverage(t.Context(), asset.Spot, currency.NewBTCUSDT(), margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage must reject spot, which has no leverage")
	_, err = e.GetLeverage(t.Context(), asset.Futures, xbtusdtm, margin.Isolated, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "GetLeverage must reject isolated margin, whose leverage KuCoin sets per order")
	_, err = e.GetLeverage(t.Context(), asset.Futures, currency.EMPTYPAIR, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must require a pair")
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.GetLeverage(t.Context(), asset.Futures, xbtusdtm, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetLeverage must reject a disabled asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		got, err := e.GetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Multi, order.UnknownSide)
		require.NoError(t, err, "GetLeverage must not error")
		assert.Positive(t, got, "GetLeverage should return the cross margin leverage")
		return
	}

	_, err = e.GetLeverage(cancelledContext(t), asset.Futures, xbtusdtm, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, context.Canceled, "GetLeverage must return the request error")
	_, err = e.GetLeverage(t.Context(), asset.Futures, currency.NewPair(currency.XBT, currency.USDM), margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, common.ErrNoResponse, "GetLeverage must reject a reply without data")
	got, err := e.GetLeverage(t.Context(), asset.Futures, xbtusdtm, margin.Multi, order.Buy)
	require.NoError(t, err, "GetLeverage must not error")
	assert.Equal(t, 3.0, got, "GetLeverage should return the contract's cross margin leverage")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	for _, tc := range []struct {
		name       string
		a          asset.Item
		pair       currency.Pair
		marginType margin.Type
		err        error
	}{
		{name: "an asset KuCoin does not trade", a: asset.Options, pair: btcusdt, marginType: margin.Multi, err: asset.ErrNotSupported},
		{name: "spot, which has no leverage", a: asset.Spot, pair: btcusdt, marginType: margin.Multi, err: asset.ErrNotSupported},
		{name: "isolated futures, whose leverage KuCoin sets per order", a: asset.Futures, pair: xbtusdtm, marginType: margin.Isolated, err: margin.ErrMarginTypeUnsupported},
		{name: "cross futures without a pair", a: asset.Futures, marginType: margin.Multi, err: currency.ErrCurrencyPairEmpty},
		{name: "isolated margin without a pair", a: asset.Margin, marginType: margin.Isolated, err: currency.ErrCurrencyPairEmpty},
	} {
		require.ErrorIsf(t, e.SetLeverage(t.Context(), tc.a, tc.pair, tc.marginType, 10, order.UnknownSide), tc.err, "SetLeverage must reject %s", tc.name)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		assert.NoError(t, e.SetLeverage(t.Context(), asset.Futures, futuresTradablePair, margin.Multi, 3, order.UnknownSide), "SetLeverage should not error")
		return
	}

	require.ErrorIs(t, e.SetLeverage(cancelledContext(t), asset.Futures, xbtusdtm, margin.Multi, 10, order.UnknownSide), context.Canceled, "SetLeverage must return the request error")
	require.ErrorIs(t, e.SetLeverage(t.Context(), asset.Futures, xbtusdtm, margin.Multi, 20, order.UnknownSide), errLeverageNotModified, "SetLeverage must report leverage KuCoin did not modify")
	for _, tc := range []struct {
		name       string
		a          asset.Item
		pair       currency.Pair
		marginType margin.Type
		leverage   float64
	}{
		{name: "futures cross margin", a: asset.Futures, pair: xbtusdtm, marginType: margin.Multi, leverage: 10},
		{name: "the cross margin account", a: asset.Margin, marginType: margin.Multi, leverage: 5},
		{name: "an isolated margin pair", a: asset.Margin, pair: btcusdt, marginType: margin.Isolated, leverage: 3.5},
	} {
		assert.NoErrorf(t, e.SetLeverage(t.Context(), tc.a, tc.pair, tc.marginType, tc.leverage, order.UnknownSide), "SetLeverage should set the leverage of %s", tc.name)
	}
}

func TestSetMarginType(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	require.ErrorIs(t, e.SetMarginType(t.Context(), asset.Margin, currency.NewBTCUSDT(), margin.Isolated), asset.ErrNotSupported, "SetMarginType must reject margin, whose orders choose their margin type")
	require.ErrorIs(t, e.SetMarginType(t.Context(), asset.Futures, xbtusdtm, margin.Unset), margin.ErrMarginTypeUnsupported, "SetMarginType must require a margin type")
	require.ErrorIs(t, e.SetMarginType(t.Context(), asset.Futures, currency.EMPTYPAIR, margin.Multi), currency.ErrCurrencyPairEmpty, "SetMarginType must require a pair")
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	require.ErrorIs(t, ex.SetMarginType(t.Context(), asset.Futures, xbtusdtm, margin.Multi), asset.ErrNotEnabled, "SetMarginType must reject a disabled asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		assert.NoError(t, e.SetMarginType(t.Context(), asset.Futures, futuresTradablePair, margin.Multi), "SetMarginType should not error")
		return
	}
	assert.NoError(t, e.SetMarginType(t.Context(), asset.Futures, xbtusdtm, margin.Multi), "SetMarginType should switch the contract to cross margin")
}

func TestSetCollateralMode(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.SetCollateralMode(t.Context(), asset.Futures, collateral.MultiMode), common.ErrFunctionNotSupported, "SetCollateralMode should report that KuCoin has no collateral modes to set")
}

func TestGetCollateralMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetCollateralMode(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetCollateralMode should report that KuCoin has no collateral modes to return")
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.ChangePositionMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ChangePositionMargin must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *margin.PositionChangeRequest
		err  error
	}{
		{name: "an asset other than futures", req: &margin.PositionChangeRequest{Asset: asset.Spot, Pair: xbtusdtm, MarginType: margin.Isolated, NewAllocatedMargin: 2}, err: futures.ErrNotFuturesAsset},
		{name: "a request without a pair", req: &margin.PositionChangeRequest{Asset: asset.Futures, MarginType: margin.Isolated, NewAllocatedMargin: 2}, err: currency.ErrCurrencyPairEmpty},
		{name: "a cross margin position", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Multi, NewAllocatedMargin: 2}, err: margin.ErrMarginTypeUnsupported},
		{name: "a request without a new margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated}, err: margin.ErrNewAllocatedMarginRequired},
		{name: "an unchanged margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 2, NewAllocatedMargin: 2}, err: errMarginUnchanged},
	} {
		_, err = e.ChangePositionMargin(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ChangePositionMargin must reject %s", tc.name)
	}
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, NewAllocatedMargin: 2})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "ChangePositionMargin must reject a disabled asset")

	t.Run("local server", func(t *testing.T) {
		t.Parallel()
		// The business number that makes KuCoin add margin once is random, so the recorded mock cannot match the request
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			var params map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&params), "decoding the request body should not error")
			reply := `{"code":"200000","data":null}`
			switch {
			case r.URL.Path == "/v1/position/margin/deposit-margin" && params["symbol"] == "XBTUSDTM":
				bizNo, ok := params["bizNo"].(string)
				assert.True(t, ok, "the business number should be a string")
				assert.NotEmpty(t, bizNo, "the business number should be set")
				assert.LessOrEqual(t, len(bizNo), 36, "the business number should be at most 36 characters")
				delete(params, "bizNo")
				assert.Equal(t, map[string]any{"symbol": "XBTUSDTM", "margin": 0.2, "positionSide": "LONG"}, params, "ChangePositionMargin should add the margin difference")
				reply = `{"code":"200000","data":{"id":"6200c9b83aecfb000152ddcd","symbol":"XBTUSDTM","posMargin":20.4728500039,"settleCurrency":"USDT"}}`
			case r.URL.Path == "/v1/margin/withdrawMargin":
				assert.Equal(t, map[string]any{"symbol": "XBTUSDTM", "withdrawAmount": "0.5"}, params, "ChangePositionMargin should withdraw the margin difference")
				reply = `{"code":"200000","data":"0.5"}`
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.ETH, currency.USDTM), MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2})
		require.ErrorIs(t, err, common.ErrNoResponse, "ChangePositionMargin must reject a reply without the position")
		resp, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 1.1, NewAllocatedMargin: 1.3, MarginSide: "long"})
		require.NoError(t, err, "ChangePositionMargin must not error adding margin")
		exp := &margin.PositionChangeResponse{Exchange: ex.Name, Pair: xbtusdtm, Asset: asset.Futures, AllocatedMargin: 20.4728500039, MarginType: margin.Isolated}
		assert.Equal(t, exp, resp, "ChangePositionMargin should return the position's margin after adding to it")
		resp, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 1.5, NewAllocatedMargin: 1})
		require.NoError(t, err, "ChangePositionMargin must not error withdrawing margin")
		exp = &margin.PositionChangeResponse{Exchange: ex.Name, Pair: xbtusdtm, Asset: asset.Futures, AllocatedMargin: 1, MarginType: margin.Isolated}
		assert.Equal(t, exp, resp, "ChangePositionMargin should return the margin left after withdrawing")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		_, err = e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: futuresTradablePair, MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 1.1})
		assert.NoError(t, err, "ChangePositionMargin should add margin to an isolated margin position")
		return
	}

	for _, tc := range []struct {
		name     string
		original float64
	}{
		{name: "added", original: 1},
		{name: "withdrawn", original: 3},
	} {
		_, err = e.ChangePositionMargin(cancelledContext(t), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: tc.original, NewAllocatedMargin: 2})
		require.ErrorIsf(t, err, context.Canceled, "ChangePositionMargin must return the request error of margin %s", tc.name)
	}
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.GetFuturesPositionSummary(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionSummary must reject a nil request")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: xbtusdtm})
	require.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetFuturesPositionSummary must reject an asset other than futures")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPositionSummary must require a pair")
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdtm})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetFuturesPositionSummary must reject a disabled asset")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		// An account may hold no position in the contract
		_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: futuresTradablePair, Direction: order.Long})
		assert.Truef(t, err == nil || errors.Is(err, futures.ErrPositionNotFound), "GetFuturesPositionSummary should not error: %v", err)
		return
	}

	_, err = e.GetFuturesPositionSummary(cancelledContext(t), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdtm})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesPositionSummary must return the request error")
	for _, tc := range []struct {
		name string
		req  *futures.PositionSummaryRequest
		err  error
	}{
		{name: "a hedge mode position without a direction", req: &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdtm}, err: errPositionDirectionRequired},
		{name: "a position whose account KuCoin does not return", req: &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.DOT, currency.USDM)}, err: common.ErrNoResponse},
	} {
		_, err = e.GetFuturesPositionSummary(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetFuturesPositionSummary must reject %s", tc.name)
	}
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.ADA, currency.USDTM)})
	require.ErrorContains(t, err, "400100", "GetFuturesPositionSummary must return the position request's error")
	_, err = e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: currency.NewPair(currency.LTC, currency.USDM)})
	require.ErrorContains(t, err, "400100", "GetFuturesPositionSummary must return the account request's error")
	xbtusdm := currency.NewPair(currency.XBT, currency.USDM)
	for _, tc := range []struct {
		name string
		req  *futures.PositionSummaryRequest
		exp  *futures.PositionSummary
	}{
		{
			// The long position of a hedge mode contract, with the account of its settlement currency, USDT
			name: "hedge mode long",
			req:  &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdtm, Direction: order.Long},
			exp: &futures.PositionSummary{
				Pair: xbtusdtm, Asset: asset.Futures, MarginType: margin.Isolated, CollateralMode: collateral.MultiMode,
				Currency: currency.USDT, StartDate: time.UnixMilli(1759395308117),
				AvailableEquity:              decimal.MustFromFloat(198.733127406),
				MarginBalance:                decimal.MustFromFloat(197.233127406),
				NotionalSize:                 decimal.MustFromFloat(118.6042),
				Leverage:                     decimal.MustFromFloat(6.01),
				MaintenanceMarginRequirement: decimal.MustFromFloat(0.004),
				InitialMarginRequirement:     decimal.MustFromFloat(19.7728500039),
				EstimatedLiquidationPrice:    decimal.MustFromFloat(99321.13),
				CollateralUsed:               decimal.MustFromFloat(118.6371),
				MarkPrice:                    decimal.MustFromFloat(118604.2),
				CurrentSize:                  decimal.NewFromInt(1),
				ContractSize:                 decimal.NewFromInt(1).Mul(decimal.MustFromFloat(0.001)),
				ContractMultiplier:           decimal.MustFromFloat(0.001),
				ContractSettlementType:       futures.Linear,
				AverageOpenPrice:             decimal.MustFromFloat(118637.1),
				UnrealisedPNL:                decimal.MustFromFloat(-0.0329),
				RealisedPNL:                  decimal.MustFromFloat(-0.04968226),
				MaintenanceMarginFraction:    decimal.MustFromFloat(0.004),
				FreeCollateral:               decimal.MustFromFloat(172.933127406),
				TotalCollateral:              decimal.MustFromFloat(198.733127406),
				FrozenBalance:                decimal.MustFromFloat(0.5),
			},
		},
		{
			// The contract is not listed, so its multiplier and settlement type are unknown
			name: "one-way mode",
			req:  &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdm},
			exp: &futures.PositionSummary{
				Pair: xbtusdm, Asset: asset.Futures, MarginType: margin.Multi, CollateralMode: collateral.MultiMode,
				Currency: currency.XBT, StartDate: time.UnixMilli(1759401696659),
				AvailableEquity:              decimal.MustFromFloat(0.0025),
				MarginBalance:                decimal.MustFromFloat(0.0024),
				NotionalSize:                 decimal.MustFromFloat(0.00084225),
				Leverage:                     decimal.MustFromFloat(1),
				MaintenanceMarginRequirement: decimal.MustFromFloat(0.0040000133),
				InitialMarginRequirement:     decimal.MustFromFloat(0.00084226),
				EstimatedLiquidationPrice:    decimal.MustFromFloat(0.01),
				CollateralUsed:               decimal.MustFromFloat(0.00084226),
				MarkPrice:                    decimal.MustFromFloat(118728.55),
				CurrentSize:                  decimal.NewFromInt(100),
				ContractSize:                 decimal.NewFromInt(100).Mul(decimal.MustFromFloat(0)),
				ContractMultiplier:           decimal.MustFromFloat(0),
				AverageOpenPrice:             decimal.MustFromFloat(118727.9),
				UnrealisedPNL:                decimal.MustFromFloat(-0.00000001),
				RealisedPNL:                  decimal.MustFromFloat(-0.00000051),
				MaintenanceMarginFraction:    decimal.MustFromFloat(0.0040000133),
				FreeCollateral:               decimal.MustFromFloat(0.0018),
				TotalCollateral:              decimal.MustFromFloat(0.0025),
				FrozenBalance:                decimal.MustFromFloat(0.0001),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := e.GetFuturesPositionSummary(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesPositionSummary must not error")
			assert.Equal(t, tc.exp, got, "GetFuturesPositionSummary should summarise the position and its account")
		})
	}
}

func TestFuturesPosition(t *testing.T) {
	t.Parallel()
	both := FuturesPositionDetail{ID: "1", PositionSide: "BOTH"}
	long := FuturesPositionDetail{ID: "2", PositionSide: "LONG"}
	short := FuturesPositionDetail{ID: "3", PositionSide: "SHORT"}
	_, err := futuresPosition(nil, order.Long)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "futuresPosition must report a contract without positions")
	_, err = futuresPosition([]FuturesPositionDetail{long, short}, order.UnknownSide)
	require.ErrorIs(t, err, errPositionDirectionRequired, "futuresPosition must require a direction in hedge mode")
	_, err = futuresPosition([]FuturesPositionDetail{short}, order.Buy)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "futuresPosition must report a direction without a position")
	for _, tc := range []struct {
		name      string
		positions []FuturesPositionDetail
		direction order.Side
		exp       string
	}{
		{name: "the one-way mode position", positions: []FuturesPositionDetail{both}, exp: "1"},
		{name: "the one-way mode position for a direction", positions: []FuturesPositionDetail{both}, direction: order.Short, exp: "1"},
		{name: "the hedge mode long position", positions: []FuturesPositionDetail{long, short}, direction: order.Long, exp: "2"},
		{name: "the hedge mode short position", positions: []FuturesPositionDetail{long, short}, direction: order.Sell, exp: "3"},
	} {
		got, err := futuresPosition(tc.positions, tc.direction)
		require.NoErrorf(t, err, "futuresPosition must not error returning %s", tc.name)
		assert.Equalf(t, tc.exp, got.ID, "futuresPosition should return %s", tc.name)
	}
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	end := time.UnixMilli(1729500000000)
	start := end.Add(-24 * time.Hour)
	_, err := e.GetFuturesPositionOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPositionOrders must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *futures.PositionsRequest
		err  error
	}{
		{name: "an asset other than futures", req: &futures.PositionsRequest{Asset: asset.Spot, Pairs: currency.Pairs{xbtusdtm}, StartDate: start, EndDate: end}, err: futures.ErrNotPerpetualFuture},
		{name: "a request without pairs", req: &futures.PositionsRequest{Asset: asset.Futures, StartDate: start, EndDate: end}, err: currency.ErrCurrencyPairEmpty},
		{name: "a request without dates", req: &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}}, err: common.ErrDateUnset},
		{name: "a window longer than KuCoin lists", req: &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: end.Add(-30 * 24 * time.Hour), EndDate: end}, err: futures.ErrOrderHistoryTooLarge},
		{name: "a start after now without an end", req: &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: time.Now().Add(time.Hour)}, err: common.ErrStartAfterEnd},
		{name: "a start more than seven days ago without an end", req: &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: time.Now().Add(-30 * 24 * time.Hour)}, err: futures.ErrOrderHistoryTooLarge},
	} {
		_, err = e.GetFuturesPositionOrders(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetFuturesPositionOrders must reject %s", tc.name)
	}
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: start, EndDate: end})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetFuturesPositionOrders must reject a disabled asset")

	t.Run("local server", func(t *testing.T) {
		t.Parallel()
		// The order manager sets only the start, so the window ends now, which the recorded mock cannot match
		queries := make(chan url.Values, 1)
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			reply := `{"code":"200000","data":null}`
			switch r.URL.Path {
			case "/v1/contracts/active":
				reply = `{"code":"200000","data":[{"symbol":"XBTUSDTM","type":"FFWCSX","baseCurrency":"XBT","quoteCurrency":"USDT","settleCurrency":"USDT","multiplier":0.001}]}`
			case "/v1/orders":
				select {
				case queries <- r.URL.Query():
				default:
					assert.Fail(t, "GetFuturesPositionOrders should request one page of orders")
				}
				reply = `{"code":"200000","data":{"currentPage":1,"pageSize":1000,"totalNum":0,"totalPage":1,"items":[]}}`
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		before := time.Now()
		req := &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: before.Add(-30 * 24 * time.Hour), RespectOrderHistoryLimits: true}
		got, err := ex.GetFuturesPositionOrders(t.Context(), req)
		require.NoError(t, err, "GetFuturesPositionOrders must not error without an end date")
		assert.WithinRange(t, req.EndDate, before, time.Now(), "GetFuturesPositionOrders should end the window now")
		assert.Equal(t, req.EndDate.Add(-7*24*time.Hour), req.StartDate, "GetFuturesPositionOrders should shorten the window to seven days")
		query := <-queries
		assert.Equal(t, strconv.FormatInt(req.StartDate.UnixMilli(), 10), query.Get("startAt"), "GetFuturesPositionOrders should request orders from the window's start")
		assert.Equal(t, strconv.FormatInt(req.EndDate.UnixMilli(), 10), query.Get("endAt"), "GetFuturesPositionOrders should request orders up to now")
		exp := []futures.PositionResponse{{Pair: xbtusdtm, Asset: asset.Futures, ContractSettlementType: futures.Linear, Orders: []order.Detail{}}}
		assert.Equal(t, exp, got, "GetFuturesPositionOrders should return the pair without orders")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
		now := time.Now()
		_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{futuresTradablePair}, StartDate: now.Add(-24 * time.Hour), EndDate: now})
		assert.NoError(t, err, "GetFuturesPositionOrders should not error")
		return
	}

	_, err = e.GetFuturesPositionOrders(cancelledContext(t), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: start, EndDate: end})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesPositionOrders must return the request error")
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPair(currency.ADA, currency.USDTM)}, StartDate: start, EndDate: end})
	require.ErrorContains(t, err, "400100", "GetFuturesPositionOrders must return the order request's error")
	_, err = e.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{currency.NewPair(currency.ETH, currency.USDTM)}, StartDate: start, EndDate: end})
	require.ErrorIs(t, err, currency.ErrPairNotFound, "GetFuturesPositionOrders must reject an order of an unlisted contract")
	// KuCoin lists closed orders over seven days at most, so the window is shortened from its start
	req := &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, StartDate: end.Add(-30 * 24 * time.Hour), EndDate: end, RespectOrderHistoryLimits: true}
	got, err := e.GetFuturesPositionOrders(t.Context(), req)
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	assert.Equal(t, end.Add(-7*24*time.Hour), req.StartDate, "GetFuturesPositionOrders should shorten the window to seven days")
	exp := []futures.PositionResponse{
		{
			Pair: xbtusdtm, Asset: asset.Futures, ContractSettlementType: futures.Linear,
			Orders: []order.Detail{
				{
					// Amount is the four lots in the base currency, at the contract's multiplier of 0.001
					Exchange: e.Name, OrderID: "235919172150825005", ClientOrderID: "gct-position-order",
					Pair:      currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter),
					AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.Limit, Side: order.Buy,
					Status: order.Filled, Price: 65000, Amount: 0.004, ContractAmount: 4, ExecutedAmount: 4,
					AverageExecutedPrice: 65000, Cost: 260, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
					Leverage: 3, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729300000000),
					LastUpdated: time.UnixMilli(1729400000000), CloseTime: time.UnixMilli(1729400000000),
				},
			},
		},
	}
	assert.Equal(t, exp, got, "GetFuturesPositionOrders should return the pair's closed orders")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.ETH.Item, Quote: currency.USDT.Item, Asset: asset.USDTMarginedFutures})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject an asset KuCoin does not trade")
	_, err = e.GetOpenInterest(cancelledContext(t))
	require.ErrorIs(t, err, context.Canceled, "GetOpenInterest must return the failed request's error")
	ex := marketTestServerExchange(t, map[string]string{"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"ETH"}]}`})
	_, err = ex.GetOpenInterest(t.Context())
	require.ErrorIs(t, err, errContractSymbolMismatch, "GetOpenInterest must reject a contract whose symbol does not start with its base currency")

	ensureTradablePairs(t)
	xbt := futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, asset.Futures, currency.NewPair(currency.XBT, currency.USDTM)), OpenInterest: 9510596}
	for _, tc := range []struct {
		name string
		keys []key.PairAsset
		exp  []futures.OpenInterest
	}{
		{
			name: "every contract",
			exp: []futures.OpenInterest{
				xbt,
				{Key: key.NewExchangeAssetPair(e.Name, asset.Futures, currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDTM)), OpenInterest: 291418},
				{Key: key.NewExchangeAssetPair(e.Name, asset.Futures, currency.NewPair(currency.XBT, currency.NewCode("MZ26"))), OpenInterest: 17480},
			},
		},
		{
			name: "requested contracts",
			keys: []key.PairAsset{
				{Base: futuresTradablePair.Base.Item, Quote: futuresTradablePair.Quote.Item, Asset: asset.Futures},
				{Base: currency.ETH.Item, Quote: currency.USDCM.Item, Asset: asset.Futures},
			},
			exp: []futures.OpenInterest{xbt},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOpenInterest(t.Context(), tc.keys...)
			require.NoError(t, err, "GetOpenInterest must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOpenInterest should return the contracts' open interest")
				return
			}
			assert.NotEmpty(t, result, "GetOpenInterest should return open interest")
		})
	}
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Options, currency.NewBTCUSDT())
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an asset KuCoin does not trade")
	_, err = e.GetCurrencyTradeURL(t.Context(), asset.Spot, currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCurrencyTradeURL must reject an empty pair")
	ex := testInstance(t)
	_, err = ex.GetCurrencyTradeURL(t.Context(), asset.Options, storeOptionsPair(t, ex))
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL must reject an enabled asset KuCoin does not trade")

	ensureTradablePairs(t)
	for a, exp := range map[asset.Item]string{
		asset.Spot:    "https://www.kucoin.com/trade/BTC-USDT",
		asset.Margin:  "https://www.kucoin.com/trade/margin/BTC-USDT",
		asset.Futures: "https://www.kucoin.com/futures/trade/XBTUSDTM",
	} {
		result, err := e.GetCurrencyTradeURL(t.Context(), a, assertToTradablePairMap[a])
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		if mockTests {
			assert.Equalf(t, exp, result, "GetCurrencyTradeURL should return the %s trade page", a)
			continue
		}
		assert.NotEmptyf(t, result, "GetCurrencyTradeURL should return a %s trade page", a)
	}
}

func TestOrderSideString(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]string{
		order.Buy:     "buy",
		order.Bid:     "buy",
		order.Long:    "buy",
		order.Sell:    "sell",
		order.Ask:     "sell",
		order.Short:   "sell",
		order.AnySide: "",
	} {
		got, err := e.OrderSideString(side)
		require.NoErrorf(t, err, "OrderSideString must not error for %s", side)
		assert.Equalf(t, exp, got, "OrderSideString should return KuCoin's side for %s", side)
	}
	_, err := e.OrderSideString(order.CouldNotShort)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "OrderSideString should reject a side KuCoin does not take")
}

func TestOrderTypeToString(t *testing.T) {
	t.Parallel()
	for oType, exp := range map[order.Type]string{
		order.Limit:       "limit",
		order.Market:      "market",
		order.StopLimit:   "limit_stop",
		order.StopMarket:  "market_stop",
		order.AnyType:     "",
		order.UnknownType: "",
	} {
		got, err := OrderTypeToString(oType)
		require.NoErrorf(t, err, "OrderTypeToString must not error for %s", oType)
		assert.Equalf(t, exp, got, "OrderTypeToString should return KuCoin's order list type for %s", oType)
	}
	_, err := OrderTypeToString(order.OCO)
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "OrderTypeToString should reject an order type KuCoin does not list")
}

func TestAccountToTradeTypeString(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a          asset.Item
		marginType margin.Type
		exp        string
	}{
		{a: asset.Spot, exp: SpotTradeType},
		{a: asset.Spot, marginType: margin.Isolated, exp: SpotTradeType},
		{a: asset.Margin, marginType: margin.Isolated, exp: IsolatedMarginTradeType},
		{a: asset.Margin, marginType: margin.Multi, exp: CrossMarginTradeType},
		{a: asset.Margin, exp: CrossMarginTradeType},
		{a: asset.Futures, marginType: margin.Isolated, exp: ""},
	} {
		assert.Equalf(t, tc.exp, e.AccountToTradeTypeString(tc.a, tc.marginType), "AccountToTradeTypeString should return %q for %s with margin type %q", tc.exp, tc.a, tc.marginType)
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1791763200000), time.UnixMilli(1791820800000)
	pair := currency.NewPair(currency.XBT, currency.USDTM)
	request := func(a asset.Item, p currency.Pair, start, end time.Time) *fundingrate.HistoricalRatesRequest {
		return &fundingrate.HistoricalRatesRequest{Asset: a, Pair: p, StartDate: start, EndDate: end}
	}
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	_, err = e.GetHistoricalFundingRates(t.Context(), request(asset.Spot, pair, start, end))
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates must reject a non-futures asset")
	_, err = e.GetHistoricalFundingRates(t.Context(), request(asset.Futures, currency.EMPTYPAIR, start, end))
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalFundingRates must reject an empty pair")
	_, err = e.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, end, start))
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalFundingRates must reject a window that ends before it starts")
	_, err = e.GetHistoricalFundingRates(cancelledContext(t), request(asset.Futures, pair, start, end))
	require.ErrorIs(t, err, context.Canceled, "GetHistoricalFundingRates must return the failed request's error")
	ex := testInstance(t)
	ex.CurrencyPairs.UseGlobalFormat = true
	_, err = ex.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, start, end))
	require.ErrorIs(t, err, currency.ErrPairFormatIsNil, "GetHistoricalFundingRates must return the pair formatting error")
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/contract/funding-rates": `{"code":"200000","data":[]}`})
	_, err = ex.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, start, end))
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates must report a window without funding rates")
	ex = marketTestServerExchange(t, map[string]string{"/api/v1/contract/funding-rates": `{"code":"200000","data":[{"symbol":"XBTUSDTM","fundingRate":0.000067,"timepoint":1791849600000}]}`})
	_, err = ex.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, start, end))
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates must leave out a funding rate after the window")
	_, err = ex.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, time.UnixMilli(1791856800000), time.UnixMilli(1791907200000)))
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates must leave out a funding rate before the window")

	result, err := e.GetHistoricalFundingRates(t.Context(), request(asset.Futures, pair, start, end))
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	if mockTests {
		rates := []fundingrate.Rate{
			{Time: time.UnixMilli(1791820800000), Rate: decimal.MustFromFloat(0.000067)},
			{Time: time.UnixMilli(1791792000000), Rate: decimal.MustFromFloat(0.000031)},
			{Time: time.UnixMilli(1791763200000), Rate: decimal.MustFromFloat(0.000035)},
		}
		exp := &fundingrate.HistoricalRates{
			Exchange:     e.Name,
			Asset:        asset.Futures,
			Pair:         pair,
			StartDate:    rates[2].Time,
			EndDate:      rates[0].Time,
			LatestRate:   rates[0],
			FundingRates: rates,
		}
		assert.Equal(t, exp, result, "GetHistoricalFundingRates should return the window's funding rates, latest first")
		return
	}
	assert.NotEmpty(t, result.FundingRates, "GetHistoricalFundingRates should return funding rates")
}

func TestWithdrawInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot, Pair: btcusdt, Price: 1})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "ModifyOrder must require an order ID or client order ID")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Options, Pair: btcusdt, OrderID: "1", Price: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must reject an asset KuCoin does not trade")
	_, err = e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Futures, Pair: currency.NewPair(currency.XBT, currency.USDTM), OrderID: "1", Price: 1})
	require.ErrorIs(t, err, asset.ErrNotSupported, "ModifyOrder must reject a futures order, which KuCoin cannot modify")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
		_, err = e.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, AssetType: asset.Spot, Pair: spotTradablePair, OrderID: "6717422bd51c29000775ea99", Price: 1})
		assert.Error(t, err, "ModifyOrder should report an order KuCoin does not have")
		return
	}

	_, err = e.ModifyOrder(cancelledContext(t), &order.Modify{AssetType: asset.Spot, Pair: btcusdt, OrderID: "670fd33bf9406e0007ab3945", Price: 30000})
	require.ErrorIs(t, err, context.Canceled, "ModifyOrder must return the request error")

	_, err = e.ModifyOrder(t.Context(), &order.Modify{AssetType: asset.Spot, Pair: btcusdt, OrderID: "gct-modify-null", Price: 1})
	require.ErrorIs(t, err, common.ErrNoResponse, "ModifyOrder must reject a reply without data")
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *order.ModifyResponse
	}{
		{
			// KuCoin takes one of the order ID and the client order ID, and replaces the order with a new one
			name:   "by order ID",
			modify: &order.Modify{Exchange: e.Name, OrderID: "670fd33bf9406e0007ab3945", ClientOrderID: "5c52e11203aa677f33e493fb", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: btcusdt, Price: 30000, Amount: 0.0001},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e0007408827", ClientOrderID: "5c52e11203aa677f33e493fb", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: btcusdt, Price: 30000, Amount: 0.0001},
		},
		{
			name:   "by client order ID",
			modify: &order.Modify{Exchange: e.Name, ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Amount: 0.0002},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e0007408828", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Amount: 0.0002},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := e.ModifyOrder(t.Context(), tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
			assert.Equal(t, tc.exp, resp, "ModifyOrder should return the replacement order")
		})
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported, as KuCoin serves only recent trades")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := e.CancelBatchOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelBatchOrders must reject no cancellations")
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "250445104152670209"}})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "CancelBatchOrders must reject cancellations while futures are disabled")
	for _, tc := range []struct {
		name   string
		cancel order.Cancel
		err    error
	}{
		{name: "a spot order", cancel: order.Cancel{AssetType: asset.Spot, Pair: currency.NewBTCUSDT(), OrderID: "671124f9365ccb00073debd4"}, err: asset.ErrNotSupported},
		{name: "an order without an ID", cancel: order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm}, err: order.ErrOrderIDNotSet},
		{name: "an order by client order ID without a pair", cancel: order.Cancel{AssetType: asset.Futures, ClientOrderID: "gct-futures-client"}, err: currency.ErrCurrencyPairEmpty},
	} {
		_, err = e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "250445104152670209"}, tc.cancel})
		require.ErrorIsf(t, err, tc.err, "CancelBatchOrders must reject %s before sending anything", tc.name)
	}

	t.Run("batches", func(t *testing.T) {
		t.Parallel()
		// The recorded mock matches DELETE requests by query only, so a local server checks each request's body
		orderIDs := make([]string, 12)
		cancels := make([]order.Cancel, 0, len(orderIDs)+1)
		exp := &order.CancelBatchResponse{Status: make(map[string]string, len(orderIDs)+1)}
		for i := range orderIDs {
			orderIDs[i] = strconv.FormatUint(250445104152670201+uint64(i), 10)
			cancels = append(cancels, order.Cancel{AssetType: asset.Futures, OrderID: orderIDs[i]})
			exp.Status[orderIDs[i]] = order.Cancelled.String()
		}
		exp.Status[orderIDs[11]] = "Order is in not cancelable state"
		cancels = append(cancels, order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm, ClientOrderID: "gct-futures-client"})
		exp.Status["gct-futures-client"] = order.Cancelled.String()
		replies := []struct{ body, reply string }{
			{body: `{"orderIdsList":["` + strings.Join(orderIDs[:10], `","`) + `"]}`, reply: batchCancelReply(orderIDs[:10], "")},
			{body: `{"orderIdsList":["` + strings.Join(orderIDs[10:], `","`) + `"]}`, reply: batchCancelReply(orderIDs[10:11], orderIDs[11])},
			{body: `{"clientOidsList":[{"symbol":"XBTUSDTM","clientOid":"gct-futures-client"}]}`, reply: `{"code":"200000","data":[{"orderId":null,"clientOid":"gct-futures-client","code":"200","msg":"success"}]}`},
		}
		var calls atomic.Int32
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			call := int(calls.Add(1)) - 1
			if !assert.Less(t, call, len(replies), "CancelBatchOrders should send three requests") {
				return
			}
			assert.Equal(t, http.MethodDelete, r.Method, "CancelBatchOrders should send DELETE requests")
			assert.Equal(t, "/v1/orders/multi-cancel", r.URL.Path, "CancelBatchOrders should send Batch Cancel Orders requests")
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err, "reading the request body should not error")
			assert.JSONEqf(t, replies[call].body, string(body), "CancelBatchOrders should send request %d's orders", call)
			_, err = io.WriteString(w, replies[call].reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		resp, err := ex.CancelBatchOrders(t.Context(), cancels)
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, exp, resp, "CancelBatchOrders should report each order's outcome")
		assert.EqualValues(t, len(replies), calls.Load(), "CancelBatchOrders should cancel ten orders a request, by order ID and by client order ID apart")
	})

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "250445104152670999"}})
		require.NoError(t, err, "CancelBatchOrders must not error")
		for id, status := range resp.Status {
			assert.NotEqualf(t, order.Cancelled.String(), status, "CancelBatchOrders should report that unknown order %s was not cancelled", id)
		}
		return
	}

	_, err = e.CancelBatchOrders(cancelledContext(t), []order.Cancel{{AssetType: asset.Futures, OrderID: "250445104152670209"}})
	require.ErrorIs(t, err, context.Canceled, "CancelBatchOrders must return the request error")
	resp, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{AssetType: asset.Futures, OrderID: "250445104152670209"}, {AssetType: asset.Futures, OrderID: "250445181751463936"}})
	require.NoError(t, err, "CancelBatchOrders must not error")
	exp := &order.CancelBatchResponse{Status: map[string]string{"250445104152670209": order.Cancelled.String(), "250445181751463936": "Order is in not cancelable state"}}
	assert.Equal(t, exp, resp, "CancelBatchOrders should report each order's outcome")
}

// batchCancelReply returns a Batch Cancel Orders reply cancelling the orders given, and refusing failed when it is set
func batchCancelReply(cancelled []string, failed string) string {
	results := make([]string, 0, len(cancelled)+1)
	for _, orderID := range cancelled {
		results = append(results, `{"orderId":"`+orderID+`","clientOid":null,"code":"200","msg":"success"}`)
	}
	if failed != "" {
		results = append(results, `{"orderId":"`+failed+`","clientOid":null,"code":"100004","msg":"Order is in not cancelable state"}`)
	}
	return `{"code":"200000","data":[` + strings.Join(results, ",") + `]}`
}

// newLocalServerExchange returns an exchange that sends its REST requests, signed with placeholder credentials, to a
// local server, for replies and request checks the recorded fixtures cannot give
func newLocalServerExchange(tb testing.TB, handler http.HandlerFunc) *Exchange {
	tb.Helper()
	server := httptest.NewServer(handler)
	tb.Cleanup(server.Close)
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	require.NoError(tb, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	require.NoError(tb, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	for _, u := range []exchange.URL{exchange.RestSpot, exchange.RestFutures} {
		require.NoError(tb, ex.API.Endpoints.SetRunningURL(u.String(), server.URL), "SetRunningURL must not error")
	}
	return ex
}

func TestStringToTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		tif      string
		postOnly bool
		exp      order.TimeInForce
	}{
		{tif: "GTC", exp: order.GoodTillCancel},
		{tif: "GTC", postOnly: true, exp: order.GoodTillCancel | order.PostOnly},
		{tif: "GTT", exp: order.GoodTillTime},
		{tif: "GTT", postOnly: true, exp: order.GoodTillTime | order.PostOnly},
		{tif: "IOC", exp: order.ImmediateOrCancel},
		{tif: "ioC", exp: order.ImmediateOrCancel},
		{tif: "Fok", exp: order.FillOrKill},
	} {
		assert.Equalf(t, tc.exp, StringToTimeInForce(tc.tif, tc.postOnly), "StringToTimeInForce should convert %s with post only %t", tc.tif, tc.postOnly)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateOrderExecutionLimits must reject an asset KuCoin does not trade")
	for _, a := range []asset.Item{asset.Spot, asset.Futures} {
		require.ErrorIsf(t, e.UpdateOrderExecutionLimits(cancelledContext(t), a), context.Canceled, "UpdateOrderExecutionLimits must return the failed %s request's error", a)
	}
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v2/symbols":          `{"code":"200000","data":[{"symbol":"AB","isMarginEnabled":true}]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"ETH"}]}`,
	})
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Spot), currency.ErrCreatingPair, "UpdateOrderExecutionLimits must reject a symbol too short to be a pair")
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Futures), errContractSymbolMismatch, "UpdateOrderExecutionLimits must reject a contract whose symbol does not start with its base currency")
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v2/symbols":          `{"code":"200000","data":[{"symbol":"ETH-BTC","isMarginEnabled":false}]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"1000PEPEUSDTM","baseCurrency":"1000PEPE","tickSize":0.000001,"lotSize":1,"maxOrderQty":1000000,"maxPrice":1000000}]}`,
	})
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Margin), limits.ErrEmptyLevels, "UpdateOrderExecutionLimits must report that no symbol trades on margin")
	require.NoError(t, ex.UpdateOrderExecutionLimits(t.Context(), asset.Futures), "UpdateOrderExecutionLimits must not error")
	l, err := ex.GetOrderExecutionLimits(asset.Futures, currency.NewPair(currency.NewCode("1000PEPE"), currency.USDTM))
	require.NoError(t, err, "GetOrderExecutionLimits must not error")
	assert.Equal(t, 1000.0, l.PriceDivisor, "UpdateOrderExecutionLimits should divide the price of a 1000 unit contract by 1000")

	ensureTradablePairs(t)
	ex = testInstance(t)
	ex.Name = t.Name()
	limit := func(a asset.Item, p currency.Pair, l limits.MinMaxLevel) limits.MinMaxLevel {
		l.Key = key.NewExchangeAssetPair(ex.Name, a, p)
		return l
	}
	btcUSDT := func(a asset.Item) limits.MinMaxLevel {
		return limit(a, currency.NewBTCUSDT(), limits.MinMaxLevel{
			PriceStepIncrementSize:  0.1,
			MinimumBaseAmount:       0.00001,
			MaximumBaseAmount:       10000000000,
			MinimumQuoteAmount:      0.1,
			MaximumQuoteAmount:      99999999,
			AmountStepIncrementSize: 0.00000001,
			QuoteStepIncrementSize:  0.000001,
		})
	}
	exp := map[asset.Item][]limits.MinMaxLevel{
		asset.Spot: {
			btcUSDT(asset.Spot),
			limit(asset.Spot, currency.NewPair(currency.NewCode("DOLO"), currency.USDT), limits.MinMaxLevel{
				PriceStepIncrementSize:  0.00001,
				MinimumBaseAmount:       10,
				MaximumBaseAmount:       10000000000,
				MinimumQuoteAmount:      0.1,
				MaximumQuoteAmount:      99999999,
				AmountStepIncrementSize: 0.1,
				QuoteStepIncrementSize:  0.00001,
				Listed:                  time.UnixMilli(1745499600000),
			}),
			limit(asset.Spot, currency.NewPair(currency.BTC, currency.EUR), limits.MinMaxLevel{
				PriceStepIncrementSize:  0.01,
				MinimumBaseAmount:       0.00001,
				MaximumBaseAmount:       10000000000,
				MinimumQuoteAmount:      0.1,
				MaximumQuoteAmount:      99999999,
				AmountStepIncrementSize: 0.00001,
				QuoteStepIncrementSize:  0.01,
				Listed:                  time.UnixMilli(1677826800000),
			}),
		},
		asset.Margin: {btcUSDT(asset.Margin)},
		asset.Futures: {
			// Futures amounts are in lots
			limit(asset.Futures, currency.NewPair(currency.XBT, currency.USDTM), limits.MinMaxLevel{
				MaxPrice:                1000000,
				PriceStepIncrementSize:  0.1,
				PriceDivisor:            1,
				MultiplierDecimal:       0.001,
				MinimumBaseAmount:       1,
				MaximumBaseAmount:       1000000,
				AmountStepIncrementSize: 1,
				MarketMaxQty:            1000000,
				Listed:                  time.UnixMilli(1585555200000),
			}),
			limit(asset.Futures, currency.NewPair(currency.NewCode("ANTHROPIC"), currency.USDTM), limits.MinMaxLevel{
				MaxPrice:                1000000,
				PriceStepIncrementSize:  0.01,
				PriceDivisor:            1,
				MultiplierDecimal:       0.001,
				MinimumBaseAmount:       1,
				MaximumBaseAmount:       100000,
				AmountStepIncrementSize: 1,
				MarketMaxQty:            100000,
				Listed:                  time.UnixMilli(1780380000000),
			}),
			limit(asset.Futures, currency.NewPair(currency.XBT, currency.NewCode("MZ26")), limits.MinMaxLevel{
				MaxPrice:                1000000,
				PriceStepIncrementSize:  0.1,
				PriceDivisor:            1,
				MultiplierDecimal:       -1,
				MinimumBaseAmount:       1,
				MaximumBaseAmount:       100000,
				AmountStepIncrementSize: 1,
				MarketMaxQty:            100000,
				Listed:                  time.UnixMilli(1790071200000),
				Delisted:                time.UnixMilli(1798185600000),
				Expiry:                  time.UnixMilli(1798185600000),
			}),
		},
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin, asset.Futures} {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		if !mockTests {
			l, err := ex.GetOrderExecutionLimits(a, assertToTradablePairMap[a])
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", a)
			assert.Positivef(t, l.AmountStepIncrementSize, "UpdateOrderExecutionLimits should load a %s amount step", a)
			continue
		}
		for _, want := range exp[a] {
			l, err := ex.GetOrderExecutionLimits(a, want.Key.Pair())
			require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s %s", a, want.Key.Pair())
			assert.WithinDurationf(t, time.Now(), l.UpdatedAt, time.Minute, "UpdateOrderExecutionLimits should stamp the %s %s limits when loaded", a, want.Key.Pair())
			l.UpdatedAt = time.Time{}
			assert.Equalf(t, want, l, "UpdateOrderExecutionLimits should load the %s %s limits", a, want.Key.Pair())
		}
	}
}
