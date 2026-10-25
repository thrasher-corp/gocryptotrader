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
	"sync"
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
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
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
		// GetDepositAddress cannot choose among addresses on several chains
		MultiChainDepositRequiresChainSet: true,
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
		pair, err := futuresContractPair(&ContractResponse{Symbol: tc.symbol, BaseCurrency: tc.base})
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
			BaseVolume:   2508.307,
			QuoteVolume:  207495496.3685,
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

	t.Run("futures volumes agree with the symbol snapshot push", func(t *testing.T) {
		t.Parallel()
		// XBTUSDTM's figures from Get All Symbols and the Symbol Snapshot push, live on 2026-10-16
		ex := marketTestServerExchange(t, map[string]string{
			"/api/v1/allTickers":       `{"code":"200000","data":[{"symbol":"XBTUSDTM","price":"81969.8","size":16,"bestBidPrice":"81969.8","bestBidSize":624,"bestAskPrice":"81969.9","bestAskSize":555,"ts":1792155956873000000}]}`,
			"/api/v1/contracts/active": `{"code":"200000","data":[{"symbol":"XBTUSDTM","baseCurrency":"XBT","isInverse":false,"volumeOf24h":4737.538,"turnoverOf24h":387534567.598,"highPrice":82897.5,"lowPrice":79726.9}]}`,
		})
		pair := currency.NewPairWithDelimiter("XBT", "USDTM", "_")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, currency.Pairs{pair}, false), "StorePairs must not error for available pairs")
		require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, currency.Pairs{pair}, true), "StorePairs must not error for enabled pairs")
		require.NoError(t, ex.UpdateTickers(t.Context(), asset.Futures), "UpdateTickers must not error")
		rest, err := ticker.GetTicker(ex.Name, pair, asset.Futures)
		require.NoError(t, err, "UpdateTickers must store the ticker")
		push := []byte(`{"topic":"/contractMarket/snapshot:XBTUSDTM","type":"message","subject":"snapshot.24h","id":"6ad221f187fbf60001a47e5b","data":{"highPrice":82897.5,"lastPrice":81923.6,"lowPrice":79726.9,"price24HoursBefore":80433.5,"symbol":"XBTUSDTM","ts":1792156145005323906,"turnover":193714564.5099,"volume":2368.107}}`)
		require.NoError(t, ex.wsHandleData(t.Context(), nil, push), "wsHandleData must not error for a symbol snapshot")
		ws, err := ticker.GetTicker(ex.Name, pair, asset.Futures)
		require.NoError(t, err, "GetTicker must not error after the snapshot push")
		assert.InDelta(t, ws.BaseVolume, rest.BaseVolume, 0.001*ws.BaseVolume, "UpdateTickers should store the base volume the snapshot push stores")
		assert.InDelta(t, ws.QuoteVolume, rest.QuoteVolume, 0.001*ws.QuoteVolume, "UpdateTickers should store the quote volume the snapshot push stores")
	})
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
			AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Sell, Status: order.Open,
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

	t.Run("spot type filter", func(t *testing.T) {
		t.Parallel()
		queries := make(chan url.Values, 8)
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/v1/hf/orders/done", r.URL.Path, "closedHFOrders should list closed spot orders")
			queries <- r.URL.Query()
			_, err := io.WriteString(w, `{"code":"200000","data":{"lastId":0,"items":[]}}`)
			assert.NoError(t, err, "writing the reply should not error")
		})
		_, err := ex.closedHFOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{btcusdt}, Type: order.Market}, spotFormat, "sell")
		require.NoError(t, err, "closedHFOrders must not error")
		require.Len(t, queries, 1, "the local server must receive one request")
		assert.Equal(t, url.Values{"symbol": {"BTC-USDT"}, "side": {"sell"}, "type": {"market"}, "limit": {"100"}}, <-queries, "closedHFOrders should list the closed spot orders of the request's side and type")
	})

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

// TestGetOrderInfoConditionalOrders covers the spot and margin stop and OCO orders, which KuCoin looks up apart from the
// other orders: an order the order lookup misses, by returning no data or an error response, is looked up as a stop
// order and then as an OCO order
func TestGetOrderInfoConditionalOrders(t *testing.T) {
	t.Parallel()
	const (
		orderID    = "vs8hoo8q2ceshiue003b67c0"
		null       = `{"code":"200000","data":null}`
		notExist   = `{"code":"400100","msg":"order not exist."}`
		unexpected = `{"code":"400100","msg":"unexpected request"}`
	)
	tradeTypes := map[asset.Item]string{asset.Spot: SpotTradeType, asset.Margin: IsolatedMarginTradeType}
	stopOrder := func(a asset.Item, status, stop string) string {
		return `{"code":"200000","data":{"id":"` + orderID + `","symbol":"BTC-USDT","userId":"60fe4956c43cbc0006562c2c","status":"` + status + `","type":"limit","side":"buy",` +
			`"price":"50000","size":"0.001","funds":null,"stp":null,"timeInForce":"GTC","cancelAfter":-1,"postOnly":false,"hidden":false,"iceberg":false,` +
			`"visibleSize":null,"channel":"API","clientOid":"gct-stop","remark":null,"tags":null,"orderTime":1792500000000000000,"domainId":"kucoin",` +
			`"tradeSource":"USER","tradeType":"` + tradeTypes[a] + `","feeCurrency":"USDT","takerFeeRate":"0.001","makerFeeRate":"0.001","createdAt":1792500000000,` +
			`"stop":"` + stop + `","stopTriggerTime":null,"stopPrice":"51000"}}`
	}
	const ocoOrder = `{"orderId":"` + orderID + `","symbol":"BTC-USDT","clientOid":"gct-oco","orderTime":1792500000000,"status":"NEW"}`
	btcusdt := currency.NewBTCUSDT()
	marginTypes := map[asset.Item]margin.Type{asset.Spot: margin.Unset, asset.Margin: margin.Isolated}
	stopDetail := func(name string, a asset.Item, oType order.Type, status order.Status) *order.Detail {
		return &order.Detail{
			Exchange: name, OrderID: orderID, ClientOrderID: "gct-stop", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "-"),
			AssetType: a, MarginType: marginTypes[a], Type: oType, Side: order.Buy, Status: status, Price: 50000,
			TriggerPrice: 51000, Amount: 0.001, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1792500000000),
			LastUpdated: time.Unix(0, 1792500000000000000),
		}
	}
	ocoDetail := func(name string, a asset.Item) *order.Detail {
		return &order.Detail{
			Exchange: name, OrderID: orderID, ClientOrderID: "gct-oco", Pair: currency.NewPairWithDelimiter("BTC", "USDT", "-"),
			AssetType: a, Type: order.OCO, Status: order.New, Date: time.UnixMilli(1792500000000), LastUpdated: time.UnixMilli(1792500000000),
		}
	}
	paths := map[asset.Item][]string{
		asset.Spot:   {"/v1/hf/orders/" + orderID + "?symbol=BTC-USDT", "/v1/stop-order/" + orderID, "/v3/oco/orders?orderIds=" + orderID},
		asset.Margin: {"/v3/hf/margin/orders/" + orderID + "?symbol=BTC-USDT", "/v3/hf/margin/stop-order/orderId?orderId=" + orderID, "/v3/hf/margin/oco-order/orderId?orderId=" + orderID},
	}
	ocoReplies := map[asset.Item]string{
		// The list holds another order first, so the lookup must pick the order of its ID
		asset.Spot:   `{"code":"200000","data":{"currentPage":1,"pageSize":50,"totalNum":2,"totalPage":1,"items":[{"orderId":"674c3b6e688dea0007c7bab2","symbol":"ETH-USDT","clientOid":"gct-other","orderTime":1792400000000,"status":"DONE"},` + ocoOrder + `]}}`,
		asset.Margin: `{"code":"200000","data":` + ocoOrder + `}`,
	}
	ocoMisses := map[asset.Item]string{
		asset.Spot:   `{"code":"200000","data":{"currentPage":1,"pageSize":50,"totalNum":0,"totalPage":0,"items":[]}}`,
		asset.Margin: null,
	}
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		for _, tc := range []struct {
			name     string
			replies  [3]string
			exp      func(string) *order.Detail
			err      error
			errText  string
			requests int
		}{
			{name: "a stop order the order lookup returns no data for", replies: [3]string{null, stopOrder(a, "NEW", "entry"), unexpected}, exp: func(n string) *order.Detail { return stopDetail(n, a, order.StopLimit, order.New) }, requests: 2},
			{name: "a triggered stop order", replies: [3]string{notExist, stopOrder(a, "TRIGGERED", "entry"), unexpected}, exp: func(n string) *order.Detail { return stopDetail(n, a, order.StopLimit, order.Closed) }, requests: 2},
			// KuCoin keeps OCO orders with the stop orders, which can return one
			{name: "an OCO order the stop order lookup returns", replies: [3]string{notExist, stopOrder(a, "NEW", "oco"), unexpected}, exp: func(n string) *order.Detail { return stopDetail(n, a, order.OCO, order.New) }, requests: 2},
			{name: "an OCO order", replies: [3]string{notExist, notExist, ocoReplies[a]}, exp: func(n string) *order.Detail { return ocoDetail(n, a) }, requests: 3},
			{name: "an OCO order the stop order lookup returns no data for", replies: [3]string{null, null, ocoReplies[a]}, exp: func(n string) *order.Detail { return ocoDetail(n, a) }, requests: 3},
			// The order lookup's answer is the one returned when no lookup finds the order
			{name: "an order no lookup returns data for", replies: [3]string{null, null, ocoMisses[a]}, err: order.ErrOrderNotFound, requests: 3},
			{name: "an order every lookup refuses", replies: [3]string{notExist, `{"code":"400100","msg":"stop order not exist"}`, `{"code":"400100","msg":"oco order not exist"}`}, errText: "code: 400100 message: order not exist.", requests: 3},
			// A stop order lookup that gets no answer fails the call, which could not tell whether the order is a stop order
			{name: "a stop order lookup without an answer", replies: [3]string{notExist, "", unexpected}, errText: "500", requests: 2},
		} {
			t.Run(a.String()+" "+tc.name, func(t *testing.T) {
				t.Parallel()
				var requests atomic.Int32
				ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					i := slices.Index(paths[a], r.URL.RequestURI())
					if !assert.NotEqualf(t, -1, i, "GetOrderInfo should send no request but the %s order lookups, not %s", a, r.URL.RequestURI()) {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if tc.replies[i] == "" {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					_, err := io.WriteString(w, tc.replies[i])
					assert.NoError(t, err, "writing the reply should not error")
				})
				ex.Name = t.Name()
				got, err := ex.GetOrderInfo(t.Context(), orderID, btcusdt, a)
				switch {
				case tc.err != nil:
					require.ErrorIs(t, err, tc.err, "GetOrderInfo must return the order lookup's error")
				case tc.errText != "":
					require.ErrorContains(t, err, tc.errText, "GetOrderInfo must return the error of the lookup that failed")
				default:
					require.NoError(t, err, "GetOrderInfo must not error")
					assert.Equal(t, tc.exp(ex.Name), got, "GetOrderInfo should return the order the lookup found")
				}
				assert.EqualValues(t, tc.requests, requests.Load(), "GetOrderInfo should stop looking once an answer settles the order")
			})
		}
	}
}

func TestLookupMissed(t *testing.T) {
	t.Parallel()
	respErr := Error{Code: "400100", Message: "order not exist."}.GetError()
	for err, exp := range map[error]bool{
		order.ErrOrderNotFound:                        true,
		fmt.Errorf("%w: 123", order.ErrOrderNotFound): true,
		respErr: true,
		fmt.Errorf("error fetching order: %w", respErr): true,
		context.Canceled: false,
		errors.New("unexpected HTTP status code 500"):            false,
		fmt.Errorf("error fetching order: %w", context.Canceled): false,
	} {
		assert.Equalf(t, exp, lookupMissed(err), "lookupMissed should return %t for %v", exp, err)
	}
}

// TestCancelAllOrdersFuturesFailure covers a futures cancellation whose later request fails: the orders earlier requests
// cancelled stay cancelled, so their statuses are returned with the error
func TestCancelAllOrdersFuturesFailure(t *testing.T) {
	t.Parallel()
	const activeOrders = `{"code":"200000","data":{"currentPage":1,"pageSize":1000,"totalNum":2,"totalPage":1,"items":[{"id":"1","symbol":"XBTUSDTM"},{"id":"3","symbol":"ETHUSDTM"}]}}`
	cancelled := order.Cancelled.String()
	for _, tc := range []struct {
		name    string
		failing string
		exp     map[string]string
		err     string
	}{
		{name: "a later symbol's limit orders", failing: "/v3/orders?symbol=ETHUSDTM", exp: map[string]string{"1": cancelled, "2": cancelled}, err: "400100"},
		{name: "the stop orders", failing: "/v1/stopOrders", exp: map[string]string{"1": cancelled, "2": cancelled, "3": cancelled}, err: "400100"},
		{name: "a context cancelled during a later request", failing: "cancel /v3/orders?symbol=ETHUSDTM", exp: map[string]string{"1": cancelled, "2": cancelled}, err: context.Canceled.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				reply := `{"code":"400100","msg":"unexpected request"}`
				switch r.URL.RequestURI() {
				case tc.failing:
				case "/v1/orders?currentPage=1&pageSize=1000&status=active":
					reply = activeOrders
				case "/v3/orders?symbol=XBTUSDTM":
					reply = `{"code":"200000","data":{"cancelledOrderIds":["1","2"]}}`
				case "/v3/orders?symbol=ETHUSDTM":
					if tc.failing == "cancel "+r.URL.RequestURI() {
						// The client abandons the request once its context is cancelled, which closes the connection
						cancel()
						<-r.Context().Done()
						return
					}
					reply = `{"code":"200000","data":{"cancelledOrderIds":["3"]}}`
				}
				_, err := io.WriteString(w, reply)
				assert.NoError(t, err, "writing the reply should not error")
			})
			resp, err := ex.CancelAllOrders(ctx, &order.Cancel{AssetType: asset.Futures})
			require.ErrorContains(t, err, tc.err, "CancelAllOrders must return the failed request's error")
			assert.Equal(t, tc.exp, resp.Status, "CancelAllOrders should return the statuses of the orders cancelled before the failure")
		})
	}
}

// TestGetActiveOrdersFuturesTakeProfitOrders lists untriggered and triggered futures take profit orders by their types
func TestGetActiveOrdersFuturesTakeProfitOrders(t *testing.T) {
	t.Parallel()
	futuresOrder := func(id, side, oType, stop string, triggered bool) string {
		state := "false"
		if triggered {
			state = "true"
		}
		return `{"id":"` + id + `","symbol":"XBTUSDTM","type":"` + oType + `","side":"` + side + `","price":"100000","size":2,"value":"200","dealValue":"0","dealSize":0,"stop":"` + stop + `","stopPriceType":"TP","stopTriggered":` + state + `,"stopPrice":"99000","leverage":"3","clientOid":"gct-` + id + `","isActive":true,"cancelExist":false,"createdAt":1792500000000,"updatedAt":1792500000000,"settleCurrency":"USDT","marginMode":"ISOLATED","status":"open"}`
	}
	var mu sync.Mutex
	var requests []string
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests = append(requests, r.URL.RequestURI())
		mu.Unlock()
		var items string
		switch r.URL.Path {
		case "/v1/stopOrders":
			items = futuresOrder("tp-sell", "sell", "limit", "up", false) + "," + futuresOrder("tpm-buy", "buy", "market", "down", false) + "," + futuresOrder("sl-sell", "sell", "limit", "down", false)
		case "/v1/orders":
			items = futuresOrder("tp-triggered", "sell", "limit", "up", true)
		default:
			assert.Failf(t, "unexpected request", "request path %s should be one the test serves", r.URL.Path)
		}
		_, err := io.WriteString(w, `{"code":"200000","data":{"currentPage":1,"pageSize":1000,"totalNum":1,"totalPage":1,"items":[`+items+`]}}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	for _, tc := range []struct {
		oType    order.Type
		exp      []string
		requests []string
	}{
		{oType: order.TakeProfit, exp: []string{"tp-triggered", "tp-sell"}, requests: []string{
			"/v1/stopOrders?currentPage=1&pageSize=1000&symbol=XBTUSDTM&type=limit",
			"/v1/orders?currentPage=1&pageSize=1000&status=active&symbol=XBTUSDTM&type=limit_stop",
		}},
		{oType: order.TakeProfitMarket, exp: []string{"tpm-buy"}, requests: []string{
			"/v1/stopOrders?currentPage=1&pageSize=1000&symbol=XBTUSDTM&type=market",
			"/v1/orders?currentPage=1&pageSize=1000&status=active&symbol=XBTUSDTM&type=market_stop",
		}},
	} {
		mu.Lock()
		requests = nil
		mu.Unlock()
		got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, Type: tc.oType, Side: order.AnySide})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s orders", tc.oType)
		ids := make([]string, len(got))
		for i := range got {
			ids[i] = got[i].OrderID
		}
		assert.ElementsMatchf(t, tc.exp, ids, "GetActiveOrders should list the active %s orders", tc.oType)
		mu.Lock()
		assert.Equalf(t, tc.requests, requests, "GetActiveOrders should ask both lists for the stop orders of %s's order type", tc.oType)
		mu.Unlock()
	}
}

// TestGetActiveOrdersFuturesStopOrders covers the untriggered futures stop orders, which KuCoin lists apart from the
// active orders: they are listed with them, one page after another, without repeating an order that triggered between
// the two lists
func TestGetActiveOrdersFuturesStopOrders(t *testing.T) {
	t.Parallel()
	futuresOrder := func(id, side, oType, stop string, triggered bool) string {
		return `{"id":"` + id + `","symbol":"XBTUSDTM","type":"` + oType + `","side":"` + side + `","price":"100000","size":2,"value":"200","dealValue":"0","dealSize":0,` +
			`"stp":"","stop":"` + stop + `","stopPriceType":"TP","stopTriggered":` + strconv.FormatBool(triggered) + `,"stopPrice":"99000","timeInForce":"GTC","postOnly":false,` +
			`"hidden":false,"iceberg":false,"leverage":"3","forceHold":false,"closeOrder":false,"visibleSize":0,"clientOid":"gct-` + id + `","remark":null,"tags":"",` +
			`"isActive":true,"cancelExist":false,"createdAt":1792500000000,"updatedAt":1792500000000,"endAt":null,"orderTime":1792500000000000000,` +
			`"settleCurrency":"USDT","marginMode":"ISOLATED","avgDealPrice":"0","filledSize":0,"filledValue":"0","status":"open","reduceOnly":false}`
	}
	page := func(currentPage, totalPage int, items ...string) string {
		return `{"code":"200000","data":{"currentPage":` + strconv.Itoa(currentPage) + `,"pageSize":1000,"totalNum":3,"totalPage":` + strconv.Itoa(totalPage) + `,"items":[` + strings.Join(items, ",") + `]}}`
	}
	active := page(1, 1, futuresOrder("active-limit", "buy", "limit", "", false), futuresOrder("triggered", "sell", "limit", "down", true))
	stopPages := []string{
		// The triggered order is listed as untriggered too, as it triggered after this page was read
		page(1, 2, futuresOrder("stop-buy", "buy", "limit", "up", false), futuresOrder("triggered", "sell", "limit", "down", false)),
		page(2, 2, futuresOrder("take-profit-sell", "sell", "market", "up", false)),
	}
	newServer := func(t *testing.T, stopsFail bool) (*Exchange, func() []string) {
		t.Helper()
		var mu sync.Mutex
		var requests []string
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			repeated := slices.Contains(requests, r.URL.RequestURI())
			requests = append(requests, r.URL.RequestURI())
			mu.Unlock()
			if repeated {
				// A repeated request would loop forever on a broken pager
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			reply := `{"code":"400100","msg":"unexpected request"}`
			switch r.URL.Path {
			case "/v1/orders":
				reply = active
			case "/v1/stopOrders":
				currentPage, err := strconv.Atoi(r.URL.Query().Get("currentPage"))
				switch {
				case stopsFail:
					reply = `{"code":"400100","msg":"stop orders unavailable"}`
				case err == nil && currentPage >= 1 && currentPage <= len(stopPages):
					reply = stopPages[currentPage-1]
				}
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		return ex, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return slices.Clone(requests)
		}
	}
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	type listed struct {
		OrderID string
		Type    order.Type
	}
	for _, tc := range []struct {
		name       string
		oType      order.Type
		side       order.Side
		start, end time.Time
		exp        []listed
		requests   []string
	}{
		{
			name: "any type", oType: order.AnyType, side: order.AnySide,
			exp: []listed{{"active-limit", order.Limit}, {"triggered", order.StopLimit}, {"stop-buy", order.StopLimit}, {"take-profit-sell", order.TakeProfitMarket}},
			requests: []string{
				"/v1/stopOrders?currentPage=1&pageSize=1000&symbol=XBTUSDTM",
				"/v1/stopOrders?currentPage=2&pageSize=1000&symbol=XBTUSDTM",
				"/v1/orders?currentPage=1&pageSize=1000&status=active&symbol=XBTUSDTM",
			},
		},
		{
			// The side and the order type a stop order places once triggered go to the stop order list too
			name: "buy stop limit", oType: order.StopLimit, side: order.Buy,
			exp: []listed{{"stop-buy", order.StopLimit}},
			requests: []string{
				"/v1/stopOrders?currentPage=1&pageSize=1000&side=buy&symbol=XBTUSDTM&type=limit",
				"/v1/stopOrders?currentPage=2&pageSize=1000&side=buy&symbol=XBTUSDTM&type=limit",
				"/v1/orders?currentPage=1&pageSize=1000&side=buy&status=active&symbol=XBTUSDTM&type=limit_stop",
			},
		},
		{
			// A stop market order places a market order once triggered; the fake lists orders of every type and side, so
			// the request alone shows the filters, and the filters leave none of the fake's orders
			name: "sell stop market", oType: order.StopMarket, side: order.Sell,
			exp: []listed{},
			requests: []string{
				"/v1/stopOrders?currentPage=1&pageSize=1000&side=sell&symbol=XBTUSDTM&type=market",
				"/v1/stopOrders?currentPage=2&pageSize=1000&side=sell&symbol=XBTUSDTM&type=market",
				"/v1/orders?currentPage=1&pageSize=1000&side=sell&status=active&symbol=XBTUSDTM&type=market_stop",
			},
		},
		{
			name: "any type in a window", oType: order.AnyType, side: order.AnySide, start: time.UnixMilli(1792400000000), end: time.UnixMilli(1792600000000),
			exp: []listed{{"active-limit", order.Limit}, {"triggered", order.StopLimit}, {"stop-buy", order.StopLimit}, {"take-profit-sell", order.TakeProfitMarket}},
			requests: []string{
				"/v1/stopOrders?currentPage=1&endAt=1792600000000&pageSize=1000&startAt=1792400000000&symbol=XBTUSDTM",
				"/v1/stopOrders?currentPage=2&endAt=1792600000000&pageSize=1000&startAt=1792400000000&symbol=XBTUSDTM",
				"/v1/orders?currentPage=1&endAt=1792600000000&pageSize=1000&startAt=1792400000000&status=active&symbol=XBTUSDTM",
			},
		},
		{
			name: "limit", oType: order.Limit, side: order.AnySide,
			exp:      []listed{{"active-limit", order.Limit}},
			requests: []string{"/v1/orders?currentPage=1&pageSize=1000&status=active&symbol=XBTUSDTM&type=limit"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, requests := newServer(t, false)
			got, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, Side: tc.side, Type: tc.oType, StartTime: tc.start, EndTime: tc.end})
			require.NoError(t, err, "GetActiveOrders must not error")
			gotListed := make([]listed, len(got))
			for i := range got {
				gotListed[i] = listed{got[i].OrderID, got[i].Type}
			}
			assert.Equal(t, tc.exp, gotListed, "GetActiveOrders should list the active and untriggered stop orders once each")
			assert.Equal(t, tc.requests, requests(), "GetActiveOrders should read every page of the lists the request needs")
		})
	}

	ex, _ := newServer(t, true)
	_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Pairs: currency.Pairs{xbtusdtm}, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorContains(t, err, "stop orders unavailable", "GetActiveOrders should fail when the stop order list fails")
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
	exp := []FuturesOrderDetailResponse{
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
	ex := marketTestServerExchange(t, map[string]string{
		"/api/v1/trade-fees":         `{"code":"200000","data":[]}`,
		"/api/v1/withdrawals/quotas": `{"code":"200000","data":null}`,
	})
	// KuCoin has no bank transfers, so neither bank fee is estimated, with or without a fiat currency, and the fake's
	// routes fail any request sent for one
	for _, feeType := range []exchange.FeeType{exchange.InternationalBankDepositFee, exchange.InternationalBankWithdrawalFee} {
		for _, fiat := range []currency.Code{currency.EMPTYCODE, currency.USD} {
			_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSDT(), FiatCurrency: fiat, Amount: 2})
			require.ErrorIsf(t, err, errFeeTypeNotSupported, "GetFeeByType must reject fee type %d with fiat currency %q", feeType, fiat)
		}
	}
	for _, feeType := range []exchange.FeeType{exchange.CryptocurrencyTradeFee, exchange.CryptocurrencyWithdrawalFee} {
		_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: feeType, Pair: currency.NewBTCUSDT(), FiatCurrency: currency.USD})
		require.ErrorIsf(t, err, common.ErrNoResponse, "GetFeeByType must reject a reply without fees for fee type %d", feeType)
	}
	for _, feeType := range []exchange.FeeType{exchange.CryptocurrencyTradeFee, exchange.CryptocurrencyWithdrawalFee} {
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
			// Futures amounts are in lots; a buy triggered as the price falls is a take profit order
			name: "futures", orderID: "236655147005071361", pair: xbtusdtm, a: asset.Futures,
			exp: &order.Detail{
				Exchange: e.Name, OrderID: "236655147005071361", ClientOrderID: "5c52e11203aa677f33e493fb",
				Pair:      currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter),
				AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.TakeProfit, Side: order.Buy,
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
				AssetType: asset.Margin, MarginType: margin.Multi, Type: order.Limit, Side: order.Buy,
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
		o    *HFOrderResponse
		err  error
	}{
		{name: "a malformed symbol", o: &HFOrderResponse{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &HFOrderResponse{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &HFOrderResponse{Symbol: "BTC-USDT", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.hfOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "hfOrderDetail should reject %s", tc.name)
	}

	btcusdt := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter)
	for _, tc := range []struct {
		name string
		o    *HFOrderResponse
		exp  order.Detail
	}{
		{
			name: "partially filled open order",
			o: &HFOrderResponse{
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
			o: &HFOrderResponse{
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
		{
			name: "filled order with its fee in KCS",
			o: &HFOrderResponse{
				ID: "6717422bd51c29000775ea04", ClientOrderID: "gct-kcs-fee", Symbol: "BTC-USDT", Type: "market", Side: "buy",
				Size: 0.5, DealSize: 0.5, DealFunds: 30000, Fee: 2.4, FeeCurrency: currency.KCS, TimeInForce: "GTC",
				CreatedAt: types.Time(time.UnixMilli(1729577515444)), LastUpdatedAt: types.Time(time.UnixMilli(1729577515481)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "6717422bd51c29000775ea04", ClientOrderID: "gct-kcs-fee", Pair: btcusdt,
				AssetType: asset.Spot, Type: order.Market, Side: order.Buy, Status: order.Filled, Amount: 0.5,
				ExecutedAmount: 0.5, AverageExecutedPrice: 60000, Cost: 30000, CostAsset: currency.USDT, Fee: 2.4,
				FeeAsset: currency.KCS, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729577515444),
				LastUpdated: time.UnixMilli(1729577515481), CloseTime: time.UnixMilli(1729577515481),
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
		o    *MarginHFOrderResponse
		err  error
	}{
		{name: "a malformed symbol", o: &MarginHFOrderResponse{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &MarginHFOrderResponse{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &MarginHFOrderResponse{Symbol: "BTC-USDT", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.marginHFOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "marginHFOrderDetail should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name string
		o    *MarginHFOrderResponse
		exp  order.Detail
	}{
		{
			name: "open cross margin order",
			o: &MarginHFOrderResponse{
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
			// A triggered stop order carries on as the market order it placed, as its order pushes report it
			name: "filled isolated margin stop order",
			o: &MarginHFOrderResponse{
				ID: "671667306afcdb0007231094", ClientOrderID: "gct-margin-filled", Symbol: "ETH-USDT", Type: "market",
				Side: "sell", Size: 0.5, DealSize: 0.5, DealFunds: 1500, Fee: 1.5, FeeCurrency: currency.USDT,
				Stop: "loss", StopTriggered: true, StopPrice: 3100, TimeInForce: "GTC", TradeType: IsolatedMarginTradeType,
				CreatedAt: types.Time(time.UnixMilli(1729521456248)), LastUpdatedAt: types.Time(time.UnixMilli(1729651011877)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "671667306afcdb0007231094", ClientOrderID: "gct-margin-filled",
				Pair:      currency.NewPairWithDelimiter(currency.ETH.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.Market, Side: order.Sell,
				Status: order.Filled, TriggerPrice: 3100, Amount: 0.5, ExecutedAmount: 0.5, AverageExecutedPrice: 3000,
				Cost: 1500, CostAsset: currency.USDT, Fee: 1.5, FeeAsset: currency.USDT, TimeInForce: order.GoodTillCancel,
				Date: time.UnixMilli(1729521456248), LastUpdated: time.UnixMilli(1729651011877),
				CloseTime: time.UnixMilli(1729651011877),
			},
		},
		{
			name: "untriggered isolated margin stop order",
			o: &MarginHFOrderResponse{
				ID: "671667306afcdb0007231095", ClientOrderID: "gct-margin-untriggered", Symbol: "ETH-USDT", Type: "market",
				Side: "sell", Size: 0.5, RemainSize: 0.5, Stop: "loss", StopPrice: 3100, TimeInForce: "GTC", Active: true,
				TradeType: IsolatedMarginTradeType, CreatedAt: types.Time(time.UnixMilli(1729521456248)),
				LastUpdatedAt: types.Time(time.UnixMilli(1729521456248)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "671667306afcdb0007231095", ClientOrderID: "gct-margin-untriggered",
				Pair:      currency.NewPairWithDelimiter(currency.ETH.String(), currency.USDT.String(), currency.DashDelimiter),
				AssetType: asset.Margin, MarginType: margin.Isolated, Type: order.StopMarket, Side: order.Sell,
				Status: order.New, TriggerPrice: 3100, Amount: 0.5, RemainingAmount: 0.5, CostAsset: currency.USDT,
				TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729521456248), LastUpdated: time.UnixMilli(1729521456248),
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
		o    *FuturesOrderDetailResponse
		err  error
	}{
		{name: "an unlisted contract", o: &FuturesOrderDetailResponse{Symbol: "UNLISTEDUSDTM", Type: "limit", Side: "buy"}, err: currency.ErrPairNotFound},
		{name: "an unknown type", o: &FuturesOrderDetailResponse{Symbol: "XBTUSDTM", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &FuturesOrderDetailResponse{Symbol: "XBTUSDTM", Type: "limit"}, err: order.ErrSideIsInvalid},
	} {
		_, err := e.futuresOrderDetail(tc.o)
		assert.ErrorIsf(t, err, tc.err, "futuresOrderDetail should reject %s", tc.name)
	}

	pair := currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter)
	for _, tc := range []struct {
		name string
		o    *FuturesOrderDetailResponse
		exp  order.Detail
	}{
		{
			name: "open cross margin order",
			o: &FuturesOrderDetailResponse{
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
			// A sell triggered as the price rises is a take profit order
			name: "filled isolated margin take profit order",
			o: &FuturesOrderDetailResponse{
				ID: "235919172150825000", ClientOrderID: "gct-futures-done", Symbol: "XBTUSDTM", Type: "market", Side: "sell",
				Size: 2, DealSize: 2, DealValue: 120, AverageDealPrice: 60000, Stop: "up", StopPrice: 60000,
				TimeInForce: "GTC", Leverage: 5, ReduceOnly: true, SettleCurrency: currency.USDT, MarginMode: "ISOLATED",
				CreatedAt: types.Time(time.UnixMilli(1729236185949)), UpdatedAt: types.Time(time.UnixMilli(1729236200000)),
				EndAt: types.Time(time.UnixMilli(1729236200000)),
			},
			exp: order.Detail{
				Exchange: e.Name, OrderID: "235919172150825000", ClientOrderID: "gct-futures-done", Pair: pair,
				AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.TakeProfitMarket, Side: order.Sell,
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

	for _, tc := range []struct {
		side, kind, stop string
		exp              order.Type
	}{
		{side: "sell", kind: "limit", stop: "up", exp: order.TakeProfit},
		{side: "buy", kind: "market", stop: "down", exp: order.TakeProfitMarket},
		{side: "buy", kind: "limit", stop: "down", exp: order.TakeProfit},
		{side: "sell", kind: "market", stop: "down", exp: order.StopMarket},
		{side: "buy", kind: "limit", stop: "up", exp: order.StopLimit},
		{side: "buy", kind: "market", stop: "up", exp: order.StopMarket},
	} {
		got, err := e.futuresOrderDetail(&FuturesOrderDetailResponse{ID: "235919172150825001", Symbol: "XBTUSDTM", Type: tc.kind, Side: tc.side, Stop: tc.stop, Size: 1, IsActive: true})
		require.NoErrorf(t, err, "futuresOrderDetail must not error for a %s %s order triggered %s", tc.side, tc.kind, tc.stop)
		assert.Equalf(t, tc.exp, got.Type, "futuresOrderDetail should convert a %s %s order triggered %s to a %s order", tc.side, tc.kind, tc.stop, tc.exp)
	}
}

func TestStopOrderDetail(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		o    *StopOrderResponse
		oco  bool
		err  error
	}{
		{name: "a malformed symbol", o: &StopOrderResponse{Symbol: "BT", Type: "limit", Side: "buy"}, err: currency.ErrCreatingPair},
		{name: "an unknown type", o: &StopOrderResponse{Symbol: "BTC-USDT", Type: "stop", Side: "buy"}, err: order.ErrUnrecognisedOrderType},
		{name: "an unknown side", o: &StopOrderResponse{Symbol: "BTC-USDT", Type: "limit"}, oco: true, err: order.ErrSideIsInvalid},
	} {
		_, err := e.stopOrderDetail(tc.o, asset.Spot, tc.oco)
		assert.ErrorIsf(t, err, tc.err, "stopOrderDetail should reject %s", tc.name)
	}

	btcusdt := currency.NewPairWithDelimiter(currency.BTC.String(), currency.USDT.String(), currency.DashDelimiter)
	for _, tc := range []struct {
		name string
		o    *StopOrderResponse
		a    asset.Item
		oco  bool
		exp  order.Detail
	}{
		{
			name: "spot stop limit order",
			o: &StopOrderResponse{
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
			o: &StopOrderResponse{
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
			o: &StopOrderResponse{
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

// TestStopOrderDetailRemembersMarginOrders checks an untriggered margin stop order REST lists is remembered as a live
// margin order, as it places its order under its own order ID once triggered and the Order V2 pushes follow it
func TestStopOrderDetailRemembersMarginOrders(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, status string
		a            asset.Item
		oco          bool
		remembered   bool
	}{
		{name: "an untriggered margin stop order", status: "NEW", a: asset.Margin, remembered: true},
		{name: "a triggered margin stop order", status: "TRIGGERED", a: asset.Margin},
		{name: "a margin OCO order", status: "NEW", a: asset.Margin, oco: true},
		{name: "an untriggered spot stop order", status: "NEW", a: asset.Spot},
	} {
		ex := wsTestInstance(t)
		_, err := ex.stopOrderDetail(&StopOrderResponse{ID: "vs93gptvr9t2fsql003l8k65", Symbol: "BTC-USDT", Status: tc.status, Type: "limit", Side: "sell", TradeType: CrossMarginTradeType}, tc.a, tc.oco)
		require.NoErrorf(t, err, "stopOrderDetail must not error for %s", tc.name)
		_, remembered := ex.marginOrderIDs.Load("vs93gptvr9t2fsql003l8k65")
		assert.Equalf(t, tc.remembered, remembered, "stopOrderDetail should remember %s only while it is a margin stop order waiting for its trigger", tc.name)
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
	ex := marketTestServerExchange(t, map[string]string{"/api/v3/deposit-addresses": `{"code":"200000","data":[{"address":"r***","memo":"2085123456","chainId":"xrp","chainName":"XRP"}]}`})
	result, err := ex.GetDepositAddress(t.Context(), currency.XRP, "", "")
	require.NoError(t, err, "GetDepositAddress must not error for a currency with one address")
	assert.Equal(t, &deposit.Address{Address: "r***", Tag: "2085123456", Chain: "xrp"}, result, "GetDepositAddress should return the only address with its memo")

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
		{name: "chain ID", currency: currency.USDT, chain: "ton", exp: &deposit.Address{Address: "***", Tag: "2085123456", Chain: "ton"}},
		// TRC20 is the name of the trx chain, so only the name can match it
		{name: "chain name", currency: currency.USDT, chain: "TRC20", exp: &deposit.Address{Address: "***", Chain: "trx"}},
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

	t.Run("futures orders with take profit and stop loss triggers", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"234125150956625922","clientOid":"gct-futures-tpsl"}}`)
		cases := []struct {
			name   string
			submit *order.Submit
			exp    *order.SubmitResponse
			body   string
		}{
			{
				// A buy takes profit as the price rises and stops a loss as it falls
				name: "limit buy with both triggers on the mark price",
				submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 100000, Amount: 1, Leverage: 3, MarginType: margin.Isolated, ClientOrderID: "gct-futures-tpsl", RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.MarkPrice},
					StopLoss:   order.RiskManagement{Enabled: true, Price: 90000, TriggerPriceType: order.MarkPrice},
				}},
				exp:  &order.SubmitResponse{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: xbtusdtm, AssetType: asset.Futures, Leverage: 3, Price: 100000, Amount: 1, ClientOrderID: "gct-futures-tpsl", MarginType: margin.Isolated, Status: order.New, OrderID: "234125150956625922"},
				body: `{"clientOid":"gct-futures-tpsl","side":"buy","symbol":"XBTUSDTM","leverage":3,"type":"limit","marginMode":"ISOLATED","price":"100000","size":1,"stopPriceType":"MP","triggerStopUpPrice":"110000","triggerStopDownPrice":"90000"}`,
			},
			{
				// The order fills on placing, as a plain market order does, and its stop loss guards the position
				name: "market buy by value with a stop loss",
				submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Long, Type: order.Market, QuoteAmount: 500, ClientOrderID: "gct-futures-sl", RiskManagementModes: order.RiskManagementModes{
					StopLoss: order.RiskManagement{Enabled: true, Price: 90000, OrderType: order.Market},
				}},
				exp:  &order.SubmitResponse{Exchange: ex.Name, Type: order.Market, Side: order.Long, Pair: xbtusdtm, AssetType: asset.Futures, QuoteAmount: 500, ClientOrderID: "gct-futures-sl", Status: order.Filled, OrderID: "234125150956625922"},
				body: `{"clientOid":"gct-futures-sl","side":"buy","symbol":"XBTUSDTM","type":"market","valueQty":"500","stopPriceType":"TP","triggerStopDownPrice":"90000"}`,
			},
			{
				name: "immediate or cancel limit buy with a take profit on the index price",
				submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Limit, Price: 100000, Amount: 2, TimeInForce: order.ImmediateOrCancel, ClientOrderID: "gct-futures-tp", RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.IndexPrice},
				}},
				exp:  &order.SubmitResponse{Exchange: ex.Name, Type: order.Limit, Side: order.Buy, Pair: xbtusdtm, AssetType: asset.Futures, TimeInForce: order.ImmediateOrCancel, Price: 100000, Amount: 2, ClientOrderID: "gct-futures-tp", Status: order.New, OrderID: "234125150956625922"},
				body: `{"clientOid":"gct-futures-tp","side":"buy","symbol":"XBTUSDTM","type":"limit","price":"100000","size":2,"timeInForce":"IOC","stopPriceType":"IP","triggerStopUpPrice":"110000"}`,
			},
		}
		for _, tc := range cases {
			resp, err := ex.SubmitOrder(t.Context(), tc.submit)
			require.NoErrorf(t, err, "SubmitOrder must not error for a %s", tc.name)
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equalf(t, tc.exp, resp, "SubmitOrder should report a %s as the order placed", tc.name)
		}
		require.Len(t, *got, len(cases), "the local server must receive every order")
		for i, tc := range cases {
			assert.Equalf(t, "/api/v1/st-orders", (*got)[i].requestURI, "SubmitOrder should place a %s through Add Take Profit And Stop Loss Order", tc.name)
			assert.JSONEqf(t, tc.body, (*got)[i].body, "SubmitOrder should send a %s with its triggers", tc.name)
		}
	})

	t.Run("futures risk management modes KuCoin cannot place", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"234125150956625923"}}`)
		for _, tc := range []struct {
			name   string
			submit *order.Submit
		}{
			{name: "a sell with a take profit and a stop loss", submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Sell, Type: order.Market, Amount: 1, ClientOrderID: "gct-futures-sell-tpsl", RiskManagementModes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Enabled: true, Price: 90000},
				StopLoss:   order.RiskManagement{Enabled: true, Price: 110000},
			}}},
			{name: "a stop limit order with a stop loss", submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.StopLimit, Price: 101000, TriggerPrice: 100500, Amount: 1, ClientOrderID: "gct-futures-stop-sl", RiskManagementModes: order.RiskManagementModes{
				StopLoss: order.RiskManagement{Enabled: true, Price: 90000},
			}}},
			{name: "a stop entry", submit: &order.Submit{Exchange: ex.Name, Pair: xbtusdtm, AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1, ClientOrderID: "gct-futures-stop-entry", RiskManagementModes: order.RiskManagementModes{
				StopEntry: order.RiskManagement{Enabled: true, Price: 110000},
			}}},
		} {
			_, err := ex.SubmitOrder(t.Context(), tc.submit)
			assert.ErrorIsf(t, err, errRiskModeUnsupported, "SubmitOrder should refuse %s", tc.name)
		}
		assert.Empty(t, *got, "SubmitOrder should send no order it refuses")
	})

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
		{name: "stop entry", submit: &order.Submit{Type: order.Market, Amount: 1, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 110000}}}, err: errRiskModeUnsupported},
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

func TestFuturesTakeProfitStopLossRequest(t *testing.T) {
	t.Parallel()
	limitBuy := FuturesOrderRequest{ClientOrderID: "gct-futures-tpsl", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Price: 100000, Size: 1}
	takeProfit := order.RiskManagement{Enabled: true, Price: 110000}
	stopLoss := order.RiskManagement{Enabled: true, Price: 90000}
	for _, tc := range []struct {
		name   string
		modify func(*FuturesOrderRequest)
		modes  order.RiskManagementModes
		err    error
	}{
		{name: "a stop order", modify: func(r *FuturesOrderRequest) { r.Stop, r.StopPrice, r.StopPriceType = "up", 100500, "TP" }, modes: order.RiskManagementModes{StopLoss: stopLoss}, err: errRiskModeUnsupported},
		// KuCoin's trigger names and their descriptions disagree for a sell, whose take profit is below its stop loss
		{name: "a sell", modify: func(r *FuturesOrderRequest) { r.Side = "sell" }, modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 90000}, StopLoss: order.RiskManagement{Enabled: true, Price: 110000}}, err: errRiskModeUnsupported},
		{name: "a sell with a stop loss only", modify: func(r *FuturesOrderRequest) { r.Side = "sell" }, modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 110000}}, err: errRiskModeUnsupported},
		{name: "a fill or kill order", modify: func(r *FuturesOrderRequest) { r.TimeInForce = "FOK" }, modes: order.RiskManagementModes{TakeProfit: takeProfit}, err: order.ErrUnsupportedTimeInForce},
		{name: "a take profit without a price", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}, StopLoss: stopLoss}, err: errTakeProfitPriceRequired},
		{name: "a take profit with a negative price", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: -110000}}, err: errTakeProfitPriceRequired},
		{name: "a stop loss without a price", modes: order.RiskManagementModes{TakeProfit: takeProfit, StopLoss: order.RiskManagement{Enabled: true}}, err: errStopLossPriceRequired},
		{name: "a stop loss with a negative price", modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: -90000}}, err: errStopLossPriceRequired},
		{name: "a take profit with a limit price", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, LimitPrice: 109900}}, err: errRiskModeUnsupported},
		{name: "a stop loss with a negative limit price", modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90000, LimitPrice: -1}}, err: errRiskModeUnsupported},
		{name: "a stop loss limit order", modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90000, OrderType: order.Limit}}, err: errRiskModeUnsupported},
		{name: "a take profit stop market order", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, OrderType: order.StopMarket}}, err: errRiskModeUnsupported},
		// KuCoin takes one trigger price type for both triggers
		{name: "triggers on different price types", modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.MarkPrice}, StopLoss: order.RiskManagement{Enabled: true, Price: 90000, TriggerPriceType: order.IndexPrice}}, err: errRiskModeUnsupported},
		{name: "a take profit below its stop loss", modes: order.RiskManagementModes{TakeProfit: stopLoss, StopLoss: takeProfit}, err: errOCOPricesInverted},
		{name: "a take profit at its stop loss", modes: order.RiskManagementModes{TakeProfit: takeProfit, StopLoss: takeProfit}, err: errOCOPricesInverted},
	} {
		r := limitBuy
		if tc.modify != nil {
			tc.modify(&r)
		}
		_, err := futuresTakeProfitStopLossRequest(&r, &tc.modes)
		assert.ErrorIsf(t, err, tc.err, "futuresTakeProfitStopLossRequest should reject %s", tc.name)
	}

	for _, tc := range []struct {
		name  string
		req   FuturesOrderRequest
		modes order.RiskManagementModes
		exp   *FuturesTakeProfitStopLossOrderRequest
	}{
		{
			name: "isolated post only limit buy with both triggers on the mark price",
			req:  FuturesOrderRequest{ClientOrderID: "gct-futures-tpsl", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit", MarginMode: "ISOLATED", Price: 100000, Size: 2, PostOnly: true},
			modes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.MarkPrice},
				StopLoss:   order.RiskManagement{Enabled: true, Price: 90000, TriggerPriceType: order.MarkPrice, OrderType: order.Market},
			},
			exp: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "gct-futures-tpsl", Side: "buy", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit", MarginMode: "ISOLATED", Price: 100000, Size: 2, PostOnly: true, StopPriceType: "MP", TriggerStopUpPrice: 110000, TriggerStopDownPrice: 90000},
		},
		{
			name:  "reduce only immediate or cancel limit buy with a take profit on the index price",
			req:   FuturesOrderRequest{ClientOrderID: "gct-futures-tp", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", ReduceOnly: true, Price: 100000, Size: 1, TimeInForce: "IOC"},
			modes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.IndexPrice}},
			exp:   &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "gct-futures-tp", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", ReduceOnly: true, Price: 100000, Size: 1, TimeInForce: "IOC", StopPriceType: "IP", TriggerStopUpPrice: 110000},
		},
		{
			name:  "cross margin market buy by value with a stop loss on the last price",
			req:   FuturesOrderRequest{ClientOrderID: "gct-futures-sl", Side: "buy", Symbol: "XBTUSDTM", Type: "market", MarginMode: "CROSS", ValueQuantity: 500},
			modes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 90000, TriggerPriceType: order.LastPrice}},
			exp:   &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "gct-futures-sl", Side: "buy", Symbol: "XBTUSDTM", Type: "market", MarginMode: "CROSS", ValueQuantity: 500, StopPriceType: "TP", TriggerStopDownPrice: 90000},
		},
		{
			// An unknown price type triggers on the last price, as a futures stop order's does, so it agrees with it
			name: "triggers on price types KuCoin reads alike",
			req:  limitBuy,
			modes: order.RiskManagementModes{
				TakeProfit: order.RiskManagement{Enabled: true, Price: 110000, TriggerPriceType: order.LastPrice},
				StopLoss:   order.RiskManagement{Enabled: true, Price: 90000, TriggerPriceType: order.UnknownPriceType},
			},
			exp: &FuturesTakeProfitStopLossOrderRequest{ClientOrderID: "gct-futures-tpsl", Side: "buy", Symbol: "XBTUSDTM", Type: "limit", Price: 100000, Size: 1, StopPriceType: "TP", TriggerStopUpPrice: 110000, TriggerStopDownPrice: 90000},
		},
	} {
		got, err := futuresTakeProfitStopLossRequest(&tc.req, &tc.modes)
		require.NoErrorf(t, err, "futuresTakeProfitStopLossRequest must not error for a %s", tc.name)
		assert.Equalf(t, tc.exp, got, "futuresTakeProfitStopLossRequest should place a %s with its take profit up and its stop loss down", tc.name)
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
	oco := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Price: 70000, Amount: 0.001, ClientOrderID: "gct-spot-oco"}
	_, err := e.submitSpotOrder(t.Context(), oco, "BTC-USDT", "sell")
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "submitSpotOrder must validate an OCO order's prices")
	oco.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, Price: 70000},
		StopLoss:   order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 49900},
	}
	_, err = e.submitSpotOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.TrailingStop, Amount: 1, ClientOrderID: "gct-spot-trailing"}, "BTC-USDT", "buy")
	require.ErrorIs(t, err, order.ErrUnsupportedOrderType, "submitSpotOrder must reject an order type KuCoin does not place")
	stop := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.StopLimit, Price: 61000, Amount: 0.001, ClientOrderID: "gct-spot-stop", RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 60000}}}
	limit := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, TimeInForce: order.FillOrKill, ClientOrderID: "gct-spot-limit"}

	t.Run("stop orders of limit and market submissions", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"vs93gptvr9t2fsql003l8k62"}}`)
		for _, tc := range []struct {
			name   string
			submit *order.Submit
			side   string
			exp    *order.SubmitResponse
		}{
			{
				name:   "market",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, Amount: 0.001, ClientOrderID: "gct-spot-stop-market", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}},
				side:   "sell",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.StopMarket, Side: order.Sell, Pair: btcusdt, AssetType: asset.Spot, Amount: 0.001, TriggerPrice: 40000, ClientOrderID: "gct-spot-stop-market", Status: order.New, OrderID: "vs93gptvr9t2fsql003l8k62"},
			},
			{
				name:   "limit",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Price: 61000, Amount: 0.001, ClientOrderID: "gct-spot-stop-limit", RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 60000}}},
				side:   "buy",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.StopLimit, Side: order.Buy, Pair: btcusdt, AssetType: asset.Spot, Price: 61000, Amount: 0.001, TriggerPrice: 60000, ClientOrderID: "gct-spot-stop-limit", Status: order.New, OrderID: "vs93gptvr9t2fsql003l8k62"},
			},
		} {
			resp, err := ex.submitSpotOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
			require.NoErrorf(t, err, "submitSpotOrder must not error for a %s submission with a stop condition", tc.name)
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equalf(t, tc.exp, resp, "submitSpotOrder should report a %s submission with a stop condition as a new stop order", tc.name)
		}
		require.Len(t, *got, 2, "the local server must receive both orders")
		for _, r := range *got {
			assert.Equal(t, "/api/v1/stop-order", r.requestURI, "submitSpotOrder should place a submission with a stop condition as a stop order")
		}
	})

	t.Run("OCO orders", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"674c316e688dea0007c7b992"}}`)
		cases := []struct {
			name   string
			submit *order.Submit
			side   string
			exp    *order.SubmitResponse
			body   string
		}{
			{
				// The stop order sells at its limit price once the price falls to the stop loss price
				name:   "sell",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Price: 70000, Amount: 0.001, ClientOrderID: "gct-spot-oco-sell", RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 49900}}},
				side:   "sell",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.OCO, Side: order.Sell, Pair: btcusdt, AssetType: asset.Spot, Price: 70000, Amount: 0.001, ClientOrderID: "gct-spot-oco-sell", Status: order.New, OrderID: "674c316e688dea0007c7b992"},
				body:   `{"clientOid":"gct-spot-oco-sell","side":"sell","symbol":"BTC-USDT","price":"70000","size":"0.001","stopPrice":"50000","limitPrice":"49900"}`,
			},
			{
				// Price may be left unset, the take profit price being the limit order's
				name:   "buy without a price",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Buy, Type: order.OCO, Amount: 0.001, ClientOrderID: "gct-spot-oco-buy", RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 70100}}},
				side:   "buy",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.OCO, Side: order.Buy, Pair: btcusdt, AssetType: asset.Spot, Price: 50000, Amount: 0.001, ClientOrderID: "gct-spot-oco-buy", Status: order.New, OrderID: "674c316e688dea0007c7b992"},
				body:   `{"clientOid":"gct-spot-oco-buy","side":"buy","symbol":"BTC-USDT","price":"50000","size":"0.001","stopPrice":"70000","limitPrice":"70100"}`,
			},
		}
		for _, tc := range cases {
			resp, err := ex.submitSpotOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
			require.NoErrorf(t, err, "submitSpotOrder must not error for a %s OCO order", tc.name)
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equalf(t, tc.exp, resp, "submitSpotOrder should report a %s OCO order as a new OCO order at its take profit price", tc.name)
		}
		require.Len(t, *got, len(cases), "the local server must receive every order")
		for i, tc := range cases {
			assert.Equalf(t, "/api/v3/oco/order", (*got)[i].requestURI, "submitSpotOrder should place a %s submission as an OCO order", tc.name)
			assert.JSONEqf(t, tc.body, (*got)[i].body, "submitSpotOrder should place a %s OCO order's limit order at its take profit and its stop order at its stop loss", tc.name)
		}
	})

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
		{name: "OCO", side: "sell", submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Spot, Side: order.Sell, Type: order.OCO, Price: 70000, Amount: 0.001, ClientOrderID: "gct-spot-oco-null", RiskManagementModes: oco.RiskManagementModes}},
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
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.OCO, Side: order.Sell, Pair: btcusdt, AssetType: asset.Spot, Price: 70000, Amount: 0.001, ClientOrderID: "gct-spot-oco", Status: order.New, OrderID: "674c316e688dea0007c7b990"},
		},
		{
			name:   "stop limit",
			submit: stop,
			side:   "buy",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.StopLimit, Side: order.Buy, Pair: btcusdt, AssetType: asset.Spot, Price: 61000, Amount: 0.001, TriggerPrice: 60000, ClientOrderID: "gct-spot-stop", Status: order.New, OrderID: "vs93gptvr9t2fsql003l8k60"},
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

// marginPlacementsLen returns how many client order IDs an exchange holds as being placed as margin orders
func marginPlacementsLen(ex *Exchange) int {
	ex.marginPlacementsMu.Lock()
	defer ex.marginPlacementsMu.Unlock()
	return len(ex.marginPlacements)
}

// syncMapLen returns how many entries a sync.Map holds
func syncMapLen(m *sync.Map) int {
	var n int
	m.Range(func(any, any) bool {
		n++
		return true
	})
	return n
}

// TestSubmitMarginOrderPushesBeforeReply checks a margin order's pushes are read as margin when they arrive before
// KuCoin's reply names the order's ID, and that an order already finished by the reply is not remembered as live
func TestSubmitMarginOrderPushesBeforeReply(t *testing.T) {
	t.Parallel()
	const orderID, clientOrderID = "6720da3fa30a360007f5f900", "gct-margin-early"
	ltcusdt := currency.NewPairWithDelimiter("LTC", "USDT", "-")
	push := func(event, status string) []byte {
		return []byte(`{"clientOid":"` + clientOrderID + `","orderId":"` + orderID + `","orderTime":1730206271588,"orderType":"market","originSize":"0.1","filledSize":"0.1","remainSize":"0","size":"0.1","side":"buy","status":"` + status + `","symbol":"LTC-USDT","ts":1730206271616000000,"type":"` + event + `"}`)
	}
	pushAsset := func(t *testing.T, ex *Exchange, event, status string) asset.Item {
		t.Helper()
		require.NoError(t, ex.processOrderChange(t.Context(), push(event, status)), "processOrderChange must not error")
		d, ok := relayed(t, ex).(*order.Detail)
		require.True(t, ok, "processOrderChange must relay an order detail")
		return d.AssetType
	}
	for _, tc := range []struct {
		name       string
		early      [][2]string
		refused    bool
		remembered bool
	}{
		{name: "an order received before the reply", early: [][2]string{{"received", "new"}}, remembered: true},
		// A reply failing after the order was placed leaves the push as the order's only record of its ID
		{name: "an order received before a failed reply", early: [][2]string{{"received", "new"}}, refused: true, remembered: true},
		{name: "an order filled before the reply", early: [][2]string{{"received", "new"}, {"filled", "done"}}},
		// The done push alone is read by the client order ID, as nothing remembers a finished order's ID
		{name: "an order whose only push before the reply is its fill", early: [][2]string{{"filled", "done"}}},
		{name: "an order filled before a failed reply", early: [][2]string{{"filled", "done"}}, refused: true},
		// Pushes of one order can arrive out of order, and a push that follows the order's fill does not bring it back
		{name: "an order whose open push follows its fill before the reply", early: [][2]string{{"filled", "done"}, {"open", "open"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var ex *Exchange
			ex = newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/v3/hf/margin/order", r.URL.Path, "SubmitOrder should place a margin HF order")
				for _, p := range tc.early {
					assert.NoError(t, ex.processOrderChange(r.Context(), push(p[0], p[1])), "processOrderChange should not error")
				}
				reply := `{"code":"200000","data":{"orderId":"` + orderID + `","clientOid":"` + clientOrderID + `"}}`
				if tc.refused {
					reply = `{"code":"500000","msg":"system error"}`
				}
				_, err := io.WriteString(w, reply)
				assert.NoError(t, err, "writing the reply should not error")
			})
			ex.Websocket.DataHandler = stream.NewRelay(10)
			_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: ltcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.1, ClientOrderID: clientOrderID})
			if tc.refused {
				require.Error(t, err, "SubmitOrder must return KuCoin's failed reply")
			} else {
				require.NoError(t, err, "SubmitOrder must not error")
			}
			require.Len(t, ex.Websocket.DataHandler.C, len(tc.early), "processOrderChange must relay each push that came before the reply")
			for range tc.early {
				d, ok := relayed(t, ex).(*order.Detail)
				require.True(t, ok, "processOrderChange must relay an order detail")
				assert.Equal(t, asset.Margin, d.AssetType, "a margin order's push before the reply should be a margin order update")
			}
			_, remembered := ex.marginOrderIDs.Load(orderID)
			assert.Equal(t, tc.remembered, remembered, "SubmitOrder should remember a margin order as live only while it is")
			assert.Zero(t, marginPlacementsLen(ex), "SubmitOrder should forget the client order ID, and any order finished before the reply, once KuCoin replies")
			exp := asset.Spot
			if tc.remembered {
				exp = asset.Margin
			}
			assert.Equal(t, exp, pushAsset(t, ex, "open", "open"), "a later push should be read by whether the order is still live")
		})
	}

	// The first request's reply is held until released, so a second request reusing its client order ID can fail first
	heldServer := func(t *testing.T, second string) (ex *Exchange, started, release chan struct{}) {
		t.Helper()
		started, release = make(chan struct{}), make(chan struct{})
		var requests atomic.Int32
		ex = newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			reply := second
			if requests.Add(1) == 1 {
				close(started)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
				reply = `{"code":"200000","data":{"orderId":"` + orderID + `","clientOid":"` + clientOrderID + `"}}`
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		ex.Websocket.DataHandler = stream.NewRelay(10)
		return ex, started, release
	}
	submitHeld := func(t *testing.T, ex *Exchange, started chan struct{}) chan error {
		t.Helper()
		first := make(chan error, 1)
		go func() {
			_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: ltcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.1, ClientOrderID: clientOrderID})
			first <- err
		}()
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the first order's request must reach KuCoin")
		}
		return first
	}
	waitHeld := func(t *testing.T, first chan error, release chan struct{}) {
		t.Helper()
		close(release)
		select {
		case err := <-first:
			require.NoError(t, err, "SubmitOrder must not error for the first order")
		case <-time.After(5 * time.Second):
			require.FailNow(t, "SubmitOrder must return once KuCoin replies to the first order")
		}
	}

	t.Run("a second request reusing the client order ID", func(t *testing.T) {
		t.Parallel()
		ex, started, release := heldServer(t, `{"code":"400100","msg":"duplicate clientOid"}`)
		first := submitHeld(t, ex, started)
		_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: ltcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.1, ClientOrderID: clientOrderID})
		require.Error(t, err, "SubmitOrder must return KuCoin's refusal of the second order")
		assert.Equal(t, asset.Margin, pushAsset(t, ex, "received", "new"), "the first order's push should be a margin order update after the second request failed")
		waitHeld(t, first, release)
		_, remembered := ex.marginOrderIDs.Load(orderID)
		assert.True(t, remembered, "SubmitOrder should remember the first order as live after the second request failed")
	})

	t.Run("the first request failing while a second reusing the client order ID is out", func(t *testing.T) {
		t.Parallel()
		secondArrived, release := make(chan struct{}), make(chan struct{})
		var releaseOnce sync.Once
		var requests atomic.Int32
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			reply := `{"code":"200000","data":{"orderId":"` + orderID + `","clientOid":"` + clientOrderID + `"}}`
			if requests.Add(1) == 1 {
				// The first request fails once the second is out
				select {
				case <-secondArrived:
				case <-time.After(5 * time.Second):
				}
				reply = `{"code":"500000","msg":"system error"}`
			} else {
				close(secondArrived)
				select {
				case <-release:
				case <-time.After(5 * time.Second):
				}
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
		ex.Websocket.DataHandler = stream.NewRelay(10)
		submit := func() chan error {
			done := make(chan error, 1)
			go func() {
				_, err := ex.SubmitOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: ltcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Market, Amount: 0.1, ClientOrderID: clientOrderID})
				done <- err
			}()
			return done
		}
		wait := func(done chan error) error {
			select {
			case err := <-done:
				return err
			case <-time.After(5 * time.Second):
				require.FailNow(t, "SubmitOrder must return")
			}
			return nil
		}
		first := submit()
		require.Eventually(t, func() bool { return requests.Load() == 1 }, 5*time.Second, time.Millisecond, "the first request must reach KuCoin")
		second := submit()
		require.Error(t, wait(first), "SubmitOrder must return KuCoin's failed reply to the first request")
		assert.Equal(t, asset.Margin, pushAsset(t, ex, "filled", "done"), "the second order's done push should be a margin order update while it is being placed")
		releaseOnce.Do(func() { close(release) })
		require.NoError(t, wait(second), "SubmitOrder must not error for the second request")
		_, remembered := ex.marginOrderIDs.Load(orderID)
		assert.False(t, remembered, "SubmitOrder should not remember an order that finished before its reply")
		assert.Zero(t, marginPlacementsLen(ex), "SubmitOrder should forget the client order ID once every request using it has its reply")
	})

	t.Run("a done push for another order with the client order ID", func(t *testing.T) {
		t.Parallel()
		ex, started, release := heldServer(t, "")
		first := submitHeld(t, ex, started)
		other := []byte(`{"clientOid":"` + clientOrderID + `","orderId":"6720da3fa30a360007f5f901","orderTime":1730206271588,"orderType":"market","originSize":"0.1","filledSize":"0.1","remainSize":"0","size":"0.1","side":"buy","status":"done","symbol":"LTC-USDT","ts":1730206271616000000,"type":"filled"}`)
		require.NoError(t, ex.processOrderChange(t.Context(), other), "processOrderChange must not error")
		require.Len(t, ex.Websocket.DataHandler.C, 1, "processOrderChange must relay the other order's push")
		relayed(t, ex)
		waitHeld(t, first, release)
		_, remembered := ex.marginOrderIDs.Load(orderID)
		assert.True(t, remembered, "another order's done push should not make SubmitOrder forget the order it placed")
	})

	t.Run("an order finished by an update", func(t *testing.T) {
		t.Parallel()
		// An update push with status done is a partly filled order whose rest was cancelled, or an STP cancellation
		ex := wsTestInstance(t)
		ex.Websocket.DataHandler = stream.NewRelay(10)
		ex.marginOrderIDs.Store(orderID, false)
		assert.Equal(t, asset.Margin, pushAsset(t, ex, "update", "done"), "the update finishing a margin order should be a margin order update")
		_, remembered := ex.marginOrderIDs.Load(orderID)
		assert.False(t, remembered, "processOrderChange should forget a margin order an update finishes")
	})
}

// TestRememberMarginOrder checks a placement KuCoin refuses is not remembered, even when its reply names an order ID
func TestRememberMarginOrder(t *testing.T) {
	t.Parallel()
	ex := wsTestInstance(t)
	errRefused := errors.New("refused")
	err := ex.rememberMarginOrder("gct-refused", func() (string, error) { return "6720da3fa30a360007f5f901", errRefused })
	require.ErrorIs(t, err, errRefused, "rememberMarginOrder must return the placement's error")
	_, remembered := ex.marginOrderIDs.Load("6720da3fa30a360007f5f901")
	assert.False(t, remembered, "rememberMarginOrder should not remember an order whose placement failed")
	assert.Zero(t, marginPlacementsLen(ex), "rememberMarginOrder should forget the client order ID once the placement returns")
	require.NoError(t, ex.rememberMarginOrder("gct-placed", func() (string, error) { return "6720da3fa30a360007f5f902", nil }), "rememberMarginOrder must not error")
	_, remembered = ex.marginOrderIDs.Load("6720da3fa30a360007f5f902")
	assert.True(t, remembered, "rememberMarginOrder should remember an order it placed")
}

func TestSubmitMarginOrder(t *testing.T) {
	t.Parallel()
	btcusdt := currency.NewBTCUSDT()
	oco := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.OCO, Price: 50000, Amount: 0.001, MarginType: margin.Isolated, AutoBorrow: true, ClientOrderID: "gct-margin-oco"}
	_, err := e.submitMarginOrder(t.Context(), oco, "BTC-USDT", "buy")
	require.ErrorIs(t, err, errTakeProfitPriceRequired, "submitMarginOrder must validate an OCO order's prices")
	oco.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, Price: 50000},
		StopLoss:   order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 70100},
	}
	_, err = e.submitMarginOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.StopLimit, Price: 1, Amount: 1, ClientOrderID: "gct-margin-stop-no-trigger"}, "BTC-USDT", "buy")
	require.ErrorIs(t, err, errStopPriceRequired, "submitMarginOrder must reject a stop order without a trigger")
	stop := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Sell, Type: order.Limit, Price: 39000, Amount: 0.001, AutoRepay: true, ClientOrderID: "gct-margin-stop", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}}
	limit := &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.Limit, Price: 50000, Amount: 0.001, MarginType: margin.Isolated, TimeInForce: order.PostOnly, AutoBorrow: true, AutoRepay: true, ClientOrderID: "gct-margin-limit"}

	t.Run("stop order of a market submission", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"vs8hoq2h6t43dlku003vff31"}}`)
		resp, err := ex.submitMarginOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Margin, MarginType: margin.Isolated, Side: order.Sell, Type: order.Market, Amount: 0.001, ClientOrderID: "gct-margin-stop-market", RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, Price: 40000}}}, "BTC-USDT", "sell")
		require.NoError(t, err, "submitMarginOrder must not error")
		resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
		exp := &order.SubmitResponse{Exchange: ex.Name, Type: order.StopMarket, Side: order.Sell, Pair: btcusdt, AssetType: asset.Margin, Amount: 0.001, TriggerPrice: 40000, ClientOrderID: "gct-margin-stop-market", MarginType: margin.Isolated, Status: order.New, OrderID: "vs8hoq2h6t43dlku003vff31"}
		assert.Equal(t, exp, resp, "submitMarginOrder should report a market submission with a stop loss as a new stop market order")
		require.Len(t, *got, 1, "the local server must receive the order")
		assert.Equal(t, "/api/v3/hf/margin/stop-order", (*got)[0].requestURI, "submitMarginOrder should place a submission with a stop loss as a stop order")
	})

	t.Run("auto repay without auto borrow", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"671bb90194422f00073ff4f3","clientOid":"gct-margin-auto-repay"}}`)
		_, err := ex.submitMarginOrder(t.Context(), &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Sell, Type: order.Limit, Price: 60000, Amount: 0.001, MarginType: margin.Isolated, AutoRepay: true, ClientOrderID: "gct-margin-auto-repay"}, "BTC-USDT", "sell")
		require.NoError(t, err, "submitMarginOrder must not error")
		require.Len(t, *got, 1, "the local server must receive the order")
		assert.JSONEq(t, `{"clientOid":"gct-margin-auto-repay","side":"sell","symbol":"BTC-USDT","type":"limit","price":"60000","size":"0.001","isIsolated":true,"autoRepay":true}`, (*got)[0].body, "submitMarginOrder should send auto repay apart from auto borrow")
	})

	t.Run("OCO orders", func(t *testing.T) {
		t.Parallel()
		ex, got := newSigningTestInstance(t, `{"code":"200000","data":{"orderId":"674c316e688dea0007c7b993"}}`)
		cases := []struct {
			name   string
			submit *order.Submit
			side   string
			exp    *order.SubmitResponse
			body   string
		}{
			{
				// The stop order sells at its limit price once the price falls to the stop loss price
				name:   "isolated sell",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Margin, MarginType: margin.Isolated, AutoRepay: true, Side: order.Sell, Type: order.OCO, Price: 70000, Amount: 0.001, ClientOrderID: "gct-margin-oco-sell", RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 49900}}},
				side:   "sell",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.OCO, Side: order.Sell, Pair: btcusdt, AssetType: asset.Margin, Price: 70000, Amount: 0.001, ClientOrderID: "gct-margin-oco-sell", MarginType: margin.Isolated, Status: order.New, OrderID: "674c316e688dea0007c7b993"},
				body:   `{"clientOid":"gct-margin-oco-sell","side":"sell","symbol":"BTC-USDT","price":"70000","size":"0.001","stopPrice":"50000","limitPrice":"49900","isIsolated":true,"autoRepay":true}`,
			},
			{
				// Price may be left unset, the take profit price being the limit order's
				name:   "cross buy without a price",
				submit: &order.Submit{Exchange: ex.Name, Pair: btcusdt, AssetType: asset.Margin, AutoBorrow: true, Side: order.Buy, Type: order.OCO, Amount: 0.001, ClientOrderID: "gct-margin-oco-buy", RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 70100}}},
				side:   "buy",
				exp:    &order.SubmitResponse{Exchange: ex.Name, Type: order.OCO, Side: order.Buy, Pair: btcusdt, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: "gct-margin-oco-buy", Status: order.New, OrderID: "674c316e688dea0007c7b993"},
				body:   `{"clientOid":"gct-margin-oco-buy","side":"buy","symbol":"BTC-USDT","price":"50000","size":"0.001","stopPrice":"70000","limitPrice":"70100","isIsolated":false,"autoBorrow":true}`,
			},
		}
		for _, tc := range cases {
			resp, err := ex.submitMarginOrder(t.Context(), tc.submit, "BTC-USDT", tc.side)
			require.NoErrorf(t, err, "submitMarginOrder must not error for an %s OCO order", tc.name)
			resp.Date, resp.LastUpdated = time.Time{}, time.Time{}
			assert.Equalf(t, tc.exp, resp, "submitMarginOrder should report an %s OCO order as a new OCO order at its take profit price", tc.name)
		}
		require.Len(t, *got, len(cases), "the local server must receive every order")
		for i, tc := range cases {
			assert.Equalf(t, "/api/v3/hf/margin/oco-order", (*got)[i].requestURI, "submitMarginOrder should place an %s submission as an OCO order", tc.name)
			assert.JSONEqf(t, tc.body, (*got)[i].body, "submitMarginOrder should place an %s OCO order's limit order at its take profit and its stop order at its stop loss", tc.name)
		}
	})

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
		{name: "an OCO order reply without data", side: "buy", err: common.ErrNoResponse, submit: &order.Submit{Exchange: e.Name, Pair: btcusdt, AssetType: asset.Margin, Side: order.Buy, Type: order.OCO, Price: 50000, Amount: 0.001, ClientOrderID: "gct-margin-oco-null", RiskManagementModes: oco.RiskManagementModes}},
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
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.OCO, Side: order.Buy, Pair: btcusdt, AssetType: asset.Margin, Price: 50000, Amount: 0.001, ClientOrderID: "gct-margin-oco", MarginType: margin.Isolated, Status: order.New, OrderID: "674c316e688dea0007c7b991"},
		},
		{
			name:   "cross stop loss",
			submit: stop,
			side:   "sell",
			exp:    &order.SubmitResponse{Exchange: e.Name, Type: order.StopLimit, Side: order.Sell, Pair: btcusdt, AssetType: asset.Margin, Price: 39000, Amount: 0.001, TriggerPrice: 40000, ClientOrderID: "gct-margin-stop", Status: order.New, OrderID: "vs8hoq2h6t43dlku003vff30"},
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
	sellTakeProfit := order.RiskManagement{Enabled: true, Price: 70000}
	sellStopLoss := order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 49900}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		err    error
	}{
		{name: "no take profit", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{StopLoss: sellStopLoss}}, err: errTakeProfitPriceRequired},
		{name: "a take profit without a price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true}, StopLoss: sellStopLoss}}, err: errTakeProfitPriceRequired},
		{name: "a take profit with a negative price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: -70000}, StopLoss: sellStopLoss}}, err: errTakeProfitPriceRequired},
		{name: "a take profit that is not enabled", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Price: 70000}, StopLoss: sellStopLoss}}, err: errTakeProfitPriceRequired},
		{name: "a price other than the take profit price", submit: &order.Submit{Side: order.Sell, Price: 69000, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: sellStopLoss}}, err: errOCOPriceMismatch},
		{name: "a negative price", submit: &order.Submit{Side: order.Sell, Price: -70000, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: sellStopLoss}}, err: errOCOPriceMismatch},
		{name: "no stop loss", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit}}, err: errStopLossPriceRequired},
		{name: "a stop loss without a price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: order.RiskManagement{Enabled: true, LimitPrice: 49900}}}, err: errStopLossPriceRequired},
		{name: "a stop loss with a negative price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: order.RiskManagement{Enabled: true, Price: -50000, LimitPrice: 49900}}}, err: errStopLossPriceRequired},
		{name: "a stop loss that is not enabled", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: order.RiskManagement{Price: 50000, LimitPrice: 49900}}}, err: errStopLossPriceRequired},
		// KuCoin's OCO order has no market stop order, so its stop order's limit price is required
		{name: "a stop loss without a limit price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: order.RiskManagement{Enabled: true, Price: 50000}}}, err: errLimitPriceRequired},
		{name: "a stop loss with a negative limit price", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: -1}}}, err: errLimitPriceRequired},
		{name: "a sell stop loss above its take profit", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 69900}}}, err: errOCOPricesInverted},
		{name: "a sell stop loss at its take profit", submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 49900}}}, err: errOCOPricesInverted},
		{name: "a buy stop loss below its take profit", submit: &order.Submit{Side: order.Buy, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 50000, LimitPrice: 50100}}}, err: errOCOPricesInverted},
		{name: "a buy stop loss at its take profit", submit: &order.Submit{Side: order.Buy, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 70000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 70100}}}, err: errOCOPricesInverted},
	} {
		_, _, _, err := ocoPrices(tc.submit)
		assert.ErrorIsf(t, err, tc.err, "ocoPrices should reject %s", tc.name)
	}

	type ocoOrderPrices struct {
		price, stopPrice, limitPrice float64
	}
	for _, tc := range []struct {
		name   string
		submit *order.Submit
		exp    ocoOrderPrices
	}{
		{
			// A sell OCO order takes profit above the market and stops a loss below it
			name:   "sell",
			submit: &order.Submit{Side: order.Sell, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: sellStopLoss}},
			exp:    ocoOrderPrices{price: 70000, stopPrice: 50000, limitPrice: 49900},
		},
		{
			name:   "sell priced at its take profit",
			submit: &order.Submit{Side: order.Sell, Price: 70000, RiskManagementModes: order.RiskManagementModes{TakeProfit: sellTakeProfit, StopLoss: sellStopLoss}},
			exp:    ocoOrderPrices{price: 70000, stopPrice: 50000, limitPrice: 49900},
		},
		{
			name:   "buy",
			submit: &order.Submit{Side: order.Bid, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, Price: 50000}, StopLoss: order.RiskManagement{Enabled: true, Price: 70000, LimitPrice: 70100}}},
			exp:    ocoOrderPrices{price: 50000, stopPrice: 70000, limitPrice: 70100},
		},
	} {
		price, stopPrice, limitPrice, err := ocoPrices(tc.submit)
		require.NoErrorf(t, err, "ocoPrices must not error for a %s order", tc.name)
		assert.Equalf(t, tc.exp, ocoOrderPrices{price: price, stopPrice: stopPrice, limitPrice: limitPrice}, "ocoPrices should place a %s order's limit order at its take profit and its stop order at its stop loss", tc.name)
	}
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
	for _, tc := range []struct {
		name, reply string
		err         error
	}{
		{name: "an isolated margin reply without accounts", reply: `{"code":"200000","data":null}`, err: common.ErrNoResponse},
		// An isolated margin account without a symbol would be saved over the cross margin account, sub-account ""
		{name: "an isolated margin account without a symbol", reply: `{"code":"200000","data":{"assets":[{"symbol":"","baseAsset":{"currency":"BTC","total":"1"},"quoteAsset":{"currency":"USDT","total":"1"}}]}}`, err: currency.ErrSymbolStringEmpty},
	} {
		ex = marketTestServerExchange(t, map[string]string{
			"/api/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[]}}`,
			"/api/v3/isolated/accounts": tc.reply,
		})
		_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
		require.ErrorIsf(t, err, tc.err, "UpdateAccountBalances must reject %s", tc.name)
	}
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[]}}`,
		"/api/v3/isolated/accounts": `{"code":"400100","msg":"isolated accounts unavailable"}`,
	})
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.ErrorContains(t, err, "isolated accounts unavailable", "UpdateAccountBalances must return the failed isolated margin request's error")

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

	// A listing replaces the one before: a currency or account it leaves out goes, and a later push does not bring it back
	listings := []string{
		`{"code":"200000","data":[
			{"id":"548674591753","currency":"ETH","type":"trade","balance":"2","available":"2","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"ETH","type":"main","balance":"3","available":"3","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"BTC","type":"main","balance":"1","available":"1","holds":"0"}]}`,
		`{"code":"200000","data":[{"id":"548674591753","currency":"ETH","type":"trade","balance":"2","available":"2","holds":"0"}]}`,
		`{"code":"200000","data":[
			{"id":"548674591753","currency":"ETH","type":"trade","balance":"2","available":"2","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"ETH","type":"main","balance":"30","available":"30","holds":"0"}]}`,
		`{"code":"200000","data":[
			{"id":"548674591753","currency":"SOL","type":"trade_hf","balance":"10","available":"10","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"SOL","type":"main","balance":"20","available":"20","holds":"0"},
			{"id":"548674591753","currency":"XRP","type":"trade","balance":"7","available":"7","holds":"0"},
			{"id":"548674591753","currency":"XRP","type":"trade_hf","balance":"10","available":"10","holds":"0"},
			{"id":"63355cd156298d0001b66e61","currency":"XRP","type":"main","balance":"20","available":"20","holds":"0"}]}`,
	}
	var reply atomic.Pointer[string]
	reply.Store(&listings[0])
	ex = newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, *reply.Load())
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	reply.Store(&listings[1])
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for a shorter listing")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	_, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "UpdateAccountBalances should drop a currency the listing leaves out")
	pushed := time.UnixMilli(time.Now().Add(time.Minute).UnixMilli())
	push := `{"currency":"ETH","total":"2.5","available":"2.5","hold":"0","relationEvent":"trade.hold","time":"` + strconv.FormatInt(pushed.UnixMilli(), 10) + `"}`
	require.NoError(t, ex.processAccountBalance(t.Context(), []byte(push)), "processAccountBalance must not error")
	eth, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "the ETH balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.ETH, Total: 2.5, Free: 2.5, UpdatedAt: pushed}, eth, "a push should not bring back the share of an account the listing left out")

	// A listing that leaves a currency out of an account leaves no share of it, so a push older than the listing, which
	// it holds the account's state after, must still be refused rather than add back the share the listing removed
	twoAccounts := `{"code":"200000","data":[
		{"id":"548674591753","currency":"USDT","type":"trade","balance":"10","available":"10","holds":"0"},
		{"id":"63355cd156298d0001b66e61","currency":"USDT","type":"main","balance":"20","available":"20","holds":"0"}]}`
	mainOnly := `{"code":"200000","data":[{"id":"63355cd156298d0001b66e61","currency":"USDT","type":"main","balance":"20","available":"20","holds":"0"}]}`
	var staleReply atomic.Pointer[string]
	staleReply.Store(&twoAccounts)
	stale := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, *staleReply.Load())
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err = stale.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	between := time.UnixMilli(time.Now().UnixMilli() + 1)
	time.Sleep(time.Until(between.Add(2 * time.Millisecond)))
	staleReply.Store(&mainOnly)
	_, err = stale.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for a listing without the trade account")
	tradePush := func(at time.Time) []byte {
		return []byte(`{"currency":"USDT","total":"10","available":"10","hold":"0","relationEvent":"trade.hold","time":"` + strconv.FormatInt(at.UnixMilli(), 10) + `"}`)
	}
	err = stale.processAccountBalance(t.Context(), tradePush(between))
	require.ErrorIs(t, err, errBalancePushOutOfSequence, "processAccountBalance must refuse a push older than the listing that left its account's currency out")
	staleCreds, err := stale.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	usdt, err := stale.Accounts.GetBalance("", staleCreds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the USDT balance must be stored")
	assert.Equal(t, 20.0, usdt.Total, "a push older than the listing should not add back the share the listing removed")
	later := time.UnixMilli(time.Now().Add(time.Minute).UnixMilli())
	require.NoError(t, stale.processAccountBalance(t.Context(), tradePush(later)), "processAccountBalance must not error for a push newer than the listing")
	usdt, err = stale.Accounts.GetBalance("", staleCreds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the USDT balance must be stored")
	assert.Equal(t, 30.0, usdt.Total, "a push newer than the listing should add its account's share")

	// A listing older than a push for a currency it does not list keeps that push
	btcPush := `{"currency":"BTC","total":"1","available":"1","hold":"0","relationEvent":"main.deposit","time":"` + strconv.FormatInt(pushed.UnixMilli(), 10) + `"}`
	require.NoError(t, ex.processAccountBalance(t.Context(), []byte(btcPush)), "processAccountBalance must not error")
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	btc, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.BTC)
	require.NoError(t, err, "the BTC balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.BTC, Total: 1, Free: 1, UpdatedAt: pushed}, btc, "a listing older than a push should keep the push for a currency it does not list")

	// A listing the store refuses leaves the shares as they were
	future := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	future[0].Balances.Set(currency.ETH, accounts.Balance{Total: 2.5, Free: 2.5, UpdatedAt: pushed.Add(time.Hour)})
	require.NoError(t, ex.Accounts.Save(t.Context(), future, false), "Save must not error")
	reply.Store(&listings[2])
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	assert.ErrorContains(t, err, "out of sequence", "UpdateAccountBalances should return the store's refusal")
	push = `{"currency":"ETH","total":"3","available":"3","hold":"0","relationEvent":"trade.hold","time":"` + strconv.FormatInt(pushed.Add(2*time.Hour).UnixMilli(), 10) + `"}`
	require.NoError(t, ex.processAccountBalance(t.Context(), []byte(push)), "processAccountBalance must not error")
	eth, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.ETH)
	require.NoError(t, err, "the ETH balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.ETH, Total: 3, Free: 3, UpdatedAt: pushed.Add(2 * time.Hour)}, eth, "a listing the store refused should not change the shares")

	// The HF trade account is the trade account, whether REST or a push names it
	reply.Store(&listings[3])
	ex = newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, *reply.Load())
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	xrp, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.XRP)
	require.NoError(t, err, "the XRP balance must be stored")
	assert.Equal(t, 37.0, xrp.Total, "UpdateAccountBalances should add every account it lists, trade and HF trade alike")
	push = `{"currency":"SOL","total":"11","available":"11","hold":"0","relationEvent":"trade_hf.setted","time":"` + strconv.FormatInt(pushed.UnixMilli(), 10) + `"}` // codespell:ignore setted
	require.NoError(t, ex.processAccountBalance(t.Context(), []byte(push)), "processAccountBalance must not error")
	sol, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.SOL)
	require.NoError(t, err, "the SOL balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.SOL, Total: 31, Free: 31, UpdatedAt: pushed}, sol, "an HF trade account push should replace the HF trade account's listed share")

	// A push that arrives while the listing is answered keeps its account's share
	var inFlight time.Time
	ex = newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(2 * time.Millisecond) // So the push lands a whole millisecond after the request starts
		inFlight = time.UnixMilli(time.Now().UnixMilli())
		push := `{"currency":"SOL","total":"11","available":"11","hold":"0","relationEvent":"trade.hold","time":"` + strconv.FormatInt(inFlight.UnixMilli(), 10) + `"}`
		assert.NoError(t, ex.processAccountBalance(t.Context(), []byte(push)), "processAccountBalance should not error")
		_, err := io.WriteString(w, *reply.Load())
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	sol, err = ex.Accounts.GetBalance("", creds, asset.Spot, currency.SOL)
	require.NoError(t, err, "the SOL balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.SOL, Total: 31, Free: 31, UpdatedAt: inFlight}, sol, "a listing should keep the share of a push that arrived while it was answered")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ex = testInstance(t)
	for _, tc := range []struct {
		asset asset.Item
		exp   accounts.SubAccounts
	}{
		{
			asset: asset.Spot,
			exp: accounts.SubAccounts{{AssetType: asset.Spot, Balances: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.01, Free: 0.01},
				currency.USDT: {Currency: currency.USDT, Total: 26.66759503, Hold: 6, Free: 20.66759503},
			}}},
		},
		{
			// The cross margin account; the isolated margin accounts are saved beside it under their symbols
			asset: asset.Margin,
			exp: accounts.SubAccounts{{AssetType: asset.Margin, Balances: accounts.CurrencyBalances{
				currency.BTC:  {Currency: currency.BTC, Total: 0.0001, Hold: 0.00002, Free: 0.00008, Borrowed: 0.000001},
				currency.USDT: {Currency: currency.USDT, Total: 38.68855864, Hold: 18.66939173, Free: 20.01916691, Borrowed: 12.5},
			}}},
		},
		{
			// Hold is the position margin, order margin and frozen funds
			asset: asset.Futures,
			exp: accounts.SubAccounts{{AssetType: asset.Futures, Balances: accounts.CurrencyBalances{
				currency.USDT: {Currency: currency.USDT, Total: 198.733127406, Hold: 25.8, Free: 172.933127406},
				currency.USDC: {Currency: currency.USDC, Total: 50.5, Hold: 12.5, Free: 38},
				currency.XBT:  {Currency: currency.XBT, Total: 0.0025, Hold: 0.0007000000000000001, Free: 0.0018},
				currency.ETH:  {Currency: currency.ETH, Total: 0.15, Hold: 0.034999999999999996, Free: 0.115},
				currency.SOL:  {Currency: currency.SOL, Total: 3.5, Hold: 1.5, Free: 2},
				currency.XRP:  {Currency: currency.XRP, Total: 120, Hold: 45, Free: 75},
			}}},
		},
	} {
		result, err := ex.UpdateAccountBalances(t.Context(), tc.asset)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", tc.asset)
		if mockTests {
			for _, s := range result {
				clearSavedTimes(t, s)
			}
			assert.Equalf(t, tc.exp, result, "UpdateAccountBalances should return the %s balances", tc.asset)
			continue
		}
		require.NotEmptyf(t, result, "UpdateAccountBalances must return a %s sub-account", tc.asset)
		assert.Equalf(t, tc.asset, result[0].AssetType, "UpdateAccountBalances should return a %s sub-account", tc.asset)
	}
}

// balancePushAt returns an account balance push of one figure for every amount, made at the given time
func balancePushAt(event, ccy string, total float64, at time.Time) []byte {
	amount := strconv.FormatFloat(total, 'f', -1, 64)
	return []byte(`{"currency":"` + ccy + `","total":"` + amount + `","available":"` + amount + `","hold":"0","relationEvent":"` + event + `","time":"` + strconv.FormatInt(at.UnixMilli(), 10) + `"}`)
}

// TestProcessAccountBalanceMarginPushOlderThanListing checks a margin push older than the listing that left out its
// isolated margin account or cross margin currency is refused, as no balance is left for the store to refuse it by
func TestProcessAccountBalanceMarginPushOlderThanListing(t *testing.T) {
	t.Parallel()
	first := []string{
		`{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"500","available":"500","hold":"0","liability":"50"},{"currency":"ETH","total":"2","available":"2","hold":"0","liability":"0"}]}}`,
		`{"code":"200000","data":{"assets":[
			{"symbol":"BTC-USDT","status":"BORROW","baseAsset":{"currency":"BTC","total":"0.01","available":"0.01","hold":"0","liability":"0"},"quoteAsset":{"currency":"USDT","total":"100","available":"100","hold":"0","liability":"25"}},
			{"symbol":"ETH-USDT","status":"EFFECTIVE","baseAsset":{"currency":"ETH","total":"0.5","available":"0.5","hold":"0","liability":"0"},"quoteAsset":{"currency":"USDT","total":"40","available":"40","hold":"0","liability":"0"}}]}}`,
	}
	second := []string{
		`{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"500","available":"500","hold":"0","liability":"50"}]}}`,
		`{"code":"200000","data":{"assets":[
			{"symbol":"BTC-USDT","status":"BORROW","baseAsset":{"currency":"BTC","total":"0.01","available":"0.01","hold":"0","liability":"0"},"quoteAsset":{"currency":"USDT","total":"100","available":"100","hold":"0","liability":"25"}}]}}`,
	}
	var listing atomic.Pointer[[]string]
	listing.Store(&first)
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"400100","msg":"unexpected request"}`
		switch r.URL.Path {
		case "/v3/margin/accounts":
			reply = (*listing.Load())[0]
		case "/v3/isolated/accounts":
			reply = (*listing.Load())[1]
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	// Pushes carry KuCoin's millisecond times, so the push falls between the two listings by a whole millisecond
	between := time.UnixMilli(time.Now().UnixMilli() + 1)
	time.Sleep(time.Until(between.Add(2 * time.Millisecond)))
	listing.Store(&second)
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error for a listing without ETH-USDT and cross margin ETH")
	for _, tc := range []struct {
		event, subAccount string
		code              currency.Code
	}{
		{event: "isolatedV2_ETH-USDT.transfer", subAccount: "ETH-USDT", code: currency.USDT},
		{event: "marginV2.transfer", code: currency.ETH},
	} {
		err := ex.processAccountBalance(t.Context(), balancePushAt(tc.event, tc.code.String(), 40, between))
		require.ErrorIsf(t, err, errBalancePushOutOfSequence, "processAccountBalance must refuse the %s push older than the listing that left it out", tc.event)
		_, err = ex.Accounts.GetBalance(tc.subAccount, creds, asset.Margin, tc.code)
		assert.ErrorIsf(t, err, accounts.ErrNoBalances, "the %s push older than the listing should not bring back the balance the listing left out", tc.event)
		later := time.UnixMilli(time.Now().Add(time.Minute).UnixMilli())
		require.NoErrorf(t, ex.processAccountBalance(t.Context(), balancePushAt(tc.event, tc.code.String(), 40, later)), "processAccountBalance must not error for the %s push newer than the listing", tc.event)
	}
}

// TestProcessAccountBalanceRefusedPushKeepsShare checks a push whose sum the store refuses leaves its account's share as it was
func TestProcessAccountBalanceRefusedPushKeepsShare(t *testing.T) {
	t.Parallel()
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"code":"200000","data":[{"id":"1","currency":"USDT","type":"trade","balance":"10","available":"10","holds":"0"},{"id":"2","currency":"USDT","type":"main","balance":"20","available":"20","holds":"0"}]}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	pushed := time.UnixMilli(time.Now().Add(time.Minute).UnixMilli())
	later := accounts.SubAccounts{accounts.NewSubAccount(asset.Spot, "")}
	later[0].Balances.Set(currency.USDT, accounts.Balance{Total: 99, Free: 99, UpdatedAt: pushed.Add(time.Hour)})
	require.NoError(t, ex.Accounts.Save(t.Context(), later, false), "Save must not error")
	err = ex.processAccountBalance(t.Context(), balancePushAt("trade.hold", "USDT", 50, pushed))
	require.ErrorContains(t, err, "out of sequence", "processAccountBalance must return the store's refusal")
	require.NoError(t, ex.processAccountBalance(t.Context(), balancePushAt("main.deposit", "USDT", 25, pushed.Add(2*time.Hour))), "processAccountBalance must not error for a push newer than the stored balance")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	b, err := ex.Accounts.GetBalance("", creds, asset.Spot, currency.USDT)
	require.NoError(t, err, "the spot USDT balance must be stored")
	assert.Equal(t, 35.0, b.Total, "a push the store refused should not replace its account's share")
}

// balanceListingCases are a spot and a margin listing, each holding USDT alone, with the relation event of a push for a
// currency or account the listing leaves out
var balanceListingCases = []struct {
	a                asset.Item
	replies          map[string]string
	event, ccy       string
	pushedTotal, exp float64
}{
	{
		a:       asset.Spot,
		replies: map[string]string{"/v1/accounts": `{"code":"200000","data":[{"id":"2","currency":"USDT","type":"main","balance":"20","available":"20","holds":"0"}]}`},
		event:   "trade.hold", ccy: "USDT", pushedTotal: 10, exp: 30,
	},
	{
		a: asset.Margin,
		replies: map[string]string{
			"/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"20","available":"20","hold":"0","liability":"0"}]}}`,
			"/v3/isolated/accounts": `{"code":"200000","data":{"assets":[]}}`,
		},
		event: "marginV2.transfer", ccy: "ETH", pushedTotal: 40, exp: 40,
	},
}

// heldListing starts listing an asset's balances against a local KuCoin answering each path with its reply, holding the
// first request until the returned release is called, and returns once that request has arrived; wait returns the
// listing's error once released
func heldListing(t *testing.T, a asset.Item, replies map[string]string) (ex *Exchange, release func(), wait func() error) {
	t.Helper()
	started, held := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release = func() { releaseOnce.Do(func() { close(held) }) }
	var requests atomic.Int32
	ex = newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			close(started)
			<-held
		}
		reply, ok := replies[r.URL.Path]
		if !assert.Truef(t, ok, "request path %s should be one the test serves", r.URL.Path) {
			reply = `{"code":"400100","msg":"unexpected request"}`
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	t.Cleanup(release)
	done := make(chan error, 1)
	go func() {
		_, err := ex.UpdateAccountBalances(t.Context(), a)
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the held listing's request must reach the server")
	}
	return ex, release, func() error {
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			require.FailNow(t, "the held listing must return once its reply is released")
		}
		return nil
	}
}

// TestUpdateAccountBalancesNewestListingTime checks a listing answered after a newer one does not move the listing time back
func TestUpdateAccountBalancesNewestListingTime(t *testing.T) {
	t.Parallel()
	for _, tc := range balanceListingCases {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex, release, wait := heldListing(t, tc.a, tc.replies)
			time.Sleep(2 * time.Millisecond)
			between := time.UnixMilli(time.Now().UnixMilli())
			time.Sleep(2 * time.Millisecond)
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "the newer listing must not error")
			release()
			require.NoError(t, wait(), "the older listing must not error")
			err = ex.processAccountBalance(t.Context(), balancePushAt(tc.event, tc.ccy, tc.pushedTotal, between))
			require.ErrorIs(t, err, errBalancePushOutOfSequence, "processAccountBalance must refuse a push older than the newest listing that left its currency out")
		})
	}
}

// TestUpdateAccountBalancesListingTimeMilliseconds checks a listing's time is taken to the millisecond, as pushes carry
// their times, so a push made in the listing's millisecond is not refused as older than the listing
func TestUpdateAccountBalancesListingTimeMilliseconds(t *testing.T) {
	t.Parallel()
	for _, tc := range balanceListingCases {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				_, err := io.WriteString(w, tc.replies[r.URL.Path])
				assert.NoError(t, err, "writing the reply should not error")
			})
			_, err := ex.UpdateAccountBalances(t.Context(), tc.a)
			require.NoError(t, err, "UpdateAccountBalances must not error")
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			var listedAt time.Time
			if tc.a == asset.Spot {
				ex.spotAccountBalancesMu.Lock()
				listedAt = ex.spotAccountsListedAt[*creds]
				ex.spotAccountBalancesMu.Unlock()
			} else {
				ex.marginBalanceMu.Lock()
				listedAt = ex.marginAccountsListedAt[*creds]
				ex.marginBalanceMu.Unlock()
			}
			require.False(t, listedAt.IsZero(), "the listing's time must be recorded")
			sameMillisecond := time.UnixMilli(listedAt.UnixMilli())
			assert.NoError(t, ex.processAccountBalance(t.Context(), balancePushAt(tc.event, tc.ccy, tc.pushedTotal, sameMillisecond)), "processAccountBalance should save a push made in the listing's millisecond")
		})
	}
}

// TestUpdateAccountBalancesMarginKeepsNewerPushes checks a margin listing keeps the balances that pushes saved while it
// was answered, stamped after its time, over its figures, with its own liability, and that a push made while it was
// answered is saved once it has been, as for spot
func TestUpdateAccountBalancesMarginKeepsNewerPushes(t *testing.T) {
	t.Parallel()
	ex, release, wait := heldListing(t, asset.Margin, map[string]string{
		"/v3/margin/accounts":   `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"20","available":"15","hold":"5","liability":"5"}]}}`,
		"/v3/isolated/accounts": `{"code":"200000","data":{"assets":[{"symbol":"BTC-USDT","status":"BORROW","baseAsset":{"currency":"BTC","total":"0.01","available":"0.01","hold":"0","liability":"0.001"},"quoteAsset":{"currency":"USDT","total":"100","available":"100","hold":"0","liability":"25"}}]}}`,
	})
	time.Sleep(2 * time.Millisecond)
	pushedAt := time.UnixMilli(time.Now().UnixMilli())
	for _, p := range []struct {
		event, ccy string
		total      float64
	}{
		{event: "marginV2.transfer", ccy: "USDT", total: 40},
		{event: "marginV2.transfer", ccy: "ETH", total: 2},
		{event: "isolatedV2_BTC-USDT.transfer", ccy: "USDT", total: 90},
		{event: "isolatedV2_ETH-USDT.transfer", ccy: "USDT", total: 30},
	} {
		require.NoErrorf(t, ex.processAccountBalance(t.Context(), balancePushAt(p.event, p.ccy, p.total, pushedAt)), "processAccountBalance must not error for the %s %s push", p.event, p.ccy)
	}
	time.Sleep(2 * time.Millisecond)
	during := time.UnixMilli(time.Now().UnixMilli())
	release()
	require.NoError(t, wait(), "the listing must not error")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	for _, tc := range []struct {
		subAccount string
		exp        accounts.Balance
	}{
		{exp: accounts.Balance{Currency: currency.USDT, Total: 40, Free: 40, Borrowed: 5, UpdatedAt: pushedAt}},
		{exp: accounts.Balance{Currency: currency.ETH, Total: 2, Free: 2, UpdatedAt: pushedAt}},
		{subAccount: "BTC-USDT", exp: accounts.Balance{Currency: currency.USDT, Total: 90, Free: 90, Borrowed: 25, UpdatedAt: pushedAt}},
		{subAccount: "ETH-USDT", exp: accounts.Balance{Currency: currency.USDT, Total: 30, Free: 30, UpdatedAt: pushedAt}},
	} {
		b, err := ex.Accounts.GetBalance(tc.subAccount, creds, asset.Margin, tc.exp.Currency)
		require.NoErrorf(t, err, "the margin %q %s balance must be stored", tc.subAccount, tc.exp.Currency)
		assert.Equalf(t, tc.exp, b, "the listing should keep the margin %q %s balance a push saved while it was answered, with the listing's liability", tc.subAccount, tc.exp.Currency)
	}
	require.NoError(t, ex.processAccountBalance(t.Context(), balancePushAt("isolatedV2_BTC-USDT.transfer", "BTC", 0.02, during)), "processAccountBalance must save a push made while the listing was answered")
	b, err := ex.Accounts.GetBalance("BTC-USDT", creds, asset.Margin, currency.BTC)
	require.NoError(t, err, "the isolated margin BTC balance must be stored")
	assert.Equal(t, accounts.Balance{Currency: currency.BTC, Total: 0.02, Free: 0.02, Borrowed: 0.001, UpdatedAt: during}, b, "a push made while the listing was answered should replace the listing's figures, keeping its liability")

	// A later listing is newer than those pushes, so its figures replace them
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "the later listing must not error")
	b, err = ex.Accounts.GetBalance("", creds, asset.Margin, currency.USDT)
	require.NoError(t, err, "the cross margin USDT balance must be stored")
	assert.Equal(t, 20.0, b.Total, "a listing newer than a push should replace its figures")
	_, err = ex.Accounts.GetBalance("", creds, asset.Margin, currency.ETH)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "a listing newer than a push should remove the currency it no longer lists")
	_, err = ex.Accounts.GetBalance("ETH-USDT", creds, asset.Margin, currency.USDT)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "a listing newer than a push should empty the isolated margin account it no longer lists")
}

// TestUpdateAccountBalancesMarginKeptPushLiability checks a balance a push saved while a margin listing was answered is
// kept with that listing's liability, none for a currency the listing leaves out, rather than the earlier liability the
// push kept
func TestUpdateAccountBalancesMarginKeptPushLiability(t *testing.T) {
	t.Parallel()
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var listings atomic.Int32
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"200000","data":{"assets":[]}}`
		if r.URL.Path == "/v3/margin/accounts" {
			reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"20","available":"20","hold":"0","liability":"5"},{"currency":"ETH","total":"1","available":"1","hold":"0","liability":"3"}]}}`
			if listings.Add(1) == 2 {
				close(started)
				<-release
				reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"30","available":"30","hold":"0","liability":"7"}]}}`
			}
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "the first listing must not error")
	type listing struct {
		subAccts accounts.SubAccounts
		err      error
	}
	second := make(chan listing, 1)
	go func() {
		subAccts, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
		second <- listing{subAccts: subAccts, err: err}
	}()
	received(t, started, "the second listing's request must reach the server")
	time.Sleep(2 * time.Millisecond)
	pushedAt := time.UnixMilli(time.Now().UnixMilli())
	for _, p := range []struct {
		ccy   string
		total float64
	}{{ccy: "USDT", total: 40}, {ccy: "ETH", total: 2}} {
		require.NoErrorf(t, ex.processAccountBalance(t.Context(), balancePushAt("marginV2.transfer", p.ccy, p.total, pushedAt)), "processAccountBalance must not error for the %s push", p.ccy)
	}
	releaseOnce.Do(func() { close(release) })
	got := received(t, second, "the second listing must return once its reply is released")
	require.NoError(t, got.err, "the second listing must not error")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	exp := accounts.SubAccounts{accounts.NewSubAccount(asset.Margin, "")}
	exp[0].Balances.Set(currency.USDT, accounts.Balance{Total: 40, Free: 40, Borrowed: 7, UpdatedAt: pushedAt})
	exp[0].Balances.Set(currency.ETH, accounts.Balance{Total: 2, Free: 2, UpdatedAt: pushedAt})
	for c, expBalance := range exp[0].Balances {
		b, err := ex.Accounts.GetBalance("", creds, asset.Margin, c)
		require.NoErrorf(t, err, "the cross margin %s balance must be stored", c)
		assert.Equalf(t, expBalance, b, "the listing should keep the %s balance a push saved while it was answered, with the listing's liability", c)
	}
	assert.Equal(t, exp, got.subAccts, "UpdateAccountBalances should return the cross margin account as saved, with the balances kept from pushes")
}

// TestUpdateAccountBalancesMarginOlderListingAnsweredLast checks a margin listing answered after a newer one was saved
// leaves the newer listing's figures, liability included, and brings back no currency the newer listing left out
func TestUpdateAccountBalancesMarginOlderListingAnsweredLast(t *testing.T) {
	t.Parallel()
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var requests atomic.Int32
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"200000","data":{"assets":[]}}`
		if r.URL.Path == "/v3/margin/accounts" {
			reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"30","available":"30","hold":"0","liability":"9"}]}}`
			if requests.Add(1) == 1 {
				close(started)
				<-release
				reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"20","available":"20","hold":"0","liability":"5"},{"currency":"ETH","total":"2","available":"2","hold":"0","liability":"0"}]}}`
			}
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	older := make(chan error, 1)
	go func() {
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
		older <- err
	}()
	received(t, started, "the older listing's request must reach the server")
	time.Sleep(2 * time.Millisecond)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "the newer listing must not error")
	releaseOnce.Do(func() { close(release) })
	require.NoError(t, received(t, older, "the older listing must return once its reply is released"), "the older listing must not error")
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	b, err := ex.Accounts.GetBalance("", creds, asset.Margin, currency.USDT)
	require.NoError(t, err, "the cross margin USDT balance must be stored")
	assert.Equal(t, 30.0, b.Total, "an older listing answered last should leave the newer listing's figure")
	assert.Equal(t, 9.0, b.Borrowed, "an older listing answered last should leave the newer listing's liability")
	_, err = ex.Accounts.GetBalance("", creds, asset.Margin, currency.ETH)
	assert.ErrorIs(t, err, accounts.ErrNoBalances, "an older listing answered last should bring back no currency the newer listing left out")
}

// TestUpdateAccountBalancesMarginListingTimePerCredentials checks a margin listing is saved though another set of
// credentials' listing, newer than it, was saved while it was answered, as each set's accounts are their own
func TestUpdateAccountBalancesMarginListingTimePerCredentials(t *testing.T) {
	t.Parallel()
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	var requests atomic.Int32
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"200000","data":{"assets":[]}}`
		if r.URL.Path == "/v3/margin/accounts" {
			reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"20","available":"20","hold":"0","liability":"5"}]}}`
			if requests.Add(1) == 1 {
				close(started)
				<-release
			}
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	other := accounts.DeployCredentialsToContext(t.Context(), &accounts.Credentials{Key: "other-api-key", Secret: "other-api-secret", ClientID: "other-passphrase"})
	held := make(chan error, 1)
	go func() {
		_, err := ex.UpdateAccountBalances(other, asset.Margin)
		held <- err
	}()
	received(t, started, "the other credentials' listing must reach the server")
	time.Sleep(2 * time.Millisecond)
	_, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "the newer listing must not error")
	releaseOnce.Do(func() { close(release) })
	require.NoError(t, received(t, held, "the other credentials' listing must return once its reply is released"), "the other credentials' listing must not error")
	otherCreds, err := ex.GetCredentials(other)
	require.NoError(t, err, "GetCredentials must not error for the other credentials")
	b, err := ex.Accounts.GetBalance("", otherCreds, asset.Margin, currency.USDT)
	require.NoError(t, err, "the other credentials' listing must be saved though another set's newer listing was saved first")
	assert.Equal(t, 20.0, b.Total, "the other credentials' listing should save its figures")
}

// TestUpdateAccountBalancesStampedBeforeRequests checks a listing's time is taken before its requests are sent, so a push
// made while the listing is answered is not refused as older than the listing, which could not hold it
func TestUpdateAccountBalancesStampedBeforeRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range balanceListingCases {
		t.Run(tc.a.String(), func(t *testing.T) {
			t.Parallel()
			ex, release, wait := heldListing(t, tc.a, tc.replies)
			time.Sleep(2 * time.Millisecond)
			during := time.UnixMilli(time.Now().UnixMilli())
			time.Sleep(2 * time.Millisecond)
			release()
			require.NoError(t, wait(), "the listing must not error")
			require.NoError(t, ex.processAccountBalance(t.Context(), balancePushAt(tc.event, tc.ccy, tc.pushedTotal, during)), "processAccountBalance must save a push made while the listing was answered")
			creds, err := ex.GetCredentials(t.Context())
			require.NoError(t, err, "GetCredentials must not error")
			b, err := ex.Accounts.GetBalance("", creds, tc.a, currency.NewCode(tc.ccy))
			require.NoError(t, err, "the pushed currency's balance must be stored")
			assert.Equal(t, tc.exp, b.Total, "the push made while the listing was answered should be saved")
		})
	}
}

// TestUpdateAccountBalancesIsolatedMargin covers the isolated margin accounts, each the margin sub-account of its
// symbol: a REST listing replaces them, emptying a symbol it no longer lists, and a push updates its own symbol's
// balance, keeping the liability REST saved, apart from the cross margin account
func TestUpdateAccountBalancesIsolatedMargin(t *testing.T) {
	t.Parallel()
	listings := []string{
		`{"code":"200000","data":{"assets":[
			{"symbol":"BTC-USDT","status":"BORROW","baseAsset":{"currency":"BTC","total":"0.01","available":"0.008","hold":"0.002","liability":"0.001"},"quoteAsset":{"currency":"USDT","total":"100","available":"90","hold":"10","liability":"25"}},
			{"symbol":"ETH-USDT","status":"EFFECTIVE","baseAsset":{"currency":"ETH","total":"0.5","available":"0.5","hold":"0","liability":"0"},"quoteAsset":{"currency":"USDT","total":"40","available":"40","hold":"0","liability":"0"}}]}}`,
		`{"code":"200000","data":{"assets":[
			{"symbol":"BTC-USDT","status":"BORROW","baseAsset":{"currency":"BTC","total":"0.02","available":"0.02","hold":"0","liability":"0.001"},"quoteAsset":{"currency":"USDT","total":"60","available":"60","hold":"0","liability":"30"}}]}}`,
	}
	var listing atomic.Int32
	var queries []string
	var mu sync.Mutex
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"400100","msg":"unexpected request"}`
		switch r.URL.Path {
		case "/v3/margin/accounts":
			reply = `{"code":"200000","data":{"accounts":[{"currency":"USDT","total":"500","available":"400","hold":"100","liability":"50"}]}}`
		case "/v3/isolated/accounts":
			mu.Lock()
			queries = append(queries, r.URL.RawQuery)
			mu.Unlock()
			reply = listings[listing.Load()]
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	ex.Name = t.Name()
	creds, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	stored := func(subAccount string, c currency.Code) accounts.Balance {
		t.Helper()
		b, err := ex.Accounts.GetBalance(subAccount, creds, asset.Margin, c)
		require.NoErrorf(t, err, "the %q %s margin balance must be stored", subAccount, c)
		b.UpdatedAt = time.Time{}
		return b
	}

	result, err := ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error")
	assert.Equal(t, []string{"queryType=ISOLATED"}, queries, "UpdateAccountBalances should list the isolated margin accounts KuCoin prefers")
	// The backtester's live funding sets each currency from one returned sub-account after another, so an isolated
	// margin account returned after the cross margin account would replace the cross margin balance
	require.Len(t, result, 1, "UpdateAccountBalances must return the cross margin account alone")
	assert.Empty(t, result[0].ID, "UpdateAccountBalances should return the cross margin account")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 500, Hold: 100, Free: 400, Borrowed: 50}, stored("", currency.USDT), "UpdateAccountBalances should save the cross margin account as sub-account \"\"")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 100, Hold: 10, Free: 90, Borrowed: 25}, stored("BTC-USDT", currency.USDT), "UpdateAccountBalances should save an isolated margin account under its symbol")
	assert.Equal(t, accounts.Balance{Currency: currency.BTC, Total: 0.01, Hold: 0.002, Free: 0.008, Borrowed: 0.001}, stored("BTC-USDT", currency.BTC), "UpdateAccountBalances should save an isolated margin account's base currency")
	assert.Equal(t, accounts.Balance{Currency: currency.ETH, Total: 0.5, Free: 0.5}, stored("ETH-USDT", currency.ETH), "UpdateAccountBalances should save each isolated margin account")
	marginBalances, err := ex.Accounts.CurrencyBalances(creds, asset.Margin)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, 640.0, marginBalances[currency.USDT].Total, "the margin balance should be the cross and isolated margin accounts together")

	// A push changes its own account's balance alone. Pushes carry KuCoin's millisecond times, which must fall after the
	// listing before them and before the one after, as the store refuses an older balance
	pushed := time.UnixMilli(time.Now().UnixMilli() + 2)
	frame := func(event string, total float64) []byte {
		amount := strconv.FormatFloat(total, 'f', -1, 64)
		return []byte(`{"currency":"USDT","total":"` + amount + `","available":"` + amount + `","hold":"0","relationEvent":"` + event + `","time":"` + strconv.FormatInt(pushed.UnixMilli(), 10) + `"}`)
	}
	require.NoError(t, ex.processAccountBalance(t.Context(), frame("isolated_BTC-USDT.hold", 95)), "processAccountBalance must not error for an isolated margin push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 95, Free: 95, Borrowed: 25}, stored("BTC-USDT", currency.USDT), "an isolated margin push should replace its symbol's balance and keep the liability REST saved")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 500, Hold: 100, Free: 400, Borrowed: 50}, stored("", currency.USDT), "an isolated margin push should leave the cross margin account")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame("isolatedV2_ETH-USDT.transfer", 45)), "processAccountBalance must not error for an isolated margin V2 push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 45, Free: 45}, stored("ETH-USDT", currency.USDT), "an isolated margin V2 push should replace its symbol's balance")
	require.NoError(t, ex.processAccountBalance(t.Context(), frame("margin.transfer", 480)), "processAccountBalance must not error for a cross margin push")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 95, Free: 95, Borrowed: 25}, stored("BTC-USDT", currency.USDT), "a cross margin push should leave the isolated margin accounts")

	// A later listing replaces each listed account and empties one it leaves out
	time.Sleep(time.Until(pushed.Add(2 * time.Millisecond)))
	listing.Store(1)
	_, err = ex.UpdateAccountBalances(t.Context(), asset.Margin)
	require.NoError(t, err, "UpdateAccountBalances must not error for a later listing")
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 60, Free: 60, Borrowed: 30}, stored("BTC-USDT", currency.USDT), "a later listing should replace a listed isolated margin account's balances")
	for _, c := range []currency.Code{currency.ETH, currency.USDT} {
		_, err = ex.Accounts.GetBalance("ETH-USDT", creds, asset.Margin, c)
		assert.ErrorIsf(t, err, accounts.ErrNoBalances, "a later listing should empty the %s of an isolated margin account it leaves out", c)
	}
	assert.Equal(t, accounts.Balance{Currency: currency.USDT, Total: 500, Hold: 100, Free: 400, Borrowed: 50}, stored("", currency.USDT), "a later listing should replace the cross margin account's balance")
	marginBalances, err = ex.Accounts.CurrencyBalances(creds, asset.Margin)
	require.NoError(t, err, "CurrencyBalances must not error")
	assert.Equal(t, 560.0, marginBalances[currency.USDT].Total, "the margin balance should leave out an isolated margin account the listing leaves out")
}

func TestContractFundingRate(t *testing.T) {
	t.Parallel()
	next := time.UnixMilli(1791907200000)
	for _, tc := range []struct {
		name     string
		contract *ContractResponse
		exp      fundingrate.Rate
	}{
		{
			name:     "current interval",
			contract: &ContractResponse{FundingFeeRate: 0.0001, NextFundingRateDateTime: types.Time(next), FundingRateGranularity: 28800000, CurrentFundingRateGranularity: 14400000},
			exp:      fundingrate.Rate{Time: next.Add(-4 * time.Hour), Rate: decimal.MustFromFloat(0.0001)},
		},
		{
			name:     "configured interval when the current one is null",
			contract: &ContractResponse{FundingFeeRate: -0.0007, NextFundingRateDateTime: types.Time(next), FundingRateGranularity: 28800000},
			exp:      fundingrate.Rate{Time: next.Add(-8 * time.Hour), Rate: decimal.MustFromFloat(-0.0007)},
		},
		{
			name:     "no funding",
			contract: &ContractResponse{},
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
	// A server answering every request makes a request that should have been rejected fail its check, not the mock
	answering := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, `{"code":"200000","data":{"id":"6200c9b83aecfb000152ddcd","symbol":"XBTUSDTM","posMargin":2,"settleCurrency":"USDT"}}`)
		assert.NoError(t, err, "writing the reply should not error")
	})
	for _, tc := range []struct {
		name string
		req  *margin.PositionChangeRequest
		err  error
	}{
		{name: "an asset other than futures", req: &margin.PositionChangeRequest{Asset: asset.Spot, Pair: xbtusdtm, MarginType: margin.Isolated, NewAllocatedMargin: 2}, err: futures.ErrNotFuturesAsset},
		{name: "a request without a pair", req: &margin.PositionChangeRequest{Asset: asset.Futures, MarginType: margin.Isolated, NewAllocatedMargin: 2}, err: currency.ErrCurrencyPairEmpty},
		{name: "a cross margin position", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Multi, NewAllocatedMargin: 2}, err: margin.ErrMarginTypeUnsupported},
		{name: "a request without a new margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated}, err: margin.ErrNewAllocatedMarginRequired},
		{name: "a request without the original margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, NewAllocatedMargin: 2}, err: margin.ErrOriginalPositionMarginRequired},
		{name: "a request with a negative original margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: -1, NewAllocatedMargin: 2}, err: margin.ErrOriginalPositionMarginRequired},
		{name: "an unchanged margin", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 2, NewAllocatedMargin: 2}, err: errMarginUnchanged},
		{name: "a margin side that is no position side", req: &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2, MarginSide: "up"}, err: order.ErrSideIsInvalid},
	} {
		_, err = answering.ChangePositionMargin(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "ChangePositionMargin must reject %s", tc.name)
	}
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err = ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: 1, NewAllocatedMargin: 2})
	require.ErrorIs(t, err, asset.ErrNotEnabled, "ChangePositionMargin must reject a disabled asset")

	t.Run("position sides", func(t *testing.T) {
		t.Parallel()
		type marginRequest struct {
			path         string
			positionSide any
		}
		sent := make(chan marginRequest, 1)
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			var params map[string]any
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&params), "decoding the request body should not error")
			select {
			case sent <- marginRequest{path: r.URL.Path, positionSide: params["positionSide"]}:
			default:
				assert.Fail(t, "ChangePositionMargin should send one request per change")
			}
			reply := `{"code":"200000","data":"1"}`
			if r.URL.Path == "/v1/position/margin/deposit-margin" {
				reply = `{"code":"200000","data":{"id":"6200c9b83aecfb000152ddcd","symbol":"XBTUSDTM","posMargin":2,"settleCurrency":"USDT"}}`
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		for _, tc := range []struct {
			marginSide string
			exp        any
		}{
			{marginSide: "buy", exp: "LONG"},
			{marginSide: "Long", exp: "LONG"},
			{marginSide: "sell", exp: "SHORT"},
			{marginSide: "SHORT", exp: "SHORT"},
			{marginSide: "both", exp: "BOTH"},
			{marginSide: "", exp: nil},
		} {
			for _, change := range []struct {
				name, path              string
				original, newAllocation float64
			}{
				{name: "adding", path: "/v1/position/margin/deposit-margin", original: 1, newAllocation: 2},
				{name: "withdrawing", path: "/v1/margin/withdrawMargin", original: 2, newAllocation: 1},
			} {
				_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.Futures, Pair: xbtusdtm, MarginType: margin.Isolated, OriginalAllocatedMargin: change.original, NewAllocatedMargin: change.newAllocation, MarginSide: tc.marginSide})
				require.NoErrorf(t, err, "ChangePositionMargin must not error %s margin for margin side %q", change.name, tc.marginSide)
				got := received(t, sent, "ChangePositionMargin must send its request")
				assert.Equalf(t, marginRequest{path: change.path, positionSide: tc.exp}, got, "ChangePositionMargin should send margin side %q as KuCoin's position side when %s margin", tc.marginSide, change.name)
			}
		}
	})

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
				MaintenanceMarginRequirement: decimal.MustFromFloat(0.54557932),
				InitialMarginRequirement:     decimal.MustFromFloat(19.7728500039),
				EstimatedLiquidationPrice:    decimal.MustFromFloat(99321.13),
				CollateralUsed:               decimal.MustFromFloat(20.2728500039),
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
				MaintenanceMarginRequirement: decimal.MustFromFloat(0.00000337),
				InitialMarginRequirement:     decimal.MustFromFloat(0.00084226),
				EstimatedLiquidationPrice:    decimal.MustFromFloat(0.01),
				CollateralUsed:               decimal.MustFromFloat(0.00084225),
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

// TestGetFuturesPositionSummaryInverse covers the summary of an inverse contract's position, whose lot is worth 1 USD
// although KuCoin reports the contract's multiplier as -1, so the contract size has the position's sign
func TestGetFuturesPositionSummaryInverse(t *testing.T) {
	t.Parallel()
	// XBTUSDM's specification as Get Symbol returned it live on 2026-10-21
	const contracts = `{"code":"200000","data":[{"symbol":"XBTUSDM","rootSymbol":"XBT","type":"FFWCSX","baseCurrency":"XBT","quoteCurrency":"USD","settleCurrency":"XBT","multiplier":-1,"isInverse":true,"lotSize":1,"maintainMargin":0.007,"initialMargin":0.014}]}`
	const account = `{"code":"200000","data":{"accountEquity":0.0025,"unrealisedPNL":-0.00000116,"marginBalance":0.00250116,"positionMargin":0.00060366,"orderMargin":0,"frozenFunds":0.0001,"availableBalance":0.0018,"currency":"XBT","riskRatio":0.0034,"maxWithdrawAmount":0.0018}}`
	xbtusdm := currency.NewPair(currency.XBT, currency.USDM)
	for _, tc := range []struct {
		name     string
		position string
		lots     int64
	}{
		{
			name:     "long",
			position: `{"id":"500000000001533800","symbol":"XBTUSDM","maintMarginReq":"0.007","crossMode":true,"openingTimestamp":1792500000000,"currentTimestamp":1792540000000,"currentQty":100,"markPrice":"82828.2","markValue":"0.00120732","posCost":"0.00120848","posInit":"0.00060424","posMargin":"0.00060366","posMaint":"0.00000845","realisedPnl":"-0.00000073","unrealisedPnl":"-0.00000116","avgEntryPrice":"82748.6","liquidationPrice":"41500.1","settleCurrency":"XBT","isInverse":true,"maintainMargin":"0.007","marginMode":"CROSS","positionSide":"BOTH","leverage":"2","imr":"0.5"}`,
			lots:     100,
		},
		{
			name:     "short",
			position: `{"id":"500000000001533801","symbol":"XBTUSDM","maintMarginReq":"0.007","crossMode":true,"openingTimestamp":1792500000000,"currentTimestamp":1792540000000,"currentQty":-100,"markPrice":"82828.2","markValue":"-0.00120732","posCost":"-0.00120848","posInit":"0.00060424","posMargin":"0.00060366","posMaint":"0.00000845","realisedPnl":"-0.00000073","unrealisedPnl":"-0.00000116","avgEntryPrice":"82748.6","liquidationPrice":"41500.1","settleCurrency":"XBT","isInverse":true,"maintainMargin":"0.007","marginMode":"CROSS","positionSide":"BOTH","leverage":"2","imr":"0.5"}`,
			lots:     -100,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				reply := `{"code":"200000","data":null}`
				switch r.URL.Path {
				case "/v1/contracts/active":
					reply = contracts
				case "/v2/position":
					assert.Equal(t, "XBTUSDM", r.URL.Query().Get("symbol"), "GetFuturesPositionSummary should request the contract's position")
					reply = `{"code":"200000","data":[` + tc.position + `]}`
				case "/v1/account-overview":
					assert.Equal(t, "XBT", r.URL.Query().Get("currency"), "GetFuturesPositionSummary should request the account of the position's settlement currency")
					reply = account
				}
				_, err := io.WriteString(w, reply)
				assert.NoError(t, err, "writing the reply should not error")
			})
			got, err := ex.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.Futures, Pair: xbtusdm})
			require.NoError(t, err, "GetFuturesPositionSummary must not error")
			notional := decimal.MustFromFloat(0.00120732)
			if tc.lots < 0 {
				notional = notional.Neg()
			}
			exp := &futures.PositionSummary{
				Pair: xbtusdm, Asset: asset.Futures, MarginType: margin.Multi, CollateralMode: collateral.MultiMode,
				Currency: currency.XBT, StartDate: time.UnixMilli(1792500000000),
				AvailableEquity:              decimal.MustFromFloat(0.0025),
				MarginBalance:                decimal.MustFromFloat(0.00250116),
				NotionalSize:                 notional,
				Leverage:                     decimal.MustFromFloat(2),
				MaintenanceMarginRequirement: decimal.MustFromFloat(0.00000845),
				InitialMarginRequirement:     decimal.MustFromFloat(0.00060424),
				EstimatedLiquidationPrice:    decimal.MustFromFloat(41500.1),
				CollateralUsed:               decimal.MustFromFloat(0.00060366),
				MarkPrice:                    decimal.MustFromFloat(82828.2),
				CurrentSize:                  decimal.NewFromInt(tc.lots),
				ContractSize:                 decimal.NewFromInt(tc.lots).Mul(decimal.MustFromFloat(1)),
				ContractMultiplier:           decimal.MustFromFloat(1),
				ContractSettlementType:       futures.Inverse,
				AverageOpenPrice:             decimal.MustFromFloat(82748.6),
				UnrealisedPNL:                decimal.MustFromFloat(-0.00000116),
				RealisedPNL:                  decimal.MustFromFloat(-0.00000073),
				MaintenanceMarginFraction:    decimal.MustFromFloat(0.007),
				FreeCollateral:               decimal.MustFromFloat(0.0018),
				TotalCollateral:              decimal.MustFromFloat(0.0025),
				FrozenBalance:                decimal.MustFromFloat(0.0001),
			}
			assert.Equal(t, exp, got, "GetFuturesPositionSummary should report the contract size in USD with the position's sign")
		})
	}
}

func TestFuturesPosition(t *testing.T) {
	t.Parallel()
	both := FuturesPositionDetailResponse{ID: "1", PositionSide: "BOTH"}
	long := FuturesPositionDetailResponse{ID: "2", PositionSide: "LONG"}
	short := FuturesPositionDetailResponse{ID: "3", PositionSide: "SHORT"}
	_, err := futuresPosition(nil, order.Long)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "futuresPosition must report a contract without positions")
	_, err = futuresPosition([]FuturesPositionDetailResponse{long, short}, order.UnknownSide)
	require.ErrorIs(t, err, errPositionDirectionRequired, "futuresPosition must require a direction in hedge mode")
	_, err = futuresPosition([]FuturesPositionDetailResponse{short}, order.Buy)
	require.ErrorIs(t, err, futures.ErrPositionNotFound, "futuresPosition must report a direction without a position")
	for _, tc := range []struct {
		name      string
		positions []FuturesPositionDetailResponse
		direction order.Side
		exp       string
	}{
		{name: "the one-way mode position", positions: []FuturesPositionDetailResponse{both}, exp: "1"},
		{name: "the one-way mode position for a direction", positions: []FuturesPositionDetailResponse{both}, direction: order.Short, exp: "1"},
		{name: "the hedge mode long position", positions: []FuturesPositionDetailResponse{long, short}, direction: order.Long, exp: "2"},
		{name: "the hedge mode short position", positions: []FuturesPositionDetailResponse{long, short}, direction: order.Sell, exp: "3"},
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
		query := received(t, queries, "GetFuturesPositionOrders must request the orders")
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
					// Amount and ExecutedAmount are the four lots in the base currency, at the contract's multiplier of 0.001
					Exchange: e.Name, OrderID: "235919172150825005", ClientOrderID: "gct-position-order",
					Pair:      currency.NewPairWithDelimiter(currency.XBT.String(), currency.USDTM.String(), currency.UnderscoreDelimiter),
					AssetType: asset.Futures, MarginType: margin.Isolated, Type: order.Limit, Side: order.Buy,
					Status: order.Filled, Price: 65000, Amount: 0.004, ContractAmount: 4, ExecutedAmount: 0.004,
					AverageExecutedPrice: 65000, Cost: 260, CostAsset: currency.USDT, SettlementCurrency: currency.USDT,
					Leverage: 3, TimeInForce: order.GoodTillCancel, Date: time.UnixMilli(1729300000000),
					LastUpdated: time.UnixMilli(1729400000000), CloseTime: time.UnixMilli(1729400000000),
				},
			},
		},
	}
	assert.Equal(t, exp, got, "GetFuturesPositionOrders should return the pair's closed orders")
}

// TestGetFuturesPositionOrdersAmounts covers the unit of position orders' amounts: a lot of a linear contract is its
// multiplier of the base currency, and a lot of an inverse contract, whose multiplier KuCoin reports as -1, is 1 USD
func TestGetFuturesPositionOrdersAmounts(t *testing.T) {
	t.Parallel()
	// XBTUSDTM's and XBTUSDM's specifications as Get Symbol returned them live on 2026-10-21. ETHUSDM is left out, as
	// a contract the list of active contracts no longer holds, such as a settled one
	const contracts = `{"code":"200000","data":[` +
		`{"symbol":"XBTUSDTM","rootSymbol":"USDT","type":"FFWCSX","baseCurrency":"XBT","quoteCurrency":"USDT","settleCurrency":"USDT","multiplier":0.001,"isInverse":false,"lotSize":1},` +
		`{"symbol":"XBTUSDM","rootSymbol":"XBT","type":"FFWCSX","baseCurrency":"XBT","quoteCurrency":"USD","settleCurrency":"XBT","multiplier":-1,"isInverse":true,"lotSize":1}]}`
	orders := map[string]string{
		// 13 lots of 0.001 XBT, 9 of them filled before the rest was cancelled
		"XBTUSDTM": `{"id":"235919172150825010","symbol":"XBTUSDTM","type":"limit","side":"buy","price":"82874.8","size":13,"value":"1077.3724","dealValue":"745.8732","dealSize":9,"timeInForce":"GTC","leverage":"5","clientOid":"gct-linear","isActive":false,"cancelExist":true,"createdAt":1792500000000,"updatedAt":1792510000000,"endAt":1792510000000,"settleCurrency":"USDT","marginMode":"ISOLATED","avgDealPrice":"82874.8","filledSize":9,"filledValue":"745.8732","status":"done"}`,
		// 1000 lots of 1 USD, 400 of them filled before the rest was cancelled
		"XBTUSDM": `{"id":"235919172150825011","symbol":"XBTUSDM","type":"limit","side":"sell","price":"82828.2","size":1000,"value":"0.01207318","dealValue":"0.00482927","dealSize":400,"timeInForce":"GTC","leverage":"3","clientOid":"gct-inverse","isActive":false,"cancelExist":true,"createdAt":1792500000000,"updatedAt":1792510000000,"endAt":1792510000000,"settleCurrency":"XBT","marginMode":"CROSS","avgDealPrice":"82828.2","filledSize":400,"filledValue":"0.00482927","status":"done"}`,
		"ETHUSDM": `{"id":"235919172150825012","symbol":"ETHUSDM","type":"limit","side":"buy","price":"2442.21","size":50,"value":"0.02047326","dealValue":"0.00818930","dealSize":20,"timeInForce":"GTC","leverage":"2","clientOid":"gct-unlisted","isActive":false,"cancelExist":true,"createdAt":1792500000000,"updatedAt":1792510000000,"endAt":1792510000000,"settleCurrency":"ETH","marginMode":"CROSS","avgDealPrice":"2442.21","filledSize":20,"filledValue":"0.00818930","status":"done"}`,
	}
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply := `{"code":"200000","data":null}`
		switch r.URL.Path {
		case "/v1/contracts/active":
			reply = contracts
		case "/v1/orders":
			symbol := r.URL.Query().Get("symbol")
			item, ok := orders[symbol]
			assert.Truef(t, ok, "GetFuturesPositionOrders should list the orders of a requested contract, not %q", symbol)
			reply = `{"code":"200000","data":{"currentPage":1,"pageSize":1000,"totalNum":1,"totalPage":1,"items":[` + item + `]}}`
		}
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	end := time.UnixMilli(1792540800000)
	xbtusdtm := currency.NewPair(currency.XBT, currency.USDTM)
	xbtusdm := currency.NewPair(currency.XBT, currency.USDM)
	ethusdm := currency.NewPair(currency.ETH, currency.USDM)
	got, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.Futures, Pairs: currency.Pairs{xbtusdtm, xbtusdm, ethusdm}, StartDate: end.Add(-24 * time.Hour), EndDate: end})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	closed := func(o *order.Detail) order.Detail {
		o.Exchange, o.AssetType, o.Type, o.Status, o.TimeInForce = ex.Name, asset.Futures, order.Limit, order.PartiallyFilledCancelled, order.GoodTillCancel
		o.Date, o.LastUpdated, o.CloseTime = time.UnixMilli(1792500000000), time.UnixMilli(1792510000000), time.UnixMilli(1792510000000)
		return *o
	}
	exp := []futures.PositionResponse{
		{
			Pair: xbtusdtm, Asset: asset.Futures, ContractSettlementType: futures.Linear,
			Orders: []order.Detail{closed(&order.Detail{
				// 13 lots of 0.001 XBT with 9 filled, exactly, as float multiplication gives 0.013000000000000001
				OrderID: "235919172150825010", ClientOrderID: "gct-linear",
				Pair: currency.NewPairWithDelimiter("XBT", "USDTM", "_"), MarginType: margin.Isolated, Side: order.Buy,
				Price: 82874.8, Amount: 0.013, ContractAmount: 13, ExecutedAmount: 0.009, RemainingAmount: 0.004,
				AverageExecutedPrice: 82874.8, Cost: 745.8732, CostAsset: currency.USDT, SettlementCurrency: currency.USDT, Leverage: 5,
			})},
		},
		{
			Pair: xbtusdm, Asset: asset.Futures, ContractSettlementType: futures.Inverse,
			Orders: []order.Detail{closed(&order.Detail{
				// 1000 lots of 1 USD with 400 filled, positive although KuCoin's multiplier is -1
				OrderID: "235919172150825011", ClientOrderID: "gct-inverse",
				Pair: currency.NewPairWithDelimiter("XBT", "USDM", "_"), MarginType: margin.Multi, Side: order.Sell,
				Price: 82828.2, Amount: 1000, ContractAmount: 1000, ExecutedAmount: 400, RemainingAmount: 600,
				AverageExecutedPrice: 82828.2, Cost: 0.00482927, CostAsset: currency.XBT, SettlementCurrency: currency.XBT, Leverage: 3,
			})},
		},
		{
			// The lot of a contract no longer listed is unknown, so its lots are left in ContractAmount alone
			Pair: ethusdm, Asset: asset.Futures,
			Orders: []order.Detail{closed(&order.Detail{
				OrderID: "235919172150825012", ClientOrderID: "gct-unlisted",
				Pair: currency.NewPairWithDelimiter("ETH", "USDM", "_"), MarginType: margin.Multi, Side: order.Buy,
				Price: 2442.21, ContractAmount: 50,
				AverageExecutedPrice: 2442.21, Cost: 0.0081893, CostAsset: currency.ETH, SettlementCurrency: currency.ETH, Leverage: 2,
			})},
		},
	}
	assert.Equal(t, exp, got, "GetFuturesPositionOrders should report amounts in the unit a lot is worth")
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

	// The recorded mock has no replacement orders to read back, so a local server answers the modifications
	replacements := map[string]string{
		"670fd33bf9406e0007ab3945": "67112258f9406e0007408827",
		"5c52e11203aa677f33e493fb": "67112258f9406e0007408828",
		"gct-modify-unread":        "67112258f9406e0007408829",
		"gct-modify-unreachable":   "67112258f9406e000740882a",
		"gct-modify-filled":        "67112258f9406e000740882b",
		"gct-modify-cancelled":     "67112258f9406e000740882c",
		"gct-modify-price":         "67112258f9406e000740882d",
		"gct-modify-tracked":       "67112258f9406e000740882e",
	}
	placed := map[string]string{
		"/v1/hf/orders/67112258f9406e0007408827": `{"code":"200000","data":{"id":"67112258f9406e0007408827","symbol":"BTC-USDT","type":"limit","side":"buy","price":"30000","size":"0.00007","dealSize":"0","remainSize":"0.00007","clientOid":"5c52e11203aa677f33e493fb","active":true}}`,
		"/v1/hf/orders/67112258f9406e0007408828": `{"code":"200000","data":{"id":"67112258f9406e0007408828","symbol":"BTC-USDT","type":"limit","side":"buy","price":"29000","size":"0.0002","dealSize":"0.00005","remainSize":"0.00015","clientOid":"5c52e11203aa677f33e493fb","active":true}}`,
		"/v1/hf/orders/67112258f9406e0007408829": `{"code":"200000","data":null}`,
		"/v1/hf/orders/67112258f9406e000740882b": `{"code":"200000","data":{"id":"67112258f9406e000740882b","symbol":"BTC-USDT","type":"limit","side":"buy","price":"31000","size":"0.00007","dealSize":"0.00007","remainSize":"0","clientOid":"5c52e11203aa677f33e493fb","active":false}}`,
		"/v1/hf/orders/67112258f9406e000740882c": `{"code":"200000","data":{"id":"67112258f9406e000740882c","symbol":"BTC-USDT","type":"limit","side":"buy","price":"32000","size":"0.00007","dealSize":"0","remainSize":"0","clientOid":"5c52e11203aa677f33e493fb","active":false,"cancelExist":true}}`,
		"/v1/hf/orders/67112258f9406e000740882d": `{"code":"200000","data":{"id":"67112258f9406e000740882d","symbol":"BTC-USDT","type":"limit","side":"buy","price":"30000","size":"0.00007","dealSize":"0","remainSize":"0.00007","clientOid":"5c52e11203aa677f33e493fb","active":true}}`,
		"/v1/hf/orders/67112258f9406e000740882e": `{"code":"200000","data":{"id":"67112258f9406e000740882e","symbol":"BTC-USDT","type":"limit","side":"buy","price":"30000","size":"0.00002","dealSize":"0","remainSize":"0.00002","clientOid":"5c52e11203aa677f33e493fb","active":true}}`,
	}
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		reply, ok := placed[r.URL.Path]
		switch r.URL.Path {
		case "/v1/hf/orders/67112258f9406e000740882a":
			w.WriteHeader(http.StatusInternalServerError)
			return
		case "/v1/hf/orders/alter":
			var params map[string]string
			assert.NoError(t, json.NewDecoder(r.Body).Decode(&params), "decoding the request body should not error")
			modified := params["orderId"]
			if modified == "" {
				modified = params["clientOid"]
			}
			reply, ok = `{"code":"200000","data":{"newOrderId":"`+replacements[modified]+`","clientOid":"5c52e11203aa677f33e493fb"}}`, true
		default:
			assert.Equal(t, "BTC-USDT", r.URL.Query().Get("symbol"), "ModifyOrder should read the replacement back by the order's symbol")
		}
		assert.Truef(t, ok, "request path %s should be one the test serves", r.URL.Path)
		_, err := io.WriteString(w, reply)
		assert.NoError(t, err, "writing the reply should not error")
	})
	for _, tc := range []struct {
		name   string
		modify *order.Modify
		exp    *order.ModifyResponse
	}{
		{
			// KuCoin takes one of the order ID and the client order ID, and replaces the order with a new one sized by what
			// the requested amount leaves after the original's fills, so the amount stays the one requested, which the
			// fills the order's executed amount holds are part of
			name:   "by order ID",
			modify: &order.Modify{Exchange: e.Name, OrderID: "670fd33bf9406e0007ab3945", ClientOrderID: "5c52e11203aa677f33e493fb", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: btcusdt, Price: 30000, Amount: 0.0001},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e0007408827", ClientOrderID: "5c52e11203aa677f33e493fb", Type: order.Limit, Side: order.Buy, AssetType: asset.Spot, Pair: btcusdt, Price: 30000, Amount: 0.0001, RemainingAmount: 0.00007, Status: order.Open},
		},
		{
			// A price change requests no amount, so the order keeps its own
			name:   "of the price alone",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-price", AssetType: asset.Spot, Pair: btcusdt, Price: 30000},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e000740882d", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 30000, RemainingAmount: 0.00007, Status: order.Open},
		},
		{
			name:   "by client order ID",
			modify: &order.Modify{Exchange: e.Name, ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Amount: 0.0002},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e0007408828", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 29000, Amount: 0.0002, RemainingAmount: 0.00015, Status: order.PartiallyFilled},
		},
		{
			name:   "without a replacement",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-none", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
			exp:    &order.ModifyResponse{Exchange: e.Name, ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
		},
		{
			// KuCoin has replaced the order, so it is reported as requested under its new order ID
			name:   "with a replacement KuCoin does not return",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-unread", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e0007408829", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
		},
		{
			// A replacement already filled is reported filled, so the engine stops tracking it
			name:   "with a replacement already filled",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-filled", AssetType: asset.Spot, Pair: btcusdt, Price: 31000, Amount: 0.00007},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e000740882b", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 31000, Amount: 0.00007, Status: order.Filled},
		},
		{
			name:   "with a replacement already cancelled",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-cancelled", AssetType: asset.Spot, Pair: btcusdt, Price: 32000, Amount: 0.00007},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e000740882c", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 32000, Amount: 0.00007, Status: order.Cancelled},
		},
		{
			name:   "with a replacement that cannot be read back",
			modify: &order.Modify{Exchange: e.Name, OrderID: "gct-modify-unreachable", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
			exp:    &order.ModifyResponse{Exchange: e.Name, OrderID: "67112258f9406e000740882a", ClientOrderID: "5c52e11203aa677f33e493fb", AssetType: asset.Spot, Pair: btcusdt, Price: 1, Amount: 0.00001},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			resp, err := ex.ModifyOrder(t.Context(), tc.modify)
			require.NoError(t, err, "ModifyOrder must not error")
			assert.Equal(t, tc.exp, resp, "ModifyOrder should return the replacement order")
		})
	}

	t.Run("tracked by the engine", func(t *testing.T) {
		t.Parallel()
		// The engine keeps the original order's executed amount, here more than the replacement's own size, so the
		// replacement reported at that size would look filled and no longer be tracked
		tracked := &order.Detail{Exchange: e.Name, OrderID: "gct-modify-tracked", AssetType: asset.Spot, Pair: btcusdt, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, Price: 29000, Amount: 0.0001, ExecutedAmount: 0.00008, RemainingAmount: 0.00002}
		resp, err := ex.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, OrderID: tracked.OrderID, AssetType: asset.Spot, Pair: btcusdt, Price: 30000, Amount: 0.0001})
		require.NoError(t, err, "ModifyOrder must not error")
		tracked.UpdateOrderFromModifyResponse(resp)
		assert.True(t, tracked.IsActive(), "the engine should keep tracking an open replacement")
		assert.Equal(t, 0.00002, tracked.RemainingAmount, "the engine should hold the replacement's remaining amount")
	})
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), currency.NewBTCUSDT(), asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported, as KuCoin serves only recent trades")
}

// TestGetHistoricalFundingRatesPages covers windows longer than the hundred rates KuCoin returns a request: the rates
// are read back from the window's end, including windows of whole pages and one rate longer
func TestGetHistoricalFundingRatesPages(t *testing.T) {
	t.Parallel()
	const interval = 8 * time.Hour
	first := time.UnixMilli(1788393600000)
	timepoint := func(k int) time.Time { return first.Add(time.Duration(k) * interval) }
	type record struct {
		Symbol      string  `json:"symbol"`
		FundingRate float64 `json:"fundingRate"`
		Timepoint   int64   `json:"timepoint"`
	}
	records := make([]record, 250)
	for k := range records {
		records[k] = record{Symbol: "XBTUSDTM", FundingRate: float64(k+1) / 1e6, Timepoint: timepoint(k).UnixMilli()}
	}
	newServer := func(t *testing.T) (*Exchange, *atomic.Int32) {
		t.Helper()
		var mu sync.Mutex
		seen := make(map[string]bool)
		var requests atomic.Int32
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			requests.Add(1)
			mu.Lock()
			repeated := seen[r.URL.RawQuery]
			seen[r.URL.RawQuery] = true
			mu.Unlock()
			from, fromErr := strconv.ParseInt(r.URL.Query().Get("from"), 10, 64)
			to, toErr := strconv.ParseInt(r.URL.Query().Get("to"), 10, 64)
			if repeated || fromErr != nil || toErr != nil || r.URL.Path != "/v1/contract/funding-rates" {
				// A repeated request would loop forever on a broken pager
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			// KuCoin returns the newest hundred rates of the window, newest first, its edges included
			var page []record
			for k := len(records) - 1; k >= 0 && len(page) < 100; k-- {
				if records[k].Timepoint >= from && records[k].Timepoint <= to {
					page = append(page, records[k])
				}
			}
			reply := `{"code":"200000","data":null}`
			if len(page) != 0 {
				data, err := json.Marshal(page)
				assert.NoError(t, err, "marshalling the page should not error")
				reply = `{"code":"200000","data":` + string(data) + `}`
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		return ex, &requests
	}
	pair := currency.NewPair(currency.XBT, currency.USDTM)
	for _, tc := range []struct {
		name        string
		start, end  time.Time
		first, last int
		requests    int32
	}{
		{name: "a window between funding times", start: timepoint(10).Add(-time.Hour), end: timepoint(239).Add(time.Hour), first: 10, last: 239, requests: 3},
		{name: "a window of two whole pages", start: timepoint(20), end: timepoint(219), first: 20, last: 219, requests: 2},
		{name: "a window one rate longer than two pages", start: timepoint(19), end: timepoint(219), first: 19, last: 219, requests: 3},
		// A whole page reaching the window's start leaves nothing to ask for
		{name: "a window of one page", start: timepoint(0), end: timepoint(99), first: 0, last: 99, requests: 1},
		{name: "a window of one page and no earlier rates", start: timepoint(0).AddDate(0, 0, -1), end: timepoint(99), first: 0, last: 99, requests: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex, requests := newServer(t)
			got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: tc.start, EndDate: tc.end})
			require.NoError(t, err, "GetHistoricalFundingRates must not error")
			rates := make([]fundingrate.Rate, 0, tc.last-tc.first+1)
			for k := tc.last; k >= tc.first; k-- {
				rates = append(rates, fundingrate.Rate{Time: timepoint(k), Rate: decimal.MustFromFloat(records[k].FundingRate)})
			}
			exp := &fundingrate.HistoricalRates{
				Exchange: ex.Name, Asset: asset.Futures, Pair: pair, FundingRates: rates,
				StartDate: rates[len(rates)-1].Time, EndDate: rates[0].Time, LatestRate: rates[0],
			}
			assert.Equal(t, exp, got, "GetHistoricalFundingRates should return every rate of the window once, newest first")
			assert.Equal(t, tc.requests, requests.Load(), "GetHistoricalFundingRates should read the window a page at a time")
		})
	}

	ex, requests := newServer(t)
	_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: first.AddDate(0, 0, -30), EndDate: first.Add(-time.Hour)})
	assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should report a window without funding rates")
	// KuCoin's window needs an end, which the RPC server sets to now when none is given
	_, err = ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: first})
	assert.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricalFundingRates should refuse a window without an end")
	assert.EqualValues(t, 1, requests.Load(), "GetHistoricalFundingRates should send no request for a window without an end")
}

// TestGetHistoricalFundingRatesWithoutProgress covers a venue answering full pages of rates after the window's end, as
// one ignoring the end would: such a page cannot move the window back, so reading stops rather than asking again
func TestGetHistoricalFundingRatesWithoutProgress(t *testing.T) {
	t.Parallel()
	first := time.UnixMilli(1788393600000)
	rates := make([]string, 100)
	for i := range rates {
		rates[i] = `{"symbol":"XBTUSDTM","fundingRate":0.0001,"timepoint":` + strconv.FormatInt(first.Add(time.Duration(200-i)*8*time.Hour).UnixMilli(), 10) + `}`
	}
	page := `{"code":"200000","data":[` + strings.Join(rates, ",") + `]}`
	var requests atomic.Int32
	ex := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) > 10 {
			// A pager that keeps asking would otherwise run until the test times out
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err := io.WriteString(w, page)
		assert.NoError(t, err, "writing the reply should not error")
	})
	pair := currency.NewPair(currency.XBT, currency.USDTM)
	_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: first, EndDate: first.Add(50 * 8 * time.Hour)})
	assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should find no rate of the window in rates after it")
	assert.EqualValues(t, 1, requests.Load(), "GetHistoricalFundingRates should stop on a page that cannot move the window back")

	// The second request gets the first page again, all of whose rates are after that request's end
	requests.Store(0)
	got, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.Futures, Pair: pair, StartDate: first, EndDate: first.Add(300 * 8 * time.Hour)})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	assert.Len(t, got.FundingRates, 100, "GetHistoricalFundingRates should keep each rate once when a later page repeats it")
	assert.EqualValues(t, 2, requests.Load(), "GetHistoricalFundingRates should stop on a page that cannot move the window back")
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

	t.Run("replies without a result for every order", func(t *testing.T) {
		t.Parallel()
		cancels := []order.Cancel{
			{AssetType: asset.Futures, OrderID: "250445104152670201"},
			{AssetType: asset.Futures, OrderID: "250445104152670202"},
			{AssetType: asset.Futures, Pair: xbtusdtm, ClientOrderID: "gct-futures-client"},
		}
		missing := func(id string) string { return fmt.Errorf("%w for order %s", common.ErrNoResponse, id).Error() }
		for _, tc := range []struct {
			name, orderReply, clientReply string
			exp                           map[string]string
		}{
			{name: "no results", orderReply: `[]`, clientReply: `[]`, exp: map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": missing("250445104152670202"), "gct-futures-client": missing("gct-futures-client")}},
			{name: "null results", orderReply: `null`, clientReply: `null`, exp: map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": missing("250445104152670202"), "gct-futures-client": missing("gct-futures-client")}},
			{name: "a null result", orderReply: `[null]`, clientReply: `[null]`, exp: map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": missing("250445104152670202"), "gct-futures-client": missing("gct-futures-client")}},
			{name: "a result naming no order", orderReply: `[{}]`, clientReply: `[{"code":"200","msg":"success"}]`, exp: map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": missing("250445104152670202"), "gct-futures-client": missing("gct-futures-client")}},
			{
				name:        "results for some of the orders",
				orderReply:  `[{"orderId":"250445104152670202","clientOid":null,"code":"200","msg":"success"}]`,
				clientReply: `[{"orderId":null,"clientOid":"gct-futures-client","code":"200","msg":"success"}]`,
				exp:         map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": order.Cancelled.String(), "gct-futures-client": order.Cancelled.String()},
			},
			{
				name:        "results for orders the request did not send",
				orderReply:  `[{"orderId":"250445104152670202","clientOid":null,"code":"200","msg":"success"},{"orderId":"250445104152670299","clientOid":null,"code":"200","msg":"success"}]`,
				clientReply: `[{"orderId":null,"clientOid":"gct-futures-client","code":"200","msg":"success"},{"orderId":null,"clientOid":"250445104152670201","code":"200","msg":"success"}]`,
				exp:         map[string]string{"250445104152670201": missing("250445104152670201"), "250445104152670202": order.Cancelled.String(), "gct-futures-client": order.Cancelled.String()},
			},
		} {
			ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				reply := tc.orderReply
				if strings.Contains(string(body), "clientOidsList") {
					reply = tc.clientReply
				}
				_, err = io.WriteString(w, `{"code":"200000","data":`+reply+`}`)
				assert.NoError(t, err, "writing the reply should not error")
			})
			resp, err := ex.CancelBatchOrders(t.Context(), cancels)
			assert.ErrorIsf(t, err, common.ErrNoResponse, "CancelBatchOrders should return an error for the orders a reply with %s leaves out", tc.name)
			require.NotNilf(t, resp, "CancelBatchOrders must return the statuses with %s", tc.name)
			assert.Equalf(t, tc.exp, resp.Status, "CancelBatchOrders should report each order the reply with %s leaves out as unanswered", tc.name)
		}
	})

	t.Run("orders listed more than once", func(t *testing.T) {
		t.Parallel()
		// Each order is cancelled once, so a request repeating it cannot fail after it is cancelled and overwrite that
		const orderID = "250445104152670201"
		cancels := make([]order.Cancel, 0, batchCancelFuturesOrdersLimit+3)
		for range batchCancelFuturesOrdersLimit + 1 {
			cancels = append(cancels, order.Cancel{AssetType: asset.Futures, OrderID: orderID})
		}
		cancels = append(cancels,
			order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm, ClientOrderID: "gct-futures-client"},
			order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm, ClientOrderID: "gct-futures-client"},
		)
		replies := []struct{ body, reply string }{
			{body: `{"orderIdsList":["` + orderID + `"]}`, reply: batchCancelReply([]string{orderID}, "")},
			{body: `{"clientOidsList":[{"symbol":"XBTUSDTM","clientOid":"gct-futures-client"}]}`, reply: `{"code":"200000","data":[{"orderId":null,"clientOid":"gct-futures-client","code":"200","msg":"success"}]}`},
		}
		var calls atomic.Int32
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			call := int(calls.Add(1)) - 1
			reply := `{"code":"429000","msg":"too many requests"}`
			if call < len(replies) {
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				assert.JSONEqf(t, replies[call].body, string(body), "CancelBatchOrders should send request %d's orders once each", call)
				reply = replies[call].reply
			}
			_, err := io.WriteString(w, reply)
			assert.NoError(t, err, "writing the reply should not error")
		})
		resp, err := ex.CancelBatchOrders(t.Context(), cancels)
		require.NoError(t, err, "CancelBatchOrders must not error")
		assert.Equal(t, &order.CancelBatchResponse{Status: map[string]string{orderID: order.Cancelled.String(), "gct-futures-client": order.Cancelled.String()}}, resp, "CancelBatchOrders should report each order cancelled")
		assert.EqualValues(t, len(replies), calls.Load(), "CancelBatchOrders should send each order in one request")
	})

	t.Run("failed requests", func(t *testing.T) {
		t.Parallel()
		orderIDs := make([]string, 12)
		cancels := make([]order.Cancel, 0, len(orderIDs)+1)
		for i := range orderIDs {
			orderIDs[i] = strconv.FormatUint(250445104152670201+uint64(i), 10)
			cancels = append(cancels, order.Cancel{AssetType: asset.Futures, OrderID: orderIDs[i]})
		}
		cancels = append(cancels, order.Cancel{AssetType: asset.Futures, Pair: xbtusdtm, ClientOrderID: "gct-futures-client"})
		const failure = `{"code":"429000","msg":"too many requests"}`
		failed := Error{Code: "429000", Message: "too many requests"}.GetError().Error()
		firstBatch, secondBatch := batchCancelReply(orderIDs[:10], ""), batchCancelReply(orderIDs[10:], "")
		clientBatch := `{"code":"200000","data":[{"orderId":null,"clientOid":"gct-futures-client","code":"200","msg":"success"}]}`
		statuses := func(first, second, client string) map[string]string {
			exp := map[string]string{"gct-futures-client": client}
			for i, orderID := range orderIDs {
				exp[orderID] = first
				if i >= 10 {
					exp[orderID] = second
				}
			}
			return exp
		}
		cancelled := order.Cancelled.String()
		for _, tc := range []struct {
			name    string
			replies []string
			exp     map[string]string
			err     string
		}{
			{name: "a failed client order ID request", replies: []string{firstBatch, secondBatch, failure}, exp: statuses(cancelled, cancelled, failed), err: failed},
			{name: "a failed request before later ones", replies: []string{failure, secondBatch, clientBatch}, exp: statuses(failed, cancelled, cancelled), err: failed},
			{name: "every request failing", replies: []string{failure, failure, failure}, exp: statuses(failed, failed, failed), err: failed + ", " + failed + ", " + failed},
		} {
			var calls atomic.Int32
			ex := newLocalServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				call := int(calls.Add(1)) - 1
				if !assert.Lessf(t, call, len(tc.replies), "CancelBatchOrders should send three requests with %s", tc.name) {
					return
				}
				_, err := io.WriteString(w, tc.replies[call])
				assert.NoError(t, err, "writing the reply should not error")
			})
			resp, err := ex.CancelBatchOrders(t.Context(), cancels)
			assert.EqualErrorf(t, err, tc.err, "CancelBatchOrders should return the errors of the failed requests with %s", tc.name)
			require.NotNilf(t, resp, "CancelBatchOrders must return the statuses with %s", tc.name)
			assert.Equalf(t, tc.exp, resp.Status, "CancelBatchOrders should report each order's outcome, or its request's error, with %s", tc.name)
			assert.EqualValuesf(t, len(tc.replies), calls.Load(), "CancelBatchOrders should send every request with %s", tc.name)
		}

		// The context is cancelled while the second request waits for its reply
		ctx, cancel := context.WithCancel(t.Context())
		var calls atomic.Int32
		ex := newLocalServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
			if calls.Add(1) > 1 {
				// The server notices the client going away only once the request body is read
				_, err := io.Copy(io.Discard, r.Body)
				assert.NoError(t, err, "reading the request body should not error")
				cancel()
				<-r.Context().Done()
				return
			}
			_, err := io.WriteString(w, firstBatch)
			assert.NoError(t, err, "writing the reply should not error")
		})
		resp, err := ex.CancelBatchOrders(ctx, cancels)
		require.ErrorIs(t, err, context.Canceled, "CancelBatchOrders must return the error of the requests a cancelled context stopped")
		require.NotNil(t, resp, "CancelBatchOrders must return the statuses when its context is cancelled")
		assert.EqualValues(t, 2, calls.Load(), "CancelBatchOrders should send no request once its context is cancelled")
		for i, orderID := range orderIDs {
			if i < 10 {
				assert.Equalf(t, cancelled, resp.Status[orderID], "CancelBatchOrders should report order %s, cancelled before the context was, as cancelled", orderID)
				continue
			}
			assert.Containsf(t, resp.Status[orderID], context.Canceled.Error(), "CancelBatchOrders should report the cancelled context for order %s", orderID)
		}
		assert.Contains(t, resp.Status["gct-futures-client"], context.Canceled.Error(), "CancelBatchOrders should report the cancelled context for the order by client order ID")
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

	// minFunds is an order's least value, apart from quoteMinSize, a market buy's least funds
	ex = marketTestServerExchange(t, map[string]string{
		"/api/v2/symbols": `{"code":"200000","data":[{"symbol":"ETH-USDT","isMarginEnabled":true,"baseMinSize":"0.0001","quoteMinSize":"0.1","minFunds":"5","baseIncrement":"0.0001","priceIncrement":"0.01"}]}`,
	})
	for _, a := range []asset.Item{asset.Spot, asset.Margin} {
		require.NoErrorf(t, ex.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		l, err = ex.GetOrderExecutionLimits(a, currency.NewPair(currency.ETH, currency.USDT))
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", a)
		assert.Equalf(t, 5.0, l.MinNotional, "UpdateOrderExecutionLimits should take minFunds as the %s minimum order value", a)
		assert.Equalf(t, 0.1, l.MinimumQuoteAmount, "UpdateOrderExecutionLimits should keep quoteMinSize as the %s minimum quote amount", a)
		assert.ErrorIsf(t, l.Validate(3000, 0.001, order.Limit), limits.ErrNotionalValue, "a %s limit order worth less than minFunds should fail validation", a)
		assert.NoErrorf(t, l.Validate(3000, 0.002, order.Limit), "a %s limit order worth more than minFunds should pass validation", a)
	}

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
			MinNotional:             0.1,
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
				MinNotional:             0.1,
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
