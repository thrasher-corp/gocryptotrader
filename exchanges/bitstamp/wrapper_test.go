package bitstamp

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
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
	testsubs "github.com/thrasher-corp/gocryptotrader/internal/testing/subscriptions"
	testutils "github.com/thrasher-corp/gocryptotrader/internal/testing/utils"
	"github.com/thrasher-corp/gocryptotrader/portfolio/banking"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// newTestInstance returns an exchange instance named after the test, so tests which process tickers and orderbooks do
// not share global state
func newTestInstance(t *testing.T) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	if mockTests {
		require.NoError(t, testexch.MockHTTPInstance(ex, "api"), "MockHTTPInstance must not error")
		require.NoError(t, ex.DisableRateLimiter(), "DisableRateLimiter must not error")
	}
	return ex
}

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        5,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.LTC, currency.BTC),
		PurchasePrice: 1800,
	}
}

func TestGetFeeByTypeOfflineTradeFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType should error on a nil fee builder")

	skipIfLiveWithoutCredentials(t)
	feeBuilder := setFeeBuilder()
	_, err = e.GetFeeByType(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFeeByType must not error")
	if mockTests {
		assert.Equal(t, exchange.OfflineTradeFee, feeBuilder.FeeType, "FeeType should be offline when authentication checks are skipped")
	} else {
		assert.Equal(t, exchange.CryptocurrencyTradeFee, feeBuilder.FeeType, "FeeType should not change with valid credentials")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	_, err := e.GetFee(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetFee should error on a nil fee builder")

	skipIfLiveWithoutCredentials(t)
	feeBuilder := setFeeBuilder()
	fee, err := e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error")
	if mockTests {
		assert.InDelta(t, 0.4/100*1800*5, fee, 1e-9, "taker fee should be correct")
	}

	feeBuilder.IsMaker = true
	fee, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error for a maker fee")
	if mockTests {
		assert.InDelta(t, 0.3/100*1800*5, fee, 1e-9, "maker fee should be correct")
	}

	feeBuilder = setFeeBuilder()
	feeBuilder.PurchasePrice = -1000
	fee, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error for a negative purchase price")
	assert.Zero(t, fee, "negative fees should be zero")

	feeBuilder = setFeeBuilder()
	feeBuilder.FeeType = exchange.CryptocurrencyWithdrawalFee
	fee, err = e.GetFee(t.Context(), feeBuilder)
	require.NoError(t, err, "GetFee must not error for a withdrawal fee")
	if mockTests {
		assert.Equal(t, 0.001, fee, "withdrawal fee should be correct")
	}

	for feeType, exp := range map[exchange.FeeType]float64{
		exchange.CryptocurrencyDepositFee:       0,
		exchange.InternationalBankDepositFee:    7.5,
		exchange.InternationalBankWithdrawalFee: 15,
		exchange.OfflineTradeFee:                0.0025 * 1800 * 5,
	} {
		feeBuilder = setFeeBuilder()
		feeBuilder.FeeType = feeType
		fee, err = e.GetFee(t.Context(), feeBuilder)
		require.NoErrorf(t, err, "GetFee must not error for %v", feeType)
		assert.InDeltaf(t, exp, fee, 1e-9, "GetFee should return the fee for %v", feeType)
	}
}

func TestGetInternationalBankFees(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 7.5, getInternationalBankDepositFee(1000), "deposit fee should have a minimum")
	assert.Equal(t, 10.0, getInternationalBankDepositFee(20000), "deposit fee should be a percentage")
	assert.Equal(t, 300.0, getInternationalBankDepositFee(1000000), "deposit fee should have a maximum")
	assert.Equal(t, 15.0, getInternationalBankWithdrawalFee(1000), "withdrawal fee should have a minimum")
	assert.Equal(t, 18.0, getInternationalBankWithdrawalFee(20000), "withdrawal fee should be a percentage")
}

func TestGetOfflineTradeFee(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 22.5, getOfflineTradeFee(1800, 5), 1e-9, "getOfflineTradeFee should charge 0.25 percent")
	assert.Zero(t, getOfflineTradeFee(1800, 0), "getOfflineTradeFee should be zero without an amount")
}

func TestGetTradingFeeFromFeeBuilder(t *testing.T) {
	t.Parallel()
	_, err := e.getTradingFee(t.Context(), &exchange.FeeBuilder{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "getTradingFee should error without a pair")

	skipIfLiveWithoutCredentials(t)
	for isMaker, exp := range map[bool]float64{false: 0.4 / 100 * 1800 * 5, true: 0.3 / 100 * 1800 * 5} {
		feeBuilder := setFeeBuilder()
		feeBuilder.IsMaker = isMaker
		fee, err := e.getTradingFee(t.Context(), feeBuilder)
		require.NoErrorf(t, err, "getTradingFee must not error for maker %t", isMaker)
		if mockTests {
			assert.InDeltaf(t, exp, fee, 1e-9, "getTradingFee should apply the fee percentage for maker %t", isMaker)
		}
	}
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	assert.Equal(t, exchange.AutoWithdrawCryptoText+" & "+exchange.AutoWithdrawFiatText, e.FormatWithdrawPermissions(), "FormatWithdrawPermissions should be correct")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Bitstamp", ex.Name, "Name should be set")
	assert.True(t, ex.Enabled, "Enabled should be set")
	assert.Equal(t, config.APICredentialsValidatorConfig{RequiresKey: true, RequiresSecret: true}, ex.API.CredentialsValidator, "only a key and secret should be required")
	assert.ElementsMatch(t, asset.Items{asset.Spot, asset.PerpetualContract}, ex.GetAssetTypes(false), "spot and perpetual contracts should be supported")
	for u, exp := range map[exchange.URL]string{exchange.RestSpot: bitstampAPIURL, exchange.WebsocketSpot: bitstampWSURL} {
		got, err := ex.API.Endpoints.GetURL(u)
		require.NoErrorf(t, err, "GetURL must not error for %s", u)
		assert.Equalf(t, exp, got, "%s should be the default URL", u)
	}
	expREST := protocol.Features{
		TickerBatching:        true,
		AutoPairUpdates:       true,
		AccountBalance:        true,
		CryptoDeposit:         true,
		CryptoWithdrawal:      true,
		FiatWithdraw:          true,
		GetOrder:              true,
		GetOrders:             true,
		CancelOrders:          true,
		CancelOrder:           true,
		SubmitOrder:           true,
		DepositHistory:        true,
		WithdrawalHistory:     true,
		UserTradeHistory:      true,
		TradeFee:              true,
		FiatDepositFee:        true,
		FiatWithdrawalFee:     true,
		CryptoDepositFee:      true,
		CryptoWithdrawalFee:   true,
		TickerFetching:        true,
		KlineFetching:         true,
		TradeFetching:         true,
		OrderbookFetching:     true,
		FiatDeposit:           true,
		FundingRateFetching:   true,
		MultiChainDeposits:    true,
		MultiChainWithdrawals: true,
	}
	assert.Equal(t, expREST, ex.Features.Supports.RESTCapabilities, "RESTCapabilities should be correct")
	expWebsocket := protocol.Features{
		GetOrders:              true,
		TradeFetching:          true,
		OrderbookFetching:      true,
		FundingRateFetching:    true,
		Subscribe:              true,
		Unsubscribe:            true,
		AuthenticatedEndpoints: true,
	}
	assert.Equal(t, expWebsocket, ex.Features.Supports.WebsocketCapabilities, "WebsocketCapabilities should be correct")
	assert.Equal(t, maxHistoryAge, ex.Features.Supports.MaximumOrderHistory, "MaximumOrderHistory should be the retention period")
	expFutures := exchange.FuturesCapabilities{
		FundingRates:                    true,
		SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.EightHour: true},
		Positions:                       true,
		Leverage:                        true,
		OpenInterest:                    exchange.OpenInterestSupport{Supported: true, SupportedViaTicker: true, SupportsRestBatch: true},
	}
	assert.Equal(t, expFutures, ex.Features.Supports.FuturesCapabilities, "FuturesCapabilities should be correct")
	testsubs.EqualLists(t, defaultSubscriptions, ex.Features.Subscriptions)
	assert.NotNil(t, ex.Requester, "Requester should be set")
	assert.NotNil(t, ex.Websocket, "Websocket should be set")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Error(t, ex.Setup(nil), "Setup should error on a nil config")
	require.NoError(t, ex.Setup(&config.Exchange{}), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should disable the exchange")

	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	cfg := &config.Config{}
	require.NoError(t, cfg.LoadConfig(filepath.Join(root, "testdata", "configtest.json"), true), "LoadConfig must not error")
	exchCfg, err := cfg.GetExchangeConfig("Bitstamp")
	require.NoError(t, err, "GetExchangeConfig must not error")
	ex = new(Exchange)
	ex.SetDefaults()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
	assert.True(t, ex.IsEnabled(), "Setup should enable the exchange")
	urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	assert.Equal(t, []string{bitstampWSURL}, urls, "Setup should configure the websocket URL")
	assert.Equal(t, maxSubscriptionsPerConnection, ex.Websocket.MaxSubscriptionsPerConnection, "Setup should limit each connection to the documented number of subscriptions")
	assert.ElementsMatch(t, asset.Items{asset.Spot, asset.PerpetualContract}, ex.GetAssetTypes(true), "Setup should enable the configured assets")

	const configuredWebsocketURL = "wss://ws.example.com"
	exchCfg.API.Endpoints[exchange.WebsocketSpot.String()] = configuredWebsocketURL
	ex = new(Exchange)
	ex.SetDefaults()
	require.NoError(t, ex.Setup(exchCfg), "Setup must not error with a configured websocket URL")
	urls, err = ex.Websocket.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	assert.Equal(t, []string{configuredWebsocketURL}, urls, "Setup should configure the websocket URL from the configuration")
}

// previousVersionConfig writes the test config as config version 17 saved it, with Bitstamp's subscriptions replaced
// and authentication set up with only a key and secret; v17 required a client ID, so configs saved before the v2 API
// update keep that requirement and the default client ID placeholder, and hold no perpetual contract pairs
func previousVersionConfig(t *testing.T, subscriptions string) string {
	t.Helper()
	root, err := testutils.RootPathFromCWD()
	require.NoError(t, err, "RootPathFromCWD must not error")
	data, err := os.ReadFile(filepath.Join(root, "testdata", "configtest.json"))
	require.NoError(t, err, "ReadFile must not error")
	var saved map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(data, &saved), "Unmarshal must not error for the config")
	var exchanges []map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(saved["exchanges"], &exchanges), "Unmarshal must not error for the exchanges")
	i := slices.IndexFunc(exchanges, func(e map[string]json.RawMessage) bool { return string(e["name"]) == `"Bitstamp"` })
	require.NotEqual(t, -1, i, "the test config must hold Bitstamp")
	var api, features, currencyPairs, pairs map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(exchanges[i]["api"], &api), "Unmarshal must not error for the API settings")
	require.NoError(t, json.Unmarshal(exchanges[i]["features"], &features), "Unmarshal must not error for the features")
	require.NoError(t, json.Unmarshal(exchanges[i]["currencyPairs"], &currencyPairs), "Unmarshal must not error for the currency pairs")
	require.NoError(t, json.Unmarshal(currencyPairs["pairs"], &pairs), "Unmarshal must not error for the pairs")
	delete(pairs, "perpetualcontract")
	currencyPairs["pairs"], err = json.Marshal(pairs)
	require.NoError(t, err, "Marshal must not error for the pairs")
	api["authenticatedSupport"] = json.RawMessage(`true`)
	api["authenticatedWebsocketApiSupport"] = json.RawMessage(`true`)
	api["credentials"] = json.RawMessage(`{"key":"testkey","secret":"testsecret","clientID":"ClientID"}`)
	api["credentialsValidator"] = json.RawMessage(`{"requiresKey":true,"requiresSecret":true,"requiresClientID":true}`)
	features["subscriptions"] = json.RawMessage(subscriptions)
	for k, v := range map[string]any{"api": api, "features": features, "currencyPairs": currencyPairs} {
		exchanges[i][k], err = json.Marshal(v)
		require.NoErrorf(t, err, "Marshal must not error for %s", k)
	}
	saved["exchanges"], err = json.Marshal(exchanges)
	require.NoError(t, err, "Marshal must not error for the exchanges")
	saved["version"] = json.RawMessage(`17`)
	data, err = json.Marshal(saved)
	require.NoError(t, err, "Marshal must not error for the config")
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, data, 0o600), "WriteFile must not error")
	return path
}

func TestLoadPreviousVersionConfig(t *testing.T) {
	t.Parallel()
	const previousDefaults = `[{"enabled":true,"channel":"orderbook","asset":"spot","interval":"100ms"},{"enabled":true,"channel":"allTrades","asset":"spot"},{"enabled":true,"channel":"myOrders","asset":"spot","authenticated":true},{"enabled":true,"channel":"myTrades","asset":"spot","authenticated":true}]`
	qualified := func(markets []string, channels ...string) []string {
		names := make([]string, 0, len(markets)*len(channels))
		for _, c := range channels {
			for _, m := range markets {
				names = append(names, c+"_"+m)
			}
		}
		return names
	}
	spot := []string{"btcusd", "btceur", "eurusd", "xrpusd", "xrpeur"}
	customised := strings.Replace(previousDefaults, `{"enabled":true,"channel":"allTrades"`, `{"enabled":false,"channel":"allTrades"`, 1)
	var customisedList subscription.List
	require.NoError(t, json.Unmarshal([]byte(customised), &customisedList), "Unmarshal must not error for the customised subscriptions")
	for _, tc := range []struct {
		name             string
		subscriptions    string
		expSubscriptions subscription.List
		exp              []string
	}{
		{
			// The new defaults cover every enabled asset, which for a previous version config is only spot
			name:             "previous defaults",
			subscriptions:    previousDefaults,
			expSubscriptions: defaultSubscriptions,
			exp:              qualified(spot, "order_book", "live_trades", "my_orders", "my_trades"),
		},
		{
			// A customised list is the user's choice, so it is kept as it is
			name:             "customised",
			subscriptions:    customised,
			expSubscriptions: customisedList,
			exp:              qualified(spot, "order_book", "my_orders", "my_trades"),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := &config.Config{}
			require.NoError(t, cfg.LoadConfig(previousVersionConfig(t, tc.subscriptions), true), "LoadConfig must upgrade a previous version config")
			exchCfg, err := cfg.GetExchangeConfig("Bitstamp")
			require.NoError(t, err, "GetExchangeConfig must not error")
			expSubscriptions, err := json.Marshal(tc.expSubscriptions)
			require.NoError(t, err, "Marshal must not error for the expected subscriptions")
			subscriptions, err := json.Marshal(exchCfg.Features.Subscriptions)
			require.NoError(t, err, "Marshal must not error for the upgraded subscriptions")
			assert.JSONEq(t, string(expSubscriptions), string(subscriptions), "the upgraded config should hold the expected subscriptions")
			// v2 authentication only needs a key and secret, so the previous client ID requirement must not disable it
			assert.True(t, exchCfg.API.AuthenticatedSupport, "AuthenticatedSupport should remain enabled without a client ID")
			assert.True(t, exchCfg.API.AuthenticatedWebsocketSupport, "AuthenticatedWebsocketSupport should remain enabled without a client ID")
			assert.Equal(t, &config.APICredentialsValidatorConfig{RequiresKey: true, RequiresSecret: true}, exchCfg.API.CredentialsValidator, "CredentialsValidator should not require a client ID")

			ex := new(Exchange)
			ex.SetDefaults()
			ex.Websocket = sharedtestvalues.NewTestWebsocket()
			require.NoError(t, ex.Setup(exchCfg), "Setup must not error")
			ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
			subs, err := ex.generateSubscriptions()
			require.NoError(t, err, "generateSubscriptions must not error")
			channels := make([]string, len(subs))
			for i := range subs {
				channels[i] = subs[i].QualifiedChannel
			}
			assert.ElementsMatch(t, tc.exp, channels, "the upgraded subscriptions should expand to the expected channels")
		})
	}
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs should error on an unsupported asset")

	for a, exp := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		pairs, err := e.FetchTradablePairs(t.Context(), a)
		require.NoErrorf(t, err, "FetchTradablePairs must not error for %s", a)
		assert.Truef(t, pairs.Contains(exp, true), "FetchTradablePairs should return %s for %s", exp, a)
		for _, p := range pairs {
			assert.Equalf(t, a == asset.PerpetualContract, p.Quote.Upper().String() == "USD-PERP" || len(p.Quote.String()) > 5 && p.Quote.Upper().String()[len(p.Quote.String())-5:] == "-PERP", "pair %s should belong to %s", p, a)
		}
	}
}

func TestTradablePairs(t *testing.T) {
	t.Parallel()
	markets := []MarketResponse{
		{Name: "BTC/USD", MarketType: MarketTypeSpot, TradingEnabled: true},
		{Name: "ETH/USD", MarketType: MarketTypeSpot},
		{Name: "BTC/USD-PERP", MarketType: MarketTypePerpetual, TradingEnabled: true},
	}
	pairs, err := tradablePairs(markets, MarketTypeSpot)
	require.NoError(t, err, "tradablePairs must not error")
	assert.Equal(t, currency.Pairs{currency.NewPairWithDelimiter("BTC", "USD", "/")}, pairs, "tradablePairs should skip disabled and other market types")

	_, err = tradablePairs([]MarketResponse{{Name: "BTCUSD", MarketType: MarketTypeSpot, TradingEnabled: true}}, MarketTypeSpot)
	assert.Error(t, err, "tradablePairs should error on a malformed market name")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	ex := newTestInstance(t)
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	for a, exp := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		pairs, err := ex.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		assert.Truef(t, pairs.Contains(exp, true), "available pairs should contain %s for %s", exp, a)
		assert.Equalf(t, a == asset.PerpetualContract, pairs.Contains(perpetualPair, true), "perpetual contracts should only be available for %s", asset.PerpetualContract)
	}
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.UpdateOrderExecutionLimits(t.Context(), asset.Binary), asset.ErrNotSupported, "UpdateOrderExecutionLimits should error on an unsupported asset")

	for _, tc := range []struct {
		a    asset.Item
		pair currency.Pair
		exp  limits.MinMaxLevel
	}{
		{
			a:    asset.Spot,
			pair: spotPair,
			exp: limits.MinMaxLevel{
				Key:                     key.ExchangeAssetPair{Exchange: e.Name, Asset: asset.Spot, Base: currency.BTC.Item, Quote: currency.USD.Item},
				PriceStepIncrementSize:  0.01,
				MinimumQuoteAmount:      10,
				AmountStepIncrementSize: 0.00000001,
			},
		},
		{
			a:    asset.PerpetualContract,
			pair: perpetualPair,
			exp: limits.MinMaxLevel{
				Key:                     key.ExchangeAssetPair{Exchange: e.Name, Asset: asset.PerpetualContract, Base: currency.BTC.Item, Quote: perpetualPair.Quote.Item},
				PriceStepIncrementSize:  1,
				MinimumBaseAmount:       0.00001,
				MaximumBaseAmount:       20,
				MinimumQuoteAmount:      10,
				MaximumQuoteAmount:      1000000,
				AmountStepIncrementSize: 0.00001,
			},
		},
	} {
		a, pair := tc.a, tc.pair
		require.NoErrorf(t, e.UpdateOrderExecutionLimits(t.Context(), a), "UpdateOrderExecutionLimits must not error for %s", a)
		l, err := e.GetOrderExecutionLimits(a, pair)
		require.NoErrorf(t, err, "GetOrderExecutionLimits must not error for %s", pair)
		if mockTests {
			assert.NotZerof(t, l.UpdatedAt, "UpdatedAt should be set for %s", pair)
			l.UpdatedAt = time.Time{}
			assert.Equalf(t, tc.exp, l, "GetOrderExecutionLimits should return the market's limits for %s", pair)
			continue
		}
		assert.Positivef(t, l.PriceStepIncrementSize, "PriceStepIncrementSize should be positive for %s", pair)
		assert.Positivef(t, l.AmountStepIncrementSize, "AmountStepIncrementSize should be positive for %s", pair)
		assert.Positivef(t, l.MinimumQuoteAmount, "MinimumQuoteAmount should be positive for %s", pair)
		if a == asset.PerpetualContract {
			assert.Positivef(t, l.MinimumBaseAmount, "MinimumBaseAmount should be positive for %s", pair)
			assert.Greaterf(t, l.MaximumBaseAmount, l.MinimumBaseAmount, "MaximumBaseAmount should exceed MinimumBaseAmount for %s", pair)
			assert.Positivef(t, l.MaximumQuoteAmount, "MaximumQuoteAmount should be positive for %s", pair)
		}
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newTestInstance(t)
	assert.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Futures), asset.ErrNotSupported, "UpdateTickers should error on an unsupported asset")

	// Tickers are stored with the request formatted pair
	exp := map[asset.Item]*ticker.Price{
		asset.Spot: {
			Last:                       84246.67,
			VolumeWeightedAveragePrice: 84052.46,
			High:                       84900,
			Low:                        82873.04,
			Bid:                        84250.32,
			Ask:                        84250.33,
			BaseVolume:                 1588.01930719,
			Open:                       84383.03,
			Open24Hour:                 84110.88,
			PercentChange24Hour:        0.16,
			Pair:                       currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd")),
			ExchangeName:               ex.Name,
			AssetType:                  asset.Spot,
			LastUpdated:                time.Unix(1790317267, 0),
		},
		asset.PerpetualContract: {
			Last:                       84055,
			VolumeWeightedAveragePrice: 84071,
			High:                       84922,
			Low:                        82895,
			Bid:                        84260,
			Ask:                        84261,
			BaseVolume:                 64.26666,
			Open:                       84431,
			Open24Hour:                 84221,
			PercentChange24Hour:        -0.2,
			OpenInterest:               48.09714,
			OpenInterestValue:          4052232.0365504245,
			MarkPrice:                  84250.99780466,
			IndexPrice:                 84241.79066666667,
			Pair:                       currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
			ExchangeName:               ex.Name,
			AssetType:                  asset.PerpetualContract,
			LastUpdated:                time.Unix(1790317267, 0),
		},
	}
	for a, pair := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
		tick, err := ticker.GetTicker(ex.Name, pair, a)
		require.NoErrorf(t, err, "GetTicker must not error for %s", pair)
		if mockTests {
			assert.Equalf(t, exp[a], tick, "UpdateTickers should store the ticker for %s", pair)
			continue
		}
		assert.Positivef(t, tick.Last, "Last should be positive for %s", pair)
		assert.NotZerof(t, tick.LastUpdated, "LastUpdated should be set for %s", pair)
		if a == asset.PerpetualContract {
			assert.Positivef(t, tick.MarkPrice, "MarkPrice should be positive for %s", pair)
			assert.Positivef(t, tick.IndexPrice, "IndexPrice should be positive for %s", pair)
			assert.Positivef(t, tick.OpenInterest, "OpenInterest should be positive for %s", pair)
		}
	}
	_, err := ticker.GetTicker(ex.Name, perpetualPair, asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should not store perpetual tickers as spot")
}

func TestUpdateTickersMarketSelection(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/ticker/", r.URL.Path, "path should be correct")
		_, err := w.Write([]byte(`[
			{"market":"XRP/EUR","market_type":"SPOT","bid":"0.50","ask":"0.50","last":"0.50","side":"0","timestamp":"1790338043"},
			{"market":"BTC/USD","market_type":"SPOT","bid":"84250.32","ask":"84250.33","last":"84246.67","high":"84900.00","low":"82873.04","vwap":"84052.46","volume":"1588.01930719","open":"84383.03","open_24":"84110.88","percent_change_24":"","side":"1","timestamp":"1790317267"},
			{"market":"BTC/USD-PERP","market_type":"PERPETUAL","bid":"84260","ask":"84261","last":"84055","mark_price":"84250.99780466","index_price":"84241.79066666667","open_interest":"48.09714","open_interest_value":"4052232.0365504246724","side":"0","timestamp":"1790317267"},
			{"market":"ZRX/USD","market_type":"SPOT","bid":"1.00","ask":"1.10","last":"1.05","side":"0","timestamp":"1790338043"}
		]`))
		assert.NoError(t, err, "Write should not error")
	})

	err := ex.UpdateTickers(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrBidEqualsAsk, "UpdateTickers should report a locked enabled market")
	got, err := ticker.GetTicker(ex.Name, spotPair, asset.Spot)
	require.NoError(t, err, "UpdateTickers must store markets listed after a locked market")
	assert.Equal(t, &ticker.Price{
		Last:                       84246.67,
		VolumeWeightedAveragePrice: 84052.46,
		High:                       84900,
		Low:                        82873.04,
		Bid:                        84250.32,
		Ask:                        84250.33,
		BaseVolume:                 1588.01930719,
		Open:                       84383.03,
		Open24Hour:                 84110.88,
		Pair:                       currency.NewBTCUSD().Lower(),
		ExchangeName:               ex.Name,
		AssetType:                  asset.Spot,
		LastUpdated:                time.Unix(1790317267, 0),
	}, got, "UpdateTickers should store every ticker field, decoding an empty percentage change as zero")
	_, err = ticker.GetTicker(ex.Name, currency.NewPair(currency.ZRX, currency.USD), asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should not store disabled markets")
	_, err = ticker.GetTicker(ex.Name, perpetualPair, asset.Spot)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should not store perpetual markets as spot")

	require.NoError(t, ex.UpdateTickers(t.Context(), asset.PerpetualContract), "UpdateTickers must not error for perpetual contracts")
	got, err = ticker.GetTicker(ex.Name, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "UpdateTickers must store enabled perpetual markets")
	assert.Equal(t, &ticker.Price{
		Last:              84055,
		Bid:               84260,
		Ask:               84261,
		MarkPrice:         84250.99780466,
		IndexPrice:        84241.79066666667,
		OpenInterest:      48.09714,
		OpenInterestValue: 4052232.0365504246724,
		Pair:              currency.NewPairWithDelimiter("btc", "usd-perp", ""),
		ExchangeName:      ex.Name,
		AssetType:         asset.PerpetualContract,
		LastUpdated:       time.Unix(1790317267, 0),
	}, got, "UpdateTickers should store every perpetual ticker field")
	_, err = ticker.GetTicker(ex.Name, spotPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, ticker.ErrTickerNotFound, "UpdateTickers should not store spot markets as perpetual contracts")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestInstance(t)
	_, err := ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker should error on an empty pair")
	_, err = ex.UpdateTicker(t.Context(), spotPair, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker should error on an unsupported asset")

	exp := map[asset.Item]*ticker.Price{
		asset.Spot: {
			Last:                       84246.67,
			VolumeWeightedAveragePrice: 84052.46,
			High:                       84900,
			Low:                        82873.04,
			Bid:                        84250.32,
			Ask:                        84250.33,
			BaseVolume:                 1588.01336151,
			Open:                       84383.03,
			Open24Hour:                 84118.33,
			PercentChange24Hour:        0.15,
			Pair:                       currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd")),
			ExchangeName:               ex.Name,
			AssetType:                  asset.Spot,
			LastUpdated:                time.Unix(1790317268, 0),
		},
		asset.PerpetualContract: {
			Last:                       84055,
			VolumeWeightedAveragePrice: 84071,
			High:                       84922,
			Low:                        82895,
			Bid:                        84260,
			Ask:                        84261,
			BaseVolume:                 64.26666,
			Open:                       84431,
			Open24Hour:                 84221,
			PercentChange24Hour:        -0.2,
			OpenInterest:               48.09714,
			OpenInterestValue:          4052171.530272792,
			MarkPrice:                  84249.73980309,
			IndexPrice:                 84240.324,
			Pair:                       currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
			ExchangeName:               ex.Name,
			AssetType:                  asset.PerpetualContract,
			LastUpdated:                time.Unix(1790317270, 0),
		},
	}
	for a, pair := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		tick, err := ex.UpdateTicker(t.Context(), pair, a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", pair)
		if mockTests {
			assert.Equalf(t, exp[a], tick, "UpdateTicker should return the ticker for %s", pair)
			continue
		}
		assert.Truef(t, tick.Pair.Equal(pair), "Pair should be correct for %s", pair)
		assert.Equalf(t, a, tick.AssetType, "AssetType should be correct for %s", pair)
		assert.Positivef(t, tick.Last, "Last should be positive for %s", pair)
		assert.Positivef(t, tick.BaseVolume, "BaseVolume should be positive for %s", pair)
	}
}

func TestTickerPrice(t *testing.T) {
	t.Parallel()
	tick := &TickerResponse{
		Last:                       84010,
		High:                       85000,
		Low:                        83000,
		VolumeWeightedAveragePrice: 84052.46,
		Volume:                     12.5,
		BestBid:                    84000,
		BestAsk:                    84020,
		Timestamp:                  types.Time(time.Unix(1790318552, 0)),
		Open:                       83500,
		Open24Hour:                 83400,
		PercentChange24Hour:        -0.2,
		MarkPrice:                  84005,
		IndexPrice:                 84001,
		OpenInterest:               150.25,
		OpenInterestValue:          12621751.25,
	}
	assert.Equal(t, &ticker.Price{
		Last:                       84010,
		VolumeWeightedAveragePrice: 84052.46,
		High:                       85000,
		Low:                        83000,
		Bid:                        84000,
		Ask:                        84020,
		BaseVolume:                 12.5,
		Open:                       83500,
		Open24Hour:                 83400,
		PercentChange24Hour:        -0.2,
		MarkPrice:                  84005,
		IndexPrice:                 84001,
		OpenInterest:               150.25,
		OpenInterestValue:          12621751.25,
		Pair:                       perpetualPair,
		ExchangeName:               e.Name,
		AssetType:                  asset.PerpetualContract,
		LastUpdated:                time.Unix(1790318552, 0),
	}, e.tickerPrice(tick, perpetualPair, asset.PerpetualContract), "tickerPrice should convert the ticker")
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestInstance(t)
	_, err := ex.UpdateOrderbook(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateOrderbook should error on an empty pair")
	_, err = ex.UpdateOrderbook(t.Context(), spotPair, asset.Futures)
	assert.Error(t, err, "UpdateOrderbook should error on an unsupported asset")

	// The mock books' best levels; books are fetched grouped by price
	exp := map[asset.Item]orderbook.Book{
		asset.Spot: {
			Bids:              orderbook.Levels{{Price: 84250.32, Amount: 0.46625964}},
			Asks:              orderbook.Levels{{Price: 84250.33, Amount: 0.57086879}},
			Exchange:          ex.Name,
			Pair:              spotPair,
			Asset:             asset.Spot,
			LastUpdated:       time.UnixMicro(1790317272953412),
			ValidateOrderbook: true,
			RestSnapshot:      true,
		},
		asset.PerpetualContract: {
			Bids:              orderbook.Levels{{Price: 84258, Amount: 0.33625}},
			Asks:              orderbook.Levels{{Price: 84259, Amount: 0.45819}},
			Exchange:          ex.Name,
			Pair:              perpetualPair,
			Asset:             asset.PerpetualContract,
			LastUpdated:       time.UnixMicro(1790317277763643),
			ValidateOrderbook: true,
			RestSnapshot:      true,
		},
	}
	for a, pair := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		ob, err := ex.UpdateOrderbook(t.Context(), pair, a)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s", pair)
		require.NotEmptyf(t, ob.Bids, "Bids must not be empty for %s", pair)
		require.NotEmptyf(t, ob.Asks, "Asks must not be empty for %s", pair)
		if mockTests {
			require.Lenf(t, ob.Bids, 10, "UpdateOrderbook must store every bid for %s", pair)
			require.Lenf(t, ob.Asks, 10, "UpdateOrderbook must store every ask for %s", pair)
			assert.NotZerof(t, ob.InsertedAt, "InsertedAt should be set for %s", pair)
			top := *ob
			top.Bids, top.Asks, top.InsertedAt = top.Bids[:1], top.Asks[:1], time.Time{}
			assert.Equalf(t, exp[a], top, "UpdateOrderbook should store the book for %s", pair)
			continue
		}
		assert.Greaterf(t, ob.Asks[0].Price, ob.Bids[0].Price, "best ask should exceed the best bid for %s", pair)
		assert.NotZerof(t, ob.LastUpdated, "LastUpdated should be set for %s", pair)
	}
}

func TestOrderbookLevels(t *testing.T) {
	t.Parallel()
	assert.Empty(t, orderbookLevels(nil), "orderbookLevels should return no levels without levels")
	levels := []OrderbookLevel{{Price: 84010, Amount: 0.0119, OrderID: 1458532827766784}, {Price: 84006, Amount: 0.33643}}
	assert.Equal(t, orderbook.Levels{{Price: 84010, Amount: 0.0119}, {Price: 84006, Amount: 0.33643}}, orderbookLevels(levels), "orderbookLevels should convert the levels")
}

func TestFilterOrderbookZeroBidPrice(t *testing.T) {
	t.Parallel()
	ob := &orderbook.Book{}
	filterOrderbookZeroBidPrice(ob)
	assert.Empty(t, ob.Bids, "empty bids should remain empty")

	ob.Bids = orderbook.Levels{{Price: 69, Amount: 1337}, {Price: 0, Amount: 69}}
	filterOrderbookZeroBidPrice(ob)
	assert.Equal(t, orderbook.Levels{{Price: 69, Amount: 1337}}, ob.Bids, "the trailing zero priced bid should be removed")

	ob.Bids = orderbook.Levels{{Price: 59, Amount: 1337}, {Price: 42, Amount: 8595}}
	filterOrderbookZeroBidPrice(ob)
	assert.Equal(t, orderbook.Levels{{Price: 59, Amount: 1337}, {Price: 42, Amount: 8595}}, ob.Bids, "priced bids should be retained")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateAccountBalances(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances should error on an unsupported asset")

	skipIfLiveWithoutCredentials(t)
	ctx := t.Context()
	if mockTests {
		// Balances are stored against the credentials used, which the mock instance does not have
		ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "key", Secret: "secret"})
	}
	// Perpetual contract balances are the derivatives account's collateral
	for a, exp := range map[asset.Item]accounts.CurrencyBalances{
		asset.Spot: {
			currency.BTC: {Currency: currency.BTC, Total: 1.5, Hold: 0.25, Free: 1.25},
			currency.USD: {Currency: currency.USD, Total: 100, Hold: 10, Free: 90},
		},
		asset.PerpetualContract: {
			currency.BTC: {Currency: currency.BTC, Total: 0.1, Hold: 0.01, Free: 0.09},
			currency.USD: {Currency: currency.USD, Total: 15000, Hold: 1500, Free: 13500},
		},
	} {
		subAccts, err := e.UpdateAccountBalances(ctx, a)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s", a)
		require.Lenf(t, subAccts, 1, "UpdateAccountBalances must return a sub account for %s", a)
		assert.Equalf(t, a, subAccts[0].AssetType, "AssetType should be correct for %s", a)
		if !mockTests {
			continue
		}
		for c, balance := range subAccts[0].Balances {
			assert.NotZerof(t, balance.UpdatedAt, "UpdatedAt should be set for %s %s", a, c)
			balance.UpdatedAt = time.Time{}
			subAccts[0].Balances[c] = balance
		}
		assert.Equalf(t, accounts.SubAccounts{{AssetType: a, Balances: exp}}, subAccts, "UpdateAccountBalances should return the balances for %s", a)
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.ValidateAPICredentials(t.Context(), asset.Futures), asset.ErrNotSupported, "ValidateAPICredentials should error on an unsupported asset")

	skipIfLiveWithoutCredentials(t)
	ctx := t.Context()
	if mockTests {
		// Balances are stored against the credentials used, which the mock instance does not have
		ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{Key: "key", Secret: "secret"})
	}
	for _, a := range e.GetAssetTypes(false) {
		assert.NoErrorf(t, e.ValidateAPICredentials(ctx, a), "ValidateAPICredentials should not error for %s", a)
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	history, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if mockTests {
		exp := []exchange.FundingHistory{
			{
				ExchangeName:      e.Name,
				Status:            DepositStatusPending,
				TransferID:        "1",
				Timestamp:         time.Unix(1759995000, 0),
				Currency:          "BTC",
				Amount:            1.23,
				TransferType:      "deposit",
				CryptoToAddress:   "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				CryptoFromAddress: "1htZQJS6YUUPIGNNPC9425xENXQS2JwM7C",
				CryptoTxID:        "e4123d1d57df4106aaae5ec4d77eb6cd42e226d020a0a2c1c7919d14b93",
				CryptoChain:       "bitcoin",
			},
			{
				ExchangeName:    e.Name,
				Timestamp:       time.Unix(1642665114, 0),
				Currency:        "BTC",
				Amount:          0.00012,
				TransferType:    "withdrawal",
				CryptoToAddress: "3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF",
				CryptoChain:     "bitcoin",
			},
		}
		assert.Equal(t, exp, history, "GetAccountFundingHistory should return the deposit and withdrawal")
	}
}

func TestGetAccountFundingHistoryWithdrawalTransaction(t *testing.T) {
	t.Parallel()
	ex := newResponseTestExchange(t, http.StatusOK, `{"deposits":[],"withdrawals":[{"currency":"BTC","network":"bitcoin","destinationAddress":"3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF","txid":"8d3c0a2e1d","amount":0.00012,"datetime":1642665114}],"ripple_iou_transactions":[]}`)
	history, err := ex.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	assert.Equal(t, []exchange.FundingHistory{{
		ExchangeName:    ex.Name,
		Timestamp:       time.Unix(1642665114, 0),
		Currency:        "BTC",
		Amount:          0.00012,
		TransferType:    "withdrawal",
		CryptoToAddress: "3FiKkjgZ6Sj4RWp3ZsCjYh5Pt7ZCBsL7uF",
		CryptoTxID:      "8d3c0a2e1d",
		CryptoChain:     "bitcoin",
	}}, history, "GetAccountFundingHistory should return a processed withdrawal's transaction hash")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	for c, exp := range map[currency.Code]int{currency.EMPTYCODE: 2, currency.BTC: 1} {
		history, err := e.GetWithdrawalsHistory(t.Context(), c, asset.Spot)
		require.NoErrorf(t, err, "GetWithdrawalsHistory must not error for %q", c)
		if !mockTests {
			continue
		}
		require.Lenf(t, history, exp, "GetWithdrawalsHistory must return the withdrawals for %q", c)
		assert.Equal(t, exchange.WithdrawalHistory{
			Status:          "Finished",
			TransferID:      "1",
			Timestamp:       time.Date(2022, 1, 31, 16, 7, 32, 0, time.UTC),
			Currency:        "BTC",
			Amount:          0.00006,
			TransferType:    "Cryptocurrency",
			CryptoToAddress: core.BitcoinDonationAddress,
			CryptoTxID:      "NsOeFbQhRnpGzNIThWGBTkQwRJqTNOGPVhYavrVyMfkAyMUmIlUpFIwGTzSvpeOP",
			CryptoChain:     "bitcoin",
		}, history[0], "withdrawal should be correct")
	}
}

func TestWithdrawalStatusAndType(t *testing.T) {
	t.Parallel()
	for status, exp := range map[WithdrawalStatus]string{
		WithdrawalStatusOpen:      "Open",
		WithdrawalStatusInProcess: "In process",
		WithdrawalStatusFinished:  "Finished",
		WithdrawalStatusCancelled: "Cancelled",
		WithdrawalStatusFailed:    "Failed",
		WithdrawalStatusReversed:  "Reversed",
		7:                         "7",
	} {
		assert.Equalf(t, exp, withdrawalStatus(status), "withdrawalStatus should describe %d", status)
	}
	for withdrawalTypeID, exp := range map[WithdrawalType]string{WithdrawalTypeSEPA: "SEPA", WithdrawalTypeWire: "Wire transfer", 1: "Cryptocurrency", 22: "Cryptocurrency"} {
		assert.Equalf(t, exp, withdrawalType(withdrawalTypeID), "withdrawalType should describe %d", withdrawalTypeID)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetRecentTrades should error on an empty pair")
	_, err = e.GetRecentTrades(t.Context(), spotPair, asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades should error on an unsupported asset")

	for a, pair := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		trades, err := e.GetRecentTrades(t.Context(), pair, a)
		require.NoErrorf(t, err, "GetRecentTrades must not error for %s", pair)
		require.NotEmptyf(t, trades, "GetRecentTrades must return trades for %s", pair)
		if mockTests {
			// Trades are returned oldest first with the request formatted pair
			exp := map[asset.Item]trade.Data{
				asset.Spot: {
					TID:          "642950919",
					Exchange:     e.Name,
					CurrencyPair: currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd")),
					AssetType:    asset.Spot,
					Side:         order.Sell,
					Price:        84250.32,
					Amount:       0.00108,
					Timestamp:    time.Unix(1790317248, 0),
				},
				asset.PerpetualContract: {
					TID:          "2005537666656911360",
					Exchange:     e.Name,
					CurrencyPair: currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
					AssetType:    asset.PerpetualContract,
					Side:         order.Sell,
					Price:        84153,
					Amount:       0.00023,
					Timestamp:    time.Unix(1790314240, 0),
				},
			}[a]
			assert.Equalf(t, exp, trades[0], "GetRecentTrades should return the oldest trade first for %s", pair)
		}
		for i := range trades {
			assert.Equalf(t, a, trades[i].AssetType, "AssetType should be correct for %s", pair)
			assert.Truef(t, trades[i].CurrencyPair.Equal(pair), "CurrencyPair should be correct for %s", pair)
			assert.NotEmptyf(t, trades[i].TID, "TID should be set for %s", pair)
			assert.Positivef(t, trades[i].Price, "Price should be positive for %s", pair)
			assert.Positivef(t, trades[i].Amount, "Amount should be positive for %s", pair)
			if i > 0 {
				assert.Falsef(t, trades[i].Timestamp.Before(trades[i-1].Timestamp), "trades should be sorted by date for %s", pair)
			}
		}
	}
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricTrades(t.Context(), spotPair, asset.Spot, time.Now().Add(-time.Minute*15), time.Now())
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported")
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder should error on a nil submission")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotPair, AssetType: asset.Futures, Side: order.Buy, Type: order.Market, Amount: 1})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "SubmitOrder should error on an unsupported asset")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotPair, AssetType: asset.Spot, Side: order.Buy, Type: order.StopLimit, Amount: 1, Price: 1})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "SubmitOrder should error on a conditional spot order")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.StopLimit, Amount: 1, Price: 1, Leverage: 2})
	assert.ErrorIs(t, err, errTriggerPriceRequired, "SubmitOrder should error on a conditional order without a trigger price")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.TrailingStop, Amount: 1, TriggerPrice: 1, Leverage: 2})
	assert.ErrorIs(t, err, order.ErrUnknownTrackingMode, "SubmitOrder should error on a trailing stop without a percentage tracking mode")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.OCO, Amount: 1, Price: 1, Leverage: 2})
	assert.ErrorIs(t, err, order.ErrUnsupportedOrderType, "SubmitOrder should error on an unsupported order type")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, Leverage: 2, MarginType: margin.SpotIsolated})
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "SubmitOrder should error on an unsupported margin type")
	_, err = e.SubmitOrder(t.Context(), &order.Submit{Exchange: e.Name, Pair: spotPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 1, Price: 1, TimeInForce: order.GoodTillCrossing})
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "SubmitOrder should error on an unsupported time in force")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	for _, tc := range []struct {
		name string
		s    *order.Submit
		exp  string
	}{
		{name: "spot limit", s: &order.Submit{Pair: spotPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Limit, Amount: 0.5, Price: 20000, ClientOrderID: "123456789"}, exp: "1234123412341236"},
		{name: "spot market", s: &order.Submit{Pair: spotPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, Amount: 0.1}, exp: "1234123412341238"},
		{name: "spot market quote amount", s: &order.Submit{Pair: spotPair, AssetType: asset.Spot, Side: order.Buy, Type: order.Market, QuoteAmount: 100}, exp: "1234123412341239"},
		{name: "perpetual limit", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.Limit, Amount: 0.01, Price: 20000, Leverage: 5, MarginType: margin.Isolated}, exp: "1234123412341241"},
		{name: "perpetual market with current leverage", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Type: order.Market, Amount: 0.01}, exp: "1234123412341242"},
		{name: "perpetual stop loss", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Type: order.StopMarket, Amount: 0.01, TriggerPrice: 19000, TriggerPriceType: order.MarkPrice, Leverage: 2, ReduceOnly: true}, exp: "1234123412341243"},
		{name: "perpetual take profit limit", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.TakeProfit | order.Limit, Amount: 0.01, Price: 21000, TriggerPrice: 20500, Leverage: 2}, exp: "1234123412341244"},
		{name: "perpetual trailing stop", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Type: order.TrailingStop, Amount: 0.01, TriggerPrice: 21000, TrackingMode: order.Percentage, TrackingValue: 5, Leverage: 2, ReduceOnly: true}, exp: "1234123412341246"},
		{name: "perpetual trailing stop limit", s: &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Type: order.TrailingStopLimit, Amount: 0.01, Price: 19500, TriggerPrice: 19000, TrackingMode: order.Percentage, TrackingValue: 2.5, Leverage: 2, TimeInForce: order.ImmediateOrCancel}, exp: "1234123412341247"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.s.Exchange = e.Name
			resp, err := e.SubmitOrder(t.Context(), tc.s)
			require.NoError(t, err, "SubmitOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, resp.OrderID, "OrderID should be correct")
				assert.Equal(t, tc.s.AssetType, resp.AssetType, "AssetType should be correct")
			}
		})
	}
}

func TestSubmitOrderRequests(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		s    *order.Submit
		path string
		exp  url.Values
	}{
		{
			name: "market sell sized by a quote amount",
			s:    &order.Submit{Pair: spotPair, AssetType: asset.Spot, Side: order.Sell, Type: order.Market, QuoteAmount: 100},
			path: "/api/v2/sell/instant/btcusd/",
			exp:  url.Values{"amount": {"100"}, "amount_in_counter": {"True"}},
		},
		{
			name: "perpetual take profit",
			s:    &order.Submit{Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Type: order.TakeProfitMarket, Amount: 0.01, TriggerPrice: 21000, Leverage: 2},
			path: "/api/v2/sell/market/btcusd-perp/",
			exp:  url.Values{"amount": {"0.01"}, "subtype": {"TAKE_PROFIT"}, "stop_price": {"21000"}, "margin_mode": {"CROSS"}, "leverage": {"2"}, "trigger": {"LAST_TRADED_PRICE"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			requests := make(chan url.Values, 1)
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.path, r.URL.Path, "path should be correct")
				assert.NoError(t, r.ParseForm(), "ParseForm should not error")
				select {
				case requests <- r.PostForm:
				default:
					assert.Fail(t, "SubmitOrder should send only one request")
				}
				_, err := w.Write([]byte(`{"id":"1"}`))
				assert.NoError(t, err, "Write should not error")
			})
			tc.s.Exchange = ex.Name
			resp, err := ex.SubmitOrder(t.Context(), tc.s)
			require.NoError(t, err, "SubmitOrder must not error")
			assert.Equal(t, "1", resp.OrderID, "OrderID should be correct")
			require.Len(t, requests, 1, "SubmitOrder must send a request")
			assert.Equal(t, tc.exp, <-requests, "SubmitOrder should send the order's parameters")
		})
	}
}

func TestSubmitConditionalOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		s   *order.Submit
		err error
	}{
		{s: &order.Submit{Type: order.StopLimit, TriggerPrice: 1, TimeInForce: order.GoodTillCrossing}, err: order.ErrUnsupportedTimeInForce},
		{s: &order.Submit{Type: order.TrailingStopLimit, TriggerPrice: 1, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedTimeInForce},
		{s: &order.Submit{Type: order.OCO}, err: order.ErrUnsupportedOrderType},
	} {
		_, err := e.submitConditionalOrder(t.Context(), tc.s, perpetualPair, &derivativesOrderParams{})
		assert.ErrorIsf(t, err, tc.err, "submitConditionalOrder should error for %s", tc.s.Type)
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	for _, tc := range []struct {
		s   *order.Submit
		p   *derivativesOrderParams
		exp uint64
	}{
		{
			s:   &order.Submit{Side: order.Sell, Type: order.StopMarket, Amount: 0.01, TriggerPrice: 19000},
			p:   &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, Trigger: TriggerMarkPrice, ReduceOnly: true},
			exp: 1234123412341243,
		},
		{
			s:   &order.Submit{Side: order.Buy, Type: order.TakeProfit | order.Limit, Amount: 0.01, Price: 21000, TriggerPrice: 20500},
			p:   &derivativesOrderParams{MarginMode: MarginModeCross, Leverage: 2, Trigger: TriggerLastTradedPrice},
			exp: 1234123412341244,
		},
	} {
		resp, err := e.submitConditionalOrder(t.Context(), tc.s, perpetualPair, tc.p)
		require.NoErrorf(t, err, "submitConditionalOrder must not error for %s", tc.s.Type)
		if mockTests {
			assert.Equalf(t, tc.exp, resp.ID, "ID should be correct for %s", tc.s.Type)
		}
	}
}

func TestValidateOrderSubmission(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		s   *order.Submit
		err error
	}{
		{s: &order.Submit{AssetType: asset.Spot, Type: order.Limit}},
		{s: &order.Submit{AssetType: asset.Spot, Type: order.Market, QuoteAmount: 1}},
		{s: &order.Submit{AssetType: asset.Spot, Type: order.StopLimit, TriggerPrice: 1}, err: order.ErrUnsupportedOrderType},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.OCO}, err: order.ErrUnsupportedOrderType},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.StopMarket}, err: errTriggerPriceRequired},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.TakeProfit | order.Limit, TriggerPrice: 1}},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.TrailingStop, TriggerPrice: 1, TrackingMode: order.Distance, TrackingValue: 1}, err: order.ErrUnknownTrackingMode},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.TrailingStopLimit, TriggerPrice: 1, TrackingMode: order.Percentage}, err: errTrackingValueRequired},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.TrailingStop, TriggerPrice: 1, TrackingMode: order.Percentage, TrackingValue: 5}},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.Market, QuoteAmount: 1, ReduceOnly: true}, err: errReduceOnlyUnsupported},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.Market, Amount: 1, ReduceOnly: true}},
		{s: &order.Submit{AssetType: asset.Spot, Type: order.Limit, QuoteAmount: 1}, err: order.ErrAmountIsInvalid},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.StopMarket, TriggerPrice: 1, QuoteAmount: 1}, err: order.ErrAmountIsInvalid},
		{s: &order.Submit{AssetType: asset.Spot, Type: order.Market, Amount: 1, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.StopMarket, TriggerPrice: 1, Amount: 1, TimeInForce: order.PostOnly}, err: order.ErrUnsupportedTimeInForce},
		{s: &order.Submit{AssetType: asset.PerpetualContract, Type: order.TrailingStop, TriggerPrice: 1, TrackingMode: order.Percentage, TrackingValue: 5, Amount: 1, TimeInForce: order.GoodTillTime}, err: order.ErrUnsupportedTimeInForce},
		{s: &order.Submit{AssetType: asset.Spot, Type: order.Market, Amount: 1, TimeInForce: order.ImmediateOrCancel}},
	} {
		assert.ErrorIsf(t, validateOrderSubmission(tc.s), tc.err, "validateOrderSubmission should return the correct error for a %s %s order", tc.s.AssetType, tc.s.Type)
	}
}

func TestTrailingDeltaBasisPoints(t *testing.T) {
	t.Parallel()
	for percentage, exp := range map[float64]uint64{0.01: 1, 0.123: 12, 0.29: 29, 2.5: 250, 5: 500, 20: 2000} {
		assert.Equalf(t, exp, trailingDeltaBasisPoints(percentage), "trailingDeltaBasisPoints should convert %v percent", percentage)
	}
}

func TestSetConditionalOrderTimeInForce(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		tif order.TimeInForce
		exp LimitOrderRequest
		err error
	}{
		{tif: order.UnknownTIF},
		{tif: order.GoodTillCancel},
		{tif: order.ImmediateOrCancel, exp: LimitOrderRequest{ImmediateOrCancel: true}},
		{tif: order.FillOrKill, exp: LimitOrderRequest{FillOrKill: true}},
		{tif: order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{tif: order.GoodTillDay, err: order.ErrUnsupportedTimeInForce},
		{tif: order.GoodTillTime, err: order.ErrUnsupportedTimeInForce},
	} {
		var req LimitOrderRequest
		assert.ErrorIsf(t, setConditionalOrderTimeInForce(&req, tc.tif), tc.err, "setConditionalOrderTimeInForce should return the correct error for %s", tc.tif)
		assert.Equalf(t, tc.exp, req, "setConditionalOrderTimeInForce should set the flags for %s", tc.tif)
	}
}

func TestSetLimitOrderTimeInForce(t *testing.T) {
	t.Parallel()
	endTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		tif order.TimeInForce
		exp LimitOrderRequest
		err error
	}{
		{tif: order.UnknownTIF},
		{tif: order.GoodTillCancel},
		{tif: order.ImmediateOrCancel, exp: LimitOrderRequest{ImmediateOrCancel: true}},
		{tif: order.FillOrKill, exp: LimitOrderRequest{FillOrKill: true}},
		{tif: order.GoodTillCancel | order.PostOnly, exp: LimitOrderRequest{MakerOrCancel: true}},
		{tif: order.GoodTillDay, exp: LimitOrderRequest{DailyOrder: true}},
		{tif: order.GoodTillTime, exp: LimitOrderRequest{GoodTillDate: true, ExpireTime: endTime}},
		{tif: order.GoodTillCrossing, err: order.ErrUnsupportedTimeInForce},
		{tif: order.StopOrReduce, err: order.ErrUnsupportedTimeInForce},
		{tif: order.ImmediateOrCancel | order.PostOnly, err: order.ErrUnsupportedTimeInForce},
		{tif: order.GoodTillTime | order.PostOnly, err: order.ErrUnsupportedTimeInForce},
	} {
		var req LimitOrderRequest
		err := setLimitOrderTimeInForce(&req, tc.tif, endTime)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "setLimitOrderTimeInForce should error for %s", tc.tif)
			continue
		}
		require.NoErrorf(t, err, "setLimitOrderTimeInForce must not error for %s", tc.tif)
		assert.Equalf(t, tc.exp, req, "setLimitOrderTimeInForce should set the flags for %s", tc.tif)
	}
	assert.ErrorIs(t, setLimitOrderTimeInForce(&LimitOrderRequest{}, order.GoodTillTime, time.Time{}), common.ErrDateUnset, "setLimitOrderTimeInForce should error without an end time")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), &order.Modify{Exchange: e.Name, Pair: spotPair, AssetType: asset.Spot, OrderID: "1", Amount: 1, Price: 1})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "ModifyOrder should not be supported")
}

func TestExecutedAmount(t *testing.T) {
	t.Parallel()
	executed, err := executedAmount(nil, currency.BTC)
	require.NoError(t, err, "executedAmount must not error without transactions")
	assert.True(t, executed.IsZero(), "executedAmount should be zero without transactions")
	executed, err = executedAmount([]OrderTransaction{
		{TradeID: 1, Amounts: map[currency.Code]float64{currency.BTC: -0.1, currency.USD: 8400}},
		{TradeID: 2, Amounts: map[currency.Code]float64{currency.BTC: 0.22345678, currency.USD: -18773.97}},
	}, currency.NewCode("btc"))
	require.NoError(t, err, "executedAmount must not error")
	assert.Equal(t, "0.32345678", executed.String(), "executedAmount should sum the absolute base amounts without a float residue")
	_, err = executedAmount([]OrderTransaction{{TradeID: 1, Amounts: map[currency.Code]float64{currency.USD: 8400}}}, currency.BTC)
	assert.ErrorIs(t, err, errMalformedTransaction, "executedAmount should error on a transaction without a base amount")
	_, err = executedAmount([]OrderTransaction{{TradeID: 1, Amounts: map[currency.Code]float64{currency.BTC: math.NaN()}}}, currency.BTC)
	assert.ErrorIs(t, err, errMalformedTransaction, "executedAmount should error on a base amount which is not a number")
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder should error on a nil request")
	assert.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: "nope"}), order.ErrOrderIDNotSet, "CancelOrder should error on an invalid order ID")
	assert.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{}), order.ErrOrderIDNotSet, "CancelOrder should error without an order ID")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	err := e.CancelOrder(t.Context(), &order.Cancel{OrderID: "1453282316578816"})
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "CancelOrder should error for an unknown order")
		return
	}
	assert.NoError(t, err, "CancelOrder should not error")
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{ClientOrderID: "my-order-123"}), "CancelOrder should not error with a client order ID")
}

func TestCancelOrderPrefersOrderID(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/cancel_order/", r.URL.Path, "path should be correct")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		assert.Equal(t, url.Values{"id": {"1453282316578816"}}, r.PostForm, "only the order ID should be sent")
		_, err := w.Write([]byte(`{"id":1453282316578816,"amount":0.02035278,"price":2100.45,"type":0,"market":"BTC/USD","status":"Canceled"}`))
		assert.NoError(t, err, "Write should not error")
	})
	err := ex.CancelOrder(t.Context(), &order.Cancel{OrderID: "1453282316578816", ClientOrderID: "my-order-123"})
	assert.NoError(t, err, "CancelOrder should not error with both IDs")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "CancelBatchOrders should not be supported")
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders should error on a nil request")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Futures})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders should error on an unsupported asset")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	for _, req := range []*order.Cancel{{AssetType: asset.Spot, Pair: spotPair}, {AssetType: asset.Spot}} {
		resp, err := e.CancelAllOrders(t.Context(), req)
		require.NoErrorf(t, err, "CancelAllOrders must not error for %q", req.Pair)
		if mockTests {
			assert.Equalf(t, map[string]string{"1234123412341234": order.Cancelled.String()}, resp.Status, "CancelAllOrders should only cancel spot orders for %q", req.Pair)
		}
	}
	if !mockTests {
		return
	}
	resp, err := e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errCancelAllOrdersFailed, "CancelAllOrders should error when not all orders are cancelled")
	assert.Equal(t, map[string]string{"1234123412341235": order.Cancelled.String()}, resp.Status, "CancelAllOrders should report the cancelled perpetual orders")
}

func TestCancelAllOrdersOpenOrderMarkets(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var cancelled []string
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		body := `{"canceled":[],"success":true}`
		switch {
		case r.URL.Path == "/api/v2/open_orders/":
			// ZRX/EUR is not enabled, as for an order placed outside GoCryptoTrader
			body = `[{"id":"5","datetime":"2025-06-16 10:00:00","type":"0","price":"1","amount":"1","amount_at_create":"1","market":"ZRX/EUR"},{"id":"6","datetime":"2025-06-16 10:00:00","type":"0","price":"1","amount":"1","amount_at_create":"1","market":"ETH/USD-PERP"}]`
		case strings.HasPrefix(r.URL.Path, "/api/v2/cancel_all_orders/"):
			mu.Lock()
			cancelled = append(cancelled, r.URL.Path)
			mu.Unlock()
		default:
			assert.Failf(t, "unexpected request", "%s should not be requested", r.URL.Path)
		}
		_, err := w.Write([]byte(body))
		assert.NoError(t, err, "Write should not error")
	})
	_, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	require.NoError(t, err, "CancelAllOrders must not error")
	mu.Lock()
	defer mu.Unlock()
	assert.ElementsMatch(t, []string{
		"/api/v2/cancel_all_orders/btcusd/",
		"/api/v2/cancel_all_orders/btceur/",
		"/api/v2/cancel_all_orders/eurusd/",
		"/api/v2/cancel_all_orders/xrpusd/",
		"/api/v2/cancel_all_orders/xrpeur/",
		"/api/v2/cancel_all_orders/zrxeur/",
	}, cancelled, "CancelAllOrders should cancel the enabled markets and the spot markets of the open orders")
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "nope", spotPair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo should error on an invalid order ID")

	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/order_status/", r.URL.Path, "path should be the order status")
		_, err := w.Write([]byte(`{"id":1458532827766786,"datetime":"2025-06-16 10:00:00","type":"1","subtype":"TRAILING_STOP_LOSS","status":"Open","market":"BTC/USD-PERP","transactions":[],"amount_remaining":"0.01000","margin_mode":"CROSS","leverage":"3","trigger":"LAST_TRADED_PRICE","activation_price":"50000","trailing_delta":50}`))
		assert.NoError(t, err, "Write should not error")
	})
	trailing, err := ex.GetOrderInfo(t.Context(), "1458532827766786", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a trailing order")
	assert.Equal(t, &order.Detail{
		Exchange:        ex.Name,
		OrderID:         "1458532827766786",
		Type:            order.TrailingStop,
		Side:            order.Sell,
		Status:          order.Open,
		AssetType:       asset.PerpetualContract,
		Pair:            currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
		MarginType:      margin.Multi,
		Leverage:        3,
		TriggerPrice:    50000,
		Date:            time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
		LastUpdated:     time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
		Amount:          0.01,
		RemainingAmount: 0.01,
		CostAsset:       currency.USD,
		FeeAsset:        currency.USD,
		Trades:          []order.TradeHistory{},
	}, trailing, "GetOrderInfo should return the activation price of a trailing order as its trigger price")

	_, err = newResponseTestExchange(t, http.StatusNotFound, `{"status":"error","reason":"Order not found.","response_code":"404.002"}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	assert.ErrorIs(t, err, errAPIResponse, "GetOrderInfo should error when the order status cannot be fetched")
	_, err = newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Open","market":"BTCUSD","transactions":[]}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	assert.ErrorContains(t, err, "BTCUSD", "GetOrderInfo should error on a malformed market")
	_, err = newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Open","market":"BTC/USD","transactions":[{"tid":7,"price":"84000","usd":"25200","fee":"1","datetime":"2025-06-16 10:00:01","type":2}]}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	assert.ErrorIs(t, err, errMalformedTransaction, "GetOrderInfo should error on a transaction without a base amount")
	_, err = newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Open","market":"BTC/USD","transactions":[],"amount_remaining":"NaN"}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "GetOrderInfo should error on a remaining amount which is not a number")
	large, err := newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Open","market":"PEPE/USD","transactions":[{"tid":7,"price":"0.00001","pepe":"100000000.1","usd":"-1000.000001","fee":"1","datetime":"2025-06-16 10:00:01","type":2}],"amount_remaining":"10000000.1"}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	require.NoError(t, err, "GetOrderInfo must not error for a large order")
	assert.Equal(t, "110000000.2", strconv.FormatFloat(large.Amount, 'f', -1, 64), "GetOrderInfo should sum the executed and remaining amounts without a float residue")
	assert.Equal(t, "100000000.1", strconv.FormatFloat(large.ExecutedAmount, 'f', -1, 64), "GetOrderInfo should return the executed amount")
	// Float sums of these transactions carry a residue into the cost, fee and average price
	for name, tc := range map[string]struct {
		transactions []string
		exp          [4]float64
	}{
		"one transaction":   {transactions: []string{`{"tid":1,"price":"101","btc":"0.3","usd":"-30.3","fee":"0.1","datetime":"2025-06-16 10:00:01","type":2}`}, exp: [4]float64{0.3, 30.3, 0.1, 101}},
		"1000 transactions": {transactions: slices.Repeat([]string{`{"tid":1,"price":"100","btc":"0.001","usd":"-0.1","fee":"0.01","datetime":"2025-06-16 10:00:01","type":2}`}, 1000), exp: [4]float64{1, 100, 10, 100}},
	} {
		d, err := newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Finished","market":"BTC/USD","transactions":[`+strings.Join(tc.transactions, ",")+`],"amount_remaining":"0"}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
		require.NoErrorf(t, err, "GetOrderInfo must not error with %s", name)
		assert.Equalf(t, tc.exp, [4]float64{d.ExecutedAmount, d.Cost, d.Fee, d.AverageExecutedPrice}, "GetOrderInfo should sum the executed amount, cost and fee exactly and average them with %s", name)
	}
	_, err = newResponseTestExchange(t, http.StatusOK, `{"id":1,"datetime":"2025-06-16 10:00:00","type":"0","status":"Finished","market":"BTC/USD","transactions":[{"tid":7,"price":"NaN","btc":"0.3","usd":"-30.3","fee":"0.1","datetime":"2025-06-16 10:00:01","type":2}],"amount_remaining":"0"}`).GetOrderInfo(t.Context(), "1", spotPair, asset.Spot)
	assert.ErrorIs(t, err, errMalformedTransaction, "GetOrderInfo should error on a transaction price which is not a number")

	skipIfLiveWithoutCredentials(t)
	d, err := e.GetOrderInfo(t.Context(), "1458532827766784", spotPair, asset.Spot)
	if !mockTests {
		assert.ErrorIs(t, err, errAPIResponse, "GetOrderInfo should error for an unknown order")
		return
	}
	require.NoError(t, err, "GetOrderInfo must not error")
	// Spot orders have no subtype, so their type is unknown; the amount is the executed and remaining amounts
	assert.Equal(t, &order.Detail{
		Exchange:             e.Name,
		OrderID:              "1458532827766784",
		ClientOrderID:        "my-order-123",
		Side:                 order.Buy,
		Status:               order.PartiallyFilled,
		AssetType:            asset.Spot,
		Pair:                 currency.NewPairWithDelimiter("BTC", "USD", "/"),
		Date:                 time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC),
		LastUpdated:          time.Date(2022, 1, 31, 14, 46, 15, 0, time.UTC),
		Amount:               0.5,
		ExecutedAmount:       0.2,
		RemainingAmount:      0.3,
		AverageExecutedPrice: 20050,
		Cost:                 4010,
		CostAsset:            currency.USD,
		Fee:                  16.04,
		FeeAsset:             currency.USD,
		Trades: []order.TradeHistory{
			{TID: "209895701", Price: 20000, Amount: 0.1, Fee: 8, Exchange: e.Name, Side: order.Buy, Timestamp: time.Date(2022, 1, 31, 14, 45, 15, 322000000, time.UTC), FeeAsset: "USD"},
			{TID: "209895702", Price: 20100, Amount: 0.1, Fee: 8.04, Exchange: e.Name, Side: order.Buy, Timestamp: time.Date(2022, 1, 31, 14, 46, 15, 0, time.UTC), FeeAsset: "USD"},
		},
	}, d, "GetOrderInfo should return the spot order with its trades")

	d, err = e.GetOrderInfo(t.Context(), "1458532827766785", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a perpetual order")
	assert.Equal(t, &order.Detail{
		Exchange:             e.Name,
		OrderID:              "1458532827766785",
		Type:                 order.StopLimit,
		Side:                 order.Sell,
		Status:               order.PartiallyCancelled,
		AssetType:            asset.PerpetualContract,
		Pair:                 currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
		MarginType:           margin.Isolated,
		Leverage:             3.1,
		TriggerPrice:         83500,
		Date:                 time.Date(2025, 6, 16, 10, 0, 0, 123000000, time.UTC),
		LastUpdated:          time.Date(2025, 6, 16, 10, 1, 0, 0, time.UTC),
		Amount:               0.05,
		ExecutedAmount:       0.01,
		RemainingAmount:      0.04,
		AverageExecutedPrice: 84000,
		Cost:                 840,
		CostAsset:            currency.USD,
		Fee:                  0.34,
		FeeAsset:             currency.USD,
		Trades: []order.TradeHistory{
			{TID: "209895703", Price: 84000, Amount: 0.01, Fee: 0.34, Exchange: e.Name, Side: order.Sell, Timestamp: time.Date(2025, 6, 16, 10, 1, 0, 0, time.UTC), FeeAsset: "USD"},
		},
	}, d, "GetOrderInfo should return the perpetual order with its trades")
}

func TestTriggerPrice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 84500.0, triggerPrice(84500, 0), "triggerPrice should return the stop price of a stop order")
	assert.Equal(t, 50000.0, triggerPrice(0, 50000), "triggerPrice should return the activation price of a trailing order")
	assert.Equal(t, 84500.0, triggerPrice(84500, 50000), "triggerPrice should prefer the stop price")
	assert.Zero(t, triggerPrice(0, 0), "triggerPrice should be zero for an order without either price")
}

func TestOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status            string
		partiallyExecuted bool
		exp               order.Status
	}{
		{status: "Open", exp: order.Open},
		{status: "Open", partiallyExecuted: true, exp: order.PartiallyFilled},
		{status: "Finished", exp: order.Filled},
		{status: "Canceled", exp: order.Cancelled},
		{status: "Canceled", partiallyExecuted: true, exp: order.PartiallyCancelled},
		{status: "Expired", exp: order.Expired},
		{status: "Unknown", exp: order.UnknownStatus},
	} {
		assert.Equalf(t, tc.exp, orderStatus(tc.status, tc.partiallyExecuted), "orderStatus should convert %q", tc.status)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	for _, tc := range []struct {
		c     currency.Code
		chain string
		exp   *deposit.Address
	}{
		{c: currency.BTC, exp: &deposit.Address{Address: core.BitcoinDonationAddress}},
		{c: currency.ETH, chain: "ethereum", exp: &deposit.Address{Address: "0x6a56f5b80f04b4fd70d64d72e1396698635e5436", Chain: "ethereum"}},
		{c: currency.XRP, exp: &deposit.Address{Address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B", Tag: "89473951"}},
		{c: currency.XLM, exp: &deposit.Address{Address: "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A", Tag: "299576079"}},
	} {
		addr, err := e.GetDepositAddress(t.Context(), tc.c, "", tc.chain)
		require.NoErrorf(t, err, "GetDepositAddress must not error for %s", tc.c)
		if mockTests {
			assert.Equalf(t, tc.exp, addr, "GetDepositAddress should return the address for %s", tc.c)
		}
	}
}

func TestGetDepositAddressTransferID(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/cspr_address/", r.URL.Path, "path should be the Casper deposit address")
		_, err := w.Write([]byte(`{"address":"01abc","transfer_id":12345}`))
		assert.NoError(t, err, "Write should not error")
	})
	addr, err := ex.GetDepositAddress(t.Context(), currency.CSPR, "", "casper")
	require.NoError(t, err, "GetDepositAddress must not error")
	assert.Equal(t, &deposit.Address{Address: "01abc", Tag: "12345", Chain: "casper"}, addr, "GetDepositAddress should return a Casper address's transfer ID as its tag")
}

func TestGetAvailableTransferChains(t *testing.T) {
	t.Parallel()
	_, err := e.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetAvailableTransferChains should error on an empty currency")
	_, err = e.GetAvailableTransferChains(t.Context(), currency.NewCode("NOTACURRENCY"))
	assert.ErrorIs(t, err, currency.ErrCurrencyNotFound, "GetAvailableTransferChains should error on an unlisted currency")

	chains, err := e.GetAvailableTransferChains(t.Context(), currency.ETH)
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	assert.Contains(t, chains, "ethereum", "GetAvailableTransferChains should return the currency's networks")

	ex := newResponseTestExchange(t, http.StatusOK, `[{"currency":"TIA","networks":[{"network":"tia","deposit":"Disabled","withdrawal":"Enabled"},{"network":"celestia","deposit":"Enabled","withdrawal":"Disabled"},{"network":"cosmos","deposit":"Disabled","withdrawal":"Disabled"}]}]`)
	chains, err = ex.GetAvailableTransferChains(t.Context(), currency.NewCode("TIA"))
	require.NoError(t, err, "GetAvailableTransferChains must not error")
	assert.ElementsMatch(t, []string{"tia", "celestia"}, chains, "GetAvailableTransferChains should return the networks open for deposits or withdrawals")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "WithdrawCryptocurrencyFunds should validate the request")

	if !mockTests {
		t.Skip("WithdrawCryptocurrencyFunds is not tested live to avoid withdrawing funds")
	}
	for _, tc := range []struct {
		c       currency.Code
		amount  float64
		address string
		chain   string
		tag     string
		exp     string
	}{
		{c: currency.BTC, amount: 0.001, address: core.BitcoinDonationAddress, chain: "bitcoin", exp: "1"},
		{c: currency.XRP, amount: 20, address: "rvYAfWj5gh67oV6fW32ZzP3Aw4Eubs59B", tag: "12345", exp: "4"},
		{c: currency.XLM, amount: 20, address: "GAHK7EEG2WWHVKDNT4CEQFZGKF2LGDSW2IVM4S5DP42RBW3K6BTODB4A", tag: "12345", exp: "5"},
	} {
		resp, err := e.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange: e.Name,
			Type:     withdraw.Crypto,
			Currency: tc.c,
			Amount:   tc.amount,
			Crypto:   withdraw.CryptoRequest{Address: tc.address, AddressTag: tc.tag, Chain: tc.chain},
		})
		require.NoErrorf(t, err, "WithdrawCryptocurrencyFunds must not error for %s", tc.c)
		assert.Equalf(t, tc.exp, resp.ID, "ID should be correct for %s", tc.c)
	}
}

func testBankWithdrawalRequest(c currency.Code, amount float64) *withdraw.Request {
	return &withdraw.Request{
		Type:     withdraw.Fiat,
		Exchange: e.Name,
		Fiat: withdraw.FiatRequest{
			Bank: banking.Account{
				SupportedExchanges:  e.Name,
				Enabled:             true,
				AccountName:         "Satoshi Nakamoto",
				AccountNumber:       "12345",
				BankAddress:         "123 Fake St",
				BankPostalCity:      "Tarry Town",
				BankCountry:         "AU",
				BankName:            "Federal Reserve Bank",
				SWIFTCode:           "CTBAAU2S",
				BankPostalCode:      "2088",
				IBAN:                "IT60X0542811101000000123456",
				SupportedCurrencies: c.String(),
			},
			WireCurrency:               currency.USD.String(),
			IntermediaryBankAddress:    "123 Fake St",
			IntermediaryBankCity:       "Tarry Town",
			IntermediaryBankCountry:    "AU",
			IntermediaryBankName:       "Federal Reserve Bank",
			IntermediaryBankPostalCode: "2088",
		},
		Amount:      amount,
		Currency:    c,
		Description: "WITHDRAW IT ALL",
	}
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFunds(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "WithdrawFiatFunds should validate the request")

	if !mockTests {
		t.Skip("WithdrawFiatFunds is not tested live to avoid withdrawing funds")
	}
	resp, err := e.WithdrawFiatFunds(t.Context(), testBankWithdrawalRequest(currency.EUR, 10))
	require.NoError(t, err, "WithdrawFiatFunds must not error")
	assert.Equal(t, "1", resp.ID, "ID should be correct")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), &withdraw.Request{})
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "WithdrawFiatFundsToInternationalBank should validate the request")

	if !mockTests {
		t.Skip("WithdrawFiatFundsToInternationalBank is not tested live to avoid withdrawing funds")
	}
	resp, err := e.WithdrawFiatFundsToInternationalBank(t.Context(), testBankWithdrawalRequest(currency.USD, 50))
	require.NoError(t, err, "WithdrawFiatFundsToInternationalBank must not error")
	assert.Equal(t, "2", resp.ID, "ID should be correct")
}

func TestWithdrawFiatFundsToInternationalBankRequest(t *testing.T) {
	t.Parallel()
	requests := make(chan url.Values, 1)
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/withdrawal/open/", r.URL.Path, "path should be the bank withdrawal")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		select {
		case requests <- r.PostForm:
		default:
			assert.Fail(t, "WithdrawFiatFundsToInternationalBank should send only one request")
		}
		_, err := w.Write([]byte(`{"withdrawal_id":2}`))
		assert.NoError(t, err, "Write should not error")
	})
	req := testBankWithdrawalRequest(currency.USD, 50)
	req.Exchange, req.Fiat.Bank.SupportedExchanges = ex.Name, ex.Name
	req.Fiat.IntermediarySwiftCode = "CHASUS33"
	resp, err := ex.WithdrawFiatFundsToInternationalBank(t.Context(), req)
	require.NoError(t, err, "WithdrawFiatFundsToInternationalBank must not error")
	assert.Equal(t, "2", resp.ID, "ID should be correct")
	require.Len(t, requests, 1, "WithdrawFiatFundsToInternationalBank must send a request")
	assert.Equal(t, url.Values{
		"amount":                      {"50"},
		"account_currency":            {"USD"},
		"name":                        {"Satoshi Nakamoto"},
		"iban":                        {"IT60X0542811101000000123456"},
		"bic":                         {"CTBAAU2S"},
		"address":                     {"123 Fake St"},
		"postal_code":                 {"2088"},
		"city":                        {"Tarry Town"},
		"country":                     {"AU"},
		"type":                        {"international"},
		"comment":                     {"WITHDRAW IT ALL"},
		"bank_name":                   {"Federal Reserve Bank"},
		"bank_address":                {"123 Fake St"},
		"bank_postal_code":            {"2088"},
		"bank_city":                   {"Tarry Town"},
		"bank_country":                {"AU"},
		"currency":                    {"USD"},
		"intermed_routing_num_or_bic": {"CHASUS33"},
	}, <-requests, "WithdrawFiatFundsToInternationalBank should send the beneficiary, bank and intermediary bank details")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders should error on a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders should error on an unsupported asset")

	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/open_orders/", r.URL.Path, "path should be the open orders")
		_, err := w.Write([]byte(`[{"id":"1234123412341236","datetime":"2025-06-16 10:00:00","type":"1","subtype":"TRAILING_STOP_LOSS","price":"0","amount":"0.01000","amount_at_create":"0.01000","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"3.00","trigger":"LAST_TRADED_PRICE","activation_price":"50000","trailing_delta":50,"reduce_only":true}]`))
		assert.NoError(t, err, "Write should not error")
	})
	trailing, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for a trailing order")
	assert.Equal(t, order.FilteredOrders{{
		Exchange:        ex.Name,
		OrderID:         "1234123412341236",
		Type:            order.TrailingStop,
		Side:            order.Sell,
		Status:          order.Open,
		AssetType:       asset.PerpetualContract,
		Pair:            currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
		MarginType:      margin.Multi,
		Leverage:        3,
		ReduceOnly:      true,
		Date:            time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
		TriggerPrice:    50000,
		Amount:          0.01,
		RemainingAmount: 0.01,
	}}, trailing, "GetActiveOrders should return the activation price of a trailing order as its trigger price")

	spotOrders := &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}
	_, err = ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{currency.EMPTYPAIR}})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetActiveOrders should error on an empty pair")
	_, err = newResponseTestExchange(t, http.StatusForbidden, `{"status":"error","reason":"Invalid signature","code":"API0005"}`).GetActiveOrders(t.Context(), spotOrders)
	assert.ErrorIs(t, err, errAPIResponse, "GetActiveOrders should error when the open orders cannot be fetched")
	_, err = newResponseTestExchange(t, http.StatusOK, `[{"id":"1","datetime":"2025-06-16 10:00:00","type":"0","subtype":"LIMIT","price":"84000","amount":"0.40000000","market":"BTCUSD"}]`).GetActiveOrders(t.Context(), spotOrders)
	assert.ErrorContains(t, err, "BTCUSD", "GetActiveOrders should error on a malformed market")
	for _, amounts := range []string{`"amount":"NaN"`, `"amount":"NaN","amount_at_create":"0.40000000"`} {
		_, err = newResponseTestExchange(t, http.StatusOK, `[{"id":"1","datetime":"2025-06-16 10:00:00","type":"0","subtype":"LIMIT","price":"84000",`+amounts+`,"market":"BTC/USD"}]`).GetActiveOrders(t.Context(), spotOrders)
		assert.ErrorContainsf(t, err, "NaN", "GetActiveOrders should error on a remaining amount which is not a number with %s", amounts)
	}
	// As floats, 67110000.00000002 and 67109999.00000002 lose their last digit, so subtracting floats leaves 1, 0 and 1
	for _, tc := range []struct {
		atCreate, remaining string
		executed            float64
		status              order.Status
	}{
		{atCreate: "67110000.00000002", remaining: "67109999.00000001", executed: 1.00000001, status: order.PartiallyFilled},
		{atCreate: "67110000.00000002", remaining: "67110000.00000001", executed: 0.00000001, status: order.PartiallyFilled},
		{atCreate: "67110000.00000001", remaining: "67109999.00000002", executed: 0.99999999, status: order.PartiallyFilled},
		// Without its amount at creation, the order's amount is its remaining amount
		{remaining: "67110000.00000002", status: order.Open},
	} {
		amounts := `"amount":"` + tc.remaining + `"`
		if tc.atCreate != "" {
			amounts += `,"amount_at_create":"` + tc.atCreate + `"`
		}
		large, err := newResponseTestExchange(t, http.StatusOK, `[{"id":"1","datetime":"2025-06-16 10:00:00","type":"0","subtype":"LIMIT","price":"0.02263",`+amounts+`,"market":"GRT/USD"}]`).GetActiveOrders(t.Context(), spotOrders)
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", amounts)
		require.Lenf(t, large, 1, "GetActiveOrders must return the order for %s", amounts)
		assert.Equalf(t, tc.executed, large[0].ExecutedAmount, "GetActiveOrders should subtract the exact amounts for %s", amounts)
		assert.Equalf(t, tc.status, large[0].Status, "GetActiveOrders should report the order's status for %s", amounts)
	}

	ex = newResponseTestExchange(t, http.StatusOK, `[{"id":"1234123412341237","datetime":"2025-06-16 10:00:00","type":"0","subtype":"LIMIT","price":"84000","amount":"0.40000000","market":"BTC/USD"}]`)
	withoutAmountAtCreate, err := ex.GetActiveOrders(t.Context(), spotOrders)
	require.NoError(t, err, "GetActiveOrders must not error for an order without its amount at creation")
	assert.Equal(t, order.FilteredOrders{{
		Exchange:        ex.Name,
		OrderID:         "1234123412341237",
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.Open,
		AssetType:       asset.Spot,
		Pair:            currency.NewPairWithDelimiter("BTC", "USD", "/"),
		Date:            time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
		Price:           84000,
		Amount:          0.4,
		RemainingAmount: 0.4,
	}}, withoutAmountAtCreate, "GetActiveOrders should use the remaining amount as the amount of an order without its amount at creation")

	skipIfLiveWithoutCredentials(t)
	for _, req := range []*order.MultiOrderRequest{
		{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType},
		{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType, Pairs: currency.Pairs{spotPair}},
		{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType},
	} {
		orders, err := e.GetActiveOrders(t.Context(), req)
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s %s", req.AssetType, req.Pairs)
		if !mockTests {
			continue
		}
		require.Lenf(t, orders, 1, "GetActiveOrders must only return orders for %s", req.AssetType)
		o := orders[0]
		assert.Equal(t, req.AssetType, o.AssetType, "AssetType should be correct")
		if req.AssetType == asset.Spot {
			assert.Equal(t, order.Detail{
				Exchange:        e.Name,
				OrderID:         "1234123412341234",
				ClientOrderID:   "my-order-123",
				Type:            order.Limit,
				Side:            order.Buy,
				Status:          order.PartiallyFilled,
				AssetType:       asset.Spot,
				Pair:            currency.NewPairWithDelimiter("BTC", "USD", "/"),
				Date:            time.Date(2022, 1, 31, 14, 43, 15, 0, time.UTC),
				Price:           100,
				Amount:          0.5,
				ExecutedAmount:  0.1,
				RemainingAmount: 0.4,
			}, o, "spot order should be correct")
			continue
		}
		assert.Equal(t, order.Detail{
			Exchange:        e.Name,
			OrderID:         "1234123412341235",
			Type:            order.StopLimit,
			Side:            order.Sell,
			Status:          order.Open,
			AssetType:       asset.PerpetualContract,
			Pair:            currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
			MarginType:      margin.Multi,
			Leverage:        3,
			ReduceOnly:      true,
			Date:            time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
			Price:           85000,
			TriggerPrice:    84500,
			Amount:          0.01,
			RemainingAmount: 0.01,
		}, o, "perpetual order should be correct")
	}
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	assert.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory should error on a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Futures, Side: order.AnySide, Type: order.AnyType})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory should error on an unsupported asset")

	skipIfLiveWithoutCredentials(t)
	ctx := recordedPagerContext(t)
	for _, tc := range []struct {
		req *order.MultiOrderRequest
		exp []string
	}{
		{req: &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType}, exp: []string{"1458532827766784", "1458532827766790"}},
		{req: &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.Sell, Type: order.AnyType}, exp: []string{"1458532827766790"}},
		{req: &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType}, exp: []string{"1234123412341241", "1234123412341235"}},
	} {
		orders, err := e.GetOrderHistory(ctx, tc.req)
		require.NoErrorf(t, err, "GetOrderHistory must not error for %s %s orders", tc.req.AssetType, tc.req.Side)
		if !mockTests {
			continue
		}
		orderIDs := make([]string, len(orders))
		for i := range orders {
			orderIDs[i] = orders[i].OrderID
			assert.Equalf(t, tc.req.AssetType, orders[i].AssetType, "AssetType should be correct for %s", tc.req.AssetType)
		}
		assert.Equalf(t, tc.exp, orderIDs, "GetOrderHistory should return the filtered %s %s orders", tc.req.AssetType, tc.req.Side)
	}
}

func TestSpotOrderHistory(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t, func(http.ResponseWriter, *http.Request) {
		assert.Fail(t, "spotOrderHistory should not request a range which ends before the transactions' time filters reach")
	})
	none, err := ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, EndTime: time.Now().Add(-45 * 24 * time.Hour)})
	require.NoError(t, err, "spotOrderHistory must not error for a range which ends more than 30 days ago")
	assert.Empty(t, none, "spotOrderHistory should return no orders for a range which ends more than 30 days ago")
	_, err = ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{currency.EMPTYPAIR}})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "spotOrderHistory should error on an empty pair")
	_, err = newResponseTestExchange(t, http.StatusForbidden, `{"status":"error","reason":"Invalid signature","code":"API0005"}`).spotOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errAPIResponse, "spotOrderHistory should error when the transactions cannot be fetched")
	ex = newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v2/user_transactions/btcusd/", r.URL.Path, "a single pair's transactions should be requested for its market")
		_, err := w.Write([]byte(`[{"id":1,"datetime":"2025-06-16 10:00:00","type":"2","fee":"0.1","btc":"0.01","usd":"-840.0","order_id":2}]`))
		assert.NoError(t, err, "Write should not error")
	})
	withoutRate, err := ex.spotOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{spotPair}})
	require.NoError(t, err, "spotOrderHistory must not error for a single pair")
	assert.Empty(t, withoutRate, "spotOrderHistory should skip a trade without an exchange rate, which names no market")

	skipIfLiveWithoutCredentials(t)
	orders, err := e.spotOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot})
	require.NoError(t, err, "spotOrderHistory must not error")
	if !mockTests {
		return
	}
	// The side is derived from the sign of the base amount and the pair from the transaction's exchange rate
	assert.Equal(t, []order.Detail{
		{
			Exchange:             e.Name,
			OrderID:              "1458532827766784",
			Side:                 order.Buy,
			AssetType:            asset.Spot,
			Pair:                 currency.NewPairWithDelimiter("BTC", "USD", "/"),
			Date:                 time.Date(2022, 3, 1, 10, 54, 53, 849000000, time.UTC),
			LastUpdated:          time.Date(2022, 3, 1, 10, 55, 53, 0, time.UTC),
			Price:                20050,
			Amount:               0.2,
			ExecutedAmount:       0.2,
			AverageExecutedPrice: 20050,
			Cost:                 4010,
			CostAsset:            currency.USD,
			Fee:                  16.04,
			FeeAsset:             currency.USD,
			Trades: []order.TradeHistory{
				{TID: "50197020", Price: 20000, Amount: 0.1, Fee: 8, Exchange: e.Name, Side: order.Buy, Timestamp: time.Date(2022, 3, 1, 10, 54, 53, 849000000, time.UTC), FeeAsset: "USD"},
				{TID: "50197021", Price: 20100, Amount: 0.1, Fee: 8.04, Exchange: e.Name, Side: order.Buy, Timestamp: time.Date(2022, 3, 1, 10, 55, 53, 0, time.UTC), FeeAsset: "USD"},
			},
		},
		{
			Exchange:             e.Name,
			OrderID:              "1458532827766790",
			Side:                 order.Sell,
			AssetType:            asset.Spot,
			Pair:                 currency.NewPairWithDelimiter("ETH", "USD", "/"),
			Date:                 time.Date(2022, 3, 2, 11, 0, 0, 0, time.UTC),
			LastUpdated:          time.Date(2022, 3, 2, 11, 0, 0, 0, time.UTC),
			Price:                2000,
			Amount:               0.05,
			ExecutedAmount:       0.05,
			AverageExecutedPrice: 2000,
			Cost:                 100,
			CostAsset:            currency.USD,
			Fee:                  0.25,
			FeeAsset:             currency.USD,
			Trades: []order.TradeHistory{
				{TID: "50197022", Price: 2000, Amount: 0.05, Fee: 0.25, Exchange: e.Name, Side: order.Sell, Timestamp: time.Date(2022, 3, 2, 11, 0, 0, 0, time.UTC), FeeAsset: "USD"},
			},
		},
	}, orders, "spotOrderHistory should aggregate the trades into orders and skip other transactions")
}

func TestDerivativesOrderHistory(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`[{"trade_id":"1","order_id":"2","datetime":"2025-06-16 10:01:00","side":"WIBBLE","price":"84000","amount":"0.01"}]`))
		assert.NoError(t, err, "Write should not error")
	})
	_, err := ex.derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "derivativesOrderHistory should error on an invalid side")
	for liquidationFee, expErr := range map[string]error{"0.2": nil, "NaN": errMalformedTransaction} {
		ex = newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(fmt.Appendf(nil, `[{"trade_id":"1","order_id":"2","datetime":"2025-06-16 10:01:00","side":"buy","price":"100","amount":"1","fee":"0.1","liquidation_fee":%q,"fee_currency":"USD","market":"BTC/USD-PERP"}]`, liquidationFee))
			assert.NoError(t, err, "Write should not error")
		})
		orders, err := ex.derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
		if expErr != nil {
			assert.ErrorIs(t, err, expErr, "derivativesOrderHistory should error on a liquidation fee which is not a number")
			continue
		}
		require.NoError(t, err, "derivativesOrderHistory must not error with a liquidation fee")
		require.Len(t, orders, 1, "derivativesOrderHistory must return the trade's order")
		require.Len(t, orders[0].Trades, 1, "the order must hold its trade")
		// A float sum of 0.1 and 0.2 is 0.30000000000000004
		assert.Equal(t, 0.3, orders[0].Trades[0].Fee, "the trade's fee should include its liquidation fee without a float residue")
		assert.Equal(t, 0.3, orders[0].Fee, "the order's fee should include the liquidation fee without a float residue")
	}

	_, err = ex.derivativesOrderHistory(t.Context(), currency.Pairs{currency.EMPTYPAIR}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "derivativesOrderHistory should error on an empty pair")
	_, err = newResponseTestExchange(t, http.StatusForbidden, `{"status":"error","reason":"Invalid signature","code":"API0005"}`).derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errAPIResponse, "derivativesOrderHistory should error when the trades cannot be fetched")
	_, err = newResponseTestExchange(t, http.StatusOK, `[{"trade_id":"1","order_id":"2","datetime":"2025-06-16 10:01:00","side":"buy","price":"100","amount":"1","fee":"0.1","fee_currency":"USD","market":"BTCUSD-PERP"}]`).derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	assert.ErrorContains(t, err, "BTCUSD-PERP", "derivativesOrderHistory should error on a malformed market")
	_, err = newResponseTestExchange(t, http.StatusOK, `[{"trade_id":"1","order_id":"2","datetime":"2025-06-16 10:01:00","side":"buy","price":"100","amount":"NaN","fee":"0.1","fee_currency":"USD","market":"BTC/USD-PERP"}]`).derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, errMalformedTransaction, "derivativesOrderHistory should error on an amount which is not a number")

	skipIfLiveWithoutCredentials(t)
	orders, err := e.derivativesOrderHistory(recordedPagerContext(t), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	require.NoError(t, err, "derivativesOrderHistory must not error")
	if !mockTests {
		return
	}
	perpetual := currency.NewPairWithDelimiter("BTC", "USD-PERP", "/")
	assert.Equal(t, []order.Detail{
		{
			Exchange:             e.Name,
			OrderID:              "1234123412341241",
			Side:                 order.Buy,
			AssetType:            asset.PerpetualContract,
			Pair:                 perpetual,
			MarginType:           margin.Isolated,
			Leverage:             3,
			Date:                 time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
			LastUpdated:          time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
			Price:                20000,
			Amount:               0.01,
			ExecutedAmount:       0.01,
			AverageExecutedPrice: 20000,
			Cost:                 200,
			CostAsset:            currency.USD,
			Fee:                  0.08,
			FeeAsset:             currency.USD,
			Trades: []order.TradeHistory{
				{TID: "1234123412341302", Price: 20000, Amount: 0.01, Fee: 0.08, Exchange: e.Name, Side: order.Buy, Timestamp: time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC), FeeAsset: "USD"},
			},
		},
		{
			Exchange:             e.Name,
			OrderID:              "1234123412341235",
			Side:                 order.Sell,
			AssetType:            asset.PerpetualContract,
			Pair:                 perpetual,
			MarginType:           margin.Isolated,
			Leverage:             3,
			Date:                 time.Date(2025, 6, 16, 10, 1, 0, 849000000, time.UTC),
			LastUpdated:          time.Date(2025, 6, 16, 10, 1, 1, 0, time.UTC),
			Price:                84006,
			Amount:               0.01,
			ExecutedAmount:       0.01,
			AverageExecutedPrice: 84006,
			Cost:                 840.06,
			CostAsset:            currency.USD,
			Fee:                  0.33602,
			FeeAsset:             currency.USD,
			Trades: []order.TradeHistory{
				{TID: "1234123412341300", Price: 84000, Amount: 0.004, Fee: 0.1344, Exchange: e.Name, Side: order.Sell, Timestamp: time.Date(2025, 6, 16, 10, 1, 0, 849000000, time.UTC), FeeAsset: "USD"},
				{TID: "1234123412341301", Price: 84010, Amount: 0.006, Fee: 0.20162, Exchange: e.Name, Side: order.Sell, Timestamp: time.Date(2025, 6, 16, 10, 1, 1, 0, time.UTC), FeeAsset: "USD"},
			},
		},
	}, orders, "derivativesOrderHistory should aggregate the trades into orders sorted by date")
}

func TestTradedPair(t *testing.T) {
	t.Parallel()
	pair, rate, ok := tradedPair(&UserTransactionResponse{
		Amounts:       map[currency.Code]float64{currency.BTC: 0.1, currency.USD: -2000},
		ExchangeRates: map[currency.Pair]float64{currency.NewBTCUSD(): 20000},
	})
	require.True(t, ok, "tradedPair must find the traded pair")
	assert.True(t, pair.Equal(spotPair), "tradedPair should return the pair")
	assert.Equal(t, 20000.0, rate, "tradedPair should return the exchange rate")

	for name, tx := range map[string]*UserTransactionResponse{
		"no exchange rate": {Amounts: map[currency.Code]float64{currency.USD: 1000}},
		"zero rate":        {Amounts: map[currency.Code]float64{currency.BTC: 0.1}, ExchangeRates: map[currency.Pair]float64{currency.NewBTCUSD(): 0}},
		"no base amount":   {Amounts: map[currency.Code]float64{currency.USD: 1000}, ExchangeRates: map[currency.Pair]float64{currency.NewBTCUSD(): 20000}},
	} {
		_, _, ok := tradedPair(tx)
		assert.Falsef(t, ok, "tradedPair should not find a traded pair with %s", name)
	}
}

func TestAddTrade(t *testing.T) {
	t.Parallel()
	first := order.TradeHistory{TID: "1", Price: 100, Amount: 2, Fee: 0.5, Timestamp: time.Unix(300, 0)}
	second := order.TradeHistory{TID: "2", Price: 110, Amount: 1, Fee: 0.25, Timestamp: time.Unix(100, 0)}
	d := &order.Detail{Date: time.Unix(200, 0)}
	addTrade(d, &first)
	addTrade(d, &second)
	assert.Equal(t, &order.Detail{
		Date:        time.Unix(100, 0),
		LastUpdated: time.Unix(300, 0),
		Trades:      []order.TradeHistory{first, second},
	}, d, "addTrade should add the trades and widen the order's dates")
}

func TestAggregatedOrders(t *testing.T) {
	t.Parallel()
	orders, err := aggregatedOrders(nil)
	require.NoError(t, err, "aggregatedOrders must not error without orders")
	assert.Empty(t, orders, "aggregatedOrders should return no orders without orders")
	twoTrades := []order.TradeHistory{{TID: "1", Price: 100, Amount: 2, Fee: 0.5}, {TID: "2", Price: 110, Amount: 1, Fee: 0.25}}
	orders, err = aggregatedOrders(map[uint64]*order.Detail{
		1: {OrderID: "1", Date: time.Unix(200, 0), Trades: twoTrades},
		2: {OrderID: "2", Date: time.Unix(100, 0)},
	})
	require.NoError(t, err, "aggregatedOrders must not error")
	assert.Equal(t, []order.Detail{
		{OrderID: "2", Date: time.Unix(100, 0)},
		{OrderID: "1", Date: time.Unix(200, 0), Amount: 3, ExecutedAmount: 3, Cost: 310, Fee: 0.75, AverageExecutedPrice: 310.0 / 3, Price: 310.0 / 3, Trades: twoTrades},
	}, orders, "aggregatedOrders should sort the orders by date and sum their trades")

	// Float sums of these trades carry a residue into the totals and the average price
	manyTrades := make([]order.TradeHistory, 1000)
	for i := range manyTrades {
		manyTrades[i] = order.TradeHistory{TID: strconv.Itoa(i), Price: 100, Amount: 0.001, Fee: 0.01}
	}
	for _, tc := range []struct {
		name   string
		trades []order.TradeHistory
		exp    order.Detail
	}{
		{name: "1000 small trades", trades: manyTrades, exp: order.Detail{Amount: 1, ExecutedAmount: 1, Cost: 100, Fee: 10, AverageExecutedPrice: 100, Price: 100}},
		{name: "a small amount", trades: []order.TradeHistory{{TID: "1", Price: 7, Amount: 0.00000003}}, exp: order.Detail{Amount: 0.00000003, ExecutedAmount: 0.00000003, Cost: 0.00000021, AverageExecutedPrice: 7, Price: 7}},
		{name: "an inexact average", trades: []order.TradeHistory{{TID: "1", Price: 0.00000003, Amount: 0.2}, {TID: "2", Price: 101, Amount: 0.00000001}}, exp: order.Detail{Amount: 0.20000001, ExecutedAmount: 0.20000001, Cost: 0.000001016, AverageExecutedPrice: 5.079999746000013e-06, Price: 5.079999746000013e-06}},
	} {
		got, err := aggregatedOrders(map[uint64]*order.Detail{1: {Trades: tc.trades}})
		require.NoErrorf(t, err, "aggregatedOrders must not error with %s", tc.name)
		require.Lenf(t, got, 1, "aggregatedOrders must return one order with %s", tc.name)
		got[0].Trades = nil
		assert.Equalf(t, tc.exp, got[0], "aggregatedOrders should sum %s exactly", tc.name)
	}
	_, err = aggregatedOrders(map[uint64]*order.Detail{1: {Trades: []order.TradeHistory{{TID: "7", Price: 1, Amount: math.NaN()}}}})
	assert.ErrorIs(t, err, errMalformedTransaction, "aggregatedOrders should error on a trade amount which is not a number")

	sameDate := make(map[uint64]*order.Detail)
	var expIDs []string
	for id := uint64(1); id <= 12; id++ {
		sameDate[id] = &order.Detail{OrderID: strconv.FormatUint(id, 10), Date: time.Unix(100, 0)}
		expIDs = append(expIDs, strconv.FormatUint(id, 10))
	}
	orders, err = aggregatedOrders(sameDate)
	require.NoError(t, err, "aggregatedOrders must not error with equal dates")
	orderIDs := make([]string, len(orders))
	for i := range orders {
		orderIDs[i] = orders[i].OrderID
	}
	assert.Equal(t, expIDs, orderIDs, "aggregatedOrders should sort orders with equal dates by order ID")
}

func TestWithinTimeRange(t *testing.T) {
	t.Parallel()
	orders := []order.Detail{
		{OrderID: "1", Date: time.Unix(50, 0), LastUpdated: time.Unix(99, 0)},
		{OrderID: "2", Date: time.Unix(50, 0), LastUpdated: time.Unix(150, 0)},
		{OrderID: "3", Date: time.Unix(200, 0), LastUpdated: time.Unix(250, 0)},
		{OrderID: "4", Date: time.Unix(201, 0), LastUpdated: time.Unix(260, 0)},
	}
	for _, tc := range []struct {
		start, end time.Time
		exp        []string
	}{
		{exp: []string{"1", "2", "3", "4"}},
		{start: time.Unix(100, 0), exp: []string{"2", "3", "4"}},
		{end: time.Unix(200, 0), exp: []string{"1", "2", "3"}},
		{start: time.Unix(100, 0), end: time.Unix(200, 0), exp: []string{"2", "3"}},
	} {
		filtered := withinTimeRange(slices.Clone(orders), tc.start, tc.end)
		ids := make([]string, len(filtered))
		for i := range filtered {
			ids[i] = filtered[i].OrderID
		}
		assert.Equalf(t, tc.exp, ids, "withinTimeRange should return the orders active between %s and %s", tc.start, tc.end)
	}
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)
	for a, pair := range map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair} {
		for name, get := range map[string]func() (*kline.Item, error){
			"GetHistoricCandles": func() (*kline.Item, error) {
				return e.GetHistoricCandles(t.Context(), pair, a, kline.OneDay, start, end)
			},
			"GetHistoricCandlesExtended": func() (*kline.Item, error) {
				return e.GetHistoricCandlesExtended(t.Context(), pair, a, kline.OneDay, start, end)
			},
		} {
			candles, err := get()
			require.NoErrorf(t, err, "%s must not error for %s", name, pair)
			if mockTests {
				require.Lenf(t, candles.Candles, 29, "%s must return a candle per day for %s", name, pair)
				exp := kline.Item{
					Exchange: e.Name,
					Pair:     pair,
					Asset:    a,
					Interval: kline.OneDay,
					Candles: map[asset.Item][]kline.Candle{
						asset.Spot:              {{Time: time.Unix(1756684800, 0), Open: 108268, High: 109907, Low: 107270, Close: 109244, Volume: 1584.06180608}},
						asset.PerpetualContract: {{Time: time.Unix(1756684800, 0), Open: 108095, High: 109852, Low: 107378, Close: 109226, Volume: 9.93507}},
					}[a],
				}
				first := *candles
				first.Candles = first.Candles[:1]
				assert.Equalf(t, exp, first, "%s should return the first candle of the range for %s", name, pair)
			}
			assert.Truef(t, candles.Pair.Equal(pair), "%s Pair should be correct for %s", name, pair)
			require.NotEmptyf(t, candles.Candles, "%s must return candles for %s", name, pair)
			for _, c := range candles.Candles {
				assert.Falsef(t, c.Time.Before(start) || c.Time.After(end), "%s candles should be within the requested range for %s", name, pair)
				assert.Positivef(t, c.Close, "%s Close should be positive for %s", name, pair)
			}
		}
	}
}

func TestCandlesInRange(t *testing.T) {
	t.Parallel()
	start := time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC)
	_, err := e.candlesInRange(t.Context(), spotPair, kline.OneDay, start, start)
	assert.ErrorIs(t, err, errInvalidLimit, "candlesInRange should error on an empty range")

	candles, err := e.candlesInRange(t.Context(), spotPair, kline.OneDay, start, time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err, "candlesInRange must not error")
	require.Len(t, candles, 29, "candlesInRange must return a candle per day of the range")
	if mockTests {
		exp := kline.Candle{Time: time.Unix(1756684800, 0), Open: 108268, High: 109907, Low: 107270, Close: 109244, Volume: 1584.06180608}
		assert.Equal(t, exp, candles[0], "candlesInRange should return the first candle of the range")
	}
	for i := range candles {
		assert.Truef(t, candles[i].Time.Equal(start.AddDate(0, 0, i)), "candle %d should start on day %d of the range", i, i)
		assert.Positivef(t, candles[i].Close, "candle %d Close should be positive", i)
	}
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := e.GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetServerTime should not be supported")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContractDetails(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, futures.ErrNotFuturesAsset, "GetFuturesContractDetails should error on a spot asset")
	_, err = e.GetFuturesContractDetails(t.Context(), asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetFuturesContractDetails should error on an unsupported asset")

	contracts, err := e.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	for i := range contracts {
		assert.Equal(t, futures.Perpetual, contracts[i].Type, "Type should be correct")
		assert.Equal(t, asset.PerpetualContract, contracts[i].Asset, "Asset should be correct")
	}
	i := slices.IndexFunc(contracts, func(c futures.Contract) bool { return c.Name.Equal(perpetualPair) })
	require.NotEqual(t, -1, i, "GetFuturesContractDetails must return the BTC/USD-PERP contract")
	c := contracts[i]
	if !mockTests {
		assert.True(t, c.Underlying.Equal(spotPair), "Underlying should be correct")
		assert.Positive(t, c.MaxLeverage, "MaxLeverage should be positive")
		return
	}
	exp := futures.Contract{
		Exchange:           e.Name,
		Name:               currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
		Underlying:         spotPair,
		Asset:              asset.PerpetualContract,
		IsActive:           true,
		Status:             "Enabled",
		Type:               futures.Perpetual,
		SettlementType:     futures.Linear,
		SettlementCurrency: currency.USD,
		MarginCurrency:     currency.USD,
		Multiplier:         1,
		MaxLeverage:        10,
	}
	assert.Equal(t, exp, c, "GetFuturesContractDetails should return the contract")
}

func TestGetFuturesContractDetailsTradingStatus(t *testing.T) {
	t.Parallel()
	ex := newResponseTestExchange(t, http.StatusOK, `[{"name":"BTC/USD-PERP","market_type":"PERPETUAL","trading":"Disabled"},{"name":"ETH/USD-PERP","market_type":"PERPETUAL","trading":"Enabled"}]`)
	contracts, err := ex.GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "GetFuturesContractDetails must not error")
	require.Len(t, contracts, 2, "GetFuturesContractDetails must return every perpetual market")
	assert.False(t, contracts[0].IsActive, "a market whose trading is disabled should not be active")
	assert.Equal(t, "Disabled", contracts[0].Status, "a market whose trading is disabled should have the API's status")
	assert.True(t, contracts[1].IsActive, "a market whose trading is enabled should be active")
	assert.Equal(t, "Enabled", contracts[1].Status, "a market whose trading is enabled should have the API's status")
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates should error on a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot, Pair: spotPair})
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLatestFundingRates should error on a spot asset")

	for _, req := range []*fundingrate.LatestRateRequest{
		{Asset: asset.PerpetualContract, Pair: perpetualPair, IncludePredictedRate: true},
		{Asset: asset.PerpetualContract},
	} {
		rates, err := e.GetLatestFundingRates(t.Context(), req)
		require.NoError(t, err, "GetLatestFundingRates must not error")
		require.NotEmpty(t, rates, "GetLatestFundingRates must return rates")
		if mockTests {
			// The latest rate is the last settled rate rather than the current rate, which is the predicted rate
			exp := fundingrate.LatestRateResponse{
				Exchange:       e.Name,
				Asset:          asset.PerpetualContract,
				Pair:           currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
				LatestRate:     fundingrate.Rate{Time: time.Unix(1790294400, 0), Rate: decimal.MustFromFloat(0)},
				TimeOfNextRate: time.Unix(1790323200, 0),
			}
			if req.IncludePredictedRate {
				exp.PredictedUpcomingRate = fundingrate.Rate{Time: time.Unix(1790323200, 0), Rate: decimal.MustFromFloat(0.000002)}
			}
			require.Len(t, rates, 1, "GetLatestFundingRates must return the enabled pair's rate")
			assert.NotZero(t, rates[0].TimeChecked, "TimeChecked should be set")
			rates[0].TimeChecked = time.Time{}
			assert.Equal(t, []fundingrate.LatestRateResponse{exp}, rates, "GetLatestFundingRates should return the latest rate")
			continue
		}
		r := rates[0]
		assert.True(t, r.Pair.Equal(perpetualPair), "Pair should be correct")
		assert.Equal(t, asset.PerpetualContract, r.Asset, "Asset should be correct")
		assert.NotZero(t, r.LatestRate.Time, "LatestRate Time should be set")
		assert.True(t, r.TimeOfNextRate.After(r.LatestRate.Time), "TimeOfNextRate should be after the latest rate")
		if !req.IncludePredictedRate {
			assert.Zero(t, r.PredictedUpcomingRate, "PredictedUpcomingRate should not be set")
			continue
		}
		assert.Equal(t, r.TimeOfNextRate, r.PredictedUpcomingRate.Time, "PredictedUpcomingRate should apply at the next funding time")
	}
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *fundingrate.HistoricalRatesRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &fundingrate.HistoricalRatesRequest{Asset: asset.Spot, Pair: spotPair}, err: futures.ErrNotPerpetualFuture},
		{req: &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract}, err: currency.ErrCurrencyPairEmpty},
		{req: &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, IncludePayments: true}, err: common.ErrFunctionNotSupported},
		{req: &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: time.Now(), EndDate: time.Now().Add(-time.Hour)}, err: common.ErrStartAfterEnd},
	} {
		_, err := e.GetHistoricalFundingRates(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetHistoricalFundingRates should return the correct error")
	}

	rates, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	require.NotEmpty(t, rates.FundingRates, "GetHistoricalFundingRates must return rates")
	if mockTests {
		require.Len(t, rates.FundingRates, 100, "GetHistoricalFundingRates must return every rate")
		exp := fundingrate.HistoricalRates{
			Exchange:   e.Name,
			Asset:      asset.PerpetualContract,
			Pair:       currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
			StartDate:  time.Unix(1787443200, 0),
			EndDate:    time.Unix(1790294400, 0),
			LatestRate: fundingrate.Rate{Time: time.Unix(1790294400, 0), Rate: decimal.MustFromFloat(0)},
			FundingRates: []fundingrate.Rate{
				{Time: time.Unix(1787443200, 0), Rate: decimal.MustFromFloat(0.00036)},
				{Time: time.Unix(1787472000, 0), Rate: decimal.MustFromFloat(0.000524)},
			},
		}
		first := *rates
		first.FundingRates = first.FundingRates[:2]
		assert.Equal(t, exp, first, "GetHistoricalFundingRates should return the rates oldest first")
	}
	assert.True(t, rates.Pair.Equal(perpetualPair), "Pair should be correct")
	assert.Equal(t, rates.FundingRates[0].Time, rates.StartDate, "StartDate should be the first rate")
	assert.Equal(t, rates.FundingRates[len(rates.FundingRates)-1], rates.LatestRate, "LatestRate should be the last rate")
	for i := 1; i < len(rates.FundingRates); i++ {
		assert.Equal(t, 8*time.Hour, rates.FundingRates[i].Time.Sub(rates.FundingRates[i-1].Time), "rates should be eight hours apart")
	}
}

func TestGetHistoricalFundingRatesRange(t *testing.T) {
	t.Parallel()
	// The test venue settles a rate every 8 hours for 433 days and, as the API does, returns the latest 100 rates of a
	// range whose bounds are both inclusive, oldest first, and refuses a range which holds no rate
	const first, settlements = int64(1756684800), 1300
	settled := func(i int64) time.Time { return time.Unix(first+i*28800, 0) }
	rate := func(i int64) float64 { return float64(i) / 1e6 }
	unix := func(t time.Time) string { return strconv.FormatInt(t.Unix(), 10) }
	for _, tc := range []struct {
		name       string
		start, end time.Time
		ignoreEnd  bool
		predicted  bool
		from, to   int64
		pageEnds   []string
	}{
		{name: "three pages", start: settled(1000), end: settled(1200), from: 1000, to: 1200, pageEnds: []string{unix(settled(1200)), unix(settled(1101)), unix(settled(1002))}},
		{name: "start after a settlement", start: settled(1000).Add(time.Second), end: settled(1199), from: 1001, to: 1199, pageEnds: []string{unix(settled(1199)), unix(settled(1100)), unix(settled(1001))}},
		// A full page whose oldest rate is the first one after the start leaves a range without a rate before it
		{name: "start before a full page", start: settled(1001).Add(-time.Hour), end: settled(1200), from: 1001, to: 1200, pageEnds: []string{unix(settled(1200)), unix(settled(1101)), unix(settled(1002))}},
		{name: "one page from the start", start: settled(1101), end: settled(1200), from: 1101, to: 1200, pageEnds: []string{unix(settled(1200))}},
		{name: "no end", start: settled(1250), from: 1250, to: settlements - 1, pageEnds: []string{""}},
		{name: "no start", from: settlements - 100, to: settlements - 1, pageEnds: []string{""}},
		{name: "no start with an end", end: settled(1200), from: 1101, to: 1200, pageEnds: []string{unix(settled(1200))}},
		// The API takes whole seconds, so it returns the rate settled in the second the start falls in
		{name: "fractional start", start: settled(1250).Add(500 * time.Millisecond), from: 1251, to: settlements - 1, pageEnds: []string{""}},
		{name: "fractional start with an end", start: settled(1000).Add(500 * time.Millisecond), end: settled(1200), from: 1001, to: 1200, pageEnds: []string{unix(settled(1200)), unix(settled(1101)), unix(settled(1002))}},
		{name: "end ignored by the venue", start: settled(1000), end: settled(1250), ignoreEnd: true, from: settlements - 100, to: 1250, pageEnds: []string{unix(settled(1250)), unix(settled(settlements - 100))}},
		{name: "end ignored by the venue without a start", end: settled(1250).Add(-time.Second), ignoreEnd: true, from: settlements - 100, to: 1249, pageEnds: []string{unix(settled(1250).Add(-time.Second))}},
		{name: "predicted rate", start: settled(1250), predicted: true, from: 1250, to: settlements - 1, pageEnds: []string{""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var mu sync.Mutex
			var requests []url.Values
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/api/v2/funding_rate/btcusd-perp/" {
					_, err := fmt.Fprintf(w, `{"funding_rate":"0.0001","timestamp":"%d","market":"BTC/USD-PERP","next_funding_time":"%d"}`, settled(settlements-1).Unix()+600, settled(settlements).Unix())
					assert.NoError(t, err, "Write should not error")
					return
				}
				if !assert.Equal(t, "/api/v2/funding_rate_history/btcusd-perp/", r.URL.Path, "path should be the funding rate history") {
					http.Error(w, "unexpected path", http.StatusNotFound)
					return
				}
				q := r.URL.Query()
				mu.Lock()
				// A repeated range would be served again for ever, so it fails the request rather than hanging the test
				repeated := slices.ContainsFunc(requests, func(v url.Values) bool { return v.Encode() == q.Encode() })
				requests = append(requests, q)
				mu.Unlock()
				if !assert.False(t, repeated, "a range should not be requested twice") {
					http.Error(w, "repeated range", http.StatusBadRequest)
					return
				}
				limit, err := strconv.ParseInt(q.Get("limit"), 10, 64)
				if !assert.NoError(t, err, "limit should be a number") || limit > 100 {
					http.Error(w, "invalid limit", http.StatusBadRequest)
					return
				}
				since, until := int64(math.MinInt64), int64(math.MaxInt64)
				if v := q.Get("since_timestamp"); v != "" {
					since, err = strconv.ParseInt(v, 10, 64)
					assert.NoError(t, err, "since_timestamp should be a number")
				}
				if v := q.Get("until_timestamp"); v != "" && !tc.ignoreEnd {
					until, err = strconv.ParseInt(v, 10, 64)
					assert.NoError(t, err, "until_timestamp should be a number")
				}
				var page []string
				for i := int64(settlements - 1); i >= 0 && int64(len(page)) < limit; i-- {
					if ts := settled(i).Unix(); ts >= since && ts <= until {
						page = append(page, fmt.Sprintf(`{"funding_rate":%q,"timestamp":"%d"}`, strconv.FormatFloat(rate(i), 'f', -1, 64), ts))
					}
				}
				if len(page) == 0 {
					http.Error(w, `{"message": "Cannot fetch funding rate history for market BTC/USD-PERP"}`, http.StatusBadRequest)
					return
				}
				slices.Reverse(page)
				_, err = fmt.Fprintf(w, `{"market":"BTC/USD-PERP","funding_rate_history":[%s]}`, strings.Join(page, ","))
				assert.NoError(t, err, "Write should not error")
			})
			resp, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
				Asset:                asset.PerpetualContract,
				Pair:                 perpetualPair,
				StartDate:            tc.start,
				EndDate:              tc.end,
				IncludePredictedRate: tc.predicted,
			})
			require.NoError(t, err, "GetHistoricalFundingRates must not error")
			exp := &fundingrate.HistoricalRates{
				Exchange:  ex.Name,
				Asset:     asset.PerpetualContract,
				Pair:      currency.NewPair(currency.NewCode("btc"), currency.NewCode("usd-perp")),
				StartDate: settled(tc.from),
				EndDate:   settled(tc.to),
			}
			for i := tc.from; i <= tc.to; i++ {
				exp.FundingRates = append(exp.FundingRates, fundingrate.Rate{Time: settled(i), Rate: decimal.MustFromFloat(rate(i))})
			}
			exp.LatestRate = exp.FundingRates[len(exp.FundingRates)-1]
			if tc.predicted {
				exp.TimeOfNextRate = settled(settlements)
				exp.PredictedUpcomingRate = fundingrate.Rate{Time: settled(settlements), Rate: decimal.MustFromFloat(0.0001)}
			}
			assert.Equal(t, exp, resp, "GetHistoricalFundingRates should return every rate in the range, oldest first")
			mu.Lock()
			defer mu.Unlock()
			since := ""
			if !tc.start.IsZero() {
				since = unix(tc.start)
			}
			pageEnds := make([]string, len(requests))
			for i, q := range requests {
				pageEnds[i] = q.Get("until_timestamp")
				assert.Equalf(t, "100", q.Get("limit"), "request %d should ask for the most rates a page holds", i)
				assert.Equalf(t, since, q.Get("since_timestamp"), "request %d should start at the start date", i)
			}
			assert.Equal(t, tc.pageEnds, pageEnds, "each page should end at the oldest rate of the page before")
		})
	}
}

func TestGetHistoricalFundingRatesNoneInRange(t *testing.T) {
	t.Parallel()
	const refusal = `{"message": "Cannot fetch funding rate history for market BTC/USD-PERP"}`
	fullPage := make([]string, maxFundingRateHistoryLimit)
	for i := range fullPage {
		fullPage[i] = fmt.Sprintf(`{"funding_rate":"0.0001","timestamp":"%d"}`, 1790294400+int64(i)*28800)
	}
	ethPerpetualPair := currency.NewPair(currency.ETH, currency.NewCode("USD-PERP"))
	type reply struct {
		status int
		body   string
	}
	for _, tc := range []struct {
		name       string
		pair       currency.Pair
		start, end time.Time
		replies    []reply
		errs       []error
		notErr     error
	}{
		{
			// The API takes whole seconds, so a start within the second of a settlement returns that rate, which is before
			// the start
			name:    "rate before a fractional start",
			start:   time.Unix(1790294400, 0).Add(500 * time.Millisecond),
			replies: []reply{{http.StatusOK, `{"market":"BTC/USD-PERP","funding_rate_history":[{"funding_rate":"0.0001","timestamp":"1790294400"}]}`}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound},
		},
		{
			name:    "range refused",
			start:   time.Unix(1790294401, 0),
			replies: []reply{{http.StatusBadRequest, refusal}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound, errAPIResponse},
		},
		{
			name:    "bounded range refused",
			start:   time.Unix(1790294401, 0),
			end:     time.Unix(1790316000, 0),
			replies: []reply{{http.StatusBadRequest, refusal}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound, errAPIResponse},
		},
		{
			// Without a start only the latest rates are requested, which the API is assumed to refuse for a market yet to
			// settle a rate, as it does an empty range
			name:    "latest rates refused",
			replies: []reply{{http.StatusBadRequest, refusal}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound, errAPIResponse},
		},
		{
			name:    "latest rates before the end refused",
			end:     time.Unix(1790294399, 0),
			replies: []reply{{http.StatusBadRequest, refusal}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound, errAPIResponse},
		},
		{
			name:    "another market's range refused",
			pair:    ethPerpetualPair,
			start:   time.Unix(1790294401, 0),
			replies: []reply{{http.StatusBadRequest, `{"message": "Cannot fetch funding rate history for market ETH/USD-PERP"}`}},
			errs:    []error{fundingrate.ErrNoFundingRatesFound, errAPIResponse},
		},
		{
			name:    "refusal naming another market",
			start:   time.Unix(1790294401, 0),
			replies: []reply{{http.StatusBadRequest, `{"message": "Cannot fetch funding rate history for market ETH/USD-PERP"}`}},
			errs:    []error{errAPIResponse},
			notErr:  fundingrate.ErrNoFundingRatesFound,
		},
		{
			name:    "another error naming the market",
			start:   time.Unix(1790294401, 0),
			replies: []reply{{http.StatusBadRequest, `{"message": "Market BTC/USD-PERP is temporarily unavailable"}`}},
			errs:    []error{errAPIResponse},
			notErr:  fundingrate.ErrNoFundingRatesFound,
		},
		{
			name:    "other refusal",
			start:   time.Unix(1790294401, 0),
			replies: []reply{{http.StatusBadRequest, `{"code": "validation-error", "errors": [{"code": "validation-error", "message": "since_timestamp is after until_timestamp", "field": "_schema"}]}`}},
			errs:    []error{errAPIResponse},
			notErr:  fundingrate.ErrNoFundingRatesFound,
		},
		{
			// A later page ends at the oldest rate of the page before, so refusing it does not mean the range holds no rate
			name:    "later page refused",
			start:   time.Unix(1700000000, 0),
			replies: []reply{{http.StatusOK, `{"market":"BTC/USD-PERP","funding_rate_history":[` + strings.Join(fullPage, ",") + `]}`}, {http.StatusBadRequest, refusal}},
			errs:    []error{errAPIResponse},
			notErr:  fundingrate.ErrNoFundingRatesFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			pair := perpetualPair
			if !tc.pair.IsEmpty() {
				pair = tc.pair
			}
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			var mu sync.Mutex
			var paths []string
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				paths = append(paths, r.URL.Path)
				n := len(paths)
				mu.Unlock()
				if !assert.LessOrEqual(t, n, len(tc.replies), "GetHistoricalFundingRates should not request more pages than there are") {
					// Cancel the call, so a pager which retries errors stops reaching the server
					cancel()
					http.Error(w, "unexpected request", http.StatusNotFound)
					return
				}
				w.WriteHeader(tc.replies[n-1].status)
				_, err := w.Write([]byte(tc.replies[n-1].body))
				assert.NoError(t, err, "Write should not error")
			})
			var err error
			waitForCall(t, func() {
				_, err = ex.GetHistoricalFundingRates(ctx, &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: pair, StartDate: tc.start, EndDate: tc.end})
			})
			for _, exp := range tc.errs {
				assert.ErrorIs(t, err, exp, "GetHistoricalFundingRates should return the correct error")
			}
			if tc.notErr != nil {
				assert.NotErrorIs(t, err, tc.notErr, "GetHistoricalFundingRates should not report the range as holding no rate")
			}
			mu.Lock()
			defer mu.Unlock()
			assert.Len(t, paths, len(tc.replies), "GetHistoricalFundingRates should request each page once")
			for i, p := range paths {
				assert.Equalf(t, "/api/v2/funding_rate_history/"+formatMarketSymbol(pair)+"/", p, "request %d should be for the market's funding rate history", i)
			}
		})
	}

	// Only the API's own error response is its refusal, whatever another error's text holds
	t.Run("refusal text in a transport error", func(t *testing.T) {
		t.Parallel()
		ex := newTestExchange(t, func(http.ResponseWriter, *http.Request) {
			assert.Fail(t, "the test server should not be reached")
		})
		transportErr := errors.New("dial tcp: Cannot fetch funding rate history for market BTC/USD-PERP")
		require.NoError(t, ex.SetHTTPClient(&http.Client{Transport: failingTransport{err: transportErr}}), "SetHTTPClient must not error")
		var err error
		waitForCall(t, func() {
			_, err = ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: time.Unix(1790294401, 0)})
		})
		assert.ErrorIs(t, err, transportErr, "GetHistoricalFundingRates should return the transport error")
		assert.NotErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetHistoricalFundingRates should not report a transport error as a range without rates")
	})
}

// waitForCall runs call, failing rather than hanging when it does not return, as a pager which retries every error
// never does
func waitForCall(t *testing.T, call func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		call()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		require.FailNow(t, "the call must return")
	}
}

// failingTransport fails every request with its error, as a broken connection does
type failingTransport struct {
	err error
}

// RoundTrip implements http.RoundTripper
func (f failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, f.err
}

func TestIsPerpetualFutureCurrency(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]bool{asset.Spot: false, asset.PerpetualContract: true, asset.Futures: false} {
		is, err := e.IsPerpetualFutureCurrency(a, perpetualPair)
		require.NoErrorf(t, err, "IsPerpetualFutureCurrency must not error for %s", a)
		assert.Equalf(t, exp, is, "IsPerpetualFutureCurrency should be correct for %s", a)
	}
}

func TestGetCollateralCurrencyForContract(t *testing.T) {
	t.Parallel()
	_, _, err := e.GetCollateralCurrencyForContract(asset.Spot, spotPair)
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetCollateralCurrencyForContract should error on a spot asset")
	_, _, err = e.GetCollateralCurrencyForContract(asset.PerpetualContract, currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCollateralCurrencyForContract should error on an empty pair")

	c, a, err := e.GetCollateralCurrencyForContract(asset.PerpetualContract, perpetualPair)
	require.NoError(t, err, "GetCollateralCurrencyForContract must not error")
	assert.True(t, c.Equal(currency.USD), "GetCollateralCurrencyForContract should return the settlement currency")
	assert.Equal(t, asset.PerpetualContract, a, "GetCollateralCurrencyForContract should return the asset")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: currency.BTC.Item, Quote: currency.USD.Item, Asset: asset.Spot})
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest should error on a spot asset")

	interest, err := e.GetOpenInterest(t.Context())
	require.NoError(t, err, "GetOpenInterest must not error")
	require.NotEmpty(t, interest, "GetOpenInterest must return open interest")
	btc := futures.OpenInterest{
		Key:          key.ExchangeAssetPair{Exchange: e.Name, Asset: asset.PerpetualContract, Base: currency.BTC.Item, Quote: perpetualPair.Quote.Item},
		OpenInterest: 48.09714,
	}
	if mockTests {
		exp := []futures.OpenInterest{
			btc,
			{Key: key.ExchangeAssetPair{Exchange: e.Name, Asset: asset.PerpetualContract, Base: currency.ETH.Item, Quote: perpetualPair.Quote.Item}, OpenInterest: 2389.903},
		}
		assert.Equal(t, exp, interest, "GetOpenInterest should return every perpetual market's open interest")
	}
	for i := range interest {
		assert.Equal(t, asset.PerpetualContract, interest[i].Key.Asset, "Asset should be correct")
		assert.Positive(t, interest[i].OpenInterest, "OpenInterest should be positive")
	}

	interest, err = e.GetOpenInterest(t.Context(), key.PairAsset{Base: perpetualPair.Base.Item, Quote: perpetualPair.Quote.Item, Asset: asset.PerpetualContract})
	require.NoError(t, err, "GetOpenInterest must not error for a pair")
	require.Len(t, interest, 1, "GetOpenInterest must return the pair's open interest")
	if mockTests {
		assert.Equal(t, btc, interest[0], "GetOpenInterest should return the pair's open interest")
		return
	}
	assert.True(t, interest[0].Key.Pair().Equal(perpetualPair), "Pair should be correct")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.SetLeverage(t.Context(), asset.Spot, spotPair, margin.Isolated, 5, order.Buy), futures.ErrNotPerpetualFuture, "SetLeverage should error on a spot asset")
	assert.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.SpotIsolated, 5, order.Buy), margin.ErrMarginTypeUnsupported, "SetLeverage should error on an unsupported margin type")
	assert.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Isolated, 0, order.Buy), errLeverageRequired, "SetLeverage should error without a leverage")
	assert.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Isolated, maxLeverage+1, order.Buy), errInvalidLeverage, "SetLeverage should error on a leverage above the maximum")
	assert.ErrorIs(t, e.SetLeverage(t.Context(), asset.PerpetualContract, currency.EMPTYPAIR, margin.Isolated, 5, order.Buy), currency.ErrCurrencyPairEmpty, "SetLeverage should error on an empty pair")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	assert.NoError(t, e.SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Isolated, 5, order.Buy), "SetLeverage should not error")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, spotPair, margin.Isolated, order.Buy)
	assert.ErrorIs(t, err, futures.ErrNotPerpetualFuture, "GetLeverage should error on a spot asset")
	_, err = e.GetLeverage(t.Context(), asset.PerpetualContract, currency.EMPTYPAIR, margin.Isolated, order.Buy)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage should error on an empty pair")
	_, err = e.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.NoMargin, order.Buy)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "GetLeverage should error on an unsupported margin type")

	skipIfLiveWithoutCredentials(t)
	for marginType, exp := range map[margin.Type]float64{margin.Isolated: 5, margin.Multi: 3, margin.Unset: 3} {
		leverage, err := e.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, marginType, order.Buy)
		require.NoErrorf(t, err, "GetLeverage must not error for %s", marginType)
		if mockTests {
			assert.Equalf(t, exp, leverage, "GetLeverage should return the leverage for %s", marginType)
		}
	}
}

func TestGetFuturesPositionSummary(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *futures.PositionSummaryRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &futures.PositionSummaryRequest{Asset: asset.Spot, Pair: spotPair}, err: futures.ErrNotPerpetualFuture},
		{req: &futures.PositionSummaryRequest{Asset: asset.PerpetualContract}, err: currency.ErrCurrencyPairEmpty},
	} {
		_, err := e.GetFuturesPositionSummary(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetFuturesPositionSummary should return the correct error")
	}

	skipIfLiveWithoutCredentials(t)
	summary, err := e.GetFuturesPositionSummary(t.Context(), &futures.PositionSummaryRequest{Asset: asset.PerpetualContract, Pair: perpetualPair})
	if !mockTests {
		assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "GetFuturesPositionSummary should error without a position")
		return
	}
	require.NoError(t, err, "GetFuturesPositionSummary must not error")
	// Isolated positions are preferred, and short positions have a negative size
	exp := &futures.PositionSummary{
		Pair:                         perpetualPair,
		Asset:                        asset.PerpetualContract,
		MarginType:                   margin.Isolated,
		Currency:                     currency.USD,
		IsolatedMargin:               decimal.MustFromFloat(300),
		NotionalSize:                 decimal.MustFromFloat(840),
		Leverage:                     decimal.MustFromFloat(3),
		MaintenanceMarginRequirement: decimal.MustFromFloat(133.33),
		InitialMarginRequirement:     decimal.MustFromFloat(266.67),
		EstimatedLiquidationPrice:    decimal.MustFromFloat(55000),
		CollateralUsed:               decimal.MustFromFloat(306.67),
		MarkPrice:                    decimal.MustFromFloat(84000),
		CurrentSize:                  decimal.MustFromFloat(-0.01),
		ContractSettlementType:       futures.Linear,
		AverageOpenPrice:             decimal.MustFromFloat(80000),
		UnrealisedPNL:                decimal.MustFromFloat(40),
		RealisedPNL:                  decimal.MustFromFloat(-2.5),
		MaintenanceMarginFraction:    decimal.MustFromFloat(12.5),
	}
	assert.Equal(t, exp, summary, "GetFuturesPositionSummary should summarise the isolated position")
}

func TestSelectPosition(t *testing.T) {
	t.Parallel()
	positions := []PositionResponse{
		{ID: "1", Market: "ETH/USD-PERP", MarginMode: MarginModeIsolated},
		{ID: "2", Market: "BTC/USD-PERP", MarginMode: MarginModeCross},
	}
	p, err := selectPosition(positions, "BTC/USD-PERP")
	require.NoError(t, err, "selectPosition must not error")
	assert.Equal(t, "2", p.ID, "selectPosition should select the market's position")
	_, err = selectPosition(positions, "SOL/USD-PERP")
	assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "selectPosition should error without a position")
}

func TestGetFuturesPositionOrders(t *testing.T) {
	t.Parallel()
	start := time.Now().Add(-24 * time.Hour).Truncate(time.Second)
	end := start.Add(12 * time.Hour)
	for _, tc := range []struct {
		req *futures.PositionsRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &futures.PositionsRequest{Asset: asset.Spot}, err: futures.ErrNotPerpetualFuture},
		{req: &futures.PositionsRequest{Asset: asset.PerpetualContract}, err: currency.ErrCurrencyPairsEmpty},
		{req: &futures.PositionsRequest{Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}}, err: common.ErrDateUnset},
	} {
		_, err := e.GetFuturesPositionOrders(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "GetFuturesPositionOrders should return the correct error")
	}

	old := time.Now().Add(-45 * 24 * time.Hour).Truncate(time.Second)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		respect    bool
	}{
		{name: "recent", start: start, end: end},
		// The trade history documents no retention limit, so an older start is sent as GetOrderHistory sends it
		{name: "older than 30 days", start: old, end: old.Add(12 * time.Hour)},
		{name: "older than 30 days respecting history limits", start: old, end: old.Add(12 * time.Hour), respect: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// The orders are filtered to the requested range, so the trade falls inside it
			traded := tc.start.Add(time.Hour).UTC()
			ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v2/trade_history/btcusd-perp/", r.URL.Path, "path should be correct")
				assert.Equal(t, strconv.FormatInt(tc.start.Unix(), 10), r.URL.Query().Get("since_timestamp"), "since_timestamp should be the start date")
				assert.Equal(t, strconv.FormatInt(tc.end.Unix(), 10), r.URL.Query().Get("until_timestamp"), "until_timestamp should be the end date")
				_, err := w.Write([]byte(`[{"trade_id":"1","order_id":"2","datetime":"` + traded.Format(time.DateTime) + `","fee":"0.1","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"BUY","type":"TRADE","price":"84000","amount":"0.01"}]`))
				assert.NoError(t, err, "Write should not error")
			})
			resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, StartDate: tc.start, EndDate: tc.end, RespectOrderHistoryLimits: tc.respect})
			require.NoError(t, err, "GetFuturesPositionOrders must not error")
			exp := []futures.PositionResponse{{
				Pair:                   perpetualPair,
				Asset:                  asset.PerpetualContract,
				ContractSettlementType: futures.Linear,
				Orders: []order.Detail{{
					Exchange:             ex.Name,
					OrderID:              "2",
					Side:                 order.Buy,
					AssetType:            asset.PerpetualContract,
					Pair:                 currency.NewPairWithDelimiter("BTC", "USD-PERP", "/"),
					MarginType:           margin.Multi,
					Leverage:             2,
					Date:                 traded,
					LastUpdated:          traded,
					Price:                84000,
					Amount:               0.01,
					ExecutedAmount:       0.01,
					AverageExecutedPrice: 84000,
					Cost:                 840,
					CostAsset:            currency.USD,
					Fee:                  0.1,
					FeeAsset:             currency.USD,
					Trades:               []order.TradeHistory{{TID: "1", Price: 84000, Amount: 0.01, Fee: 0.1, Exchange: ex.Name, Side: order.Buy, Timestamp: traded, FeeAsset: "USD"}},
				}},
			}}
			assert.Equal(t, exp, resp, "GetFuturesPositionOrders should aggregate the pair's trades into orders")
		})
	}
}

func TestGetFuturesPositionOrdersFractionalStart(t *testing.T) {
	t.Parallel()
	// The request's time bounds are whole seconds, so a fractional start can also return trades earlier in its second;
	// this start is a nanosecond after the earlier trade, so truncating the start to microseconds or coarser lets that trade in
	start := time.Now().Add(-24 * time.Hour).Truncate(time.Second).Add(900*time.Millisecond + time.Nanosecond)
	before := start.Add(-time.Nanosecond).UTC().Format("2006-01-02 15:04:05.000000")
	after := start.Add(time.Hour).UTC().Format(time.DateTime)
	ex := newTestExchange(t, func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(`[{"trade_id":"1","order_id":"2","datetime":"` + before + `","fee":"0.1","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"BUY","type":"TRADE","price":"84000","amount":"0.01"},` +
			`{"trade_id":"3","order_id":"4","datetime":"` + after + `","fee":"0.1","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"BUY","type":"TRADE","price":"84000","amount":"0.01"}]`))
		assert.NoError(t, err, "Write should not error")
	})
	resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, StartDate: start, EndDate: start.Add(12 * time.Hour)})
	require.NoError(t, err, "GetFuturesPositionOrders must not error")
	require.Len(t, resp, 1, "GetFuturesPositionOrders must return the pair's position")
	orderIDs := make([]string, len(resp[0].Orders))
	for i := range resp[0].Orders {
		orderIDs[i] = resp[0].Orders[i].OrderID
	}
	assert.Equal(t, []string{"4"}, orderIDs, "GetFuturesPositionOrders should leave out an order traded before a fractional start")
}

func TestOrderHistoryTimeRange(t *testing.T) {
	t.Parallel()
	// The request's time bounds are whole seconds, so the API can also return trades from earlier in a fractional start's
	// second; an order counts only its trades within the range, and is dated by the first of them
	second := time.Now().Add(-24 * time.Hour).UTC().Truncate(time.Second)
	first := second.Add(900 * time.Millisecond)
	type trade struct {
		id, orderID, selfTradeOrderID uint64
		side                          string
		at                            time.Time
	}
	buys := func(times ...time.Time) []trade {
		trades := make([]trade, len(times))
		for i, at := range times {
			trades[i] = trade{id: uint64(2*i + 1), orderID: 2, side: "BUY", at: at}
		}
		return trades
	}
	for _, tc := range []struct {
		name       string
		start, end time.Time
		// trades is nil when the API answers with trades at the request's since_timestamp and an hour later
		trades []trade
		assets []asset.Item
		exp    map[string][]string
		// until is the request's until_timestamp, when the row checks it
		until time.Time
	}{
		// A nanosecond after the order's second trade, so truncating the start to microseconds or coarser lets that trade in
		{name: "start after a trade", start: first.Add(time.Nanosecond), end: second.Add(12 * time.Hour), trades: buys(first.Add(-time.Millisecond), first, first.Add(time.Microsecond), second.Add(time.Hour)), exp: map[string][]string{"2": {"5", "7"}}},
		{name: "start at a trade", start: first.Add(time.Microsecond), end: second.Add(12 * time.Hour), trades: buys(first.Add(-time.Millisecond), first, first.Add(time.Microsecond), second.Add(time.Hour)), exp: map[string][]string{"2": {"5", "7"}}},
		{name: "end at a trade", start: second, end: second.Add(2 * time.Second), trades: buys(second.Add(time.Second), second.Add(2*time.Second)), exp: map[string][]string{"2": {"1", "3"}}},
		{name: "fractional end", start: second, end: second.Add(2900 * time.Millisecond), trades: buys(second.Add(time.Second), second.Add(2*time.Second)), exp: map[string][]string{"2": {"1", "3"}}},
		// The end is requested up to its next whole second, which returns trades after it too. It is a nanosecond before the
		// order's third trade, so truncating it to milliseconds or coarser leaves out the second trade, and rounding it up
		// to microseconds or coarser lets the third in
		{
			name: "end before a trade", start: second, end: second.Add(2400*time.Millisecond - time.Nanosecond), until: second.Add(3 * time.Second),
			trades: buys(second.Add(time.Second), second.Add(2400*time.Millisecond-time.Microsecond), second.Add(2400*time.Millisecond), second.Add(3*time.Second)),
			exp:    map[string][]string{"2": {"1", "3"}},
		},
		{
			name: "self trade before the start", start: first.Add(time.Nanosecond), end: second.Add(12 * time.Hour), assets: []asset.Item{asset.PerpetualContract},
			trades: []trade{{id: 1, orderID: 2, selfTradeOrderID: 4, side: TradeSideSelf, at: first}, {id: 3, orderID: 2, side: "BUY", at: first.Add(time.Microsecond)}, {id: 5, orderID: 4, side: "SELL", at: second.Add(time.Hour)}},
			exp:    map[string][]string{"2": {"3"}, "4": {"5"}},
		},
		// Spot transactions only reach back 30 days, so the request's since_timestamp is moved forward, still within the range
		{name: "start older than 30 days", start: second.Add(-45 * 24 * time.Hour), end: second.Add(12 * time.Hour), exp: map[string][]string{"2": {"1", "3"}}},
	} {
		assets := tc.assets
		if assets == nil {
			assets = []asset.Item{asset.Spot, asset.PerpetualContract}
		}
		for _, a := range assets {
			for _, positions := range []bool{false, true} {
				if positions && a != asset.PerpetualContract {
					continue
				}
				name := "GetOrderHistory " + a.String()
				if positions {
					name = "GetFuturesPositionOrders"
				}
				var mu sync.Mutex
				sent := tc.trades
				ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
					assert.NoError(t, r.ParseForm(), "ParseForm should not error")
					if !tc.until.IsZero() {
						assert.Equalf(t, strconv.FormatInt(tc.until.Unix(), 10), r.Form.Get("until_timestamp"), "%s should request the end rounded up to a whole second for %s", name, tc.name)
					}
					mu.Lock()
					defer mu.Unlock()
					if tc.trades == nil {
						since, err := strconv.ParseInt(r.Form.Get("since_timestamp"), 10, 64)
						assert.NoError(t, err, "since_timestamp should be a number")
						sent = buys(time.Unix(since, 0).UTC(), time.Unix(since, 0).UTC().Add(time.Hour))
					}
					rows := make([]string, len(sent))
					for i, tr := range sent {
						datetime := tr.at.UTC().Format("2006-01-02 15:04:05.000000")
						rows[i] = `{"trade_id":"` + strconv.FormatUint(tr.id, 10) + `","order_id":"` + strconv.FormatUint(tr.orderID, 10) + `","self_trade_order_id":"` + strconv.FormatUint(tr.selfTradeOrderID, 10) + `","datetime":"` + datetime + `","fee":"0.1","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"` + tr.side + `","type":"TRADE","price":"84000","amount":"0.01"}`
						if a == asset.Spot {
							rows[i] = `{"id":` + strconv.FormatUint(tr.id, 10) + `,"datetime":"` + datetime + `","type":"2","fee":"0.1","btc":"0.01","usd":"-840.0","btc_usd":84000,"order_id":` + strconv.FormatUint(tr.orderID, 10) + `}`
						}
					}
					_, err := w.Write([]byte("[" + strings.Join(rows, ",") + "]"))
					assert.NoError(t, err, "Write should not error")
				})
				var orders []order.Detail
				if positions {
					resp, err := ex.GetFuturesPositionOrders(t.Context(), &futures.PositionsRequest{Asset: a, Pairs: currency.Pairs{perpetualPair}, StartDate: tc.start, EndDate: tc.end})
					require.NoErrorf(t, err, "%s must not error for %s", name, tc.name)
					require.Lenf(t, resp, 1, "%s must return the pair's position for %s", name, tc.name)
					orders = resp[0].Orders
				} else {
					var err error
					orders, err = ex.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType, StartTime: tc.start, EndTime: tc.end})
					require.NoErrorf(t, err, "%s must not error for %s", name, tc.name)
				}
				mu.Lock()
				executed := make(map[string]time.Time, len(sent))
				for _, tr := range sent {
					executed[strconv.FormatUint(tr.id, 10)] = tr.at
				}
				mu.Unlock()
				require.Lenf(t, orders, len(tc.exp), "%s must return each order with a trade in the range for %s", name, tc.name)
				for i := range orders {
					exp, ok := tc.exp[orders[i].OrderID]
					require.Truef(t, ok, "%s must not return order %s for %s", name, orders[i].OrderID, tc.name)
					tradeIDs := make([]string, len(orders[i].Trades))
					for j := range orders[i].Trades {
						tradeIDs[j] = orders[i].Trades[j].TID
					}
					assert.Equalf(t, exp, tradeIDs, "%s should keep only order %s's trades in the range for %s", name, orders[i].OrderID, tc.name)
					assert.Equalf(t, float64(len(exp))/100, orders[i].ExecutedAmount, "%s should count only order %s's trades in the range for %s", name, orders[i].OrderID, tc.name)
					assert.Truef(t, orders[i].Date.Equal(executed[exp[0]]), "%s should date order %s by its first trade in the range for %s", name, orders[i].OrderID, tc.name)
					assert.Truef(t, orders[i].LastUpdated.Equal(executed[exp[len(exp)-1]]), "%s should update order %s by its last trade in the range for %s", name, orders[i].OrderID, tc.name)
				}
			}
		}
	}
}

func TestChangePositionMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		req *margin.PositionChangeRequest
		err error
	}{
		{req: nil, err: common.ErrNilPointer},
		{req: &margin.PositionChangeRequest{Asset: asset.Spot}, err: futures.ErrNotPerpetualFuture},
		{req: &margin.PositionChangeRequest{Asset: asset.PerpetualContract}, err: currency.ErrCurrencyPairEmpty},
		{req: &margin.PositionChangeRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, MarginType: margin.Multi}, err: margin.ErrMarginTypeUnsupported},
		{req: &margin.PositionChangeRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, MarginType: margin.Isolated}, err: margin.ErrNewAllocatedMarginRequired},
	} {
		_, err := e.ChangePositionMargin(t.Context(), tc.req)
		assert.ErrorIs(t, err, tc.err, "ChangePositionMargin should return the correct error")
	}

	// Only an isolated position's collateral can be set, so a market without one is refused before any change is sent,
	// whether the market's positions are read alone or with every other market's
	const otherIsolated = `{"id":"1234567892","market":"ETH/USD-PERP","margin_mode":"ISOLATED","side":"LONG","size":"0.10000"}`
	const cross = `{"id":"1234567891","market":"BTC/USD-PERP","margin_mode":"CROSS","side":"LONG","size":"0.02000"}`
	for _, tc := range []struct {
		name, market, all string
	}{
		{name: "no position", market: `[]`, all: `[` + otherIsolated + `]`},
		{name: "a cross position", market: `[` + cross + `]`, all: `[` + otherIsolated + `,` + cross + `]`},
	} {
		var mu sync.Mutex
		var paths []string
		ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			paths = append(paths, r.URL.Path)
			mu.Unlock()
			body := `{}`
			switch r.URL.Path {
			case "/api/v2/open_positions/btcusd-perp/":
				body = tc.market
			case "/api/v2/open_positions/":
				body = tc.all
			}
			_, err := w.Write([]byte(body))
			assert.NoError(t, err, "Write should not error")
		})
		_, err := ex.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, MarginType: margin.Isolated, NewAllocatedMargin: 12.5})
		assert.ErrorIsf(t, err, futures.ErrNoPositionsFound, "ChangePositionMargin should error with %s for the market", tc.name)
		mu.Lock()
		assert.Truef(t, slices.ContainsFunc(paths, func(p string) bool { return strings.HasPrefix(p, "/api/v2/open_positions/") }), "ChangePositionMargin should read the open positions with %s", tc.name)
		assert.NotContainsf(t, paths, "/api/v2/adjust_position_collateral/", "ChangePositionMargin should not change any collateral with %s", tc.name)
		mu.Unlock()
	}

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.ChangePositionMargin(t.Context(), &margin.PositionChangeRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, MarginType: margin.Isolated, NewAllocatedMargin: 12.5})
	if !mockTests {
		assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "ChangePositionMargin should error without a position")
		return
	}
	require.NoError(t, err, "ChangePositionMargin must not error")
	exp := &margin.PositionChangeResponse{Exchange: e.Name, Pair: perpetualPair, Asset: asset.PerpetualContract, AllocatedMargin: 12.5, MarginType: margin.Isolated}
	assert.Equal(t, exp, resp, "ChangePositionMargin should return the position's new margin")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	for _, a := range e.GetAssetTypes(false) {
		pairs, err := e.CurrencyPairs.GetPairs(a, true)
		require.NoErrorf(t, err, "GetPairs must not error for %s", a)
		require.NotEmptyf(t, pairs, "enabled pairs must not be empty for %s", a)
		resp, err := e.GetCurrencyTradeURL(t.Context(), a, pairs[0])
		require.NoErrorf(t, err, "GetCurrencyTradeURL must not error for %s", a)
		assert.Equalf(t, tradeBaseURL+formatMarketSymbol(pairs[0])+"/", resp, "GetCurrencyTradeURL should return the market's URL for %s", a)
	}
	_, err := e.GetCurrencyTradeURL(t.Context(), asset.Futures, spotPair)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetCurrencyTradeURL should error on an unsupported asset")
}

func TestMarketTypeForAsset(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]string{asset.Spot: MarketTypeSpot, asset.PerpetualContract: MarketTypePerpetual} {
		marketType, err := marketTypeForAsset(a)
		require.NoErrorf(t, err, "marketTypeForAsset must not error for %s", a)
		assert.Equalf(t, exp, marketType, "marketTypeForAsset should be correct for %s", a)
	}
	_, err := marketTypeForAsset(asset.Futures)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "marketTypeForAsset should error on an unsupported asset")
}

func TestMarketPairAsset(t *testing.T) {
	t.Parallel()
	for market, exp := range map[string]asset.Item{"BTC/USD": asset.Spot, "BTC/USD-PERP": asset.PerpetualContract} {
		pair, a, err := marketPairAsset(market)
		require.NoErrorf(t, err, "marketPairAsset must not error for %s", market)
		assert.Equalf(t, exp, a, "marketPairAsset should return the asset for %s", market)
		assert.Equalf(t, market, pair.String(), "marketPairAsset should return the pair for %s", market)
	}
	_, _, err := marketPairAsset("BTCUSD")
	assert.Error(t, err, "marketPairAsset should error on a malformed market")
}

func TestMarginModes(t *testing.T) {
	t.Parallel()
	for marginType, exp := range map[margin.Type]string{margin.Unset: MarginModeCross, margin.Multi: MarginModeCross, margin.Isolated: MarginModeIsolated} {
		mode, err := formatMarginMode(marginType)
		require.NoErrorf(t, err, "formatMarginMode must not error for %s", marginType)
		assert.Equalf(t, exp, mode, "formatMarginMode should be correct for %s", marginType)
	}
	_, err := formatMarginMode(margin.NoMargin)
	assert.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "formatMarginMode should error on an unsupported margin type")

	for mode, exp := range map[string]margin.Type{MarginModeCross: margin.Multi, MarginModeIsolated: margin.Isolated, "": margin.Unset} {
		assert.Equalf(t, exp, marginTypeFromMode(mode), "marginTypeFromMode should be correct for %q", mode)
	}
}

func TestFormatTrigger(t *testing.T) {
	t.Parallel()
	for priceType, exp := range map[order.PriceType]string{order.LastPrice: TriggerLastTradedPrice, order.IndexPrice: TriggerIndexPrice, order.MarkPrice: TriggerMarkPrice} {
		assert.Equalf(t, exp, formatTrigger(priceType), "formatTrigger should be correct for %s", priceType)
	}
}

func TestOrderTypeFromSubtype(t *testing.T) {
	t.Parallel()
	for subtype, exp := range map[string]order.Type{
		"":                                  order.Limit,
		OrderSubtypeLimit:                   order.Limit,
		OrderSubtypeMarket:                  order.Market,
		OrderSubtypeInstant:                 order.Market,
		OrderSubtypeCash:                    order.Market,
		OrderSubtypeStopMarket:              order.StopMarket,
		OrderSubtypeStopLoss:                order.StopMarket,
		OrderSubtypeStopLimit:               order.StopLimit,
		OrderSubtypeStopLossLimit:           order.StopLimit,
		OrderSubtypeTakeProfit:              order.TakeProfitMarket,
		OrderSubtypeTakeProfitLimit:         order.TakeProfit | order.Limit,
		OrderSubtypeTrailingStopLoss:        order.TrailingStop,
		OrderSubtypeTrailingTakeProfit:      order.TrailingStop,
		OrderSubtypeTrailingStopLossLimit:   order.TrailingStopLimit,
		OrderSubtypeTrailingTakeProfitLimit: order.TrailingStopLimit,
		"UNKNOWN":                           order.UnknownType,
	} {
		assert.Equalf(t, exp, orderTypeFromSubtype(subtype), "orderTypeFromSubtype should be correct for %q", subtype)
	}
}

// recordedPagerContext returns the context for paging the recorded mock, which serves the same response to every request,
// so a pager which never stops fails at the deadline instead of hanging its test; live histories can take longer to page
func recordedPagerContext(t *testing.T) context.Context {
	t.Helper()
	if !mockTests {
		return t.Context()
	}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// pagedTestExchange returns an exchange whose test server responds to each request with the page built from its
// parameters, and a function returning the parameters of every request received
func pagedTestExchange(t *testing.T, path string, page func(params url.Values) string) (ex *Exchange, requests func() []url.Values) {
	t.Helper()
	var mu sync.Mutex
	var received []url.Values
	ex = newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, path, r.URL.Path, "path should be correct")
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		mu.Lock()
		received = append(received, r.Form)
		mu.Unlock()
		_, err := w.Write([]byte(page(r.Form)))
		assert.NoError(t, err, "Write should not error")
	})
	return ex, func() []url.Values {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(received)
	}
}

// idList returns a JSON list of objects, each formatted from an ID between first and last inclusive
func idList(format string, first, last uint64) string {
	var sb strings.Builder
	sb.WriteByte('[')
	for id := first; id <= last; id++ {
		if id > first {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, format, id)
	}
	sb.WriteByte(']')
	return sb.String()
}

// pageOffset returns the offset of a paged request, which is not sent for the first page
func pageOffset(t *testing.T, params url.Values) uint64 {
	t.Helper()
	if params.Get("offset") == "" {
		return 0
	}
	offset, err := strconv.ParseUint(params.Get("offset"), 10, 64)
	assert.NoError(t, err, "ParseUint should not error")
	return offset
}

func TestAllUserTransactions(t *testing.T) {
	t.Parallel()
	ex, requests := pagedTestExchange(t, "/api/v2/user_transactions/", func(params url.Values) string {
		offset := pageOffset(t, params)
		return idList(`{"id":%d}`, offset+1, min(offset+maxResultLimit, 1500))
	})
	txs, err := ex.allUserTransactions(t.Context(), &UserTransactionsRequest{Limit: maxResultLimit, Sort: SortAscending})
	require.NoError(t, err, "allUserTransactions must not error")
	require.Len(t, txs, 1500, "allUserTransactions must return every transaction")
	assert.Equal(t, uint64(1500), txs[1499].ID, "allUserTransactions should return the transactions in order")
	assert.Equal(t, []url.Values{
		{"limit": {"1000"}, "sort": {"asc"}},
		{"limit": {"1000"}, "offset": {"1000"}, "sort": {"asc"}},
	}, requests(), "allUserTransactions should request each page by offset")

	ex, requests = pagedTestExchange(t, "/api/v2/user_transactions/", func(url.Values) string {
		return idList(`{"id":%d}`, 1, maxResultLimit)
	})
	txs, err = ex.allUserTransactions(t.Context(), &UserTransactionsRequest{Limit: maxResultLimit, Sort: SortAscending})
	require.NoError(t, err, "allUserTransactions must not error when the offset is not applied")
	assert.Len(t, txs, maxResultLimit, "allUserTransactions should not repeat transactions")
	assert.Len(t, requests(), 2, "allUserTransactions should stop at a page without new transactions")

	// Past the maximum offset the pages continue from the last transaction's ID, which this venue ignores
	ex, requests = pagedTestExchange(t, "/api/v2/user_transactions/", func(params url.Values) string {
		offset := pageOffset(t, params)
		return idList(`{"id":%d}`, offset+1, offset+maxResultLimit)
	})
	_, err = ex.allUserTransactions(t.Context(), &UserTransactionsRequest{Limit: maxResultLimit, Sort: SortAscending})
	assert.ErrorIs(t, err, errInvalidOffset, "allUserTransactions should error when the transactions after the maximum offset are not returned")
	received := requests()
	require.Len(t, received, maxResultOffset/maxResultLimit+2, "allUserTransactions must stop at a page from a transaction ID without new transactions")
	assert.Equal(t, url.Values{"limit": {"1000"}, "offset": {"200000"}, "sort": {"asc"}}, received[len(received)-2], "allUserTransactions should page by offset up to the maximum offset")
	assert.Equal(t, url.Values{"limit": {"1000"}, "since_id": {"201000"}, "sort": {"asc"}}, received[len(received)-1], "allUserTransactions should continue from the last transaction's ID")

	// How the API applies since_id is not documented, so both an inclusive and an exclusive ID are paged through
	const transactions = 230500
	for _, inclusive := range []bool{true, false} {
		t.Run("since ID inclusive "+strconv.FormatBool(inclusive), func(t *testing.T) {
			t.Parallel()
			ex, requests := pagedTestExchange(t, "/api/v2/user_transactions/", func(params url.Values) string {
				offset := pageOffset(t, params)
				assert.LessOrEqual(t, offset, uint64(maxResultOffset), "offset should not exceed the maximum")
				first := offset + 1
				if sinceID := params.Get("since_id"); sinceID != "" {
					assert.Zero(t, offset, "a page from a transaction ID should not skip transactions")
					id, err := strconv.ParseUint(sinceID, 10, 64)
					assert.NoError(t, err, "since_id should be a number")
					first = id
					if !inclusive {
						first++
					}
				}
				return idList(`{"id":%d}`, first, min(first+maxResultLimit-1, transactions))
			})
			txs, err := ex.allUserTransactions(t.Context(), &UserTransactionsRequest{Limit: maxResultLimit, Sort: SortAscending})
			require.NoError(t, err, "allUserTransactions must not error past the maximum offset")
			require.Len(t, txs, transactions, "allUserTransactions must return every transaction once")
			for i := range txs {
				if !assert.Equalf(t, uint64(i+1), txs[i].ID, "transaction %d should be in ID order", i) {
					break
				}
			}
			assert.Len(t, requests(), 231, "allUserTransactions should request each transaction's page once")
		})
	}
}

func TestGetWithdrawalsHistoryPaging(t *testing.T) {
	t.Parallel()
	ex, requests := pagedTestExchange(t, "/api/v2/withdrawal-requests/", func(params url.Values) string {
		offset := pageOffset(t, params)
		return idList(`{"id":%d,"currency":"BTC","status":2,"type":1}`, offset+1, min(offset+maxResultLimit, 1500))
	})
	history, err := ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	require.Len(t, history, 1500, "GetWithdrawalsHistory must return every withdrawal")
	assert.Equal(t, "1500", history[1499].TransferID, "GetWithdrawalsHistory should return the withdrawals in order")
	assert.Equal(t, []url.Values{
		{"limit": {"1000"}, "offset": {"0"}, "timedelta": {"50000000"}},
		{"limit": {"1000"}, "offset": {"1000"}, "timedelta": {"50000000"}},
	}, requests(), "GetWithdrawalsHistory should request each page by offset over the longest time delta")

	ex, requests = pagedTestExchange(t, "/api/v2/withdrawal-requests/", func(url.Values) string {
		return idList(`{"id":%d,"currency":"BTC"}`, 1, maxResultLimit)
	})
	history, err = ex.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error when the offset is not applied")
	assert.Len(t, history, maxResultLimit, "GetWithdrawalsHistory should not repeat withdrawals")
	assert.Len(t, requests(), 2, "GetWithdrawalsHistory should stop at a page without new withdrawals")

	ex, _ = pagedTestExchange(t, "/api/v2/withdrawal-requests/", func(params url.Values) string {
		offset := pageOffset(t, params)
		return idList(`{"id":%d,"currency":"BTC"}`, offset+1, offset+maxResultLimit)
	})
	_, err = ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Spot)
	assert.ErrorIs(t, err, errInvalidOffset, "GetWithdrawalsHistory should error when the withdrawals exceed the maximum offset")
}

func TestAllCryptoTransactions(t *testing.T) {
	t.Parallel()
	ex, requests := pagedTestExchange(t, "/api/v2/crypto-transactions/", func(params url.Values) string {
		offset := pageOffset(t, params)
		withdrawals := `[{"txid":"1"},{"txid":"2"}]`
		if offset > 0 {
			withdrawals = `[{"txid":"3"}]`
		}
		return `{"deposits":` + idList(`{"id":%d}`, offset+1, min(offset+maxResultLimit, 1200)) + `,"withdrawals":` + withdrawals + `,"ripple_iou_transactions":[]}`
	})
	txs, err := ex.allCryptoTransactions(t.Context())
	require.NoError(t, err, "allCryptoTransactions must not error")
	assert.Len(t, txs.Deposits, 1200, "allCryptoTransactions should return every deposit")
	assert.Len(t, txs.Withdrawals, 3, "allCryptoTransactions should return the withdrawals of every page")
	assert.Equal(t, []url.Values{{"limit": {"1000"}}, {"limit": {"1000"}, "offset": {"1000"}}}, requests(), "allCryptoTransactions should request pages until no list is full")

	ex, _ = pagedTestExchange(t, "/api/v2/crypto-transactions/", func(url.Values) string {
		return `{"deposits":[],"withdrawals":[],"ripple_iou_transactions":` + idList(`{"txid":"%d"}`, 1, maxResultLimit) + `}`
	})
	_, err = ex.allCryptoTransactions(t.Context())
	assert.ErrorIs(t, err, errInvalidOffset, "allCryptoTransactions should error when the transactions exceed the maximum offset")
}

func TestAllDerivativesTrades(t *testing.T) {
	t.Parallel()
	ex, requests := pagedTestExchange(t, "/api/v2/trade_history/btcusd-perp/", func(params url.Values) string {
		first := uint64(1)
		if afterID := params.Get("after_id"); afterID != "" {
			// A page starts from the trade it was requested after, which is returned again
			var err error
			first, err = strconv.ParseUint(afterID, 10, 64)
			assert.NoError(t, err, "ParseUint should not error")
		}
		return idList(`{"trade_id":"%d"}`, first, min(first+maxResultLimit-1, 1500))
	})
	trades, err := ex.allDerivativesTrades(t.Context(), &DerivativesTradeHistoryRequest{Pair: perpetualPair, Limit: maxResultLimit, Sort: SortAscending})
	require.NoError(t, err, "allDerivativesTrades must not error")
	require.Len(t, trades, 1500, "allDerivativesTrades must return every trade once")
	assert.Equal(t, uint64(1500), trades[1499].TradeID, "allDerivativesTrades should return the trades in order")
	assert.Equal(t, []url.Values{
		{"limit": {"1000"}, "sort": {"asc"}},
		{"after_id": {"1000"}, "limit": {"1000"}, "sort": {"asc"}},
	}, requests(), "allDerivativesTrades should request each page after the last trade")

	ex, requests = pagedTestExchange(t, "/api/v2/trade_history/btcusd-perp/", func(url.Values) string {
		return idList(`{"trade_id":"%d"}`, 1, maxResultLimit)
	})
	trades, err = ex.allDerivativesTrades(t.Context(), &DerivativesTradeHistoryRequest{Pair: perpetualPair, Limit: maxResultLimit, Sort: SortAscending})
	require.NoError(t, err, "allDerivativesTrades must not error when the trade ID is not applied")
	assert.Len(t, trades, maxResultLimit, "allDerivativesTrades should not repeat trades")
	assert.Len(t, requests(), 2, "allDerivativesTrades should stop at a page without new trades")
}

func TestDerivativesOrderHistorySelfTrades(t *testing.T) {
	t.Parallel()
	ex, _ := pagedTestExchange(t, "/api/v2/trade_history/btcusd-perp/", func(url.Values) string {
		return `[
			{"trade_id":"1","order_id":"10","self_trade_order_id":"11","datetime":"2025-06-16 10:00:00","fee":"0.5","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"SELF","type":"TRADE","price":"80000","amount":"0.01"},
			{"trade_id":"2","order_id":"11","datetime":"2025-06-16 10:01:00","fee":"0.4","fee_currency":"USD","market":"BTC/USD-PERP","margin_mode":"CROSS","leverage":"2","side":"SELL","type":"TRADE","price":"81000","amount":"0.02"}
		]`
	})
	orders, err := ex.derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	require.NoError(t, err, "derivativesOrderHistory must not error")
	perpetual := currency.NewPairWithDelimiter("BTC", "USD-PERP", "/")
	selfTrade := order.TradeHistory{TID: "1", Price: 80000, Amount: 0.01, Fee: 0.5, Exchange: ex.Name, Timestamp: time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC), FeeAsset: "USD"}
	complementary := selfTrade
	complementary.Fee = 0
	// The self trade is added to both orders, with its fee only counted once; an order only seen through a self trade
	// has an unknown side, and the other order's side is revealed by its own trade
	assert.Equal(t, []order.Detail{
		{
			Exchange:             ex.Name,
			OrderID:              "10",
			Side:                 order.UnknownSide,
			AssetType:            asset.PerpetualContract,
			Pair:                 perpetual,
			MarginType:           margin.Multi,
			Leverage:             2,
			Date:                 time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
			LastUpdated:          time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
			Price:                80000,
			Amount:               0.01,
			ExecutedAmount:       0.01,
			AverageExecutedPrice: 80000,
			Cost:                 800,
			CostAsset:            currency.USD,
			Fee:                  0.5,
			FeeAsset:             currency.USD,
			Trades:               []order.TradeHistory{selfTrade},
		},
		{
			Exchange:             ex.Name,
			OrderID:              "11",
			Side:                 order.Sell,
			AssetType:            asset.PerpetualContract,
			Pair:                 perpetual,
			MarginType:           margin.Multi,
			Leverage:             2,
			Date:                 time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
			LastUpdated:          time.Date(2025, 6, 16, 10, 1, 0, 0, time.UTC),
			Price:                2420 / 0.03,
			Amount:               0.03,
			ExecutedAmount:       0.03,
			AverageExecutedPrice: 2420 / 0.03,
			Cost:                 2420,
			CostAsset:            currency.USD,
			Fee:                  0.4,
			FeeAsset:             currency.USD,
			Trades: []order.TradeHistory{
				complementary,
				{TID: "2", Price: 81000, Amount: 0.02, Fee: 0.4, Exchange: ex.Name, Side: order.Sell, Timestamp: time.Date(2025, 6, 16, 10, 1, 0, 0, time.UTC), FeeAsset: "USD"},
			},
		},
	}, orders, "derivativesOrderHistory should return both orders of a self trade")

	ex, _ = pagedTestExchange(t, "/api/v2/trade_history/btcusd-perp/", func(url.Values) string {
		return `[{"trade_id":"1","order_id":"10","self_trade_order_id":"wibble","datetime":"2025-06-16 10:00:00","market":"BTC/USD-PERP","side":"SELF","price":"80000","amount":"0.01"}]`
	})
	_, err = ex.derivativesOrderHistory(t.Context(), currency.Pairs{perpetualPair}, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "derivativesOrderHistory should error on a malformed self trade order ID")
}

func TestWithdrawCryptocurrencyFundsTags(t *testing.T) {
	t.Parallel()
	requests := make(chan url.Values, 1)
	ex := newTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, r.ParseForm(), "ParseForm should not error")
		select {
		case requests <- r.PostForm:
		default:
			assert.Fail(t, "WithdrawCryptocurrencyFunds should send only one request")
		}
		_, err := w.Write([]byte(`{"id":1}`))
		assert.NoError(t, err, "Write should not error")
	})
	for _, tc := range []struct {
		c      currency.Code
		chain  string
		tag    string
		isSelf bool
		exp    url.Values
	}{
		{c: currency.XRP, tag: "12345", exp: url.Values{"destination_tag": {"12345"}}},
		{c: currency.USDC, chain: "XRPL", tag: "12345", exp: url.Values{"network": {"XRPL"}, "destination_tag": {"12345"}}},
		{c: currency.CSPR, tag: "67890", exp: url.Values{"transfer_id": {"67890"}}},
		{c: currency.XLM, tag: "memo", exp: url.Values{"memo_id": {"memo"}}},
		{c: currency.BTC, chain: "bitcoin", isSelf: true, exp: url.Values{"network": {"bitcoin"}, "beneficiary_thirdparty": {"False"}}},
	} {
		resp, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange: ex.Name,
			Type:     withdraw.Crypto,
			Currency: tc.c,
			Amount:   1,
			Crypto:   withdraw.CryptoRequest{Address: "address", AddressTag: tc.tag, Chain: tc.chain},
			Travel:   withdraw.TravelRule{IsSelf: tc.isSelf},
		})
		require.NoErrorf(t, err, "WithdrawCryptocurrencyFunds must not error for %s", tc.c)
		assert.Equalf(t, "1", resp.ID, "ID should be correct for %s", tc.c)
		tc.exp.Set("amount", "1")
		tc.exp.Set("address", "address")
		require.Lenf(t, requests, 1, "WithdrawCryptocurrencyFunds must send a request for %s", tc.c)
		assert.Equalf(t, tc.exp, <-requests, "WithdrawCryptocurrencyFunds should send the tag in the field for the %s network", tc.c)
	}

	_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
		Exchange: ex.Name,
		Type:     withdraw.Crypto,
		Currency: currency.CSPR,
		Amount:   1,
		Crypto:   withdraw.CryptoRequest{Address: "address", AddressTag: "memo"},
	})
	assert.ErrorIs(t, err, errInvalidTransferID, "WithdrawCryptocurrencyFunds should error on a Casper transfer ID which is not a number")
	for _, chain := range []string{"", "casper"} {
		_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &withdraw.Request{
			Exchange: ex.Name,
			Type:     withdraw.Crypto,
			Currency: currency.CSPR,
			Amount:   1,
			Crypto:   withdraw.CryptoRequest{Address: "address", AddressTag: "0", Chain: chain},
		})
		assert.ErrorIsf(t, err, errZeroTransferID, "WithdrawCryptocurrencyFunds should error on a Casper transfer ID of zero on the %q network", chain)
	}
	assert.Empty(t, requests, "WithdrawCryptocurrencyFunds should not send a withdrawal with an invalid Casper transfer ID")
}

func TestGetLeverageSettingNotFound(t *testing.T) {
	t.Parallel()
	ex, _ := pagedTestExchange(t, "/api/v2/leverage_settings/", func(url.Values) string {
		return `[{"margin_mode":"CROSS","market":"ETH/USD-PERP","leverage_current":"3.00","leverage_max":"10.00"}]`
	})
	_, err := ex.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Multi, order.Buy)
	assert.ErrorIs(t, err, errLeverageSettingNotFound, "GetLeverage should error when no setting matches the market and margin mode")
}
