package hyperliquid

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
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
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/protocol"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	// testWithdrawalAddress receives the mock withdrawals and transfers
	testWithdrawalAddress = "0x0000000000000000000000000000000000000014"
	// liveOrderNotional is the USDC value of live test orders, above Hyperliquid's $10 minimum order value
	liveOrderNotional = 11
	// liveMarketOrderSlippage bounds the price of live test market orders
	liveMarketOrderSlippage = 0.01
)

var (
	// dashFormat is the configured pair format, which enabled pairs carry
	dashFormat      = currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}
	testBuilderPair = currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	// The mappings of the mock responses' BTC, HYPE spot and xyz:XYZ100 markets
	testPerpetualMapping = pairMapping{pair: perpetualPair, coin: "BTC", sizeDecimals: 5, maxLeverage: 40}
	testSpotMapping      = pairMapping{pair: spotPair, coin: "@107", assetID: 10107, sizeDecimals: 2}
	testBuilderMapping   = pairMapping{pair: testBuilderPair, coin: "xyz:XYZ100", dex: "xyz", assetID: 110000, sizeDecimals: 4, maxLeverage: 30, deployerFeeScale: 1, growthMode: true}
	// testDelistedMapping is the mapping of the mock responses' delisted MATIC market
	testDelistedMapping = pairMapping{pair: currency.NewPair(currency.MATIC, currency.USDC), coin: "MATIC", assetID: 2, sizeDecimals: 1, maxLeverage: 20}
	// testPerpetualMappings are the mappings of every active perpetual market in the mock responses
	testPerpetualMappings = []pairMapping{
		testPerpetualMapping,
		{pair: currency.NewPair(currency.ETH, currency.USDC), coin: "ETH", assetID: 1, sizeDecimals: 4, maxLeverage: 25},
		{pair: currency.NewPair(currency.NewCode("HPOS"), currency.USDC), coin: "HPOS", assetID: 3, maxLeverage: 3, onlyIsolated: true},
		testBuilderMapping,
		{pair: currency.NewPair(currency.NewCode("xyz:TSLA"), currency.USDC), coin: "xyz:TSLA", dex: "xyz", assetID: 110001, sizeDecimals: 3, maxLeverage: 20, onlyIsolated: true},
		{pair: currency.NewPair(currency.NewCode("flx:TSLA"), currency.NewCode("USDH")), coin: "flx:TSLA", dex: "flx", assetID: 120000, sizeDecimals: 2, maxLeverage: 10, deployerFeeScale: 1, growthMode: true},
	}
)

// newInfoServerExchange returns an exchange signing as the mock signing account, whose info requests receive the
// response keyed by their type and DEX as "type:dex", else by their type; the perpetual DEX registry and user roles
// default to only the default DEX and a user, and any other info request fails. Exchange actions receive the response
// actions returns for the decoded action, and fail when it returns nothing
func newInfoServerExchange(t *testing.T, responses map[string]string, actions func(action map[string]any) string) *Exchange {
	t.Helper()
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var response string
		switch r.URL.Path {
		case "/info":
			var body InfoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
				return
			}
			var ok bool
			if response, ok = responses[body.Type+":"+body.DEX]; ok {
				break
			}
			if response, ok = responses[body.Type]; ok {
				break
			}
			switch body.Type {
			case "perpDexs":
				response = `[null]`
			case "userRole":
				response = `{"role":"user"}`
			default:
				http.Error(w, "unexpected info request "+body.Type, http.StatusBadRequest)
				return
			}
		case "/exchange":
			var body SignedActionRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the exchange request should not error") {
				return
			}
			action, ok := body.Action.(map[string]any)
			if !assert.True(t, ok, "The exchange request's action should be a JSON object") {
				return
			}
			if actions != nil {
				response = actions(action)
			}
			if response == "" {
				http.Error(w, "unexpected exchange action", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	return ex
}

// clearBalanceTimes checks the times Save stamps on stored balances are current, then clears them so balances compare by
// value
func clearBalanceTimes(t *testing.T, subAccounts accounts.SubAccounts) {
	t.Helper()
	for _, subAccount := range subAccounts {
		for code, balance := range subAccount.Balances {
			assert.WithinDurationf(t, time.Now(), balance.UpdatedAt, time.Minute, "Save should stamp the %s balance with the current time", code)
			balance.UpdatedAt = time.Time{}
			subAccount.Balances[code] = balance
		}
	}
}

// balanceObserver records the stored USDC total as each Save starts. A refresh's stored balances change only inside a
// Save, so any total a reader can see between two of its Saves is still stored when the next Save starts
type balanceObserver struct {
	*Exchange
	usdcTotals []float64
}

// GetCredentials records the stored USDC total, as Save requests credentials before it stores anything. A store
// holding no balances is recorded as no USDC, since a reader then sees none
func (o *balanceObserver) GetCredentials(ctx context.Context) (*accounts.Credentials, error) {
	if balances, err := o.Accounts.CurrencyBalances(nil, asset.All); err == nil || errors.Is(err, accounts.ErrNoBalances) {
		o.usdcTotals = append(o.usdcTotals, balances[currency.USDC].Total)
	}
	return o.Exchange.GetCredentials(ctx)
}

// statusesResponse returns an exchange action response holding the statuses, which are JSON values
func statusesResponse(responseType string, statuses ...string) string {
	return `{"status":"ok","response":{"type":"` + responseType + `","data":{"statuses":[` + strings.Join(statuses, ",") + `]}}}`
}

// setTestPairs caches the mappings and makes their pairs the asset's available and enabled pairs
func setTestPairs(t *testing.T, ex *Exchange, a asset.Item, mappings ...pairMapping) {
	t.Helper()
	ex.setPairMappings(a, mappings)
	pairs := make(currency.Pairs, len(mappings))
	for i := range mappings {
		pairs[i] = mappings[i].pair
	}
	require.NoError(t, ex.UpdatePairs(pairs, a, false), "UpdatePairs must not error for available pairs")
	require.NoError(t, ex.UpdatePairs(pairs, a, true), "UpdatePairs must not error for enabled pairs")
}

func TestLogDefaultError(t *testing.T) {
	t.Parallel()
	assert.NotPanics(t, func() { logDefaultError(nil) }, "logDefaultError should not panic for a nil error")
	assert.NotPanics(t, func() { logDefaultError(assert.AnError) }, "logDefaultError should not panic for an error")
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Hyperliquid", ex.Name, "SetDefaults should name the exchange")
	assert.True(t, ex.Enabled, "SetDefaults should enable the exchange")
	assert.Equal(t, currency.Currencies{currency.USDC}, ex.BaseCurrencies, "SetDefaults should use USDC as the base currency")
	assert.True(t, ex.API.CredentialsValidator.RequiresKey, "SetDefaults should require the account address as the key")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		format, err := ex.GetPairFormat(a, true)
		require.NoErrorf(t, err, "GetPairFormat must not error for %s", a)
		assert.Equalf(t, dashFormat, format, "SetDefaults should set the %s request format", a)
	}
	expREST := protocol.Features{
		TickerBatching:                 true,
		TickerFetching:                 true,
		KlineFetching:                  true,
		TradeFetching:                  true,
		OrderbookFetching:              true,
		AutoPairUpdates:                true,
		AccountBalance:                 true,
		CryptoWithdrawal:               true,
		DepositHistory:                 true,
		WithdrawalHistory:              true,
		GetOrder:                       true,
		GetOrders:                      true,
		CancelOrders:                   true,
		CancelOrder:                    true,
		SubmitOrder:                    true,
		ModifyOrder:                    true,
		TradeFee:                       true,
		FundingRateFetching:            true,
		AuthenticatedEndpoints:         true,
		HasAssetTypeAccountSegregation: true,
	}
	assert.Equal(t, expREST, ex.Features.Supports.RESTCapabilities, "SetDefaults should set the REST capabilities")
	expWebsocket := protocol.Features{
		TickerFetching:         true,
		KlineFetching:          true,
		TradeFetching:          true,
		OrderbookFetching:      true,
		Subscribe:              true,
		Unsubscribe:            true,
		GetOrders:              true,
		AuthenticatedEndpoints: true,
	}
	assert.Equal(t, expWebsocket, ex.Features.Supports.WebsocketCapabilities, "SetDefaults should set the websocket capabilities")
	expFutures := exchange.FuturesCapabilities{
		FundingRates:                    true,
		FundingRateBatching:             map[asset.Item]bool{asset.PerpetualContract: true},
		SupportedFundingRateFrequencies: map[kline.Interval]bool{kline.OneHour: true},
		Leverage:                        true,
		OpenInterest:                    exchange.OpenInterestSupport{Supported: true, SupportsRestBatch: true},
	}
	assert.Equal(t, expFutures, ex.Features.Supports.FuturesCapabilities, "SetDefaults should set the futures capabilities")
	assert.Equal(t, exchange.AutoWithdrawCryptoWithSetup, ex.Features.Supports.WithdrawPermissions, "SetDefaults should require withdrawal setup")
	assert.Equal(t, uint64(maximumCandleCount), ex.Features.Enabled.Kline.GlobalResultLimit, "SetDefaults should limit candles to Hyperliquid's retention")
	for kind, exp := range map[exchange.URL]string{exchange.RestSpot: apiURL, exchange.WebsocketSpot: websocketURL} {
		endpoint, err := ex.API.Endpoints.GetURL(kind)
		require.NoErrorf(t, err, "GetURL must not error for %s", kind)
		assert.Equalf(t, exp, endpoint, "SetDefaults should set the %s URL", kind)
	}
	_, err := ex.API.Endpoints.GetURL(exchange.RestFutures)
	assert.Error(t, err, "SetDefaults should not set a futures URL, as one host serves every market")
	assert.NotNil(t, ex.Requester, "SetDefaults should set the requester")
	assert.NotNil(t, ex.Websocket, "SetDefaults should set the websocket manager")
	assert.NotNil(t, ex.pairMappings, "SetDefaults should initialise the pair mappings")
	assert.NotNil(t, ex.pairMappingMisses, "SetDefaults should initialise the pair mapping misses")
	assert.NotNil(t, ex.websocketPending, "SetDefaults should initialise the pending websocket operations")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	newConfig := func(t *testing.T) (*Exchange, *config.Exchange) {
		t.Helper()
		ex := new(Exchange)
		ex.SetDefaults()
		cfg, err := ex.GetStandardConfig()
		require.NoError(t, err, "GetStandardConfig must not error")
		return ex, cfg
	}

	ex, cfg := newConfig(t)
	require.Error(t, ex.Setup(nil), "Setup must reject a nil configuration")
	cfg.Enabled = false
	require.NoError(t, ex.Setup(cfg), "Setup must not error for a disabled exchange")
	assert.False(t, ex.IsEnabled(), "Setup should leave a disabled exchange disabled")

	ex, cfg = newConfig(t)
	cfg.API.AuthenticatedSupport = true
	cfg.API.AuthenticatedWebsocketSupport = true
	cfg.API.Credentials.Key = testAccountAddress
	cfg.WebsocketResponseMaxLimit = 15 * time.Second
	require.NoError(t, ex.Setup(cfg), "Setup must not error")
	assert.True(t, ex.IsEnabled(), "Setup should enable the exchange")
	assert.Equal(t, 15*time.Second, ex.WebsocketResponseMaxLimit, "Setup should apply the configured websocket response limit to subscription acknowledgements")
	credentials, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "GetCredentials must not error")
	assert.Equal(t, testAccountAddress, credentials.Key, "Setup should load the configured account address")
	assert.True(t, ex.isMainnetEnvironment(), "Setup should sign for mainnet by default")
	assert.False(t, ex.Websocket.CanUseAuthenticatedEndpoints(), "Setup should leave account feeds unusable until the account connection is up")
	urls, err := ex.Websocket.GetConfiguredWebsocketURLs()
	require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
	assert.Equal(t, []string{websocketURL}, urls, "Setup should set the public and account connections up on the websocket URL")

	for _, tc := range []struct {
		name      string
		endpoints map[string]string
		exp       map[exchange.URL]string
	}{
		{
			name: "default endpoints",
			exp:  map[exchange.URL]string{exchange.RestSpot: testnetAPIURL, exchange.WebsocketSpot: testnetWebsocketURL},
		},
		{
			name:      "default endpoints with trailing slashes",
			endpoints: map[string]string{exchange.RestSpot.String(): apiURL + "/", exchange.WebsocketSpot.String(): websocketURL + "/"},
			exp:       map[exchange.URL]string{exchange.RestSpot: testnetAPIURL, exchange.WebsocketSpot: testnetWebsocketURL},
		},
		{
			name:      "custom gateway",
			endpoints: map[string]string{exchange.RestSpot.String(): "https://hyperliquid.internal.example", exchange.WebsocketSpot.String(): "wss://hyperliquid.internal.example/ws"},
			exp:       map[exchange.URL]string{exchange.RestSpot: "https://hyperliquid.internal.example", exchange.WebsocketSpot: "wss://hyperliquid.internal.example/ws"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			sandbox, sandboxConfig := newConfig(t)
			sandboxConfig.UseSandbox = true
			sandboxConfig.API.Endpoints = tc.endpoints
			require.NoError(t, sandbox.Setup(sandboxConfig), "Setup must not error in sandbox mode")
			assert.False(t, sandbox.isMainnetEnvironment(), "Setup should sign for testnet in sandbox mode")
			for kind, exp := range tc.exp {
				endpoint, err := sandbox.API.Endpoints.GetURL(kind)
				require.NoErrorf(t, err, "GetURL must not error for %s", kind)
				assert.Equalf(t, exp, endpoint, "Setup should select the %s URL", kind)
			}
			urls, err := sandbox.Websocket.GetConfiguredWebsocketURLs()
			require.NoError(t, err, "GetConfiguredWebsocketURLs must not error")
			assert.Equal(t, []string{tc.exp[exchange.WebsocketSpot]}, urls, "Setup should set the connections up on the selected websocket URL")
		})
	}

	ex, cfg = newConfig(t)
	cfg.API.Endpoints = map[string]string{exchange.RestSpot.String(): testnetAPIURL, exchange.WebsocketSpot.String(): testnetWebsocketURL}
	require.ErrorIs(t, ex.Setup(cfg), errEndpointEnvironment, "Setup must reject testnet URLs outside sandbox mode")

	ex, cfg = newConfig(t)
	cfg.API.Endpoints = map[string]string{"invalid": "https://example.com"}
	require.Error(t, ex.Setup(cfg), "Setup must reject an unknown endpoint")

	ex, cfg = newConfig(t)
	ex.API.Endpoints = ex.NewEndpoints()
	require.Error(t, ex.Setup(cfg), "Setup must error without endpoints")

	ex, cfg = newConfig(t)
	cfg.WebsocketTrafficTimeout = time.Millisecond
	require.Error(t, ex.Setup(cfg), "Setup must reject an invalid websocket traffic timeout")

	ex, cfg = newConfig(t)
	ex.Websocket.TrafficAlert = nil
	require.Error(t, ex.Setup(cfg), "Setup must error when the websocket connection cannot be set up")
}

func TestFetchTradablePairs(t *testing.T) {
	t.Parallel()
	_, err := e.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an unsupported asset")
	_, err = new(Exchange).FetchTradablePairs(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs must reject an asset without pair storage")
	configured := newTestExchange(t)
	require.NoError(t, configured.SetAssetPairStore(asset.Options, currency.PairStore{AssetEnabled: true, RequestFormat: &dashFormat, ConfigFormat: &dashFormat}), "SetAssetPairStore must register the extra asset")
	assert.True(t, configured.SupportsAsset(asset.Options), "SupportsAsset should recognise the configured asset")
	pairs, err := configured.FetchTradablePairs(t.Context(), asset.Options)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "FetchTradablePairs should reject a configured asset without a market implementation")
	assert.Nil(t, pairs, "FetchTradablePairs should return no pairs for the unimplemented asset")

	pairs, err = e.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "FetchTradablePairs must not error for perpetuals")
	if mockTests {
		exp := currency.Pairs{
			perpetualPair,
			currency.NewPair(currency.ETH, currency.USDC),
			currency.NewPair(currency.NewCode("HPOS"), currency.USDC),
			testBuilderPair,
			currency.NewPair(currency.NewCode("xyz:TSLA"), currency.USDC),
			currency.NewPair(currency.NewCode("flx:TSLA"), currency.NewCode("USDH")),
		}
		assert.Equal(t, exp, pairs, "FetchTradablePairs should list every active market of every perpetual DEX")
	} else {
		assert.Contains(t, pairs, perpetualPair, "FetchTradablePairs should list BTC-USDC")
	}

	pairs, err = e.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error for spot")
	if mockTests {
		exp := currency.Pairs{currency.NewPair(currency.NewCode("PURR"), currency.USDC), spotPair, currency.NewPair(currency.NewCode("USDH"), currency.USDC)}
		assert.Equal(t, exp, pairs, "FetchTradablePairs should list every spot market")
	} else {
		assert.Contains(t, pairs, spotPair, "FetchTradablePairs should list HYPE-USDC")
	}

	ambiguous := newInfoServerExchange(t, map[string]string{
		"meta":     `{"universe":[{"name":"BTC","szDecimals":5},{"name":"BTC","szDecimals":5},{"name":"ETH","szDecimals":4}]}`,
		"spotMeta": `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@108","index":108},{"tokens":[152,0],"name":"@109","index":109}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"HYPE","index":151},{"name":"PURR","index":152}]}`,
	}, nil)
	pairs, err = ambiguous.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "FetchTradablePairs must not error for an ambiguous perpetual display pair")
	assert.Equal(t, currency.Pairs{currency.NewPair(currency.ETH, currency.USDC)}, pairs, "FetchTradablePairs should skip an ambiguous perpetual display pair")
	_, ok := ambiguous.lookupPairMapping(perpetualPair, asset.PerpetualContract)
	assert.False(t, ok, "FetchTradablePairs should not map an ambiguous perpetual display pair")
	pairs, err = ambiguous.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "FetchTradablePairs must not error for an ambiguous spot display pair")
	assert.Equal(t, currency.Pairs{currency.NewPair(currency.NewCode("PURR"), currency.USDC)}, pairs, "FetchTradablePairs should skip an ambiguous spot display pair")
	mapping, ok := ambiguous.lookupPairMapping(pairs[0], asset.Spot)
	require.True(t, ok, "FetchTradablePairs must map an unambiguous spot pair")
	assert.Equal(t, pairMapping{pair: pairs[0], coin: "@109", assetID: 10109}, mapping, "FetchTradablePairs should map the unambiguous spot market")

	failed := newUnavailableServerExchange(t)
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		_, err = failed.FetchTradablePairs(t.Context(), a)
		assert.ErrorIsf(t, err, request.ErrBadStatus, "FetchTradablePairs should return a %s metadata failure", a)
	}
}

func TestFetchPerpetualPairMappings(t *testing.T) {
	t.Parallel()
	mappings, err := e.fetchPerpetualPairMappings(t.Context())
	require.NoError(t, err, "fetchPerpetualPairMappings must not error")
	e.pairMappingsMu.RLock()
	delisted := slices.Clone(e.delistedPairMappings)
	e.pairMappingsMu.RUnlock()
	if mockTests {
		assert.Equal(t, testPerpetualMappings, mappings, "fetchPerpetualPairMappings should map every active market with its DEX's asset ID offset and fees")
		assert.Equal(t, []pairMapping{testDelistedMapping}, delisted, "fetchPerpetualPairMappings should keep the delisted market")
	} else {
		assert.Contains(t, mappings, testPerpetualMapping, "fetchPerpetualPairMappings should map BTC")
		assert.True(t, slices.ContainsFunc(delisted, func(m pairMapping) bool { return m.coin == "MATIC" }), "fetchPerpetualPairMappings should keep delisted MATIC")
	}

	ex := newInfoServerExchange(t, map[string]string{
		"perpDexs": `[null,{"name":"xyz"}]`,
		"meta:":    `{"universe":[{"name":"BAD COIN"},{"name":"ETH","szDecimals":4,"maxLeverage":25},{"name":"ETH","szDecimals":2,"isDelisted":true},{"name":"MATIC","szDecimals":1,"maxLeverage":20,"isDelisted":true}]}`,
		"meta:xyz": `{"universe":[{"name":"xyz:XYZ100","szDecimals":4,"maxLeverage":30,"deployerFeeScale":"0.5"},{"name":"xyz:OLD","isDelisted":true}],"collateralToken":150}`,
		"spotMeta": `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`,
	}, nil)
	mappings, err = ex.fetchPerpetualPairMappings(t.Context())
	require.NoError(t, err, "fetchPerpetualPairMappings must not error for a builder DEX with spot collateral")
	exp := []pairMapping{
		{pair: currency.NewPair(currency.ETH, currency.USDC), coin: "ETH", assetID: 1, sizeDecimals: 4, maxLeverage: 25},
		{pair: currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.HYPE), coin: "xyz:XYZ100", dex: "xyz", assetID: 110000, sizeDecimals: 4, maxLeverage: 30, deployerFeeScale: 0.5},
	}
	assert.Equal(t, exp, mappings, "fetchPerpetualPairMappings should skip invalid names and delisted markets, and quote builder markets in their collateral, outside growth mode")
	exp = []pairMapping{
		{pair: currency.NewPair(currency.MATIC, currency.USDC), coin: "MATIC", assetID: 3, sizeDecimals: 1, maxLeverage: 20},
		{pair: currency.NewPair(currency.NewCode("xyz:OLD"), currency.HYPE), coin: "xyz:OLD", dex: "xyz", assetID: 110001},
	}
	assert.Equal(t, exp, ex.delistedPairMappings, "fetchPerpetualPairMappings should keep each delisted market whose coin no live market reuses")

	for _, tc := range []struct {
		name      string
		responses map[string]string
		err       error
	}{
		{name: "missing builder entry", responses: map[string]string{"perpDexs": `[null,null]`}, err: errInvalidPerpetualDEX},
		{name: "blank builder name", responses: map[string]string{"perpDexs": `[null,{"name":" "}]`}, err: errInvalidPerpetualDEX},
		{name: "duplicate builder name", responses: map[string]string{"perpDexs": `[null,{"name":"xyz"},{"name":"xyz"}]`}, err: errInvalidPerpetualDEX},
		{name: "unscoped builder market", responses: map[string]string{"perpDexs": `[null,{"name":"xyz"}]`, "meta": `{"universe":[{"name":"XYZ100"}]}`}, err: errInvalidPerpetualDEX},
		{
			name:      "too many builder markets",
			responses: map[string]string{"perpDexs": `[null,{"name":"xyz"}]`, "meta": `{"universe":[` + strings.Repeat(`{"name":"xyz:X"},`, builderPerpetualDEXAssetStride) + `{"name":"xyz:X"}]}`},
			err:       errInvalidPerpetualDEX,
		},
		{
			name:      "missing collateral token",
			responses: map[string]string{"meta": `{"universe":[],"collateralToken":150}`, "spotMeta": `{"tokens":[{"name":"USDC","index":0}]}`},
			err:       errSpotTokenNotFound,
		},
		{
			name:      "later DEX missing collateral token",
			responses: map[string]string{"perpDexs": `[null,{"name":"xyz"},{"name":"flx"}]`, "meta:": `{"universe":[]}`, "meta:xyz": `{"universe":[],"collateralToken":150}`, "meta:flx": `{"universe":[],"collateralToken":151}`, "spotMeta": `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`},
			err:       errSpotTokenNotFound,
		},
		{name: "invalid registry", responses: map[string]string{"perpDexs": `[]`}, err: errUnexpectedResponseLength},
		{name: "metadata failure", responses: map[string]string{"meta": `{`}},
		{name: "spot metadata failure", responses: map[string]string{"meta": `{"universe":[],"collateralToken":150}`, "spotMeta": `{`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := newInfoServerExchange(t, tc.responses, nil).fetchPerpetualPairMappings(t.Context())
			if tc.err == nil {
				assert.Error(t, err, "fetchPerpetualPairMappings should return a malformed response error")
				return
			}
			assert.ErrorIs(t, err, tc.err, "fetchPerpetualPairMappings should return the expected error")
		})
	}
}

func TestGetSpotTokenNames(t *testing.T) {
	t.Parallel()
	names, err := e.getSpotTokenNames(t.Context())
	require.NoError(t, err, "getSpotTokenNames must not error")
	if mockTests {
		exp := map[uint64]currency.Code{0: currency.USDC, 1: currency.NewCode("PURR"), 150: currency.HYPE, 253: currency.NewCode("RAT"), 360: currency.NewCode("USDH")}
		assert.Equal(t, exp, names, "getSpotTokenNames should map every token index to its name")
	} else {
		assert.Equal(t, currency.USDC, names[0], "getSpotTokenNames should map index 0 to USDC")
	}

	for _, tc := range []struct {
		name     string
		spotMeta string
		err      error
	}{
		{name: "duplicate index", spotMeta: `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":0}]}`, err: errMalformedResponse},
		{name: "blank name", spotMeta: `{"tokens":[{"name":"USDC","index":0},{"name":"","index":150}]}`, err: errSpotTokenNotFound},
		{name: "non-USDC index 0", spotMeta: `{"tokens":[{"name":"USDT","index":0}]}`, err: errSpotTokenNotFound},
	} {
		_, err := newInfoServerExchange(t, map[string]string{"spotMeta": tc.spotMeta}, nil).getSpotTokenNames(t.Context())
		assert.ErrorIsf(t, err, tc.err, "getSpotTokenNames should reject spot metadata with a %s", tc.name)
	}
	_, err = newUnavailableServerExchange(t).getSpotTokenNames(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "getSpotTokenNames should return a spot metadata failure")
}

func TestFetchSpotPairMappings(t *testing.T) {
	t.Parallel()
	mappings, err := e.fetchSpotPairMappings(t.Context())
	require.NoError(t, err, "fetchSpotPairMappings must not error")
	if mockTests {
		exp := []pairMapping{
			{pair: currency.NewPair(currency.NewCode("PURR"), currency.USDC), coin: "PURR/USDC", assetID: 10000},
			testSpotMapping,
			{pair: currency.NewPair(currency.NewCode("USDH"), currency.USDC), coin: "@230", assetID: 10230, sizeDecimals: 2},
		}
		assert.Equal(t, exp, mappings, "fetchSpotPairMappings should map every spot market")
	} else {
		assert.Contains(t, mappings, testSpotMapping, "fetchSpotPairMappings should map HYPE-USDC")
	}

	purr := pairMapping{pair: currency.NewPair(currency.NewCode("PURR"), currency.USDC), coin: "@2", assetID: 10002}
	for _, tc := range []struct {
		name     string
		spotMeta string
	}{
		{name: "one token", spotMeta: `{"universe":[{"tokens":[1],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"PURR","index":2}]}`},
		{name: "missing base token", spotMeta: `{"universe":[{"tokens":[1,0],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"PURR","index":2}]}`},
		{name: "missing quote token", spotMeta: `{"universe":[{"tokens":[1,9],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"TOKEN","index":1},{"name":"PURR","index":2}]}`},
		{name: "blank token name", spotMeta: `{"universe":[{"tokens":[1,0],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"","index":1},{"name":"PURR","index":2}]}`},
	} {
		mappings, err := newInfoServerExchange(t, map[string]string{"spotMeta": tc.spotMeta}, nil).fetchSpotPairMappings(t.Context())
		require.NoErrorf(t, err, "fetchSpotPairMappings must not error for a market with a %s", tc.name)
		assert.Equalf(t, []pairMapping{purr}, mappings, "fetchSpotPairMappings should skip a market with a %s", tc.name)
	}

	for _, tc := range []struct {
		name     string
		spotMeta string
	}{
		{name: "token index", spotMeta: `{"universe":[{"tokens":[150,0],"name":"@107","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":150}]}`},
		{name: "market index", spotMeta: `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@108","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":151}]}`},
		{name: "market name", spotMeta: `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@107","index":108}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":151}]}`},
	} {
		_, err := newInfoServerExchange(t, map[string]string{"spotMeta": tc.spotMeta}, nil).fetchSpotPairMappings(t.Context())
		assert.ErrorIsf(t, err, errMalformedResponse, "fetchSpotPairMappings should reject a duplicate %s", tc.name)
	}
	_, err = newUnavailableServerExchange(t).fetchSpotPairMappings(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "fetchSpotPairMappings should return a spot metadata failure")
}

func TestUpdateTradablePairs(t *testing.T) {
	t.Parallel()
	require.NoError(t, e.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must not error")
	exp := map[asset.Item]currency.Pairs{
		asset.PerpetualContract: {
			perpetualPair,
			currency.NewPair(currency.ETH, currency.USDC),
			currency.NewPair(currency.NewCode("HPOS"), currency.USDC),
			testBuilderPair,
			currency.NewPair(currency.NewCode("xyz:TSLA"), currency.USDC),
			currency.NewPair(currency.NewCode("flx:TSLA"), currency.NewCode("USDH")),
		},
		asset.Spot: {currency.NewPair(currency.NewCode("PURR"), currency.USDC), spotPair, currency.NewPair(currency.NewCode("USDH"), currency.USDC)},
	}
	for a, testPair := range map[asset.Item]currency.Pair{asset.PerpetualContract: perpetualPair, asset.Spot: spotPair} {
		available, err := e.GetAvailablePairs(a)
		require.NoErrorf(t, err, "GetAvailablePairs must not error for %s", a)
		if mockTests {
			assert.Equalf(t, exp[a].Format(dashFormat), available, "UpdateTradablePairs should make every %s market available", a)
		} else {
			assert.Truef(t, available.Contains(testPair, true), "UpdateTradablePairs should make %s available", testPair)
		}
	}

	require.ErrorIs(t, newUnavailableServerExchange(t).UpdateTradablePairs(t.Context()), request.ErrBadStatus, "UpdateTradablePairs must return a metadata failure")
	empty := newInfoServerExchange(t, map[string]string{"meta": `{"universe":[]}`, "spotMeta": `{"universe":[],"tokens":[]}`}, nil)
	require.Error(t, empty.UpdateTradablePairs(t.Context()), "UpdateTradablePairs must error when no pair can be enabled")
}

func TestGetPerpetualDEXNames(t *testing.T) {
	t.Parallel()
	names, err := e.getPerpetualDEXNames(t.Context())
	require.NoError(t, err, "getPerpetualDEXNames must not error")
	if mockTests {
		assert.Equal(t, []string{"", "xyz", "flx"}, names, "getPerpetualDEXNames should list every DEX in registry order")
	} else {
		assert.Empty(t, names[0], "getPerpetualDEXNames should list the default DEX first")
	}

	names, err = newInfoServerExchange(t, map[string]string{"perpDexs": `[null,{"name":" xyz "},{"name":"hyna"}]`}, nil).getPerpetualDEXNames(t.Context())
	require.NoError(t, err, "getPerpetualDEXNames must not error for padded names")
	assert.Equal(t, []string{"", "xyz", "hyna"}, names, "getPerpetualDEXNames should trim DEX names")

	for _, tc := range []struct {
		registry string
		err      error
	}{
		{registry: `[]`, err: errUnexpectedResponseLength},
		{registry: `[{"name":"default"}]`, err: errUnexpectedResponseLength},
		{registry: `[null,null]`, err: errInvalidPerpetualDEX},
		{registry: `[null,{"name":" "}]`, err: errInvalidPerpetualDEX},
		{registry: `[null,{"name":"xyz"},{"name":" xyz "}]`, err: errInvalidPerpetualDEX},
	} {
		_, err := newInfoServerExchange(t, map[string]string{"perpDexs": tc.registry}, nil).getPerpetualDEXNames(t.Context())
		assert.ErrorIsf(t, err, tc.err, "getPerpetualDEXNames should reject registry %s", tc.registry)
	}
	_, err = newUnavailableServerExchange(t).getPerpetualDEXNames(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "getPerpetualDEXNames should return a registry failure")
}

func TestSetPairMappings(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	mappings := []pairMapping{testSpotMapping}
	ex.setPairMappings(asset.Spot, mappings)
	mappings[0].coin = "changed"
	mapping, ok := ex.lookupPairMapping(spotPair, asset.Spot)
	require.True(t, ok, "lookupPairMapping must find the stored mapping")
	assert.Equal(t, testSpotMapping, mapping, "setPairMappings should store a copy of the mappings")
}

func TestGetCoin(t *testing.T) {
	t.Parallel()
	coin, err := e.getCoin(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "getCoin must not error")
	assert.Equal(t, "BTC", coin, "getCoin should return the market's coin")
	coin, err = e.getCoin(t.Context(), spotPair, asset.Spot)
	require.NoError(t, err, "getCoin must not error for spot")
	assert.Equal(t, "@107", coin, "getCoin should return the spot market's coin")
	_, err = e.getCoin(t.Context(), currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "getCoin should reject an empty pair")
}

func TestGetPairMapping(t *testing.T) {
	t.Parallel()
	_, err := e.getPairMapping(t.Context(), perpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "getPairMapping must reject an unsupported asset")
	_, err = e.getPairMapping(t.Context(), currency.EMPTYPAIR, asset.PerpetualContract)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "getPairMapping must reject an empty pair")

	mapping, err := e.getPairMapping(t.Context(), testBuilderPair, asset.PerpetualContract)
	require.NoError(t, err, "getPairMapping must not error for a builder DEX market")
	if mockTests {
		assert.Equal(t, testBuilderMapping, mapping, "getPairMapping should fetch the builder DEX market's mapping")
	} else {
		assert.Equal(t, "xyz", mapping.dex, "getPairMapping should return the builder DEX market's DEX")
	}

	ex := new(Exchange)
	ex.SetDefaults()
	ex.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	mapping, err = ex.getPairMapping(t.Context(), spotPair, asset.Spot)
	require.NoError(t, err, "getPairMapping must not error for a cached mapping")
	assert.Equal(t, testSpotMapping, mapping, "getPairMapping should return a cached mapping without fetching")

	_, err = e.getPairMapping(t.Context(), testDelistedMapping.pair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errMarketDelisted, "getPairMapping should refuse a delisted market")
	delisted := newInfoServerExchange(t, map[string]string{
		"meta":     `{"universe":[{"name":"BTC","szDecimals":5},{"name":"MATIC","szDecimals":1,"isDelisted":true}]}`,
		"spotMeta": `{"universe":[],"tokens":[{"name":"USDC","index":0}]}`,
	}, nil)
	_, err = delisted.getPairMapping(t.Context(), testDelistedMapping.pair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errMarketDelisted, "getPairMapping should refuse a delisted market's pair")
	assert.ErrorIs(t, err, errPairMappingNotFound, "getPairMapping should report a delisted market's pair as unmapped")
	for a, p := range map[asset.Item]currency.Pair{asset.PerpetualContract: currency.NewPair(currency.ETH, currency.USDC), asset.Spot: testDelistedMapping.pair} {
		_, err = delisted.getPairMapping(t.Context(), p, a)
		assert.ErrorIsf(t, err, errPairMappingNotFound, "getPairMapping should report an unknown %s pair", a)
		assert.NotErrorIsf(t, err, errMarketDelisted, "getPairMapping should not report an unknown %s pair as delisted", a)
	}
	_, err = newUnavailableServerExchange(t).getPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "getPairMapping should return a metadata failure")
}

func TestLookupPairMapping(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	_, ok := ex.lookupPairMapping(spotPair, asset.Spot)
	assert.False(t, ok, "lookupPairMapping should not find an uncached mapping")
	ex.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	mapping, ok := ex.lookupPairMapping(spotPair, asset.Spot)
	require.True(t, ok, "lookupPairMapping must find a cached mapping")
	assert.Equal(t, testSpotMapping, mapping, "lookupPairMapping should return the cached mapping")
	_, ok = ex.lookupPairMapping(spotPair, asset.PerpetualContract)
	assert.False(t, ok, "lookupPairMapping should not find a mapping of another asset")
}

func TestFetchPairMapping(t *testing.T) {
	t.Parallel()
	var fetches atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		fetches.Add(1)
		response := `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40}]}`
		if body.Type == "perpDexs" {
			response = `[null]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	mapping, err := ex.fetchPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "fetchPairMapping must not error for a mapping cached while waiting")
	assert.Equal(t, testPerpetualMapping, mapping, "fetchPairMapping should return a mapping cached while waiting")
	assert.Zero(t, fetches.Load(), "fetchPairMapping should not fetch a cached mapping")

	ex.setPairMappings(asset.PerpetualContract, nil)
	mapping, err = ex.fetchPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "fetchPairMapping must not error")
	assert.Equal(t, testPerpetualMapping, mapping, "fetchPairMapping should fetch the market's mapping")

	missing := currency.NewPair(currency.ETH, currency.USDC)
	_, err = ex.fetchPairMapping(t.Context(), missing, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "fetchPairMapping must report a missing market")
	fetched := fetches.Load()
	_, err = ex.fetchPairMapping(t.Context(), missing, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "fetchPairMapping must report a cached missing market")
	assert.Equal(t, fetched, fetches.Load(), "fetchPairMapping should not refetch a recently missing market")
	ex.pairMappingMisses["pair:"+asset.PerpetualContract.String()+":"+strings.ToLower(missing.String())] = time.Now().Add(-time.Second)
	_, err = ex.fetchPairMapping(t.Context(), missing, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "fetchPairMapping must report a missing market after its miss expires")
	assert.Greater(t, fetches.Load(), fetched, "fetchPairMapping should refetch once a miss expires")

	_, err = newUnavailableServerExchange(t).fetchPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "fetchPairMapping should return a metadata failure")
}

func TestGetPairMappingByCoin(t *testing.T) {
	t.Parallel()
	_, _, err := e.getPairMappingByCoin(t.Context(), " ")
	require.ErrorIs(t, err, errCoinRequired, "getPairMappingByCoin must require a coin")

	mapping, a, err := e.getPairMappingByCoin(t.Context(), "@107")
	require.NoError(t, err, "getPairMappingByCoin must not error")
	assert.Equal(t, asset.Spot, a, "getPairMappingByCoin should return the coin's asset")
	if mockTests {
		assert.Equal(t, testSpotMapping, mapping, "getPairMappingByCoin should return the coin's mapping")
	} else {
		assert.Equal(t, spotPair, mapping.pair, "getPairMappingByCoin should return the coin's pair")
	}

	mapping, a, err = e.getPairMappingByCoin(t.Context(), "MATIC")
	require.NoError(t, err, "getPairMappingByCoin must not error for a delisted market's coin")
	assert.Equal(t, asset.PerpetualContract, a, "getPairMappingByCoin should return a delisted market's asset")
	assert.Equal(t, testDelistedMapping.pair, mapping.pair, "getPairMappingByCoin should return a delisted market's pair")

	ex := new(Exchange)
	ex.SetDefaults()
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	ex.setPairMappings(asset.Spot, []pairMapping{{pair: spotPair, coin: "BTC"}})
	_, _, err = ex.getPairMappingByCoin(t.Context(), "BTC")
	assert.ErrorIs(t, err, errAmbiguousCoinMapping, "getPairMappingByCoin should reject a cached ambiguous coin without fetching")
}

func TestLookupPairMappingByCoin(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	_, _, err := ex.lookupPairMappingByCoin("BTC")
	require.ErrorIs(t, err, errPairMappingNotFound, "lookupPairMappingByCoin must report an uncached coin")
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	mapping, a, err := ex.lookupPairMappingByCoin("BTC")
	require.NoError(t, err, "lookupPairMappingByCoin must not error")
	assert.Equal(t, testPerpetualMapping, mapping, "lookupPairMappingByCoin should return the coin's mapping")
	assert.Equal(t, asset.PerpetualContract, a, "lookupPairMappingByCoin should return the coin's asset")
	ex.delistedPairMappings = []pairMapping{testDelistedMapping, {pair: perpetualPair, coin: "BTC", assetID: 9}}
	mapping, _, err = ex.lookupPairMappingByCoin("BTC")
	require.NoError(t, err, "lookupPairMappingByCoin must not error for a coin a live and a delisted market use")
	assert.Equal(t, testPerpetualMapping, mapping, "lookupPairMappingByCoin should prefer the live market to a delisted one")
	mapping, a, err = ex.lookupPairMappingByCoin("MATIC")
	require.NoError(t, err, "lookupPairMappingByCoin must not error for a delisted market's coin")
	assert.Equal(t, testDelistedMapping, mapping, "lookupPairMappingByCoin should return a delisted market's mapping")
	assert.Equal(t, asset.PerpetualContract, a, "lookupPairMappingByCoin should return a delisted market's asset")
	ex.delistedPairMappings = append(ex.delistedPairMappings, testDelistedMapping)
	_, _, err = ex.lookupPairMappingByCoin("MATIC")
	assert.ErrorIs(t, err, errAmbiguousCoinMapping, "lookupPairMappingByCoin should reject a coin two delisted markets use")
	ex.setPairMappings(asset.Spot, []pairMapping{{pair: spotPair, coin: "BTC"}})
	_, _, err = ex.lookupPairMappingByCoin("BTC")
	assert.ErrorIs(t, err, errAmbiguousCoinMapping, "lookupPairMappingByCoin should reject a coin mapped by both assets")
}

func TestFetchPairMappingByCoin(t *testing.T) {
	t.Parallel()
	cached := new(Exchange)
	cached.SetDefaults()
	cached.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	mapping, a, err := cached.fetchPairMappingByCoin(t.Context(), "BTC")
	require.NoError(t, err, "fetchPairMappingByCoin must not error for a coin cached while waiting")
	assert.Equal(t, testPerpetualMapping, mapping, "fetchPairMappingByCoin should return a coin cached while waiting")
	assert.Equal(t, asset.PerpetualContract, a, "fetchPairMappingByCoin should return the cached coin's asset")

	_, _, err = new(Exchange).fetchPairMappingByCoin(t.Context(), "BTC")
	require.ErrorIs(t, err, errPairMappingNotFound, "fetchPairMappingByCoin must report a coin when no asset is supported")

	var fetches atomic.Int32
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		fetches.Add(1)
		var response string
		switch body.Type {
		case "perpDexs":
			response = `[null]`
		case "meta":
			response = `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40}]}`
		case "spotMeta":
			response = `{"universe":[{"tokens":[150,0],"name":"@107","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150,"szDecimals":2}]}`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	mapping, a, err = ex.fetchPairMappingByCoin(t.Context(), "BTC")
	require.NoError(t, err, "fetchPairMappingByCoin must not error")
	assert.Equal(t, testPerpetualMapping, mapping, "fetchPairMappingByCoin should fetch the coin's mapping")
	assert.Equal(t, asset.PerpetualContract, a, "fetchPairMappingByCoin should return the fetched coin's asset")

	// Each refetch maps every coin, so coins missing within a minute of it wait for the next one
	fetched := fetches.Load()
	coins := []string{"MISSING", "#90700", "OTHER"}
	for _, coin := range coins {
		_, _, err = ex.fetchPairMappingByCoin(t.Context(), coin)
		require.ErrorIsf(t, err, errPairMappingNotFound, "fetchPairMappingByCoin must report missing coin %s", coin)
	}
	assert.Equal(t, fetched, fetches.Load(), "fetchPairMappingByCoin should not refetch for coins missing within a minute of a refetch")
	assert.Equal(t, ex.pairMappingsRefreshed.Add(pairMappingMissCacheDuration), ex.pairMappingMisses["coin:missing"], "fetchPairMappingByCoin should cache a miss until the next refetch is due")
	ex.pairMappingMisses["coin:missing"] = time.Now().Add(-time.Second)
	_, _, err = ex.fetchPairMappingByCoin(t.Context(), "MISSING")
	require.ErrorIs(t, err, errPairMappingNotFound, "fetchPairMappingByCoin must report a missing coin after its miss expires")
	assert.Equal(t, fetched, fetches.Load(), "fetchPairMappingByCoin should not refetch within a minute of a refetch once a miss expires")
	ex.pairMappingsRefreshed = time.Now().Add(-pairMappingMissCacheDuration)
	clear(ex.pairMappingMisses)
	for _, coin := range coins {
		_, _, err = ex.fetchPairMappingByCoin(t.Context(), coin)
		require.ErrorIsf(t, err, errPairMappingNotFound, "fetchPairMappingByCoin must report missing coin %s once a refetch is due", coin)
	}
	assert.Equal(t, fetched+3, fetches.Load(), "fetchPairMappingByCoin should refetch the registry, perpetual and spot metadata once for every missing coin")

	_, _, err = newUnavailableServerExchange(t).fetchPairMappingByCoin(t.Context(), "BTC")
	assert.ErrorIs(t, err, request.ErrBadStatus, "fetchPairMappingByCoin should return a metadata failure")
}

func TestIsCachedPairMappingMiss(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.False(t, ex.isCachedPairMappingMiss("pair:spot:missing"), "isCachedPairMappingMiss should report an unknown key as uncached")
	ex.pairMappingMisses = map[string]time.Time{"pair:spot:missing": time.Now().Add(time.Minute)}
	assert.True(t, ex.isCachedPairMappingMiss("pair:spot:missing"), "isCachedPairMappingMiss should report a recent miss as cached")
	ex.pairMappingMisses["pair:spot:missing"] = time.Now().Add(-time.Second)
	assert.False(t, ex.isCachedPairMappingMiss("pair:spot:missing"), "isCachedPairMappingMiss should report an expired miss as uncached")
	assert.NotContains(t, ex.pairMappingMisses, "pair:spot:missing", "isCachedPairMappingMiss should drop an expired miss")
}

func TestCachePairMappingMiss(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	expiry := time.Now().Add(pairMappingMissCacheDuration)
	ex.cachePairMappingMiss("pair:spot:missing", expiry)
	require.Contains(t, ex.pairMappingMisses, "pair:spot:missing", "cachePairMappingMiss must record the miss")
	assert.Equal(t, expiry, ex.pairMappingMisses["pair:spot:missing"], "cachePairMappingMiss should expire the miss when told")
}

func TestGetPerpetualPairMappings(t *testing.T) {
	t.Parallel()
	mappings, err := e.getPerpetualPairMappings(t.Context())
	require.NoError(t, err, "getPerpetualPairMappings must not error")
	if mockTests {
		assert.Equal(t, testPerpetualMappings, mappings, "getPerpetualPairMappings should return every active perpetual market")
	} else {
		assert.Contains(t, mappings, testPerpetualMapping, "getPerpetualPairMappings should include BTC")
	}

	ex := new(Exchange)
	ex.SetDefaults()
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testBuilderMapping})
	mappings, err = ex.getPerpetualPairMappings(t.Context())
	require.NoError(t, err, "getPerpetualPairMappings must not error for cached mappings")
	assert.Equal(t, []pairMapping{testBuilderMapping}, mappings, "getPerpetualPairMappings should return cached mappings without fetching")

	_, err = newUnavailableServerExchange(t).getPerpetualPairMappings(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "getPerpetualPairMappings should return a metadata failure")
}

func TestGetPerpetualAssetContexts(t *testing.T) {
	t.Parallel()
	contexts, err := e.getPerpetualAssetContexts(t.Context(), []pairMapping{testPerpetualMapping, testBuilderMapping})
	require.NoError(t, err, "getPerpetualAssetContexts must not error")
	require.Len(t, contexts, 2, "getPerpetualAssetContexts must return a context per mapping")
	if mockTests {
		exp := []PerpetualAssetContext{
			{Funding: 0.0000125, OpenInterest: 39310.43958, PreviousDayPrice: 86248, DayNotionalVolume: 1820651593.0427696705, Premium: -0.0002691175, OraclePrice: 85836.1, MarkPrice: 85810.5, MidPrice: 85812.5, ImpactPrices: []types.Number{85812, 85813}, DayBaseVolume: 21219.11844},
			{Funding: 0.00000625, OpenInterest: 7148.95, PreviousDayPrice: 30759, DayNotionalVolume: 275526916.1308999658, Premium: 0.0000645887, OraclePrice: 31120, MarkPrice: 31123, MidPrice: 31122.5, ImpactPrices: []types.Number{31121.02, 31123}, DayBaseVolume: 8893.0929},
		}
		assert.Equal(t, exp, contexts, "getPerpetualAssetContexts should return each market's context from its DEX")
	} else {
		assert.Positive(t, contexts[0].MarkPrice.Float64(), "getPerpetualAssetContexts should return BTC's mark price")
	}

	var requested []string
	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		requested = append(requested, body.DEX)
		response := `[{"universe":[{"name":"BTC"},{"name":"OLD","isDelisted":true}]},[{"funding":"0.0001"},{"funding":"0.0002"}]]`
		if body.DEX == "xyz" {
			response = `[{"universe":[{"name":"xyz:XYZ100"}]},[{"funding":"0.0003"}]]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	contexts, err = ex.getPerpetualAssetContexts(t.Context(), []pairMapping{testPerpetualMapping, testBuilderMapping, testPerpetualMapping})
	require.NoError(t, err, "getPerpetualAssetContexts must not error for repeated DEXs")
	assert.Equal(t, []PerpetualAssetContext{{Funding: 0.0001}, {Funding: 0.0003}, {Funding: 0.0001}}, contexts, "getPerpetualAssetContexts should return each market's context")
	assert.Equal(t, []string{"", "xyz"}, requested, "getPerpetualAssetContexts should fetch each DEX once")
	_, err = ex.getPerpetualAssetContexts(t.Context(), []pairMapping{{pair: perpetualPair, coin: "OLD"}})
	assert.ErrorIs(t, err, errAssetContextNotFound, "getPerpetualAssetContexts should not return a delisted market's context")
	_, err = newUnavailableServerExchange(t).getPerpetualAssetContexts(t.Context(), []pairMapping{testPerpetualMapping})
	assert.ErrorIs(t, err, request.ErrBadStatus, "getPerpetualAssetContexts should return a context failure")
}

// expMockTicker returns the ticker the mock responses' context gives the perpetual or spot test pair
func expMockTicker(exchangeName string, a asset.Item, lastUpdated time.Time) *ticker.Price {
	if a == asset.Spot {
		return &ticker.Price{
			Last:         93.1465,
			Open:         93.102,
			BaseVolume:   677650.0300000001,
			QuoteVolume:  63321924.1126599833,
			MarkPrice:    93.146,
			Pair:         spotPair.Format(dashFormat),
			ExchangeName: exchangeName,
			AssetType:    asset.Spot,
			LastUpdated:  lastUpdated,
		}
	}
	return &ticker.Price{
		Last:         85812.5,
		Open:         86248,
		BaseVolume:   21219.11844,
		QuoteVolume:  1820651593.0427696705,
		OpenInterest: 39310.43958,
		MarkPrice:    85810.5,
		IndexPrice:   85836.1,
		Pair:         perpetualPair.Format(dashFormat),
		ExchangeName: exchangeName,
		AssetType:    asset.PerpetualContract,
		LastUpdated:  lastUpdated,
	}
}

func TestUpdateTickers(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers must reject an unsupported asset")
	require.ErrorIs(t, new(Exchange).UpdateTickers(t.Context(), asset.Spot), asset.ErrNotSupported, "UpdateTickers must reject an asset without pair storage")
	configured := newInfoServerExchange(t, nil, nil)
	require.NoError(t, configured.SetAssetPairStore(asset.Options, currency.PairStore{AssetEnabled: true, RequestFormat: &dashFormat, ConfigFormat: &dashFormat}), "SetAssetPairStore must register the extra asset")
	setTestPairs(t, configured, asset.Options, testPerpetualMapping)
	assert.True(t, configured.SupportsAsset(asset.Options), "SupportsAsset should recognise the configured asset")
	assert.ErrorIs(t, configured.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "UpdateTickers should reject a configured asset without a ticker implementation")

	for _, a := range ex.GetAssetTypes(false) {
		require.NoErrorf(t, ex.UpdateTickers(t.Context(), a), "UpdateTickers must not error for %s", a)
	}
	perpetual, err := ticker.GetTicker(ex.Name, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetTicker must not error for the perpetual ticker")
	spot, err := ticker.GetTicker(ex.Name, spotPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error for the spot ticker")
	if mockTests {
		assert.Equal(t, expMockTicker(ex.Name, asset.PerpetualContract, perpetual.LastUpdated), perpetual, "UpdateTickers should store the perpetual ticker from its context")
		assert.Equal(t, expMockTicker(ex.Name, asset.Spot, spot.LastUpdated), spot, "UpdateTickers should store the spot ticker from its context")
	} else {
		assert.Positive(t, perpetual.MarkPrice, "UpdateTickers should store the perpetual mark price")
		assert.Positive(t, spot.MarkPrice, "UpdateTickers should store the spot mark price")
	}
	assert.WithinDuration(t, time.Now(), perpetual.LastUpdated, time.Minute, "UpdateTickers should time the perpetual ticker when it is fetched")

	var requested []string
	builder := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		requested = append(requested, body.DEX)
		response := `[{"universe":[{"name":"BTC"}]},[{"openInterest":"10","prevDayPx":"99","dayNtlVlm":"1000","oraclePx":"100","markPx":"101","midPx":null,"dayBaseVlm":"10"}]]`
		if body.DEX == "xyz" {
			response = `[{"universe":[{"name":"xyz:XYZ100"}]},[{"openInterest":"20","prevDayPx":"49","dayNtlVlm":"2000","oraclePx":"50","markPx":"51","midPx":"50.5","dayBaseVlm":"40"}]]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	builder.Name = t.Name()
	setTestPairs(t, builder, asset.PerpetualContract, testPerpetualMapping, testBuilderMapping)
	require.NoError(t, builder.UpdateTickers(t.Context(), asset.PerpetualContract), "UpdateTickers must not error across DEXs")
	assert.Equal(t, []string{"", "xyz"}, requested, "UpdateTickers should fetch each DEX's contexts once")
	for _, exp := range []*ticker.Price{
		{Last: 101, Open: 99, BaseVolume: 10, QuoteVolume: 1000, OpenInterest: 10, MarkPrice: 101, IndexPrice: 100, Pair: perpetualPair.Format(dashFormat), ExchangeName: builder.Name, AssetType: asset.PerpetualContract},
		{Last: 50.5, Open: 49, BaseVolume: 40, QuoteVolume: 2000, OpenInterest: 20, MarkPrice: 51, IndexPrice: 50, Pair: testBuilderPair.Format(dashFormat), ExchangeName: builder.Name, AssetType: asset.PerpetualContract},
	} {
		price, err := ticker.GetTicker(builder.Name, exp.Pair, asset.PerpetualContract)
		require.NoErrorf(t, err, "GetTicker must not error for %s", exp.Pair)
		exp.LastUpdated = price.LastUpdated
		assert.Equalf(t, exp, price, "UpdateTickers should store %s, using the mark price without a mid price", exp.Pair)
	}

	spotContexts := `[{"universe":[],"tokens":[]},[{"prevDayPx":"9","dayNtlVlm":"100","markPx":"10","midPx":null,"coin":"@107","dayBaseVlm":"10"},{"coin":"#10","markPx":"0.5"}]]`
	spotExchange := newInfoServerExchange(t, map[string]string{"spotMetaAndAssetCtxs": spotContexts}, nil)
	setTestPairs(t, spotExchange, asset.Spot, testSpotMapping)
	require.NoError(t, spotExchange.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers must not error for spot contexts outside the universe")
	price, err := ticker.GetTicker(spotExchange.Name, spotPair, asset.Spot)
	require.NoError(t, err, "GetTicker must not error for the spot ticker")
	exp := &ticker.Price{Last: 10, Open: 9, BaseVolume: 10, QuoteVolume: 100, MarkPrice: 10, Pair: spotPair.Format(dashFormat), ExchangeName: spotExchange.Name, AssetType: asset.Spot, LastUpdated: price.LastUpdated}
	assert.Equal(t, exp, price, "UpdateTickers should match spot contexts by coin and use the mark price without a mid price")

	perpetualContexts := `[{"universe":[{"name":"BTC"}]},[{"markPx":"101"}]]`
	for _, tc := range []struct {
		name      string
		a         asset.Item
		responses map[string]string
		mappings  []pairMapping
		err       error
	}{
		{name: "mismatched perpetual contexts", a: asset.PerpetualContract, responses: map[string]string{"metaAndAssetCtxs": `[{"universe":[{"name":"BTC"}]},[]]`}, mappings: []pairMapping{testPerpetualMapping}, err: errUnexpectedResponseLength},
		{name: "missing perpetual context", a: asset.PerpetualContract, responses: map[string]string{"metaAndAssetCtxs": perpetualContexts}, mappings: []pairMapping{{pair: perpetualPair, coin: "ETH"}}, err: errAssetContextNotFound},
		{name: "missing spot context", a: asset.Spot, responses: map[string]string{"spotMetaAndAssetCtxs": spotContexts}, mappings: []pairMapping{{pair: spotPair, coin: "@108"}}, err: errAssetContextNotFound},
	} {
		ex := newInfoServerExchange(t, tc.responses, nil)
		setTestPairs(t, ex, tc.a, tc.mappings...)
		assert.ErrorIsf(t, ex.UpdateTickers(t.Context(), tc.a), tc.err, "UpdateTickers should report a %s", tc.name)
	}

	for a, responses := range map[asset.Item]map[string]string{
		asset.PerpetualContract: {"metaAndAssetCtxs": perpetualContexts, "meta": `{"universe":[{"name":"ETH"}]}`},
		asset.Spot:              {"spotMetaAndAssetCtxs": spotContexts, "spotMeta": `{"universe":[],"tokens":[]}`},
	} {
		ex := newInfoServerExchange(t, responses, nil)
		assert.ErrorIsf(t, ex.UpdateTickers(t.Context(), a), errPairMappingNotFound, "UpdateTickers should report a %s pair without a market", a)
	}

	mixed := newInfoServerExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContexts}, nil)
	ethPair := currency.NewPair(currency.ETH, currency.USDC)
	setTestPairs(t, mixed, asset.PerpetualContract, testPerpetualMapping, pairMapping{pair: ethPair, coin: "ETH"})
	assert.ErrorIs(t, mixed.UpdateTickers(t.Context(), asset.PerpetualContract), errAssetContextNotFound, "UpdateTickers should report a missing context in a batch")
	price, err = ticker.GetTicker(mixed.Name, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetTicker must not error for the batch's valid ticker")
	assert.Equal(t, 101.0, price.Last, "UpdateTickers should store the batch's valid tickers")

	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		unnamed := newInfoServerExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContexts, "spotMetaAndAssetCtxs": spotContexts}, nil)
		setTestPairs(t, unnamed, asset.Spot, testSpotMapping)
		setTestPairs(t, unnamed, asset.PerpetualContract, testPerpetualMapping)
		unnamed.Name = ""
		assert.ErrorIsf(t, unnamed.UpdateTickers(t.Context(), a), common.ErrExchangeNameNotSet, "UpdateTickers should return a %s ticker processing error", a)

		failed := newUnavailableServerExchange(t)
		setTestPairs(t, failed, a, map[asset.Item]pairMapping{asset.Spot: testSpotMapping, asset.PerpetualContract: testPerpetualMapping}[a])
		assert.ErrorIsf(t, failed.UpdateTickers(t.Context(), a), request.ErrBadStatus, "UpdateTickers should return a %s context failure", a)
	}

	disabled := newInfoServerExchange(t, nil, nil)
	require.NoError(t, disabled.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
	assert.Error(t, disabled.UpdateTickers(t.Context(), asset.Spot), "UpdateTickers should error for a disabled asset")
}

func TestUpdateTicker(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateTicker(t.Context(), spotPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateTicker must reject an unsupported asset")
	_, err = ex.UpdateTicker(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "UpdateTicker must reject an empty pair")

	for a, pair := range map[asset.Item]currency.Pair{asset.PerpetualContract: perpetualPair, asset.Spot: spotPair} {
		price, err := ex.UpdateTicker(t.Context(), pair.Format(dashFormat), a)
		require.NoErrorf(t, err, "UpdateTicker must not error for %s", a)
		if mockTests {
			assert.Equalf(t, expMockTicker(ex.Name, a, price.LastUpdated), price, "UpdateTicker should return the %s ticker from its context", a)
		} else {
			assert.Positivef(t, price.Last, "UpdateTicker should return the %s ticker's last price", a)
		}
	}
	purr := currency.NewPair(currency.NewCode("PURR"), currency.USDC)
	price, err := ex.UpdateTicker(t.Context(), purr, asset.Spot)
	require.NoError(t, err, "UpdateTicker must not error for an available pair that is not enabled")
	if mockTests {
		exp := &ticker.Price{Last: 0.150495, Open: 0.16577, BaseVolume: 14727160, QuoteVolume: 2361343.4740200015, MarkPrice: 0.15049, Pair: purr, ExchangeName: ex.Name, AssetType: asset.Spot, LastUpdated: price.LastUpdated}
		assert.Equal(t, exp, price, "UpdateTicker should return the ticker of an available pair that is not enabled")
	} else {
		assert.Positive(t, price.MarkPrice, "UpdateTicker should return the ticker of an available pair that is not enabled")
	}
	ethPair := currency.NewPair(currency.ETH, currency.USDC)
	_, err = ex.UpdateTicker(t.Context(), ethPair, asset.Spot)
	assert.ErrorIs(t, err, errPairMappingNotFound, "UpdateTicker should report a pair without a market")

	sibling := newInfoServerExchange(t, map[string]string{"metaAndAssetCtxs": `[{"universe":[{"name":"BTC"}]},[{"markPx":"101","midPx":"100.5"}]]`}, nil)
	setTestPairs(t, sibling, asset.PerpetualContract, testPerpetualMapping, pairMapping{pair: ethPair, coin: "ETH"})
	price, err = sibling.UpdateTicker(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "UpdateTicker must not error when another enabled pair has no context")
	exp := &ticker.Price{Last: 100.5, MarkPrice: 101, Pair: perpetualPair, ExchangeName: sibling.Name, AssetType: asset.PerpetualContract, LastUpdated: price.LastUpdated}
	assert.Equal(t, exp, price, "UpdateTicker should return the requested ticker regardless of its siblings")
	_, err = sibling.UpdateTicker(t.Context(), ethPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errAssetContextNotFound, "UpdateTicker should report a perpetual pair without a context")

	spotContexts := newInfoServerExchange(t, map[string]string{"spotMetaAndAssetCtxs": `[{"universe":[],"tokens":[]},[{"coin":"@108","markPx":"10"}]]`}, nil)
	setTestPairs(t, spotContexts, asset.Spot, testSpotMapping)
	_, err = spotContexts.UpdateTicker(t.Context(), spotPair, asset.Spot)
	assert.ErrorIs(t, err, errAssetContextNotFound, "UpdateTicker should report a spot pair without a context")

	unnamed := newInfoServerExchange(t, map[string]string{"metaAndAssetCtxs": `[{"universe":[{"name":"BTC"}]},[{"markPx":"101"}]]`}, nil)
	setTestPairs(t, unnamed, asset.PerpetualContract, testPerpetualMapping)
	unnamed.Name = ""
	_, err = unnamed.UpdateTicker(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, common.ErrExchangeNameNotSet, "UpdateTicker should return a ticker processing error")

	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		failed := newUnavailableServerExchange(t)
		setTestPairs(t, failed, a, map[asset.Item]pairMapping{asset.Spot: testSpotMapping, asset.PerpetualContract: testPerpetualMapping}[a])
		_, err = failed.UpdateTicker(t.Context(), map[asset.Item]currency.Pair{asset.Spot: spotPair, asset.PerpetualContract: perpetualPair}[a], a)
		assert.ErrorIsf(t, err, request.ErrBadStatus, "UpdateTicker should return a %s context failure", a)
	}
}

func TestUpdateOrderbook(t *testing.T) {
	t.Parallel()
	ex := newTestExchange(t)
	_, err := ex.UpdateOrderbook(t.Context(), perpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateOrderbook must reject an unsupported asset")

	for _, exp := range []*orderbook.Book{
		{
			Bids:        orderbook.Levels{{Price: 85812, Amount: 0.41043, OrderCount: 4}, {Price: 85811, Amount: 1.62719, OrderCount: 7}},
			Asks:        orderbook.Levels{{Price: 85813, Amount: 4.71207, OrderCount: 19}, {Price: 85814, Amount: 0.0035, OrderCount: 1}},
			Pair:        perpetualPair,
			Asset:       asset.PerpetualContract,
			LastUpdated: time.UnixMilli(1791275808695).UTC(),
		},
		{
			Bids:        orderbook.Levels{{Price: 93.144, Amount: 12.04, OrderCount: 2}},
			Asks:        orderbook.Levels{{Price: 93.149, Amount: 48.31, OrderCount: 3}},
			Pair:        spotPair,
			Asset:       asset.Spot,
			LastUpdated: time.UnixMilli(1791275812441).UTC(),
		},
		{
			Bids:        orderbook.Levels{{Price: 31122, Amount: 0.215, OrderCount: 3}},
			Asks:        orderbook.Levels{{Price: 31123, Amount: 0.8411, OrderCount: 5}},
			Pair:        testBuilderPair,
			Asset:       asset.PerpetualContract,
			LastUpdated: time.UnixMilli(1791275813002).UTC(),
		},
	} {
		book, err := ex.UpdateOrderbook(t.Context(), exp.Pair, exp.Asset)
		require.NoErrorf(t, err, "UpdateOrderbook must not error for %s %s", exp.Asset, exp.Pair)
		if mockTests {
			exp.Exchange = ex.Name
			exp.ValidateOrderbook = ex.ValidateOrderbook
			exp.RestSnapshot = true
			exp.InsertedAt = book.InsertedAt
			assert.Equalf(t, exp, book, "UpdateOrderbook should store the %s %s snapshot", exp.Asset, exp.Pair)
		} else {
			assert.NotEmptyf(t, book.Bids, "UpdateOrderbook should return %s bids", exp.Pair)
			assert.NotEmptyf(t, book.Asks, "UpdateOrderbook should return %s asks", exp.Pair)
		}
	}

	invalidBook := `{"coin":"BTC","levels":[[{"px":"100","sz":"-2","n":1}],[{"px":"101","sz":"3","n":2}]],"time":1791275808695}`
	invalid := newTradingServerExchange(t, map[string]string{"l2Book": invalidBook}, nil)
	// A stored book keeps the validation setting it was first stored with, and ex already stored this test's BTC book
	invalid.Name = t.Name() + "WithoutValidation"
	_, err = invalid.UpdateOrderbook(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.Error(t, err, "UpdateOrderbook should return a validation failure")
	invalid.ValidateOrderbook = false
	book, err := invalid.UpdateOrderbook(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "UpdateOrderbook must not error with validation disabled")
	assert.False(t, book.ValidateOrderbook, "UpdateOrderbook should keep the exchange's validation setting")

	_, err = newTradingServerExchange(t, map[string]string{"l2Book": `{"coin":"BTC","levels":[[]],"time":1791275808695}`}, nil).UpdateOrderbook(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errInvalidBookLevelCount, "UpdateOrderbook should reject a book without both sides")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.UpdateOrderbook(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "UpdateOrderbook should return a book failure")
}

func TestConvertL2Book(t *testing.T) {
	t.Parallel()
	_, err := e.convertL2Book(&L2BookResponse{Levels: [][]L2Level{{}}}, perpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errInvalidBookLevelCount, "convertL2Book must reject a book without both sides")

	book, err := e.convertL2Book(&L2BookResponse{
		Coin:   "BTC",
		Time:   milli(1791275808695),
		Levels: [][]L2Level{{{Price: 100, Size: 2, NumberOfOrders: 3}}, {{Price: 101, Size: 4, NumberOfOrders: 5}}},
	}, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "convertL2Book must not error")
	exp := &orderbook.Book{
		Exchange:          e.Name,
		Pair:              perpetualPair,
		Asset:             asset.PerpetualContract,
		LastUpdated:       time.UnixMilli(1791275808695).UTC(),
		ValidateOrderbook: e.ValidateOrderbook,
		Bids:              orderbook.Levels{{Price: 100, Amount: 2, OrderCount: 3}},
		Asks:              orderbook.Levels{{Price: 101, Amount: 4, OrderCount: 5}},
	}
	assert.Equal(t, exp, book, "convertL2Book should convert both sides of the book")
}

func TestConvertL2Level(t *testing.T) {
	t.Parallel()
	exp := orderbook.Level{Price: 85812, Amount: 0.41043, OrderCount: 4}
	assert.Equal(t, exp, convertL2Level(&L2Level{Price: 85812, Size: 0.41043, NumberOfOrders: 4}), "convertL2Level should convert the level's price, size and order count")
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetRecentTrades(t.Context(), perpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetRecentTrades must reject an unsupported asset")

	trades, err := e.GetRecentTrades(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetRecentTrades must not error")
	if mockTests {
		exp := []trade.Data{
			{TID: "284092582946010", Exchange: e.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Sell, Price: 85812, Amount: 0.00347, Timestamp: time.UnixMilli(1791275814004).UTC()},
			{TID: "744489298526402", Exchange: e.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Price: 85813, Amount: 0.01757, Timestamp: time.UnixMilli(1791275814806).UTC()},
		}
		assert.Equal(t, exp, trades, "GetRecentTrades should convert every trade in time order")
	} else {
		assert.NotEmpty(t, trades, "GetRecentTrades should return trades")
	}

	unordered := newInfoServerExchange(t, map[string]string{"recentTrades": `[{"coin":"BTC","side":"B","px":"101","sz":"3","time":1791275814806,"tid":8},{"coin":"BTC","side":"A","px":"100","sz":"2","time":1791275814004,"tid":7}]`}, nil)
	unordered.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	trades, err = unordered.GetRecentTrades(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetRecentTrades must not error for unordered trades")
	require.Len(t, trades, 2, "GetRecentTrades must return every trade")
	assert.Equal(t, "7", trades[0].TID, "GetRecentTrades should sort trades oldest first")

	invalid := newInfoServerExchange(t, map[string]string{"recentTrades": `[{"coin":"BTC","side":"X","px":"100","sz":"2","time":1791275814004,"tid":7}]`}, nil)
	invalid.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = invalid.GetRecentTrades(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "GetRecentTrades should reject an invalid trade side")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetRecentTrades(t.Context(), perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRecentTrades should return a trades failure")
}

func TestConvertTrade(t *testing.T) {
	t.Parallel()
	_, err := e.convertTrade(&RecentTradeResponse{Side: "X"}, perpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "convertTrade must reject an invalid side")
	result, err := e.convertTrade(&RecentTradeResponse{Coin: "BTC", Side: "B", Price: 100, Size: 2, Time: milli(1791275814004), Hash: "0x1", TradeID: 7}, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "convertTrade must not error")
	exp := trade.Data{TID: "7", Exchange: e.Name, CurrencyPair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.Buy, Price: 100, Amount: 2, Timestamp: time.UnixMilli(1791275814004).UTC()}
	assert.Equal(t, exp, result, "convertTrade should convert the trade")
}

func TestGetHistoricCandles(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := e.GetHistoricCandles(t.Context(), perpetualPair, asset.Options, kline.OneHour, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricCandles must reject an unsupported asset")
	_, err = e.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.PerpetualContract, kline.OneHour, start, end)
	require.Error(t, err, "GetHistoricCandles must reject an empty pair")
	_, err = e.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, kline.OneMin, end.Add(-(maximumCandleCount+1)*time.Minute), end)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "GetHistoricCandles must reject a range beyond Hyperliquid's retention")

	result, err := e.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, kline.OneHour, start, end)
	require.NoError(t, err, "GetHistoricCandles must not error")
	assert.Equal(t, kline.OneHour, result.Interval, "GetHistoricCandles should return the requested interval")
	if mockTests {
		exp := []kline.Candle{
			{Time: time.UnixMilli(1791252000000).UTC(), Open: 85641, High: 85739, Low: 85433, Close: 85513, Volume: 526.02537},
			{Time: time.UnixMilli(1791255600000).UTC(), Open: 85513, High: 85702, Low: 85390, Close: 85690, Volume: 611.8724},
		}
		for hour := 2; hour < 6; hour++ {
			exp = append(exp, kline.Candle{Time: start.Add(time.Duration(hour) * time.Hour).UTC()})
		}
		assert.Equal(t, exp, result.Candles, "GetHistoricCandles should convert every candle and pad the range's missing hours")
	} else {
		assert.NotEmpty(t, result.Candles, "GetHistoricCandles should return candles")
	}

	ex := newInfoServerExchange(t, map[string]string{"candleSnapshot": `[]`, "meta": `{"universe":[]}`}, nil)
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = ex.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, kline.OneHour, start, end)
	assert.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "GetHistoricCandles should report an empty response")
	ex.setPairMappings(asset.PerpetualContract, nil)
	_, err = ex.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, kline.OneHour, start, end)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetHistoricCandles should report a pair without a market")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, kline.OneHour, start, end)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricCandles should return a candle failure")
}

func TestGetHistoricCandlesFromDailyCandles(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		interval   kline.Interval
		start, end time.Time
		candles    int
	}{
		{name: "three days", interval: kline.ThreeDay, start: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), end: time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC), candles: 20},
		{name: "one week", interval: kline.OneWeek, start: time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC), end: time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), candles: 10},
		{name: "two weeks", interval: kline.TwoWeek, start: time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), end: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), candles: 5},
		{name: "32 days", interval: 32 * kline.OneDay, start: time.Date(2025, 5, 22, 0, 0, 0, 0, time.UTC), end: time.Date(2025, 7, 25, 0, 0, 0, 0, time.UTC), candles: 2},
		{name: "60 days", interval: 60 * kline.OneDay, start: time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC), end: time.Date(2025, 9, 7, 0, 0, 0, 0, time.UTC), candles: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var days strings.Builder
			days.WriteByte('[')
			for open := tc.start.AddDate(0, 0, -7); open.Before(tc.end.AddDate(0, 0, 7)); open = open.AddDate(0, 0, 1) {
				if days.Len() > 1 {
					days.WriteByte(',')
				}
				fmt.Fprintf(&days, `{"t":%d,"T":%d,"s":"BTC","i":"1d","o":"1","c":"2","h":"3","l":"0.5","v":"10","n":1}`, open.UnixMilli(), open.AddDate(0, 0, 1).UnixMilli()-1)
			}
			days.WriteByte(']')
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body InfoRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
					return
				}
				if !assert.Equal(t, "candleSnapshot", body.Type, "GetHistoricCandles should only request candles") || !assert.NotNil(t, body.Request, "GetHistoricCandles should send the candle request") {
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				if !assert.Equal(t, "1d", body.Request.Interval, "GetHistoricCandles should request daily candles") {
					http.Error(w, "unexpected interval", http.StatusBadRequest)
					return
				}
				_, err := w.Write([]byte(days.String()))
				assert.NoError(t, err, "Writing the response should not error")
			}))
			ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
			result, err := ex.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, tc.interval, tc.start, tc.end)
			require.NoError(t, err, "GetHistoricCandles must not error")
			assert.Equal(t, tc.interval, result.Interval, "GetHistoricCandles should return the requested interval")
			require.Len(t, result.Candles, tc.candles, "GetHistoricCandles must return each interval of the range")
			var volume float64
			for i := range result.Candles {
				assert.Equalf(t, kline.Candle{Time: result.Candles[i].Time, Open: 1, High: 3, Low: 0.5, Close: 2, Volume: result.Candles[i].Volume}, result.Candles[i], "GetHistoricCandles should build candle %d from daily candles", i)
				volume += result.Candles[i].Volume
			}
			assert.Equal(t, tc.start, result.Candles[0].Time, "GetHistoricCandles should start at the range's start")
			assert.Equal(t, 10*tc.end.Sub(tc.start).Hours()/24, volume, "GetHistoricCandles should combine every day of the range")
		})
	}

	ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		assert.Fail(t, "GetHistoricCandles should not request candles it refuses")
		http.Error(w, "unexpected request", http.StatusBadRequest)
	}))
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	for _, interval := range []kline.Interval{kline.OneMonth, kline.ThreeMonth, kline.SixMonth, kline.NineMonth, kline.OneYear} {
		_, err := ex.GetHistoricCandles(t.Context(), perpetualPair, asset.PerpetualContract, interval, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
		assert.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "GetHistoricCandles should refuse %s candles, which are not calendar aligned", interval.Word())
	}
}

func TestConvertCandle(t *testing.T) {
	t.Parallel()
	exp := kline.Candle{Time: time.UnixMilli(1791252000000).UTC(), Open: 1, High: 4, Low: 0.5, Close: 2, Volume: 3}
	assert.Equal(t, exp, convertCandle(&CandleResponse{OpenTime: milli(1791252000000), CloseTime: milli(1791255599999), Open: 1, Close: 2, High: 4, Low: 0.5, Volume: 3, NumberOfTrades: 5}), "convertCandle should convert the candle")
}

func TestGetHistoricTrades(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetHistoricTrades(t.Context(), spotPair, asset.Spot, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricTrades should not be supported")
}

func TestGetHistoricCandlesExtended(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetHistoricCandlesExtended(t.Context(), spotPair, asset.Spot, kline.OneMin, time.Time{}, time.Time{})
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetHistoricCandlesExtended should not be supported")
}

func TestGetServerTime(t *testing.T) {
	t.Parallel()
	_, err := newUnavailableServerExchange(t).GetServerTime(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetServerTime should return the exchange status failure")

	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		result, err := e.GetServerTime(t.Context(), a)
		require.NoErrorf(t, err, "GetServerTime must not error for %s", a)
		if mockTests {
			assert.Equalf(t, time.UnixMilli(1791592715097).UTC(), result, "GetServerTime should return the exchange status time for %s", a)
		} else {
			assert.WithinDurationf(t, time.Now(), result, time.Minute, "GetServerTime should return the current time for %s", a)
		}
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetDepositAddress(t.Context(), currency.USDC, "", "")
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetDepositAddress should not be supported")
}

func TestWithdrawFiatFunds(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).WithdrawFiatFunds(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFunds should not be supported")
}

func TestWithdrawFiatFundsToInternationalBank(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).WithdrawFiatFundsToInternationalBank(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "WithdrawFiatFundsToInternationalBank should not be supported")
}

func TestGetFuturesContractDetails(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetFuturesContractDetails(t.Context(), asset.PerpetualContract)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFuturesContractDetails should not be supported")
}

func TestGetCurrencyTradeURL(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetCurrencyTradeURL(t.Context(), asset.Spot, spotPair)
	assert.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetCurrencyTradeURL should not be supported")
}

func TestUpdateOrderExecutionLimits(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, new(Exchange).UpdateOrderExecutionLimits(t.Context(), asset.Spot), common.ErrNotYetImplemented, "UpdateOrderExecutionLimits should not be implemented")
}

func TestUpdateAccountBalances(t *testing.T) {
	t.Parallel()
	unconfigured := new(Exchange)
	unconfigured.SetDefaults()
	_, err := unconfigured.UpdateAccountBalances(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "UpdateAccountBalances must reject an unsupported asset before checking credentials")
	_, err = unconfigured.UpdateAccountBalances(t.Context(), asset.Spot)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "UpdateAccountBalances must require an account address")

	for _, mode := range []AccountAbstraction{AccountAbstractionUnifiedAccount, AccountAbstractionPortfolioMargin} {
		responses := map[string]string{
			"spotClearinghouseState": `{"balances":[{"coin":"USDC","token":0,"total":"30","hold":"4"},{"coin":"HYPE","token":150,"total":"5","hold":"1"}]}`,
			"userAbstraction":        `"` + string(mode) + `"`,
			"perpDexs":               `[null,{"name":"xyz"}]`,
		}
		ex := newInfoServerExchange(t, responses, nil)
		stale := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)
		stale.Balances.Set(currency.USDC, accounts.Balance{Total: 30})
		staleBuilder := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":xyz")
		staleBuilder.Balances.Set(currency.HYPE, accounts.Balance{Total: 5})
		require.NoError(t, ex.Accounts.Save(t.Context(), accounts.SubAccounts{stale, staleBuilder}, true), "Save must not error")
		_, err := ex.UpdateAccountBalances(t.Context(), asset.Spot)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s spot balances", mode)
		result, err := ex.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s perpetual balances", mode)
		exp := accounts.SubAccounts{
			accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress),
			accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":xyz"),
		}
		assert.Equalf(t, exp, result, "UpdateAccountBalances should clear every DEX's separate balance for %s, which holds them in spot", mode)
		balances, err := ex.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 30.0, balances[currency.USDC].Total, "UpdateAccountBalances should count %s USDC once", mode)
		assert.Equalf(t, 5.0, balances[currency.HYPE].Total, "UpdateAccountBalances should count %s HYPE once", mode)

		// With spot disabled the engine refreshes perpetual balances only, which must then refresh the spot balance
		perpetualOnly := newInfoServerExchange(t, map[string]string{
			"spotClearinghouseState": `{"balances":[{"coin":"USDC","token":0,"total":"30","hold":"35"},{"coin":"HYPE","token":150,"total":"5","hold":"1"}]}`,
			"userAbstraction":        responses["userAbstraction"],
			"perpDexs":               `[null]`,
		}, nil)
		require.NoError(t, perpetualOnly.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
		result, err = perpetualOnly.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s perpetual balances with spot disabled", mode)
		assert.Equalf(t, accounts.SubAccounts{accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)}, result, "UpdateAccountBalances should return %s's DEX subaccount empty with spot disabled", mode)
		balances, err = perpetualOnly.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 30.0, balances[currency.USDC].Total, "UpdateAccountBalances should refresh %s USDC with spot disabled", mode)
		assert.Equalf(t, 30.0, balances[currency.USDC].Hold, "UpdateAccountBalances should hold no more than %s's USDC total with spot disabled", mode)
		assert.Zerof(t, balances[currency.USDC].Free, "UpdateAccountBalances should leave %s's USDC fully held with spot disabled", mode)
		assert.Equalf(t, 5.0, balances[currency.HYPE].Total, "UpdateAccountBalances should refresh %s HYPE with spot disabled", mode)
		observer := &balanceObserver{Exchange: newInfoServerExchange(t, responses, nil)}
		observer.Accounts = accounts.MustNewAccounts(observer)
		require.NoError(t, observer.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
		require.NoError(t, observer.Accounts.Save(t.Context(), accounts.SubAccounts{stale, staleBuilder}, true), "Save must not error")
		observer.usdcTotals = nil
		_, err = observer.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s perpetual balances with stale DEX balances", mode)
		require.NotEmpty(t, observer.usdcTotals, "UpdateAccountBalances must save balances")
		for _, total := range observer.usdcTotals {
			assert.Equalf(t, 30.0, total, "UpdateAccountBalances should store %s's refreshed USDC and clear the stale DEX balances in one Save, so no reader sees the USDC counted twice or not at all", mode)
		}
		balances, err = observer.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 30.0, balances[currency.USDC].Total, "UpdateAccountBalances should clear %s's stale default DEX balance with spot disabled", mode)
		assert.Equalf(t, 5.0, balances[currency.HYPE].Total, "UpdateAccountBalances should clear %s's stale builder DEX balance with spot disabled", mode)
		delete(responses, "spotClearinghouseState")
		spotFailure := newInfoServerExchange(t, responses, nil)
		require.NoError(t, spotFailure.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
		stored := accounts.NewSubAccount(asset.Spot, testAccountAddress)
		stored.Balances.Set(currency.USDC, accounts.Balance{Total: 30, Free: 30})
		require.NoError(t, spotFailure.Accounts.Save(t.Context(), accounts.SubAccounts{stored}, true), "Save must not error")
		_, err = spotFailure.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		assert.ErrorIsf(t, err, request.ErrBadStatus, "UpdateAccountBalances should return %s's failed spot refresh with spot disabled", mode)
		balances, err = spotFailure.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 30.0, balances[currency.USDC].Total, "UpdateAccountBalances should keep %s's stored balance when the spot refresh fails", mode)
		// With spot enabled the engine refreshes spot itself, so this server, which fails a spot request, must get none
		_, err = newInfoServerExchange(t, responses, nil).UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		assert.NoErrorf(t, err, "UpdateAccountBalances should not request %s's spot balance with spot enabled", mode)
	}

	// The engine refreshes only the enabled assets, so with spot disabled a balance stored under spot while the account
	// shared one is cleared once it holds separate balances, in the same Save as those
	for _, mode := range []AccountAbstraction{AccountAbstractionDefault, AccountAbstractionDisabled, AccountAbstractionDEXAbstraction} {
		responses := map[string]string{
			"userAbstraction":    `"` + string(mode) + `"`,
			"spotMeta":           `{"tokens":[{"name":"USDC","index":0}]}`,
			"allPerpMetas":       `[{"universe":[],"collateralToken":0}]`,
			"meta":               `{"universe":[],"collateralToken":0}`,
			"clearinghouseState": `{"marginSummary":{"accountValue":"30","totalMarginUsed":"0"},"withdrawable":"30"}`,
		}
		shared := accounts.NewSubAccount(asset.Spot, testAccountAddress)
		shared.Balances.Set(currency.USDC, accounts.Balance{Total: 30, Free: 30})
		observer := &balanceObserver{Exchange: newInfoServerExchange(t, responses, nil)}
		observer.Accounts = accounts.MustNewAccounts(observer)
		require.NoError(t, observer.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "SetAssetEnabled must not error")
		require.NoError(t, observer.Accounts.Save(t.Context(), accounts.SubAccounts{shared}, true), "Save must not error")
		observer.usdcTotals = nil
		result, err := observer.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s perpetual balances with spot disabled", mode)
		clearBalanceTimes(t, result)
		expDEX := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)
		expDEX.Balances.Set(currency.USDC, accounts.Balance{Total: 30, Free: 30})
		assert.Equalf(t, accounts.SubAccounts{expDEX}, result, "UpdateAccountBalances should return only %s's DEX balances", mode)
		require.NotEmpty(t, observer.usdcTotals, "UpdateAccountBalances must save balances")
		for _, total := range observer.usdcTotals {
			assert.Equalf(t, 30.0, total, "UpdateAccountBalances should store %s's DEX balance and clear the shared spot balance in one Save, so no reader sees the USDC counted twice or not at all", mode)
		}
		balances, err := observer.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 30.0, balances[currency.USDC].Total, "UpdateAccountBalances should clear the shared balance kept under spot once %s holds separate balances", mode)

		// With spot enabled the engine refreshes spot itself, so the perpetual refresh leaves stored spot balances unchanged
		enabled := newInfoServerExchange(t, responses, nil)
		require.NoError(t, enabled.Accounts.Save(t.Context(), accounts.SubAccounts{shared}, true), "Save must not error")
		_, err = enabled.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		require.NoErrorf(t, err, "UpdateAccountBalances must not error for %s perpetual balances with spot enabled", mode)
		balances, err = enabled.Accounts.CurrencyBalances(nil, asset.All)
		require.NoError(t, err, "CurrencyBalances must not error")
		assert.Equalf(t, 60.0, balances[currency.USDC].Total, "UpdateAccountBalances should keep %s's spot balance with spot enabled", mode)
	}

	// Portfolio margin balances shaped as live ones, with negative holds and spot holds
	portfolio := newInfoServerExchange(t, map[string]string{
		"spotClearinghouseState": `{"balances":[` +
			`{"coin":"USDC","token":0,"total":"1784747.5","hold":"-3204581.75","entryNtl":"0.0","spotHold":"585798.25"},` +
			`{"coin":"HYPE","token":150,"total":"13.75","hold":"9.25","entryNtl":"0.0"},` +
			`{"coin":"USDT0","token":268,"total":"0.0","hold":"-659983.25","entryNtl":"0.0","spotHold":"0.0"},` +
			`{"coin":"USDH","token":360,"total":"10449.5","hold":"10178.25","entryNtl":"0.0","spotHold":"10449.5"}]}`,
	}, nil)
	portfolioBalances, err := portfolio.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for portfolio margin spot balances")
	clearBalanceTimes(t, portfolioBalances)
	expPortfolio := accounts.NewSubAccount(asset.Spot, testAccountAddress)
	expPortfolio.Balances.Set(currency.USDC, accounts.Balance{Total: 1784747.5, Hold: 585798.25, Free: 1198949.25})
	expPortfolio.Balances.Set(currency.HYPE, accounts.Balance{Total: 13.75, Hold: 9.25, Free: 4.5})
	expPortfolio.Balances.Set(currency.NewCode("USDT0"), accounts.Balance{})
	expPortfolio.Balances.Set(currency.NewCode("USDH"), accounts.Balance{Total: 10449.5, Hold: 10449.5})
	assert.Equal(t, accounts.SubAccounts{expPortfolio}, portfolioBalances, "UpdateAccountBalances should hold the larger of each balance's hold and spot hold, keeping the free amount within the total")

	builder := newInfoServerExchange(t, map[string]string{
		"userAbstraction":        `"default"`,
		"perpDexs":               `[null,{"name":"xyz"}]`,
		"spotMeta":               `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`,
		"allPerpMetas":           `[{"universe":[],"collateralToken":0}]`,
		"meta:":                  `{"universe":[],"collateralToken":0}`,
		"meta:xyz":               `{"universe":[],"collateralToken":150}`,
		"clearinghouseState:":    `{"marginSummary":{"accountValue":"20","totalMarginUsed":"3"},"withdrawable":"16"}`,
		"clearinghouseState:xyz": `{"marginSummary":{"accountValue":"7","totalMarginUsed":"2"},"withdrawable":"4"}`,
	}, nil)
	result, err := builder.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "UpdateAccountBalances must not error for a builder DEX with spot collateral")
	clearBalanceTimes(t, result)
	defaultDEX := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)
	defaultDEX.Balances.Set(currency.USDC, accounts.Balance{Total: 20, Hold: 3, Free: 16})
	builderDEX := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":xyz")
	builderDEX.Balances.Set(currency.HYPE, accounts.Balance{Total: 7, Hold: 2, Free: 4})
	assert.Equal(t, accounts.SubAccounts{defaultDEX, builderDEX}, result, "UpdateAccountBalances should hold each DEX's collateral in its own subaccount")

	underMargined := newInfoServerExchange(t, map[string]string{
		"userAbstraction":     `"default"`,
		"perpDexs":            `[null]`,
		"spotMeta":            `{"tokens":[{"name":"USDC","index":0}]}`,
		"allPerpMetas":        `[{"universe":[],"collateralToken":0}]`,
		"meta:":               `{"universe":[],"collateralToken":0}`,
		"clearinghouseState:": `{"marginSummary":{"accountValue":"81600.834161","totalMarginUsed":"81606.117731"},"withdrawable":"0.0"}`,
	}, nil)
	result, err = underMargined.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "UpdateAccountBalances must not error for an account using more margin than it is worth")
	clearBalanceTimes(t, result)
	capped := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)
	capped.Balances.Set(currency.USDC, accounts.Balance{Total: 81600.834161, Hold: 81600.834161})
	assert.Equal(t, accounts.SubAccounts{capped}, result, "UpdateAccountBalances should hold no more than an account is worth")

	const (
		defaultMeta = `{"universe":[{"name":"BTC"},{"name":"ETH"}],"collateralToken":0}`
		builderMeta = `{"universe":[{"name":"xyz:XYZ100"},{"name":"xyz:TSLA"}],"collateralToken":150}`
		otherMeta   = `{"universe":[{"name":"flx:TSLA"}],"collateralToken":360}`
	)
	for _, tc := range []struct {
		name, bulk, failureKey, fallbackBody string
		fallbacks                            []string
		err                                  error
	}{
		{name: "ordered", bulk: `[` + defaultMeta + `,` + builderMeta + `,` + otherMeta + `]`},
		{name: "reordered", bulk: `[` + otherMeta + `,` + defaultMeta + `,` + builderMeta + `]`},
		{name: "unrelated extra", bulk: `[{"universe":[{"name":"unknown:X"}],"collateralToken":999},` + otherMeta + `,` + defaultMeta + `,` + builderMeta + `]`},
		{name: "missing default", bulk: `[` + builderMeta + `,` + otherMeta + `]`, fallbacks: []string{""}},
		{name: "missing builder", bulk: `[` + defaultMeta + `,` + otherMeta + `]`, fallbacks: []string{"xyz"}},
		{name: "empty default", bulk: `[{"universe":[],"collateralToken":999},` + builderMeta + `,` + otherMeta + `]`, fallbacks: []string{""}},
		{name: "empty builder", bulk: `[` + defaultMeta + `,{"universe":[],"collateralToken":999},` + otherMeta + `]`, fallbacks: []string{"xyz"}},
		{name: "duplicate default", bulk: `[` + defaultMeta + `,` + defaultMeta + `,` + builderMeta + `,` + otherMeta + `]`, fallbacks: []string{""}},
		{name: "duplicate builder", bulk: `[` + defaultMeta + `,` + builderMeta + `,` + builderMeta + `,` + otherMeta + `]`, fallbacks: []string{"xyz"}},
		{name: "mixed builder prefixes", bulk: `[` + defaultMeta + `,{"universe":[{"name":"xyz:X"},{"name":"flx:Y"}],"collateralToken":999},` + otherMeta + `]`, fallbacks: []string{"xyz", "flx"}},
		{name: "mixed default and builder", bulk: `[{"universe":[{"name":"BTC"},{"name":"xyz:Y"}],"collateralToken":999},` + builderMeta + `,` + otherMeta + `]`, fallbacks: []string{"", "xyz"}},
		{name: "bulk network failure", failureKey: "allPerpMetas", err: request.ErrBadStatus},
		{name: "bulk empty response", bulk: `[]`, err: common.ErrNoResponse},
		{name: "fallback network failure", bulk: `[` + defaultMeta + `,` + otherMeta + `]`, failureKey: "meta:xyz", fallbacks: []string{"xyz"}, err: request.ErrBadStatus},
		{name: "fallback empty response", bulk: `[` + defaultMeta + `,` + otherMeta + `]`, fallbackBody: `null`, fallbacks: []string{"xyz"}, err: common.ErrNoResponse},
		{name: "fallback collateral mismatch", bulk: `[` + defaultMeta + `,` + otherMeta + `]`, fallbackBody: `{"collateralToken":999}`, fallbacks: []string{"xyz"}, err: errSpotTokenNotFound},
		{name: "bulk collateral mismatch", bulk: `[` + defaultMeta + `,{"universe":[{"name":"xyz:X"}],"collateralToken":999},` + otherMeta + `]`, err: errSpotTokenNotFound},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			responses := map[string]string{
				"userAbstraction":        `"default"`,
				"perpDexs":               `[null,{"name":"xyz"},{"name":"flx"}]`,
				"spotMeta":               `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"USDH","index":360}]}`,
				"allPerpMetas":           tc.bulk,
				"meta:":                  `{"universe":[],"collateralToken":0}`,
				"meta:xyz":               `{"universe":[],"collateralToken":150}`,
				"meta:flx":               `{"universe":[],"collateralToken":360}`,
				"clearinghouseState:":    `{"marginSummary":{"accountValue":"20","totalMarginUsed":"3"},"withdrawable":"16"}`,
				"clearinghouseState:xyz": `{"marginSummary":{"accountValue":"7","totalMarginUsed":"2"},"withdrawable":"4"}`,
				"clearinghouseState:flx": `{"marginSummary":{"accountValue":"9","totalMarginUsed":"1"},"withdrawable":"8"}`,
			}
			if tc.fallbackBody != "" {
				responses["meta:xyz"] = tc.fallbackBody
			}
			var mu sync.Mutex
			calls := make(map[string]int)
			ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body InfoRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decode should read the info request") {
					return
				}
				key := body.Type
				if key == "meta" || key == "clearinghouseState" {
					key += ":" + body.DEX
				}
				mu.Lock()
				calls[key]++
				mu.Unlock()
				if key == tc.failureKey {
					http.Error(w, "metadata unavailable", http.StatusServiceUnavailable)
					return
				}
				response, ok := responses[key]
				if !assert.True(t, ok, "UpdateAccountBalances should request only expected info") {
					http.Error(w, "unexpected request", http.StatusBadRequest)
					return
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "Write should return the info response")
			}))
			setTestCredentials(ex, signingCredentials())
			result, err := ex.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
			require.ErrorIs(t, err, tc.err, "UpdateAccountBalances must propagate metadata errors")
			mu.Lock()
			defer mu.Unlock()
			assert.Equal(t, 1, calls["allPerpMetas"], "UpdateAccountBalances should request bulk metadata once")
			for _, dex := range []string{"", "xyz", "flx"} {
				expectedCalls := 0
				if slices.Contains(tc.fallbacks, dex) {
					expectedCalls = 1
				}
				assert.Equalf(t, expectedCalls, calls["meta:"+dex], "UpdateAccountBalances should fall back only for unidentified DEX %q", dex)
			}
			if tc.err != nil {
				assert.Nil(t, result, "UpdateAccountBalances should return no partial balances")
				assert.Zero(t, calls["clearinghouseState:flx"], "UpdateAccountBalances should stop fetching after a metadata failure")
				return
			}
			clearBalanceTimes(t, result)
			other := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":flx")
			other.Balances.Set(currency.NewCode("USDH"), accounts.Balance{Total: 9, Hold: 1, Free: 8})
			assert.Equal(t, accounts.SubAccounts{defaultDEX, builderDEX, other}, result, "UpdateAccountBalances should attribute collateral by DEX irrespective of metadata order")
			expectedCalls := map[string]int{"userAbstraction": 1, "perpDexs": 1, "spotMeta": 1, "allPerpMetas": 1, "clearinghouseState:": 1, "clearinghouseState:xyz": 1, "clearinghouseState:flx": 1}
			for _, dex := range tc.fallbacks {
				expectedCalls["meta:"+dex] = 1
			}
			assert.Equal(t, expectedCalls, calls, "UpdateAccountBalances should use seven requests instead of nine when bulk metadata identifies all three DEXs")
		})
	}

	for _, tc := range []struct {
		name      string
		responses map[string]string
		err       error
	}{
		{name: "abstraction failure", responses: map[string]string{"userAbstraction": `"unknown"`}, err: errAccountAbstractionInvalid},
		{name: "registry failure", responses: map[string]string{"userAbstraction": `"default"`, "perpDexs": `{`}},
		{name: "spot metadata failure", responses: map[string]string{"userAbstraction": `"default"`, "spotMeta": `{`}},
		{name: "duplicate token index", responses: map[string]string{"userAbstraction": `"default"`, "spotMeta": `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":0}]}`}, err: errMalformedResponse},
		{name: "metadata failure", responses: map[string]string{"userAbstraction": `"default"`, "spotMeta": `{"tokens":[{"name":"USDC","index":0}]}`, "allPerpMetas": `{`}},
		{name: "missing collateral token", responses: map[string]string{"userAbstraction": `"default"`, "spotMeta": `{"tokens":[{"name":"USDC","index":0}]}`, "allPerpMetas": `[{"universe":[{"name":"BTC"}],"collateralToken":150}]`}, err: errSpotTokenNotFound},
		{name: "clearinghouse failure", responses: map[string]string{"userAbstraction": `"default"`, "spotMeta": `{"tokens":[{"name":"USDC","index":0}]}`, "allPerpMetas": `[{"universe":[{"name":"BTC"}],"collateralToken":0}]`, "clearinghouseState": `{`}},
	} {
		_, err := newInfoServerExchange(t, tc.responses, nil).UpdateAccountBalances(t.Context(), asset.PerpetualContract)
		if tc.err == nil {
			assert.Errorf(t, err, "UpdateAccountBalances should return a %s", tc.name)
			continue
		}
		assert.ErrorIsf(t, err, tc.err, "UpdateAccountBalances should return a %s", tc.name)
	}
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		_, err = newUnavailableServerExchange(t).UpdateAccountBalances(t.Context(), a)
		assert.ErrorIsf(t, err, request.ErrBadStatus, "UpdateAccountBalances should return a %s balance failure", a)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	spot, err := e.UpdateAccountBalances(t.Context(), asset.Spot)
	require.NoError(t, err, "UpdateAccountBalances must not error for spot")
	perpetual, err := e.UpdateAccountBalances(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "UpdateAccountBalances must not error for perpetuals")
	if !mockTests {
		assert.NotEmpty(t, perpetual, "UpdateAccountBalances should return a subaccount per perpetual DEX")
		return
	}
	clearBalanceTimes(t, spot)
	clearBalanceTimes(t, perpetual)
	expSpot := accounts.NewSubAccount(asset.Spot, testAccountAddress)
	expSpot.Balances.Set(currency.USDC, accounts.Balance{Total: 1520.25, Hold: 120.5, Free: 1399.75})
	expSpot.Balances.Set(currency.HYPE, accounts.Balance{Total: 12.5, Hold: 2.5, Free: 10})
	assert.Equal(t, accounts.SubAccounts{expSpot}, spot, "UpdateAccountBalances should return every spot balance")
	expDefault := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress)
	expDefault.Balances.Set(currency.USDC, accounts.Balance{Total: 2500.5, Hold: 42.90625, Free: 2457.59375})
	expXYZ := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":xyz")
	expXYZ.Balances.Set(currency.USDC, accounts.Balance{Total: 300, Free: 300})
	expFLX := accounts.NewSubAccount(asset.PerpetualContract, testAccountAddress+":flx")
	expFLX.Balances.Set(currency.NewCode("USDH"), accounts.Balance{Total: 120, Hold: 3.955, Free: 116.045})
	assert.Equal(t, accounts.SubAccounts{expDefault, expXYZ, expFLX}, perpetual, "UpdateAccountBalances should return each perpetual DEX's collateral balance")
}

func TestSpotSubAccount(t *testing.T) {
	t.Parallel()
	_, err := newUnavailableServerExchange(t).spotSubAccount(t.Context(), testAccountAddress)
	assert.ErrorIs(t, err, request.ErrBadStatus, "spotSubAccount should return a failed spot balance request")

	ex := newInfoServerExchange(t, map[string]string{
		"spotClearinghouseState": `{"balances":[{"coin":"USDC","token":0,"total":"30","hold":"4"},{"coin":"HYPE","token":150,"total":"5","hold":"-1","spotHold":"2"}]}`,
	}, nil)
	subAccount, err := ex.spotSubAccount(t.Context(), testTraderAddress)
	require.NoError(t, err, "spotSubAccount must not error")
	exp := accounts.NewSubAccount(asset.Spot, testTraderAddress)
	exp.Balances.Set(currency.USDC, accounts.Balance{Total: 30, Hold: 4, Free: 26})
	exp.Balances.Set(currency.HYPE, accounts.Balance{Total: 5, Hold: 2, Free: 3})
	assert.Equal(t, exp, subAccount, "spotSubAccount should return every spot balance under the address")
}

func TestPerpetualMetadataForDEX(t *testing.T) {
	t.Parallel()
	defaultMeta := PerpetualMetadataResponse{Universe: []PerpetualAssetMetadata{{Name: "BTC"}, {Name: "ETH"}}, CollateralToken: 0}
	builderMeta := PerpetualMetadataResponse{Universe: []PerpetualAssetMetadata{{Name: "xyz:X"}, {Name: "xyz:Y"}}, CollateralToken: 150}
	otherMeta := PerpetualMetadataResponse{Universe: []PerpetualAssetMetadata{{Name: "flx:X"}}, CollateralToken: 360}
	for _, tc := range []struct {
		name, dex string
		metadata  []PerpetualMetadataResponse
		index     int
	}{
		{name: "no entries", index: -1},
		{name: "default", metadata: []PerpetualMetadataResponse{defaultMeta}, index: 0},
		{name: "reordered default", metadata: []PerpetualMetadataResponse{builderMeta, defaultMeta, otherMeta}, index: 1},
		{name: "reordered builder", dex: "xyz", metadata: []PerpetualMetadataResponse{otherMeta, defaultMeta, builderMeta}, index: 2},
		{name: "missing builder", dex: "xyz", metadata: []PerpetualMetadataResponse{defaultMeta, otherMeta}, index: -1},
		{name: "missing default", metadata: []PerpetualMetadataResponse{builderMeta}, index: -1},
		{name: "empty default", metadata: []PerpetualMetadataResponse{{CollateralToken: 999}, builderMeta}, index: -1},
		{name: "empty builder", dex: "xyz", metadata: []PerpetualMetadataResponse{defaultMeta, {CollateralToken: 999}}, index: -1},
		{name: "mixed default", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: "BTC"}, {Name: "xyz:X"}}}}, index: -1},
		{name: "mixed builder", dex: "xyz", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: "xyz:X"}, {Name: "flx:Y"}}}}, index: -1},
		{name: "mixed entry after valid", dex: "xyz", metadata: []PerpetualMetadataResponse{builderMeta, {Universe: []PerpetualAssetMetadata{{Name: "xyz:X"}, {Name: "flx:Y"}}}}, index: -1},
		{name: "duplicate default", metadata: []PerpetualMetadataResponse{defaultMeta, defaultMeta}, index: -1},
		{name: "duplicate builder", dex: "xyz", metadata: []PerpetualMetadataResponse{builderMeta, otherMeta, builderMeta}, index: -1},
		{name: "three duplicate builders", dex: "xyz", metadata: []PerpetualMetadataResponse{builderMeta, builderMeta, builderMeta}, index: -1},
		{name: "unrelated extras", dex: "xyz", metadata: []PerpetualMetadataResponse{{}, otherMeta, builderMeta, defaultMeta}, index: 2},
		{name: "empty name", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: ""}}}}, index: -1},
		{name: "blank name", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: " "}}}}, index: -1},
		{name: "empty prefix", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: ":BTC"}}}}, index: -1},
		{name: "empty coin", dex: "xyz", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: "xyz:"}}}}, index: -1},
		{name: "nested prefix", dex: "xyz", metadata: []PerpetualMetadataResponse{{Universe: []PerpetualAssetMetadata{{Name: "xyz:flx:BTC"}}}}, index: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := perpetualMetadataForDEX(tc.metadata, tc.dex)
			if tc.index < 0 {
				assert.Nil(t, result, "perpetualMetadataForDEX should leave unverifiable metadata for fallback")
				return
			}
			assert.Same(t, &tc.metadata[tc.index], result, "perpetualMetadataForDEX should select the unique entry whose markets all identify the DEX")
		})
	}
}

func TestConvertSpotBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		balance SpotBalance
		exp     accounts.Balance
	}{
		{name: "an ordinary hold", balance: SpotBalance{Total: 10, Hold: 2}, exp: accounts.Balance{Total: 10, Hold: 2, Free: 8}},
		{name: "a negative hold", balance: SpotBalance{Total: 10, Hold: -3}, exp: accounts.Balance{Total: 10, Free: 10}},
		{name: "a spot hold above a negative hold", balance: SpotBalance{Total: 10, Hold: -3, SpotHold: 4}, exp: accounts.Balance{Total: 10, Hold: 4, Free: 6}},
		{name: "a hold above the spot hold", balance: SpotBalance{Total: 10, Hold: 6, SpotHold: 4}, exp: accounts.Balance{Total: 10, Hold: 6, Free: 4}},
		{name: "a hold above the total", balance: SpotBalance{Total: 10, Hold: 12, SpotHold: 5}, exp: accounts.Balance{Total: 10, Hold: 10}},
		{name: "a spot hold above the total", balance: SpotBalance{Total: 10, Hold: -3, SpotHold: 12}, exp: accounts.Balance{Total: 10, Hold: 10}},
		{name: "a negative hold on an empty balance", balance: SpotBalance{Hold: -659983.25}, exp: accounts.Balance{}},
		{name: "a negative total, leaving nothing free", balance: SpotBalance{Total: -1}, exp: accounts.Balance{Total: -1, Hold: -1}},
	} {
		assert.Equalf(t, tc.exp, convertSpotBalance(&tc.balance), "convertSpotBalance should convert %s", tc.name)
	}
}

func TestConvertPerpetualBalance(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		state ClearinghouseStateResponse
		exp   accounts.Balance
	}{
		{name: "margin used within the account value", state: ClearinghouseStateResponse{MarginSummary: MarginSummary{AccountValue: 100, TotalMarginUsed: 30}, Withdrawable: 70}, exp: accounts.Balance{Total: 100, Hold: 30, Free: 70}},
		{name: "margin used above the account value", state: ClearinghouseStateResponse{MarginSummary: MarginSummary{AccountValue: 81600.834161, TotalMarginUsed: 81606.117731}}, exp: accounts.Balance{Total: 81600.834161, Hold: 81600.834161}},
		{name: "a negative account value", state: ClearinghouseStateResponse{MarginSummary: MarginSummary{AccountValue: -5, TotalMarginUsed: 10}}, exp: accounts.Balance{Total: -5, Hold: -5}},
	} {
		assert.Equalf(t, tc.exp, convertPerpetualBalance(&tc.state), "convertPerpetualBalance should convert %s", tc.name)
	}
}

func TestDEXSubAccountID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, testAccountAddress, dexSubAccountID(testAccountAddress, ""), "dexSubAccountID should not scope the default DEX")
	assert.Equal(t, testAccountAddress+":xyz", dexSubAccountID(testAccountAddress, "xyz"), "dexSubAccountID should scope a builder DEX")
}

func TestGetAccountFundingHistory(t *testing.T) {
	t.Parallel()
	_, err := new(Exchange).GetAccountFundingHistory(t.Context())
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetAccountFundingHistory must require an account address")
	_, err = newUnavailableServerExchange(t).GetAccountFundingHistory(t.Context())
	require.ErrorIs(t, err, request.ErrBadStatus, "GetAccountFundingHistory must return a ledger failure")
	_, err = newInfoServerExchange(t, map[string]string{"userNonFundingLedgerUpdates": `[{"time":1791200000000,"hash":"0x1","delta":{"type":""}}]`}, nil).GetAccountFundingHistory(t.Context())
	require.ErrorIs(t, err, errMalformedResponse, "GetAccountFundingHistory must reject a ledger update without a type")
	const transferWithFee = `[{"time":1791200000000,"hash":"0x2","delta":{"type":"spotTransfer","token":"HYPE","amount":"2.5","usdcValue":"232.86","user":"` + testAccountAddress + `","destination":"0x000000000000000000000000000000000000000b","fee":"1.0","nativeTokenFee":"0.0","nonce":null,"feeToken":""}}]`
	fees, err := newInfoServerExchange(t, map[string]string{"userNonFundingLedgerUpdates": transferWithFee}, nil).GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error for a transfer with a fee")
	require.Len(t, fees, 2, "GetAccountFundingHistory must return a fee paid in another currency than its transfer as an entry of its own")
	assert.Equal(t, "HYPE", fees[0].Currency, "GetAccountFundingHistory should return the transfer in its own currency")
	assert.Equal(t, "USDC", fees[1].Currency, "GetAccountFundingHistory should return the fee in its own currency")
	assert.Equal(t, 1.0, fees[1].Fee, "GetAccountFundingHistory should return the fee the account paid")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "GetAccountFundingHistory must not error")
	if !mockTests {
		return
	}
	exp := []exchange.FundingHistory{
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x1f6d2c9e3a8b4d5f60718293a4b5c6d7e8f9011223344556677889900aabbcc0",
			Description:  "deposit",
			Timestamp:    time.UnixMilli(1791200000000).UTC(),
			Currency:     "USDC",
			Amount:       3000,
			TransferType: "deposit",
			CryptoTxID:   "0x1f6d2c9e3a8b4d5f60718293a4b5c6d7e8f9011223344556677889900aabbcc0",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x2a7e3d0f4b9c5e6071829304b5c6d7e8f90a1122334455667788990011aabbdd",
			Description:  "perpetual to spot",
			Timestamp:    time.UnixMilli(1791210000000).UTC(),
			Currency:     "USDC",
			Amount:       1500,
			TransferType: "accountClassTransfer",
			CryptoTxID:   "0x2a7e3d0f4b9c5e6071829304b5c6d7e8f90a1122334455667788990011aabbdd",
		},
		{
			ExchangeName:      e.Name,
			Status:            "processed",
			TransferID:        "0x3b8f4e105cad6f718293a415c6d7e8f90a1b22334455667788990011aabbccee",
			Description:       "spotTransfer",
			Timestamp:         time.UnixMilli(1791220000000).UTC(),
			Currency:          "HYPE",
			Amount:            2.5,
			TransferType:      "spotTransfer",
			CryptoToAddress:   testWithdrawalAddress,
			CryptoFromAddress: testAccountAddress,
			CryptoTxID:        "0x3b8f4e105cad6f718293a415c6d7e8f90a1b22334455667788990011aabbccee",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x4c90a5216dbe7082a3b4c526d7e8f90a1b2c33445566778899001122aabbccff",
			Description:  "withdraw",
			Timestamp:    time.UnixMilli(1791230000000).UTC(),
			Currency:     "USDC",
			Amount:       100,
			Fee:          1,
			TransferType: "withdraw",
			CryptoTxID:   "0x4c90a5216dbe7082a3b4c526d7e8f90a1b2c33445566778899001122aabbccff",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x5da1b6327ecf8193b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b",
			Description:  "spot to staking",
			Timestamp:    time.UnixMilli(1791240000000).UTC(),
			Currency:     "HYPE",
			Amount:       1,
			TransferType: "cStakingTransfer",
			CryptoTxID:   "0x5da1b6327ecf8193b4c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x6eb2c7438fd09204c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c",
			Description:  "staking to spot",
			Timestamp:    time.UnixMilli(1791250000000).UTC(),
			Currency:     "HYPE",
			Amount:       0.5,
			TransferType: "cStakingTransfer",
			CryptoTxID:   "0x6eb2c7438fd09204c5d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x7fc3d8549ae1a315d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d",
			Description:  "borrowLend: repay with 0.0125 interest",
			Timestamp:    time.UnixMilli(1791260000000).UTC(),
			Currency:     "USDC",
			Amount:       50,
			TransferType: "borrowLend",
			CryptoTxID:   "0x7fc3d8549ae1a315d6e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d",
		},
		{
			ExchangeName: e.Name,
			Status:       "processed",
			TransferID:   "0x80d4e965abf2b426e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
			Description:  "activateDexAbstraction: xyz",
			Timestamp:    time.UnixMilli(1791270000000).UTC(),
			Currency:     "USDC",
			Amount:       1,
			TransferType: "activateDexAbstraction",
			CryptoTxID:   "0x80d4e965abf2b426e7f8091a2b3c4d5e6f708192a3b4c5d6e7f8091a2b3c4d5e",
		},
	}
	assert.Equal(t, exp, result, "GetAccountFundingHistory should convert every ledger update in time order")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	t.Parallel()
	result, err := new(Exchange).GetWithdrawalsHistory(t.Context(), currency.BTC, asset.Empty)
	require.NoError(t, err, "GetWithdrawalsHistory must not error for a currency the bridge does not withdraw")
	assert.Empty(t, result, "GetWithdrawalsHistory should return no withdrawals of a currency the bridge does not withdraw")
	_, err = new(Exchange).GetWithdrawalsHistory(t.Context(), currency.USDC, asset.Empty)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetWithdrawalsHistory must require an account address")
	_, err = newUnavailableServerExchange(t).GetWithdrawalsHistory(t.Context(), currency.USDC, asset.Empty)
	require.ErrorIs(t, err, request.ErrBadStatus, "GetWithdrawalsHistory must return a ledger failure")

	sandbox := newInfoServerExchange(t, map[string]string{"userNonFundingLedgerUpdates": `[{"time":1791200000000,"hash":"0xdeposit","delta":{"type":"deposit","usdc":"3"}},{"time":1791200000001,"hash":"0xwithdraw","delta":{"type":"withdraw","usdc":"2","fee":"1","nonce":7}}]`}, nil)
	sandbox.Config.UseSandbox = true
	result, err = sandbox.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Empty)
	require.NoError(t, err, "GetWithdrawalsHistory must not error in sandbox mode")
	exp := []exchange.WithdrawalHistory{{
		Status:       "processed",
		TransferID:   "0xwithdraw",
		Description:  "Hyperliquid bridge withdrawal",
		Timestamp:    time.UnixMilli(1791200000001).UTC(),
		Currency:     "USDC",
		Amount:       2,
		Fee:          1,
		TransferType: "withdrawal",
		CryptoTxID:   "0xwithdraw",
		CryptoChain:  "Arbitrum Sepolia",
	}}
	assert.Equal(t, exp, result, "GetWithdrawalsHistory should return only bridge withdrawals, on the testnet bridge chain")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err = e.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.Spot)
	require.NoError(t, err, "GetWithdrawalsHistory must not error")
	if mockTests {
		exp = []exchange.WithdrawalHistory{{
			Status:       "processed",
			TransferID:   "0x4c90a5216dbe7082a3b4c526d7e8f90a1b2c33445566778899001122aabbccff",
			Description:  "Hyperliquid bridge withdrawal",
			Timestamp:    time.UnixMilli(1791230000000).UTC(),
			Currency:     "USDC",
			Amount:       100,
			Fee:          1,
			TransferType: "withdrawal",
			CryptoTxID:   "0x4c90a5216dbe7082a3b4c526d7e8f90a1b2c33445566778899001122aabbccff",
			CryptoChain:  "Arbitrum",
		}}
		assert.Equal(t, exp, result, "GetWithdrawalsHistory should convert the bridge withdrawal")
	}
}

func TestGetUserNonFundingLedgerUpdatesPaginated(t *testing.T) {
	t.Parallel()
	// venuePageSize is the page size Hyperliquid returned live for userNonFundingLedgerUpdates (2026-10-09), which the
	// pager must treat as a full page
	const venuePageSize = 2000
	start := time.UnixMilli(1791200000000)
	// ledgerServerExchange serves pages of deposits; page receives the request's start time and the page number, and
	// returns each record's time and hash
	ledgerServerExchange := func(t *testing.T, page func(startTime uint64, number int32) (times []uint64, hashes []string)) *Exchange {
		t.Helper()
		var pages atomic.Int32
		return newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body InfoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
				return
			}
			times, hashes := page(body.StartTime, pages.Add(1))
			updates := make([]string, len(times))
			for i := range times {
				updates[i] = fmt.Sprintf(`{"time":%d,"hash":%q,"delta":{"type":"deposit","usdc":"1"}}`, times[i], hashes[i])
			}
			_, err := w.Write([]byte("[" + strings.Join(updates, ",") + "]"))
			assert.NoError(t, err, "Writing the ledger page should not error")
		}))
	}
	// page returns size records from startTime, one millisecond apart
	page := func(startTime, size uint64, prefix string) (times []uint64, hashes []string) {
		for i := range size {
			times = append(times, startTime+i)
			hashes = append(hashes, prefix+strconv.FormatUint(i, 10))
		}
		return times, hashes
	}

	var starts []uint64
	paged := ledgerServerExchange(t, func(startTime uint64, number int32) ([]uint64, []string) {
		starts = append(starts, startTime)
		if number == 1 {
			return page(startTime, venuePageSize, "0xa")
		}
		return page(startTime+1, venuePageSize-1, "0xb")
	})
	result, err := paged.getUserNonFundingLedgerUpdatesPaginated(t.Context(), testAccountAddress, start)
	require.NoError(t, err, "getUserNonFundingLedgerUpdatesPaginated must not error")
	assert.Len(t, result, 2*venuePageSize-1, "getUserNonFundingLedgerUpdatesPaginated should return every page's records")
	assert.Equal(t, []uint64{unixMilli(start), unixMilli(start) + venuePageSize - 1}, starts, "getUserNonFundingLedgerUpdatesPaginated should request another page only after a full page, from its last time")

	repeated := ledgerServerExchange(t, func(startTime uint64, number int32) ([]uint64, []string) {
		if number == 1 {
			return page(startTime, venuePageSize, "0xa")
		}
		return []uint64{startTime}, []string{"0xa" + strconv.Itoa(venuePageSize-1)}
	})
	result, err = repeated.getUserNonFundingLedgerUpdatesPaginated(t.Context(), testAccountAddress, start)
	require.NoError(t, err, "getUserNonFundingLedgerUpdatesPaginated must not error for a repeated boundary record")
	assert.Len(t, result, venuePageSize, "getUserNonFundingLedgerUpdatesPaginated should drop a boundary record repeated on the next page")

	_, err = ledgerServerExchange(t, func(startTime uint64, _ int32) ([]uint64, []string) {
		return page(startTime, venuePageSize+1, "0xa")
	}).getUserNonFundingLedgerUpdatesPaginated(t.Context(), testAccountAddress, start)
	assert.ErrorIs(t, err, errUnexpectedResponseLength, "getUserNonFundingLedgerUpdatesPaginated should reject a page larger than the venue's")

	for _, tc := range []struct {
		name string
		page func(startTime uint64, number int32) ([]uint64, []string)
	}{
		{name: "record before the cursor", page: func(startTime uint64, _ int32) ([]uint64, []string) {
			return []uint64{startTime - 1}, []string{"0xa"}
		}},
		{name: "decreasing times", page: func(startTime uint64, _ int32) ([]uint64, []string) {
			return []uint64{startTime + 2, startTime + 1}, []string{"0xa", "0xb"}
		}},
		{name: "stalled cursor", page: func(startTime uint64, _ int32) ([]uint64, []string) {
			times, hashes := page(startTime, venuePageSize, "0xa")
			for i := range times {
				times[i] = startTime
			}
			return times, hashes
		}},
	} {
		_, err := ledgerServerExchange(t, tc.page).getUserNonFundingLedgerUpdatesPaginated(t.Context(), testAccountAddress, start)
		assert.ErrorIsf(t, err, errMalformedResponse, "getUserNonFundingLedgerUpdatesPaginated should reject a %s", tc.name)
	}
	_, err = newUnavailableServerExchange(t).getUserNonFundingLedgerUpdatesPaginated(t.Context(), testAccountAddress, start)
	assert.ErrorIs(t, err, request.ErrBadStatus, "getUserNonFundingLedgerUpdatesPaginated should return a ledger failure")
}

func TestConvertUserLedgerUpdate(t *testing.T) {
	t.Parallel()
	_, err := e.convertUserLedgerUpdate(testAccountAddress, &UserLedgerUpdateResponse{Delta: UserLedgerDelta{Type: " "}})
	require.ErrorIs(t, err, errMalformedResponse, "convertUserLedgerUpdate must reject an update without a type")

	entry := func(description, ccy string, amount, fee float64, delta *UserLedgerDelta) exchange.FundingHistory {
		return exchange.FundingHistory{
			ExchangeName:      e.Name,
			Status:            "processed",
			TransferID:        "0x1",
			Description:       description,
			Timestamp:         time.UnixMilli(1791200000000).UTC(),
			Currency:          ccy,
			Amount:            amount,
			Fee:               fee,
			TransferType:      delta.Type,
			CryptoToAddress:   delta.Destination,
			CryptoFromAddress: delta.User,
			CryptoTxID:        "0x1",
		}
	}
	// A fee without a fee token is in USDC, so it stays in a USDC movement's entry and has its own beside any other
	for _, tc := range []struct {
		delta       UserLedgerDelta
		currency    string
		amount      float64
		description string
	}{
		{delta: UserLedgerDelta{Type: "deposit", USDC: 10}, currency: "USDC", amount: 10, description: "deposit"},
		{delta: UserLedgerDelta{Type: "withdraw", USDC: 11}, currency: "USDC", amount: 11, description: "withdraw"},
		{delta: UserLedgerDelta{Type: "spotTransfer", Token: currency.HYPE, Amount: 2, USDCValue: 186}, currency: "HYPE", amount: 2, description: "spotTransfer"},
		{delta: UserLedgerDelta{Type: "spotGenesis", Token: currency.HYPE, Amount: 3}, currency: "HYPE", amount: 3, description: "spotGenesis"},
		{delta: UserLedgerDelta{Type: "send", Token: currency.USDC, Amount: 4, SourceDEX: "spot", DestinationDEX: "xyz"}, currency: "USDC", amount: 4, description: "send: spot to xyz"},
		{delta: UserLedgerDelta{Type: "send", Token: currency.USDC, Amount: 4, SourceDEX: "spot"}, currency: "USDC", amount: 4, description: "send: spot to perpetual"},
		{delta: UserLedgerDelta{Type: "send", Token: currency.USDC, Amount: 4, DestinationDEX: "xyz"}, currency: "USDC", amount: 4, description: "send: perpetual to xyz"},
		{delta: UserLedgerDelta{Type: "rewardsClaim", Token: currency.USDC, Amount: 5}, currency: "USDC", amount: 5, description: "rewardsClaim"},
		{delta: UserLedgerDelta{Type: "deployGasAuction", Token: currency.HYPE, Amount: 9}, currency: "HYPE", amount: 9, description: "deployGasAuction"},
		{delta: UserLedgerDelta{Type: "vaultWithdraw", NetWithdrawnUSD: 6, RequestedUSD: 6.5}, currency: "USDC", amount: 6, description: "vaultWithdraw"},
		{delta: UserLedgerDelta{Type: "accountClassTransfer", USDC: 7}, currency: "USDC", amount: 7, description: "perpetual to spot"},
		{delta: UserLedgerDelta{Type: "accountClassTransfer", USDC: 8, ToPerpetual: true}, currency: "USDC", amount: 8, description: "spot to perpetual"},
		{delta: UserLedgerDelta{Type: "cStakingTransfer", Token: currency.HYPE, Amount: 1, IsDeposit: true}, currency: "HYPE", amount: 1, description: "spot to staking"},
		{delta: UserLedgerDelta{Type: "cStakingTransfer", Token: currency.HYPE, Amount: 0.5}, currency: "HYPE", amount: 0.5, description: "staking to spot"},
		{delta: UserLedgerDelta{Type: "borrowLend", Token: currency.USDC, Amount: 50, Operation: "repay", InterestAmount: 0.0125}, currency: "USDC", amount: 50, description: "borrowLend: repay with 0.0125 interest"},
		{delta: UserLedgerDelta{Type: "borrowLend", Token: currency.HYPE, Amount: 2, Operation: "supply"}, currency: "HYPE", amount: 2, description: "borrowLend: supply"},
		{delta: UserLedgerDelta{Type: "activateDexAbstraction", Token: currency.USDC, Amount: 1, DEX: "xyz"}, currency: "USDC", amount: 1, description: "activateDexAbstraction: xyz"},
		{delta: UserLedgerDelta{Type: "activateDexAbstraction", Token: currency.USDC, Amount: 1}, currency: "USDC", amount: 1, description: "activateDexAbstraction: perpetual"},
	} {
		tc.delta.Fee = 0.5
		tc.delta.User = testAccountAddress
		tc.delta.Destination = testWithdrawalAddress
		result, err := e.convertUserLedgerUpdate(testAccountAddress, &UserLedgerUpdateResponse{Delta: tc.delta, Hash: "0x1", Time: milli(1791200000000)})
		require.NoErrorf(t, err, "convertUserLedgerUpdate must not error for %s", tc.description)
		exp := []exchange.FundingHistory{entry(tc.description, tc.currency, tc.amount, 0.5, &tc.delta)}
		if tc.currency != "USDC" {
			exp = []exchange.FundingHistory{entry(tc.description, tc.currency, tc.amount, 0, &tc.delta), entry(tc.description+": fee", "USDC", 0, 0.5, &tc.delta)}
		}
		assert.Equalf(t, exp, result, "convertUserLedgerUpdate should convert a %s update", tc.description)
	}

	// Live shape: a first transfer to a new account carries the sender's 1 USDC activation fee in both accounts' ledgers
	sent := UserLedgerDelta{Type: "spotTransfer", Token: currency.NewCode("UBTC"), Amount: 0.0139556728, USDCValue: 1163.568175, User: testAccountAddress, Destination: testWithdrawalAddress, Fee: 1, FeeToken: currency.USDC}
	nativeFee := UserLedgerDelta{Type: "send", Token: currency.USDC, Amount: 151, User: testAccountAddress, Destination: testWithdrawalAddress, NativeTokenFee: 0.01115548, SourceDEX: "spot", DestinationDEX: "spot"}
	ownToken := UserLedgerDelta{Type: "spotTransfer", Token: currency.HYPE, Amount: 2, User: strings.ToUpper(testAccountAddress), Destination: testWithdrawalAddress, Fee: 0.1, FeeToken: currency.HYPE}
	withdrawal := UserLedgerDelta{Type: "withdraw", USDC: 11, Fee: 1}
	for _, tc := range []struct {
		name    string
		account string
		delta   *UserLedgerDelta
		exp     []exchange.FundingHistory
	}{
		{name: "sender", account: testAccountAddress, delta: &sent, exp: []exchange.FundingHistory{entry("spotTransfer", "UBTC", 0.0139556728, 0, &sent), entry("spotTransfer: fee", "USDC", 0, 1, &sent)}},
		{name: "recipient", account: testWithdrawalAddress, delta: &sent, exp: []exchange.FundingHistory{entry("spotTransfer", "UBTC", 0.0139556728, 0, &sent)}},
		{name: "native token fee", account: testAccountAddress, delta: &nativeFee, exp: []exchange.FundingHistory{entry("send: spot to spot", "USDC", 151, 0, &nativeFee), entry("send: spot to spot: native token fee", "HYPE", 0, 0.01115548, &nativeFee)}},
		{name: "fee in the movement's token", account: testAccountAddress, delta: &ownToken, exp: []exchange.FundingHistory{entry("spotTransfer", "HYPE", 2, 0.1, &ownToken)}},
		{name: "update without a user", account: testAccountAddress, delta: &withdrawal, exp: []exchange.FundingHistory{entry("withdraw", "USDC", 11, 1, &withdrawal)}},
	} {
		result, err := e.convertUserLedgerUpdate(tc.account, &UserLedgerUpdateResponse{Delta: *tc.delta, Hash: "0x1", Time: milli(1791200000000)})
		require.NoErrorf(t, err, "convertUserLedgerUpdate must not error for the %s", tc.name)
		assert.Equalf(t, tc.exp, result, "convertUserLedgerUpdate should report the fee of the %s", tc.name)
	}
}

func TestLedgerDEXName(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "perpetual", ledgerDEXName(""), "ledgerDEXName should name the first perpetual DEX")
	assert.Equal(t, "spot", ledgerDEXName("spot"), "ledgerDEXName should keep the spot balance's name")
	assert.Equal(t, "xyz", ledgerDEXName("xyz"), "ledgerDEXName should keep a builder DEX's name")
}

// newTradingServerExchange returns an info server exchange with the BTC, xyz:XYZ100 and HYPE spot markets mapped; its
// metadata lists the same markets, so a refetch keeps the mappings, and the xyz DEX has no open orders, unless responses
// replaces them
func newTradingServerExchange(t *testing.T, responses map[string]string, actions func(action map[string]any) string) *Exchange {
	t.Helper()
	merged := map[string]string{
		"perpDexs":               `[null,{"name":"xyz"}]`,
		"meta:":                  `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40}]}`,
		"meta:xyz":               `{"universe":[{"name":"xyz:XYZ100","szDecimals":4,"maxLeverage":30}]}`,
		"spotMeta":               `{"universe":[{"tokens":[150,0],"name":"@107","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150,"szDecimals":2}]}`,
		"frontendOpenOrders:xyz": `[]`,
		"openOrders:xyz":         `[]`,
	}
	maps.Copy(merged, responses)
	ex := newInfoServerExchange(t, merged, actions)
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	ex.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	return ex
}

// newCountingServerExchange returns an exchange watching the mock signing account whose info requests receive the
// response keyed by their type and DEX as "type:dex", else by their type, with the perpetual DEX registry defaulting to
// only the default DEX, and fail otherwise; requests returns how many info requests of each key it has received
func newCountingServerExchange(t *testing.T, responses map[string]string) (ex *Exchange, requests func() map[string]int) {
	t.Helper()
	var mu sync.Mutex
	counts := make(map[string]int)
	ex = newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		key := body.Type + ":" + body.DEX
		mu.Lock()
		counts[key]++
		mu.Unlock()
		response, ok := responses[key]
		if !ok {
			response, ok = responses[body.Type]
		}
		if !ok && body.Type == "perpDexs" {
			response, ok = `[null]`, true
		}
		if !ok {
			http.Error(w, "unexpected info request "+key, http.StatusBadRequest)
			return
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	ex.Name = t.Name()
	setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress})
	return ex, func() map[string]int {
		mu.Lock()
		defer mu.Unlock()
		return maps.Clone(counts)
	}
}

// respondWith returns an exchange action handler that answers every action with the body
func respondWith(body string) func(map[string]any) string {
	return func(map[string]any) string { return body }
}

// captureAction returns an exchange action handler that stores each action as JSON and answers it with the body
func captureAction(t *testing.T, captured *string, body string) func(map[string]any) string {
	t.Helper()
	return func(action map[string]any) string {
		encoded, err := json.Marshal(action)
		assert.NoError(t, err, "Encoding the action should not error")
		*captured = string(encoded)
		return body
	}
}

// perpetualTestMarket returns the mapping and current mid price of perpetualPair's market
func perpetualTestMarket(t *testing.T) (mapping pairMapping, mid float64) {
	t.Helper()
	mapping, err := e.getPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "getPairMapping must not error for the perpetual test pair")
	mids, err := e.GetAllMids(t.Context(), mapping.dex)
	require.NoError(t, err, "GetAllMids must not error")
	mid = mids[mapping.coin].Float64()
	require.Positive(t, mid, "The perpetual test pair must have a mid price")
	return mapping, mid
}

// liveOrderSize returns the smallest size in a market's size decimals whose value at the price reaches liveOrderNotional
func liveOrderSize(price float64, sizeDecimals uint64) float64 {
	scale := 1.0
	for range sizeDecimals {
		scale *= 10
	}
	return math.Ceil(liveOrderNotional/price*scale) / scale
}

// restingOrderSubmission returns a resting perpetual buy for perpetualPair: the recorded order in mock mode, or, live, a
// minimal order with a fresh client order ID priced at half the mid, far below the market
func restingOrderSubmission(t *testing.T) *order.Submit {
	t.Helper()
	submit := &order.Submit{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          perpetualPair,
		AssetType:     asset.PerpetualContract,
		TimeInForce:   order.GoodTillCancel,
		Amount:        0.001,
		Price:         85000,
		ClientOrderID: testClientOrderID,
	}
	if mockTests {
		return submit
	}
	mapping, mid := perpetualTestMarket(t)
	var err error
	submit.Price, err = roundMarketPrice(mid/2, order.Buy, asset.PerpetualContract, mapping.sizeDecimals)
	require.NoError(t, err, "roundMarketPrice must not error for half the mid price")
	submit.Amount = liveOrderSize(submit.Price, mapping.sizeDecimals)
	submit.ClientOrderID = newClientOrderID(t)
	return submit
}

// cancelOnCleanup cancels a live test order by its client order ID when the test ends, which also covers a modified
// order's replacement; an order the test already cancelled must no longer be open
func cancelOnCleanup(t *testing.T, clientOrderID string) {
	t.Helper()
	if mockTests {
		return
	}
	t.Cleanup(func() {
		// The test's context is cancelled before its cleanups run
		ctx := context.WithoutCancel(t.Context())
		if err := e.CancelOrder(ctx, &order.Cancel{ClientOrderID: clientOrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract}); err == nil {
			return
		}
		detail, err := e.GetOrderInfo(ctx, clientOrderID, perpetualPair, asset.PerpetualContract)
		if assert.NoErrorf(t, err, "GetOrderInfo should find test order %s", clientOrderID) {
			assert.Truef(t, detail.IsInactive(), "Test order %s should not be left open", clientOrderID)
		}
	})
}

// placeRestingOrder places restingOrderSubmission, which a live test cancels when it ends
func placeRestingOrder(t *testing.T) (*order.Submit, *order.SubmitResponse) {
	t.Helper()
	submit := restingOrderSubmission(t)
	result, err := e.SubmitOrder(t.Context(), submit)
	require.NoError(t, err, "SubmitOrder must not error for a resting order")
	cancelOnCleanup(t, submit.ClientOrderID)
	require.Equal(t, order.New, result.Status, "SubmitOrder must leave the order resting")
	return submit, result
}

func TestSubmitOrder(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrSubmissionIsNil, "SubmitOrder must reject a nil submission")

	limit := &order.Submit{
		Exchange:      e.Name,
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          perpetualPair,
		AssetType:     asset.PerpetualContract,
		TimeInForce:   order.GoodTillCancel,
		Amount:        0.001,
		Price:         85000,
		ClientOrderID: testClientOrderID,
	}
	watchOnly := newTradingServerExchange(t, nil, nil)
	setTestCredentials(watchOnly, &accounts.Credentials{Key: testAccountAddress})
	_, err = watchOnly.SubmitOrder(t.Context(), limit)
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "SubmitOrder must require a private key")

	// An unavailable server fails every request, so these errors prove the parameters are checked before any request
	unavailable := newUnavailableServerExchange(t)
	for _, tc := range []struct {
		name   string
		mutate func(*order.Submit)
		err    error
	}{
		{name: "quote amount", mutate: func(s *order.Submit) { s.Amount, s.QuoteAmount = 0, 85 }, err: errQuoteAmountUnsupported},
		{name: "quote amount beside an amount", mutate: func(s *order.Submit) { s.QuoteAmount = 85 }, err: errQuoteAmountUnsupported},
		{name: "invalid client order ID", mutate: func(s *order.Submit) { s.ClientOrderID = "invalid" }, err: errClientOrderIDInvalid},
		{name: "trigger price on a limit order", mutate: func(s *order.Submit) { s.TriggerPrice = 90000 }, err: errRiskManagementUnsupported},
		{name: "spot trigger order", mutate: func(s *order.Submit) {
			s.Type, s.Pair, s.AssetType, s.TriggerPrice, s.TriggerPriceType = order.StopLimit, spotPair, asset.Spot, 90, order.MarkPrice
		}, err: errTriggerOrderPerpetualOnly},
		{name: "stop entry child", mutate: func(s *order.Submit) { s.RiskManagementModes.StopEntry.Enabled = true }, err: errRiskManagementUnsupported},
		{name: "fill or kill market order", mutate: func(s *order.Submit) {
			s.Type, s.TimeInForce, s.SlippageTolerance = order.Market, order.FillOrKill, 0.01
		}, err: order.ErrUnsupportedTimeInForce},
		{name: "immediate or cancel trigger order", mutate: func(s *order.Submit) {
			s.Type, s.TimeInForce, s.TriggerPrice, s.TriggerPriceType = order.StopLimit, order.ImmediateOrCancel, 90000, order.MarkPrice
		}, err: order.ErrUnsupportedTimeInForce},
	} {
		invalid := *limit
		tc.mutate(&invalid)
		_, err = unavailable.SubmitOrder(t.Context(), &invalid)
		assert.ErrorIsf(t, err, tc.err, "SubmitOrder should reject a %s before any request", tc.name)
	}
	invalid := *limit
	invalid.Amount = 0.000001
	_, err = newTradingServerExchange(t, nil, nil).SubmitOrder(t.Context(), &invalid)
	require.ErrorIs(t, err, errSizePrecision, "SubmitOrder must reject a size beyond the market's precision")

	stopMarket := &order.Submit{
		Exchange:          e.Name,
		Type:              order.StopMarket,
		Side:              order.Sell,
		Pair:              perpetualPair,
		AssetType:         asset.PerpetualContract,
		Amount:            0.1,
		Price:             80,
		TriggerPrice:      90,
		TriggerPriceType:  order.MarkPrice,
		ReduceOnly:        true,
		SlippageTolerance: 0.1,
	}
	var action string
	trigger := newTradingServerExchange(t, nil, captureAction(t, &action, statusesResponse("order", `{"resting":{"oid":9}}`)))
	result, err := trigger.SubmitOrder(t.Context(), stopMarket)
	require.NoError(t, err, "SubmitOrder must not error for a stop market order")
	assert.JSONEq(t, `{"type":"order","orders":[{"a":0,"b":false,"p":"80","s":"0.1","r":true,"t":{"trigger":{"isMarket":true,"triggerPx":"90","tpsl":"sl"}}}],"grouping":"na"}`, action, "SubmitOrder should send the stop market order")
	exp := &order.SubmitResponse{
		Exchange:     stopMarket.Exchange,
		Type:         order.StopMarket,
		Side:         order.Sell,
		Pair:         perpetualPair,
		AssetType:    asset.PerpetualContract,
		TimeInForce:  order.UnknownTIF,
		ReduceOnly:   true,
		Price:        80,
		Amount:       0.1,
		TriggerPrice: 90,
		Status:       order.New,
		OrderID:      "9",
		Date:         result.Date,
		LastUpdated:  result.LastUpdated,
	}
	exp.RemainingAmount = 0.1
	assert.Equal(t, exp, result, "SubmitOrder should return the resting trigger order")
	stopEntry := *stopMarket
	stopEntry.ReduceOnly = false
	result, err = trigger.SubmitOrder(t.Context(), &stopEntry)
	require.NoError(t, err, "SubmitOrder must not error for a trigger order that may open a position")
	assert.JSONEq(t, `{"type":"order","orders":[{"a":0,"b":false,"p":"80","s":"0.1","r":false,"t":{"trigger":{"isMarket":true,"triggerPx":"90","tpsl":"sl"}}}],"grouping":"na"}`, action, "SubmitOrder should send a trigger order that is not reduce-only")
	assert.False(t, result.ReduceOnly, "SubmitOrder should return a trigger order that is not reduce-only")

	bracket := *limit
	bracket.Exchange = trigger.Name
	bracket.RiskManagementModes = order.RiskManagementModes{
		TakeProfit: order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90000},
		StopLoss:   order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 80000, LimitPrice: 79000, OrderType: order.Limit},
	}
	grouped := newTradingServerExchange(t, nil, captureAction(t, &action, statusesResponse("order", `{"resting":{"oid":10}}`, `"waitingForFill"`, `"waitingForFill"`)))
	result, err = grouped.SubmitOrder(t.Context(), &bracket)
	require.NoError(t, err, "SubmitOrder must not error for a bracket order")
	assert.JSONEq(t, `{"type":"order","orders":[`+
		`{"a":0,"b":true,"p":"85000","s":"0.001","r":false,"t":{"limit":{"tif":"Gtc"}},"c":"`+testClientOrderID+`"},`+
		`{"a":0,"b":false,"p":"81000","s":"0.001","r":true,"t":{"trigger":{"isMarket":true,"triggerPx":"90000","tpsl":"tp"}}},`+
		`{"a":0,"b":false,"p":"79000","s":"0.001","r":true,"t":{"trigger":{"isMarket":false,"triggerPx":"80000","tpsl":"sl"}}}],"grouping":"normalTpsl"}`,
		action, "SubmitOrder should send the parent with its take-profit and stop-loss children")
	assert.Equal(t, "10", result.OrderID, "SubmitOrder should return the parent's order ID")
	assert.NoError(t, result.SubmissionError, "SubmitOrder should accept children that wait for their parent's fill")

	for _, tc := range []struct {
		name       string
		submit     *order.Submit
		mids       string
		statuses   []string
		status     order.Status
		remaining  float64
		submission error
		err        error
	}{
		{name: "partial GTC fill", submit: limit, statuses: []string{`{"filled":{"oid":7,"totalSz":"0.0004","avgPx":"85000"}}`}, status: order.PartiallyFilledCancelled, remaining: 0.0006},
		{name: "rejected child", submit: &bracket, statuses: []string{`{"resting":{"oid":11}}`, `{"error":"bad TP"}`, `"waitingForFill"`}, status: order.New, remaining: 0.001, submission: errGroupedOrderChildFailure},
		{name: "partial ALO fill", submit: func() *order.Submit { s := *limit; s.TimeInForce = order.PostOnly; return &s }(), statuses: []string{`{"filled":{"oid":7,"totalSz":"0.0004","avgPx":"85000"}}`}, err: errActionStatusMalformed},
		{name: "partial trigger fill", submit: stopMarket, statuses: []string{`{"filled":{"oid":9,"totalSz":"0.04","avgPx":"90"}}`}, err: errActionStatusMalformed},
		{name: "rejected batch", submit: &bracket, statuses: []string{`{"error":"batch rejected"}`}, err: order.ErrUnableToPlaceOrder},
		{name: "waiting parent", submit: &bracket, statuses: []string{`"waitingForFill"`, `"waitingForFill"`, `"waitingForFill"`}, err: errActionStatusMalformed},
		{name: "over-reported fill", submit: limit, statuses: []string{`{"filled":{"oid":7,"totalSz":"0.002","avgPx":"85000"}}`}, err: errInvalidFilledSize},
		{name: "fill beyond the market's precision", submit: limit, statuses: []string{`{"filled":{"oid":7,"totalSz":"0.000400001","avgPx":"85000"}}`}, err: errInvalidFilledSize},
		{name: "missing status", submit: limit, err: errActionStatusCount},
	} {
		ex := newTradingServerExchange(t, map[string]string{"allMids": `{"BTC":"85000"}`}, respondWith(statusesResponse("order", tc.statuses...)))
		submitted, err := ex.SubmitOrder(t.Context(), tc.submit)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "SubmitOrder should reject a %s", tc.name)
			continue
		}
		require.NoErrorf(t, err, "SubmitOrder must not error for a %s", tc.name)
		assert.Equalf(t, tc.status, submitted.Status, "SubmitOrder should return the status of a %s", tc.name)
		assert.InDeltaf(t, tc.remaining, submitted.RemainingAmount, 1e-12, "SubmitOrder should return the remaining amount of a %s", tc.name)
		assert.ErrorIsf(t, submitted.SubmissionError, tc.submission, "SubmitOrder should return the submission error of a %s", tc.name)
	}
	_, err = newTradingServerExchange(t, nil, respondWith(`{"status":"err","response":"Insufficient margin to place order."}`)).SubmitOrder(t.Context(), limit)
	assert.ErrorIs(t, err, errActionResponse, "SubmitOrder should return a rejected action")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.SubmitOrder(t.Context(), limit)
	assert.ErrorIs(t, err, request.ErrBadStatus, "SubmitOrder should return an action failure")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	resting := restingOrderSubmission(t)
	result, err = e.SubmitOrder(t.Context(), resting)
	require.NoError(t, err, "SubmitOrder must not error for a resting limit order")
	cancelOnCleanup(t, resting.ClientOrderID)
	if mockTests {
		exp = &order.SubmitResponse{
			Exchange:        e.Name,
			Type:            order.Limit,
			Side:            order.Buy,
			Pair:            perpetualPair,
			AssetType:       asset.PerpetualContract,
			TimeInForce:     order.GoodTillCancel,
			Price:           85000,
			Amount:          0.001,
			RemainingAmount: 0.001,
			ClientOrderID:   testClientOrderID,
			Status:          order.New,
			OrderID:         strconv.FormatUint(testOrderID, 10),
			Date:            result.Date,
			LastUpdated:     result.LastUpdated,
		}
		assert.Equal(t, exp, result, "SubmitOrder should return the resting limit order")
	} else {
		assert.Equal(t, order.New, result.Status, "SubmitOrder should leave the limit order resting")
		assert.NotEmpty(t, result.OrderID, "SubmitOrder should return the limit order's ID")
	}

	mapping, mid := perpetualTestMarket(t)
	market := &order.Submit{Exchange: e.Name, Type: order.Market, Side: order.Buy, Pair: perpetualPair, AssetType: asset.PerpetualContract, Amount: liveOrderSize(mid, mapping.sizeDecimals), SlippageTolerance: liveMarketOrderSlippage}
	result, err = e.SubmitOrder(t.Context(), market)
	require.NoError(t, err, "SubmitOrder must not error for a market order")
	if !mockTests {
		// The market buy changes the account's position, so its fill is sold back when the test ends
		filled := result.Amount - result.RemainingAmount
		t.Cleanup(func() {
			closing := &order.Submit{Exchange: e.Name, Type: order.Market, Side: order.Sell, Pair: perpetualPair, AssetType: asset.PerpetualContract, Amount: filled, SlippageTolerance: liveMarketOrderSlippage}
			_, err := e.SubmitOrder(context.WithoutCancel(t.Context()), closing)
			assert.NoError(t, err, "SubmitOrder should sell back the test market order's fill")
		})
		assert.Positive(t, result.AverageExecutedPrice, "SubmitOrder should return the market order's average price")
		return
	}
	exp = &order.SubmitResponse{
		Exchange:             e.Name,
		Type:                 order.Market,
		Side:                 order.Buy,
		Pair:                 perpetualPair,
		AssetType:            asset.PerpetualContract,
		TimeInForce:          order.ImmediateOrCancel,
		Price:                86670,
		Amount:               0.00013,
		AverageExecutedPrice: 85813,
		Status:               order.Filled,
		OrderID:              "566563303542",
		Date:                 result.Date,
		LastUpdated:          result.LastUpdated,
	}
	assert.Equal(t, exp, result, "SubmitOrder should return the filled market order, bounded above the mid price by the slippage tolerance")
}

func TestModifyOrder(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyOrder(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrModifyOrderIsNil, "ModifyOrder must reject a nil modification")

	const openOrderStatus = `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1791275926947,"isTrigger":false,"triggerPx":"0","reduceOnly":true,"orderType":"Limit","tif":"Gtc","cloid":"0x00000000000000000000000000000001"},"status":"open","statusTimestamp":1791275930650}}`
	watchOnly := newTradingServerExchange(t, nil, nil)
	setTestCredentials(watchOnly, &accounts.Credentials{Key: testAccountAddress})
	_, err = watchOnly.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "ModifyOrder must require a private key")
	// The wrapper standards modify an IOC limit order with a watch-only address, which accepts only the credential failure
	_, err = watchOnly.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Type: order.Limit, TimeInForce: order.ImmediateOrCancel})
	require.ErrorIs(t, err, request.ErrAuthRequestFailed, "ModifyOrder must check the credentials before the modification")

	// An unavailable server fails every request, so these errors prove the modification is checked before any request
	unavailable := newUnavailableServerExchange(t)
	for _, tc := range []struct {
		modify *order.Modify
		err    error
	}{
		{modify: &order.Modify{OrderID: "bad", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: order.ErrOrderIDNotSet},
		{modify: &order.Modify{OrderID: "0", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: order.ErrOrderIDNotSet},
		{modify: &order.Modify{ClientOrderID: "invalid", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: errClientOrderIDInvalid},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Type: order.Market}, err: order.ErrUnsupportedOrderType},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Type: order.StopMarket}, err: errModifyOrderTypeUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, TimeInForce: order.ImmediateOrCancel}, err: order.ErrUnsupportedTimeInForce},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, TimeInForce: order.FillOrKill}, err: order.ErrUnsupportedTimeInForce},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, TriggerPrice: 90}, err: errRiskManagementUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110}}}, err: errRiskManagementUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90}}}, err: errRiskManagementUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, RiskManagementModes: order.RiskManagementModes{StopEntry: order.RiskManagement{Enabled: true, Price: 90}}}, err: errRiskManagementUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, RiskManagementModes: order.RiskManagementModes{StopLoss: order.RiskManagement{Enabled: true}}}, err: errRiskManagementUnsupported},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.AnySide}, err: order.ErrSideIsInvalid},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.ClosePosition}, err: order.ErrSideIsInvalid},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.CouldNotBuy}, err: order.ErrSideIsInvalid},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: order.MissingData}, err: order.ErrSideIsInvalid},
		{modify: &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, NewClientOrderID: "invalid"}, err: errClientOrderIDInvalid},
	} {
		_, err := unavailable.ModifyOrder(t.Context(), tc.modify)
		assert.ErrorIsf(t, err, tc.err, "ModifyOrder should reject %+v before any request", tc.modify)
	}
	// These pass the checks and fail only at the order lookup
	for _, side := range []order.Side{order.Bid, order.Ask, order.Long, order.Short} {
		_, err := unavailable.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Side: side})
		assert.ErrorIsf(t, err, request.ErrBadStatus, "ModifyOrder should accept the %s side and look the order up", side)
	}
	_, err = unavailable.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, RiskManagementModes: order.RiskManagementModes{TakeProfit: order.RiskManagement{TriggerPriceType: order.MarkPrice, Price: 110}}})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ModifyOrder should accept a take profit that is not enabled and look the order up")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, nil).ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Amount: 1.000001})
	assert.ErrorIs(t, err, errSizePrecision, "ModifyOrder should reject a size beyond the market's precision")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, nil).ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Amount: 1})
	assert.ErrorIs(t, err, order.ErrAmountIsInvalid, "ModifyOrder should reject an amount the executed size already reaches")
	_, err = unavailable.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, request.ErrBadStatus, "ModifyOrder should return an order status failure")
	triggerOrder := newTradingServerExchange(t, map[string]string{"orderStatus": `{"status":"order","order":{"order":{"coin":"BTC","side":"A","limitPx":"80","sz":"1","origSz":"1","oid":12,"timestamp":1791275926947,"isTrigger":true,"triggerPx":"90","reduceOnly":true,"orderType":"Stop Market","tif":null},"status":"open","statusTimestamp":1791275930650}}`}, nil)
	_, err = triggerOrder.ModifyOrder(t.Context(), &order.Modify{OrderID: "12", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errModifyOrderTypeUnsupported, "ModifyOrder should reject a trigger order, which only always-place modifies replace")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`, "frontendOpenOrders": `[]`}, nil).ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "ModifyOrder should report an unknown order")
	filledOrder := strings.Replace(strings.Replace(openOrderStatus, `"sz":"1"`, `"sz":"0"`, 1), `"status":"open"`, `"status":"filled"`, 1)
	filled, requests := newCountingServerExchange(t, map[string]string{"orderStatus": filledOrder})
	setTestCredentials(filled, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
	filled.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	_, err = filled.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errOrderNotModifiable, "ModifyOrder should reject a filled order")
	assert.NotContains(t, requests(), "userFillsByTime:", "ModifyOrder should not read the fills of the order it looks up")

	var action string
	explicit := newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, captureAction(t, &action, statusesResponse("order", `{"resting":{"oid":8}}`)))
	result, err := explicit.ModifyOrder(t.Context(), &order.Modify{
		OrderID:          "7",
		Pair:             perpetualPair,
		AssetType:        asset.PerpetualContract,
		Price:            101,
		Amount:           1.5,
		Side:             order.Sell,
		Type:             order.Limit,
		TimeInForce:      order.PostOnly,
		NewClientOrderID: "0x00000000000000000000000000000002",
	})
	require.NoError(t, err, "ModifyOrder must not error")
	assert.JSONEq(t, `{"type":"batchModify","modifies":[{"oid":7,"order":{"a":0,"b":false,"p":"101","s":"0.5","r":true,"t":{"limit":{"tif":"Alo"}},"c":"0x00000000000000000000000000000002"}}]}`, action, "ModifyOrder should send a replacement of the amount less the executed size")
	exp := &order.ModifyResponse{
		Exchange:        explicit.Name,
		OrderID:         "8",
		ClientOrderID:   "0x00000000000000000000000000000002",
		Pair:            perpetualPair,
		Type:            order.Limit,
		Side:            order.Sell,
		Status:          order.New,
		AssetType:       asset.PerpetualContract,
		TimeInForce:     order.PostOnly,
		Price:           101,
		Amount:          0.5,
		RemainingAmount: 0.5,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     result.LastUpdated,
	}
	assert.Equal(t, exp, result, "ModifyOrder should return the resting replacement")

	for _, tc := range []struct {
		name, original, remaining string
		total, replacement        float64
		err                       error
	}{
		{name: "omitted total", original: "2", remaining: "1", replacement: 1},
		{name: "engine price only", original: "2", remaining: "1", total: 2, replacement: 1},
		{name: "unexecuted", original: "2", remaining: "2", total: 1.5, replacement: 1.5},
		{name: "below executed", original: "2", remaining: "1", total: 0.5, err: order.ErrAmountIsInvalid},
		{name: "equal executed", original: "2", remaining: "1", total: 1, err: order.ErrAmountIsInvalid},
		{name: "over precise total", original: "2", remaining: "1", total: 1.000001, err: errSizePrecision},
		{name: "over precise below executed", original: "2", remaining: "1", total: 0.000001, err: errSizePrecision},
		{name: "minimum remaining unit", original: "2", remaining: "1", total: 1.00001, replacement: 0.00001},
		{name: "scaled total below a whole unit", original: "2", remaining: "1", total: 1.00002, replacement: 0.00002},
		{name: "floating subtraction residue", original: "0.3", remaining: "0.2", total: 0.3, replacement: 0.2},
		{name: "inherited floating subtraction residue", original: "0.3", remaining: "0.2", replacement: 0.2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			status := strings.ReplaceAll(strings.ReplaceAll(openOrderStatus, `"origSz":"2"`, `"origSz":"`+tc.original+`"`), `"sz":"1"`, `"sz":"`+tc.remaining+`"`)
			var calls atomic.Int32
			ex := newTradingServerExchange(t, map[string]string{"orderStatus": status}, func(action map[string]any) string {
				calls.Add(1)
				encoded, err := json.Marshal(action)
				assert.NoError(t, err, "Marshal should encode the replacement")
				assert.Contains(t, string(encoded), `"s":"`+strconv.FormatFloat(tc.replacement, 'f', -1, 64)+`"`, "ModifyOrder should send only the unexecuted requested size")
				return statusesResponse("order", `{"resting":{"oid":8}}`)
			})
			modified, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract, Price: 99, Amount: tc.total})
			require.ErrorIs(t, err, tc.err, "ModifyOrder must return the expected error")
			if tc.err != nil {
				if tc.err == order.ErrAmountIsInvalid {
					assert.ErrorContains(t, err, fmt.Sprintf("total %v does not exceed executed size 1", tc.total), "ModifyOrder should explain the total and executed size conflict")
				}
				assert.Zero(t, calls.Load(), "ModifyOrder should reject invalid totals before an exchange request")
				assert.Nil(t, modified, "ModifyOrder should return no invalid replacement")
				return
			}
			require.NotNil(t, modified, "ModifyOrder must return a replacement")
			assert.Equal(t, int32(1), calls.Load(), "ModifyOrder should send one replacement")
			assert.Equal(t, tc.replacement, modified.Amount, "ModifyOrder should report the replacement total")
			assert.Equal(t, tc.replacement, modified.RemainingAmount, "ModifyOrder should report the resting replacement size")
		})
	}

	// OrderManager defaults an omitted amount to the stored total; each replacement starts its own execution history.
	for _, tc := range []struct {
		name, remaining string
		exp             float64
	}{
		{name: "resting replacement", remaining: "1", exp: 1},
		{name: "partly filled replacement", remaining: "0.6", exp: 0.6},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			stored := &order.Detail{OrderID: "7", Amount: 2, ExecutedAmount: 1, RemainingAmount: 1}
			for _, tc := range []struct {
				status      string
				replacement uint64
				exp         float64
			}{
				{status: openOrderStatus, replacement: 8, exp: 1},
				{status: strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(openOrderStatus, `"origSz":"2"`, `"origSz":"1"`), `"sz":"1"`, `"sz":"`+tc.remaining+`"`), `"oid":7`, `"oid":8`), replacement: 9, exp: tc.exp},
			} {
				var replacementAction string
				ex := newTradingServerExchange(t, map[string]string{"orderStatus": tc.status}, captureAction(t, &replacementAction, statusesResponse("order", `{"resting":{"oid":`+strconv.FormatUint(tc.replacement, 10)+`}}`)))
				modified, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: stored.OrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract, Price: 99, Amount: stored.Amount})
				require.NoError(t, err, "ModifyOrder must accept consecutive engine amendments")
				assert.Contains(t, replacementAction, `"s":"`+strconv.FormatFloat(tc.exp, 'f', -1, 64)+`"`, "ModifyOrder should subtract only the current venue order's execution")
				stored.UpdateOrderFromModifyResponse(modified)
				assert.Equal(t, strconv.FormatUint(tc.replacement, 10), stored.OrderID, "UpdateOrderFromModifyResponse should track the replacement ID")
				assert.Equal(t, tc.exp, stored.Amount, "UpdateOrderFromModifyResponse should track the replacement total")
			}
		})
	}

	inherited := newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, captureAction(t, &action, statusesResponse("order", `{"resting":{"oid":8}}`)))
	result, err = inherited.ModifyOrder(t.Context(), &order.Modify{ClientOrderID: "0X00000000000000000000000000000001", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	require.NoError(t, err, "ModifyOrder must not error by client order ID")
	assert.JSONEq(t, `{"type":"batchModify","modifies":[{"oid":"0x00000000000000000000000000000001","order":{"a":0,"b":true,"p":"100","s":"1","r":true,"t":{"limit":{"tif":"Gtc"}},"c":"0x00000000000000000000000000000001"}}]}`, action, "ModifyOrder should replace the order with its own remaining size and fields")
	assert.Equal(t, "8", result.OrderID, "ModifyOrder should return the order ID")

	for _, tc := range []struct {
		status    string
		exp       order.Status
		remaining float64
		err       error
	}{
		{status: `{"filled":{"oid":7,"totalSz":"1","avgPx":"100"}}`, exp: order.Filled},
		{status: `{"filled":{"oid":7,"totalSz":"0.4","avgPx":"100"}}`, exp: order.PartiallyFilledCancelled, remaining: 0.6},
		{status: `{"filled":{"oid":7,"totalSz":"2","avgPx":"100"}}`, err: errInvalidFilledSize},
		{status: `{"filled":{"oid":7,"totalSz":"0.400001","avgPx":"100"}}`, err: errInvalidFilledSize},
		{status: `{"error":"Cannot modify canceled or filled order"}`, err: order.ErrUnableToPlaceOrder},
	} {
		ex := newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, respondWith(statusesResponse("order", tc.status)))
		modified, err := ex.ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "ModifyOrder should reject status %s", tc.status)
			continue
		}
		require.NoErrorf(t, err, "ModifyOrder must not error for status %s", tc.status)
		assert.Equalf(t, tc.exp, modified.Status, "ModifyOrder should derive the status of %s", tc.status)
		assert.InDeltaf(t, tc.remaining, modified.RemainingAmount, 1e-12, "ModifyOrder should derive the remaining amount of %s", tc.status)
	}
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, respondWith(statusesResponse("order"))).ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errActionStatusCount, "ModifyOrder should reject a response without a status")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, respondWith(`{"status":"err","response":"modify failed"}`)).ModifyOrder(t.Context(), &order.Modify{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errActionResponse, "ModifyOrder should return a rejected action")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	submit, resting := placeRestingOrder(t)
	price := 84000.0
	if !mockTests {
		// The replacement must rest too, so it moves a tenth further below the market
		mapping, err := e.getPairMapping(t.Context(), perpetualPair, asset.PerpetualContract)
		require.NoError(t, err, "getPairMapping must not error for the perpetual test pair")
		price, err = roundMarketPrice(submit.Price*0.9, order.Buy, asset.PerpetualContract, mapping.sizeDecimals)
		require.NoError(t, err, "roundMarketPrice must not error for the replacement price")
	}
	result, err = e.ModifyOrder(t.Context(), &order.Modify{OrderID: resting.OrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract, Price: price})
	require.NoError(t, err, "ModifyOrder must not error")
	if !mockTests {
		assert.Equal(t, order.New, result.Status, "ModifyOrder should leave the replacement resting")
		assert.Equal(t, price, result.Price, "ModifyOrder should return the replacement's price")
		assert.Equal(t, submit.ClientOrderID, result.ClientOrderID, "ModifyOrder should keep the order's client order ID")
		return
	}
	exp = &order.ModifyResponse{
		Exchange:        e.Name,
		OrderID:         "566563303537",
		ClientOrderID:   testClientOrderID,
		Pair:            perpetualPair,
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.New,
		AssetType:       asset.PerpetualContract,
		TimeInForce:     order.PostOnly,
		Price:           84000,
		Amount:          0.001,
		RemainingAmount: 0.001,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     result.LastUpdated,
	}
	assert.Equal(t, exp, result, "ModifyOrder should return the replacement order")
}

func TestValidateModifiableOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType   order.Type
		timeInForce order.TimeInForce
		err         error
	}{
		{orderType: order.Limit},
		{orderType: order.Limit, timeInForce: order.GoodTillCancel},
		{orderType: order.Limit, timeInForce: order.PostOnly},
		{orderType: order.Market, err: errModifyOrderTypeUnsupported},
		{orderType: order.StopMarket, err: order.ErrUnsupportedOrderType},
		{orderType: order.Limit, timeInForce: order.ImmediateOrCancel, err: errModifyOrderTypeUnsupported},
		{orderType: order.Limit, timeInForce: order.FillOrKill, err: order.ErrUnsupportedTimeInForce},
	} {
		assert.ErrorIsf(t, validateModifiableOrder(tc.orderType, tc.timeInForce), tc.err, "validateModifiableOrder should return the expected error for a %s %s order", tc.orderType, tc.timeInForce)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.CancelOrder(t.Context(), nil), order.ErrCancelOrderIsNil, "CancelOrder must reject a nil cancel")
	require.ErrorIs(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: "bad", Pair: perpetualPair, AssetType: asset.PerpetualContract}), order.ErrOrderIDNotSet, "CancelOrder must reject an invalid order ID")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	_, byOrderID := placeRestingOrder(t)
	byClientOrderID, _ := placeRestingOrder(t)
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{OrderID: byOrderID.OrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract}), "CancelOrder should cancel an order by its order ID")
	assert.NoError(t, e.CancelOrder(t.Context(), &order.Cancel{ClientOrderID: byClientOrderID.ClientOrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract}), "CancelOrder should cancel an order by its client order ID")
}

func TestCancelBatchOrders(t *testing.T) {
	t.Parallel()
	_, err := e.CancelBatchOrders(t.Context(), []order.Cancel{{ClientOrderID: "invalid", Pair: perpetualPair, AssetType: asset.PerpetualContract}})
	require.ErrorIs(t, err, errClientOrderIDInvalid, "CancelBatchOrders must reject an invalid client order ID")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	_, byOrderID := placeRestingOrder(t)
	byClientOrderID, _ := placeRestingOrder(t)
	result, err := e.CancelBatchOrders(t.Context(), []order.Cancel{
		{OrderID: byOrderID.OrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract},
		{ClientOrderID: byClientOrderID.ClientOrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract},
	})
	require.NoError(t, err, "CancelBatchOrders must not error")
	if mockTests {
		exp := &order.CancelBatchResponse{Status: map[string]string{strconv.FormatUint(testOrderID, 10): "success", testClientOrderID: "success"}}
		assert.Equal(t, exp, result, "CancelBatchOrders should return each cancel's status")
		return
	}
	for _, identifier := range []string{byOrderID.OrderID, byClientOrderID.ClientOrderID} {
		assert.Equalf(t, "success", result.Status[identifier], "CancelBatchOrders should cancel order %s", identifier)
	}
}

func TestCancelOrderBatch(t *testing.T) {
	t.Parallel()
	var actions []string
	ex := newTradingServerExchange(t, nil, func(action map[string]any) string {
		encoded, err := json.Marshal(action)
		assert.NoError(t, err, "Encoding the action should not error")
		actions = append(actions, string(encoded))
		cancels, _ := action["cancels"].([]any)
		return statusesResponse("cancel", slices.Repeat([]string{`"success"`}, len(cancels))...)
	})
	statuses, err := ex.cancelOrderBatch(t.Context(), nil)
	require.NoError(t, err, "cancelOrderBatch must not error for no cancels")
	assert.Empty(t, statuses, "cancelOrderBatch should return no statuses for no cancels")
	_, err = ex.cancelOrderBatch(t.Context(), make([]order.Cancel, maximumActionBatchSize+1))
	require.ErrorIs(t, err, errActionBatchTooLarge, "cancelOrderBatch must reject an oversized batch")

	for _, tc := range []struct {
		cancel order.Cancel
		err    error
	}{
		{cancel: order.Cancel{OrderID: "1", AssetType: asset.PerpetualContract}, err: order.ErrPairIsEmpty},
		{cancel: order.Cancel{OrderID: "1", Pair: perpetualPair}, err: order.ErrAssetNotSet},
		{cancel: order.Cancel{OrderID: "1", Pair: currency.NewPair(currency.ETH, currency.USDC), AssetType: asset.PerpetualContract}, err: errPairMappingNotFound},
		{cancel: order.Cancel{OrderID: "bad", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: order.ErrOrderIDNotSet},
		{cancel: order.Cancel{OrderID: "0", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: order.ErrOrderIDNotSet},
		{cancel: order.Cancel{ClientOrderID: "invalid", Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: errClientOrderIDInvalid},
		{cancel: order.Cancel{Pair: perpetualPair, AssetType: asset.PerpetualContract}, err: order.ErrOrderIDNotSet},
	} {
		_, err := ex.cancelOrderBatch(t.Context(), []order.Cancel{tc.cancel})
		assert.ErrorIsf(t, err, tc.err, "cancelOrderBatch should reject %+v", tc.cancel)
	}
	assert.Empty(t, actions, "cancelOrderBatch should not send an invalid batch")
	// An unavailable server fails every request, so this error proves identifiers are checked before any market lookup
	_, err = newUnavailableServerExchange(t).cancelOrderBatch(t.Context(), []order.Cancel{
		{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract},
		{OrderID: "bad", Pair: perpetualPair, AssetType: asset.PerpetualContract},
	})
	assert.ErrorIs(t, err, order.ErrOrderIDNotSet, "cancelOrderBatch should check every identifier before looking up any market")

	upperClientOrderID := strings.ToUpper(testClientOrderID[2:])
	statuses, err = ex.cancelOrderBatch(t.Context(), []order.Cancel{
		{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract},
		{ClientOrderID: "0x" + upperClientOrderID, Pair: spotPair, AssetType: asset.Spot},
		{OrderID: "8", Pair: testBuilderPair, AssetType: asset.PerpetualContract},
	})
	require.NoError(t, err, "cancelOrderBatch must not error")
	assert.Equal(t, map[string]string{"7": "success", "0x" + upperClientOrderID: "success", "8": "success"}, statuses, "cancelOrderBatch should key each status by the caller's identifier")
	assert.Equal(t, []string{
		`{"cancels":[{"a":0,"o":7},{"a":110000,"o":8}],"type":"cancel"}`,
		`{"cancels":[{"asset":10107,"cloid":"` + testClientOrderID + `"}],"type":"cancelByCloid"}`,
	}, actions, "cancelOrderBatch should cancel order IDs and client order IDs in one action each")

	statuses, err = newTradingServerExchange(t, nil, respondWith(`{"status":"err","response":"cancel failed"}`)).cancelOrderBatch(t.Context(), []order.Cancel{
		{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract},
		{ClientOrderID: testClientOrderID, Pair: perpetualPair, AssetType: asset.PerpetualContract},
	})
	assert.ErrorIs(t, err, errActionResponse, "cancelOrderBatch should return a rejected action")
	assert.Empty(t, statuses, "cancelOrderBatch should not report a rejected action's cancels as successful")
	_, err = newTradingServerExchange(t, nil, respondWith(statusesResponse("cancel"))).cancelOrderBatch(t.Context(), []order.Cancel{{OrderID: "7", Pair: perpetualPair, AssetType: asset.PerpetualContract}})
	assert.ErrorIs(t, err, errActionStatusCount, "cancelOrderBatch should reject a response without a status")
}

func TestRecordCancelStatuses(t *testing.T) {
	t.Parallel()
	statuses := make(map[string]string)
	err := recordCancelStatuses(statuses, []string{"1", "2"}, []CancelActionStatusResponse{{}, {Error: "Order was never placed, already canceled, or filled."}})
	assert.ErrorIs(t, err, errActionResponse, "recordCancelStatuses should return each failed cancel")
	assert.Equal(t, map[string]string{"1": "success", "2": "Order was never placed, already canceled, or filled."}, statuses, "recordCancelStatuses should record each cancel's outcome")
}

func TestCancelAllOrders(t *testing.T) {
	// Live, CancelAllOrders cancels every open order of the test pair, so the test runs before the parallel tests place
	// their own orders on it
	if mockTests {
		t.Parallel()
	}
	_, err := e.CancelAllOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrCancelOrderIsNil, "CancelAllOrders must reject a nil cancel")
	_, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "CancelAllOrders must reject an unsupported asset")
	unconfigured := new(Exchange)
	unconfigured.SetDefaults()
	_, err = unconfigured.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "CancelAllOrders must require credentials")

	successes := func(action map[string]any) string {
		cancels, _ := action["cancels"].([]any)
		return statusesResponse("cancel", slices.Repeat([]string{`"success"`}, len(cancels))...)
	}
	openOrders := `[{"coin":"BTC","oid":7},{"coin":"@107","oid":8},{"coin":"MISSING","oid":9}]`
	ex := newTradingServerExchange(t, map[string]string{"openOrders": openOrders}, successes)
	result, err := ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, errPairMappingNotFound, "CancelAllOrders should report an order without a market")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"7": "success"}}, result, "CancelAllOrders should cancel the asset's orders")
	result, err = ex.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract, Pair: currency.NewPair(currency.ETH, currency.USDC)})
	assert.ErrorIs(t, err, errPairMappingNotFound, "CancelAllOrders should report an order without a market")
	assert.Empty(t, result.Status, "CancelAllOrders should cancel only the requested pair's orders")

	orders := make([]string, maximumActionBatchSize+1)
	for i := range orders {
		orders[i] = `{"coin":"BTC","oid":` + strconv.Itoa(i+1) + `}`
	}
	chunked := newTradingServerExchange(t, map[string]string{"openOrders": `[` + strings.Join(orders, ",") + `]`}, successes)
	result, err = chunked.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.NoError(t, err, "CancelAllOrders must not error beyond one action's batch")
	assert.Len(t, result.Status, maximumActionBatchSize+1, "CancelAllOrders should cancel every order across batches")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		_, err = newUnavailableServerExchange(t).CancelAllOrders(t.Context(), &order.Cancel{AssetType: a})
		assert.ErrorIsf(t, err, request.ErrBadStatus, "CancelAllOrders should return a %s open orders failure", a)
	}
	const missingOrder = "Order was never placed, already canceled, or filled. asset=10107"
	failed := newTradingServerExchange(t, map[string]string{"openOrders": openOrders}, respondWith(statusesResponse("cancel", `{"error":"`+missingOrder+`"}`)))
	result, err = failed.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.Spot})
	assert.ErrorIs(t, err, errActionResponse, "CancelAllOrders should report a failed cancel")
	assert.Equal(t, order.CancelAllResponse{Status: map[string]string{"8": missingOrder}}, result, "CancelAllOrders should return a failed cancel's reason")
	outcomes, requests := newCountingServerExchange(t, map[string]string{"openOrders": `[{"coin":"#90700","oid":10},{"coin":"#90411","oid":11}]`})
	outcomes.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	result, err = outcomes.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract})
	require.NoError(t, err, "CancelAllOrders must leave HIP-4 outcome orders open")
	assert.Empty(t, result.Status, "CancelAllOrders should not cancel HIP-4 outcome orders")
	assert.Equal(t, map[string]int{"perpDexs:": 1, "openOrders:": 1}, requests(), "CancelAllOrders should not look up the markets of HIP-4 outcome orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
		// The test order must be the only open order of the pair, so that it is the only order CancelAllOrders cancels
		open, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair}, Side: order.AnySide, Type: order.AnyType})
		require.NoError(t, err, "GetActiveOrders must not error")
		if len(open) != 0 {
			t.Skipf("Live CancelAllOrders would cancel the account's %d open %s orders", len(open), perpetualPair)
		}
	}
	_, resting := placeRestingOrder(t)
	result, err = e.CancelAllOrders(t.Context(), &order.Cancel{AssetType: asset.PerpetualContract, Pair: perpetualPair})
	require.NoError(t, err, "CancelAllOrders must not error for the perpetual test pair")
	if mockTests {
		exp := order.CancelAllResponse{Status: map[string]string{strconv.FormatUint(testOrderID, 10): "success", "566563303543": "success"}}
		assert.Equal(t, exp, result, "CancelAllOrders should cancel the pair's perpetual orders, including its position TP/SL, and leave the outcome order open")
	} else {
		assert.Equal(t, "success", result.Status[resting.OrderID], "CancelAllOrders should cancel the test order")
	}
}

func TestGetOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderInfo(t.Context(), "", perpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderInfo must require an order ID")
	_, err = e.GetOrderInfo(t.Context(), "7", perpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderInfo must reject an unsupported asset")
	_, err = new(Exchange).GetOrderInfo(t.Context(), "7", perpetualPair, asset.Empty)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrderInfo must require an account address")

	const openOrderStatus = `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1791275926947,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1791275930650}}`
	ex := newTradingServerExchange(t, map[string]string{"orderStatus": openOrderStatus}, nil)
	_, err = ex.GetOrderInfo(t.Context(), "invalid", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorIs(t, err, errClientOrderIDInvalid, "GetOrderInfo should reject an identifier that is neither an order ID nor a client order ID")
	_, err = ex.GetOrderInfo(t.Context(), "7", spotPair, asset.Empty)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should not return an order of another pair")
	_, err = ex.GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Spot)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should not return an order of another asset")
	spotStatusOrder, err := newTradingServerExchange(t, map[string]string{"orderStatus": strings.Replace(openOrderStatus, `"coin":"BTC"`, `"coin":"@107"`, 1)}, nil).GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Empty)
	require.NoError(t, err, "GetOrderInfo must not error for a spot order")
	assert.Equal(t, asset.Spot, spotStatusOrder.AssetType, "GetOrderInfo should return a spot order under the spot asset")
	assert.Equal(t, spotPair, spotStatusOrder.Pair, "GetOrderInfo should return a spot order under its spot pair")
	for _, response := range []string{`{"status":"unknownOid"}`, `{"status":"order","order":null}`} {
		_, err = newTradingServerExchange(t, map[string]string{"orderStatus": response, "frontendOpenOrders": `[]`}, nil).GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Empty)
		assert.ErrorIsf(t, err, order.ErrOrderNotFound, "GetOrderInfo should report %s as not found", response)
	}
	// orderStatus can report an older order that is still open as unknown, so such an order is found among the open orders
	const olderOpenOrders = `[{"coin":"BTC","side":"A","limitPx":"85260.0","sz":"0.04384","origSz":"0.05","oid":573783590636,"timestamp":1791950640557,"orderType":"Limit","tif":"Alo","reduceOnly":true,"cloid":"0x0000000000000000000000000000000a"},` +
		`{"coin":"BTC","side":"A","limitPx":"80.0","sz":"1.0","origSz":"1.0","oid":573783590638,"timestamp":1791950640557,"orderType":"Stop Market","isTrigger":true,"triggerPx":"90.0","reduceOnly":true,"tif":null},` +
		`{"coin":"@107","side":"B","limitPx":"95.5","sz":"1.5","origSz":"1.5","oid":573783590637,"timestamp":1791950640557,"orderType":"Limit","tif":"Gtc"}]`
	beyondReach := newTradingServerExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`, "frontendOpenOrders": olderOpenOrders}, nil)
	placed := time.UnixMilli(1791950640557).UTC()
	// Sizes are subtracted in float64, as the conversion subtracts them
	originalSize, remainingSize := 0.05, 0.04384
	perpetualOrder := &order.Detail{TimeInForce: order.PostOnly, ReduceOnly: true, Price: 85260, Amount: originalSize, ExecutedAmount: originalSize - remainingSize, RemainingAmount: remainingSize, Exchange: beyondReach.Name, OrderID: "573783590636", ClientOrderID: "0x0000000000000000000000000000000a", Type: order.Limit, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	triggerOrder := &order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 80, Amount: 1, TriggerPrice: 90, RemainingAmount: 1, Exchange: beyondReach.Name, OrderID: "573783590638", Type: order.StopMarket, Side: order.Sell, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	spotOrder := &order.Detail{TimeInForce: order.GoodTillCancel, Price: 95.5, Amount: 1.5, RemainingAmount: 1.5, Exchange: beyondReach.Name, OrderID: "573783590637", Type: order.Limit, Side: order.Buy, Status: order.Open, AssetType: asset.Spot, Date: placed, LastUpdated: placed, Pair: spotPair}
	for _, tc := range []struct {
		identifier string
		pair       currency.Pair
		a          asset.Item
		exp        *order.Detail
	}{
		{identifier: "573783590636", pair: perpetualPair, a: asset.PerpetualContract, exp: perpetualOrder},
		{identifier: "000573783590636", exp: perpetualOrder},
		{identifier: "0x0000000000000000000000000000000a", exp: perpetualOrder},
		{identifier: "0X0000000000000000000000000000000A", exp: perpetualOrder},
		{identifier: "573783590638", pair: perpetualPair, a: asset.PerpetualContract, exp: triggerOrder},
		{identifier: "573783590637", exp: spotOrder},
		{identifier: "573783590637", pair: spotPair, a: asset.Spot, exp: spotOrder},
	} {
		result, err := beyondReach.GetOrderInfo(t.Context(), tc.identifier, tc.pair, tc.a)
		require.NoErrorf(t, err, "GetOrderInfo must find the open order %s that orderStatus no longer resolves", tc.identifier)
		assert.Equalf(t, tc.exp, result, "GetOrderInfo should return the open order %s from the open orders", tc.identifier)
	}
	for _, tc := range []struct {
		identifier string
		pair       currency.Pair
		a          asset.Item
	}{
		{identifier: "57378359063"},
		{identifier: "573783590636", pair: spotPair},
		{identifier: "573783590636", pair: spotPair, a: asset.PerpetualContract},
		{identifier: "573783590636", a: asset.Spot},
		{identifier: "573783590636", pair: perpetualPair, a: asset.Spot},
	} {
		_, err = beyondReach.GetOrderInfo(t.Context(), tc.identifier, tc.pair, tc.a)
		assert.ErrorIsf(t, err, order.ErrOrderNotFound, "GetOrderInfo should not return another open order for %s with pair %q and asset %q", tc.identifier, tc.pair, tc.a)
	}
	failedDEX := newInfoServerExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`, "perpDexs": `[null,{"name":"xyz"}]`, "frontendOpenOrders:": `[]`}, nil)
	failedDEX.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	failedDEX.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	_, err = failedDEX.GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderInfo should report a DEX's failed open order listing though the next asset type's listing succeeds")
	unknownMarket := newTradingServerExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`, "frontendOpenOrders": `[{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":8,"timestamp":1791950640557,"orderType":"Limit","tif":"Gtc"},` + olderOpenOrders[1:]}, nil)
	found, err := unknownMarket.GetOrderInfo(t.Context(), "573783590637", currency.EMPTYPAIR, asset.Empty)
	require.NoError(t, err, "GetOrderInfo must find an open spot order though a perpetual order fails to convert")
	assert.Equal(t, "573783590637", found.OrderID, "GetOrderInfo should return the requested open spot order")
	_, err = unknownMarket.GetOrderInfo(t.Context(), "9", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetOrderInfo should report an open order it cannot convert rather than an unknown order as not found")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": `{"status":"unknownOid"}`, "frontendOpenOrders": `{`}, nil).GetOrderInfo(t.Context(), "573783590636", perpetualPair, asset.PerpetualContract)
	require.Error(t, err, "GetOrderInfo must return an open order listing failure")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should not report an order as not found when its open orders cannot be read")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": strings.Replace(openOrderStatus, `"BTC"`, `"MISSING"`, 1)}, nil).GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetOrderInfo should report an order without a market")
	_, err = newUnavailableServerExchange(t).GetOrderInfo(t.Context(), "7", currency.EMPTYPAIR, asset.Empty)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderInfo should return an order status failure")

	// orderStatus lists a position TP/SL order without a size, as live, so an open one takes the size of the position it
	// closes, as GetActiveOrders sizes it
	const positionOrderStatus = `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"99000.0","sz":"0.0","origSz":"0.0","oid":14,"timestamp":1791275926947,"triggerCondition":"Price above 90000","isTrigger":true,"triggerPx":"90000.0","isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","tif":null},"status":"open","statusTimestamp":1791275926947}}`
	const btcShortPosition = `{"assetPositions":[{"type":"oneWay","position":{"coin":"ETH","szi":"3"}},{"type":"oneWay","position":{"coin":"BTC","szi":"-0.25"}}]}`
	positioned := newTradingServerExchange(t, map[string]string{"orderStatus": positionOrderStatus, "clearinghouseState": btcShortPosition}, nil)
	positionOrder, err := positioned.GetOrderInfo(t.Context(), "14", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for an open position TP/SL order")
	positionPlaced := time.UnixMilli(1791275926947).UTC()
	expPositionOrder := &order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 99000, Amount: 0.25, TriggerPrice: 90000, RemainingAmount: 0.25, Exchange: positioned.Name, OrderID: "14", Type: order.StopMarket, Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: positionPlaced, LastUpdated: positionPlaced, Pair: perpetualPair}
	assert.Equal(t, expPositionOrder, positionOrder, "GetOrderInfo should size an open position TP/SL order from its position")
	_, err = newTradingServerExchange(t, map[string]string{"orderStatus": positionOrderStatus}, nil).GetOrderInfo(t.Context(), "14", perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOrderInfo should return the positions failure of an open position TP/SL order")
	cancelledPositionOrder, err := newTradingServerExchange(t, map[string]string{"orderStatus": strings.Replace(positionOrderStatus, `"status":"open"`, `"status":"canceled"`, 1)}, nil).GetOrderInfo(t.Context(), "14", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not read positions for a cancelled position TP/SL order")
	assert.Zero(t, cancelledPositionOrder.Amount, "GetOrderInfo should leave a cancelled position TP/SL order unsized")
	sizedPositionOrder, err := newTradingServerExchange(t, map[string]string{"orderStatus": strings.NewReplacer(`"sz":"0.0"`, `"sz":"0.1"`, `"origSz":"0.0"`, `"origSz":"0.1"`).Replace(positionOrderStatus)}, nil).GetOrderInfo(t.Context(), "14", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not read positions for a position TP/SL order with a size")
	assert.Equal(t, 0.1, sizedPositionOrder.Amount, "GetOrderInfo should keep the size of a position TP/SL order placed with one")
	unsizedOrder, err := newTradingServerExchange(t, map[string]string{"orderStatus": strings.Replace(positionOrderStatus, `"isPositionTpsl":true`, `"isPositionTpsl":false`, 1)}, nil).GetOrderInfo(t.Context(), "14", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not read positions for an order that is not a position TP/SL")
	assert.Zero(t, unsizedOrder.Amount, "GetOrderInfo should size only position TP/SL orders from their positions")
	builderPositioned := newInfoServerExchange(t, map[string]string{
		"orderStatus":            strings.Replace(positionOrderStatus, `"coin":"BTC"`, `"coin":"xyz:XYZ100"`, 1),
		"clearinghouseState:":    btcShortPosition,
		"clearinghouseState:xyz": `{"assetPositions":[{"type":"oneWay","position":{"coin":"xyz:XYZ100","szi":"0.5"}}]}`,
	}, nil)
	builderPositioned.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	builderPositionOrder, err := builderPositioned.GetOrderInfo(t.Context(), "14", testBuilderPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for an open position TP/SL order on a builder-deployed DEX")
	assert.Equal(t, 0.5, builderPositionOrder.Amount, "GetOrderInfo should size a position TP/SL order from its own DEX's position")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetOrderInfo(t.Context(), "1", perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "GetOrderInfo should report an unknown order as not found")
	if !mockTests {
		// The recorded order exists only in the mock responses, so the account's latest order is looked up instead
		history, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType})
		require.NoError(t, err, "GetOrderHistory must not error")
		if len(history) == 0 {
			t.Skip("The configured account has no perpetual orders to look up")
		}
		for _, identifier := range []string{history[0].OrderID, history[0].ClientOrderID} {
			if identifier == "" {
				continue
			}
			result, err := e.GetOrderInfo(t.Context(), identifier, history[0].Pair, asset.PerpetualContract)
			require.NoErrorf(t, err, "GetOrderInfo must not error for the account's latest order %s", identifier)
			assert.Equalf(t, history[0].OrderID, result.OrderID, "GetOrderInfo should return the account's latest order for %s", identifier)
		}
		return
	}
	for _, tc := range []struct {
		orderID string
		pair    currency.Pair
		a       asset.Item
	}{
		{orderID: strconv.FormatUint(testOrderID, 10), pair: perpetualPair, a: asset.PerpetualContract},
		{orderID: testClientOrderID},
	} {
		result, err := e.GetOrderInfo(t.Context(), tc.orderID, tc.pair, tc.a)
		require.NoErrorf(t, err, "GetOrderInfo must not error for %s", tc.orderID)
		exp := &order.Detail{
			TimeInForce:     order.GoodTillCancel,
			Price:           85000,
			Amount:          0.002,
			ExecutedAmount:  0.001,
			RemainingAmount: 0.001,
			Exchange:        e.Name,
			OrderID:         strconv.FormatUint(testOrderID, 10),
			ClientOrderID:   testClientOrderID,
			Type:            order.Limit,
			Side:            order.Buy,
			Status:          order.PartiallyFilled,
			AssetType:       asset.PerpetualContract,
			Date:            time.UnixMilli(1791275926947).UTC(),
			LastUpdated:     time.UnixMilli(1791275926947).UTC(),
			Pair:            perpetualPair,
		}
		assert.Equalf(t, exp, result, "GetOrderInfo should convert the order for %s", tc.orderID)
	}
}

func TestFindOpenOrder(t *testing.T) {
	t.Parallel()
	const listing = `[{"coin":"BTC","side":"A","limitPx":"100","sz":"1","origSz":"1","oid":7,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc","cloid":"0x0000000000000000000000000000000b"},` +
		`{"coin":"@107","side":"B","limitPx":"95.5","sz":"1.5","origSz":"1.5","oid":8,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"}]`
	ex := newTradingServerExchange(t, map[string]string{"frontendOpenOrders": listing}, nil)
	for _, tc := range []struct {
		identifier *OrderStatusRequest
		pair       currency.Pair
		a          asset.Item
		exp        string
		err        error
	}{
		{identifier: &OrderStatusRequest{OrderID: 7}, a: asset.PerpetualContract, exp: "7"},
		{identifier: &OrderStatusRequest{ClientOrderID: "0x0000000000000000000000000000000B"}, exp: "7"},
		{identifier: &OrderStatusRequest{OrderID: 8}, pair: spotPair, exp: "8"},
		{identifier: &OrderStatusRequest{OrderID: 8}, a: asset.PerpetualContract, err: order.ErrOrderNotFound},
		{identifier: &OrderStatusRequest{OrderID: 7}, pair: spotPair, err: order.ErrOrderNotFound},
		{identifier: &OrderStatusRequest{OrderID: 7}, pair: perpetualPair.Swap(), err: order.ErrOrderNotFound},
		{identifier: &OrderStatusRequest{OrderID: 9}, err: order.ErrOrderNotFound},
	} {
		result, err := ex.findOpenOrder(t.Context(), tc.identifier, tc.pair, tc.a)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "findOpenOrder should not find %+v with pair %q and asset %q", tc.identifier, tc.pair, tc.a)
			continue
		}
		require.NoErrorf(t, err, "findOpenOrder must find %+v with pair %q and asset %q", tc.identifier, tc.pair, tc.a)
		assert.Equalf(t, tc.exp, result.OrderID, "findOpenOrder should return order %s", tc.exp)
	}
	_, err := newTradingServerExchange(t, map[string]string{"frontendOpenOrders": `{`}, nil).findOpenOrder(t.Context(), &OrderStatusRequest{OrderID: 7}, currency.EMPTYPAIR, asset.Empty)
	require.Error(t, err, "findOpenOrder must return a listing failure")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "findOpenOrder should not report an order as not found when the listings fail")
}

func TestGetActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetActiveOrders must reject a nil request")
	_, err = e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetActiveOrders must reject an unsupported asset")
	perpetualRequest := &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType}
	unconfigured := new(Exchange)
	unconfigured.SetDefaults()
	_, err = unconfigured.GetActiveOrders(t.Context(), perpetualRequest)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetActiveOrders must require an account address")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		_, err = newUnavailableServerExchange(t).GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
		require.ErrorIsf(t, err, request.ErrBadStatus, "GetActiveOrders must return a %s open orders failure", a)
	}

	mixed := newTradingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":8,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"}]`,
	}, nil)
	orders, err := mixed.GetActiveOrders(t.Context(), perpetualRequest)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetActiveOrders should report an order without a market")
	require.Len(t, orders, 1, "GetActiveOrders must return the convertible orders")
	assert.Equal(t, "8", orders[0].OrderID, "GetActiveOrders should return the convertible order")

	frontendOrders := make([]string, maximumFrontendOpenOrders)
	basicOrders := make([]string, maximumFrontendOpenOrders+2)
	for i := range maximumFrontendOpenOrders + 1 {
		basicOrder := `{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"1","oid":` + strconv.Itoa(i+1) + `,"timestamp":1791275926947`
		if i < maximumFrontendOpenOrders {
			frontendOrders[i] = basicOrder + `,"orderType":"Limit","tif":"Alo"}`
		}
		basicOrders[i] = basicOrder + `}`
	}
	basicOrders[maximumFrontendOpenOrders+1] = `{"coin":"MISSING","side":"B","oid":102,"timestamp":1791275926947}`
	capped := newTradingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`,
		"openOrders":         `[` + strings.Join(basicOrders, ",") + `]`,
	}, nil)
	orders, err = capped.GetActiveOrders(t.Context(), perpetualRequest)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetActiveOrders should report an older order without a market")
	require.Len(t, orders, maximumFrontendOpenOrders+1, "GetActiveOrders must add the orders beyond the frontendOpenOrders limit")
	placed := time.UnixMilli(1791275926947).UTC()
	exp := order.Detail{Price: 100, Amount: 1, RemainingAmount: 1, Exchange: capped.Name, OrderID: "1", Type: order.Limit, TimeInForce: order.PostOnly, Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	assert.Equal(t, exp, orders[0], "GetActiveOrders should keep the frontend fields of the orders within the limit")
	exp = order.Detail{Price: 100, Amount: 1, RemainingAmount: 1, Exchange: capped.Name, OrderID: "101", Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	assert.Equal(t, exp, orders[maximumFrontendOpenOrders], "GetActiveOrders should add an order beyond the limit without its order type or time in force")
	reduceOnly := newTradingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`,
		"openOrders":         `[{"coin":"BTC","side":"A","limitPx":"100","sz":"1","origSz":"1","oid":104,"timestamp":1791275926947,"reduceOnly":true}]`,
	}, nil)
	orders, err = reduceOnly.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must not error for a reduce-only order beyond the limit")
	require.Len(t, orders, maximumFrontendOpenOrders+1, "GetActiveOrders must add the reduce-only order beyond the frontendOpenOrders limit")
	exp = order.Detail{ReduceOnly: true, Price: 100, Amount: 1, RemainingAmount: 1, Exchange: reduceOnly.Name, OrderID: "104", Side: order.Sell, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	assert.Equal(t, exp, orders[maximumFrontendOpenOrders], "GetActiveOrders should keep the reduce-only flag openOrders sends")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		t.Run(a.String()+" fallback asset filtering", func(t *testing.T) {
			t.Parallel()
			filtered, requests := newCountingServerExchange(t, map[string]string{
				"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`,
				"openOrders":         `[` + basicOrders[maximumFrontendOpenOrders] + `,{"coin":"@107","side":"A","limitPx":"95.5","sz":"1.5","origSz":"1.5","oid":103,"timestamp":1791275926947}]`,
			})
			filtered.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
			filtered.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
			filteredOrders, filterErr := filtered.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: a, Side: order.AnySide, Type: order.AnyType})
			require.NoError(t, filterErr, "GetActiveOrders must filter fallback orders by asset")
			expected := order.FilteredOrders{{Price: 95.5, Amount: 1.5, RemainingAmount: 1.5, Exchange: filtered.Name, OrderID: "103", Side: order.Sell, Status: order.Open, AssetType: asset.Spot, Date: placed, LastUpdated: placed, Pair: spotPair}}
			expectedRequests := map[string]int{"frontendOpenOrders:": 1, "openOrders:": 1}
			if a == asset.PerpetualContract {
				expected = make(order.FilteredOrders, maximumFrontendOpenOrders+1)
				for i := range expected {
					expected[i] = order.Detail{Price: 100, Amount: 1, RemainingAmount: 1, Exchange: filtered.Name, OrderID: strconv.Itoa(i + 1), Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
					if i < maximumFrontendOpenOrders {
						expected[i].Type = order.Limit
						expected[i].TimeInForce = order.PostOnly
					}
				}
				expectedRequests["perpDexs:"] = 1
			}
			assert.Equal(t, expected, filteredOrders, "GetActiveOrders should retain only orders of the requested asset")
			assert.Equal(t, expectedRequests, requests(), "GetActiveOrders should fetch fallback orders once without extra metadata requests")
		})
	}
	unavailable := newTradingServerExchange(t, map[string]string{"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`, "openOrders": `{`}, nil)
	orders, err = unavailable.GetActiveOrders(t.Context(), perpetualRequest)
	assert.Error(t, err, "GetActiveOrders should return an openOrders failure")
	assert.Len(t, orders, maximumFrontendOpenOrders, "GetActiveOrders should still return the frontendOpenOrders listing beside an openOrders failure")
	failedDEX := newTradingServerExchange(t, map[string]string{"frontendOpenOrders": `[` + frontendOrders[0] + `]`, "frontendOpenOrders:xyz": `{`}, nil)
	orders, err = failedDEX.GetActiveOrders(t.Context(), perpetualRequest)
	assert.Error(t, err, "GetActiveOrders should report a DEX whose open orders cannot be read")
	require.Len(t, orders, 1, "GetActiveOrders must return the other DEXs' orders beside a failed listing")
	assert.Equal(t, "1", orders[0].OrderID, "GetActiveOrders should return the readable DEX's order")

	for _, tc := range []struct {
		name  string
		pairs currency.Pairs
		exp   map[string]int
	}{
		{name: "no pairs", exp: map[string]int{"perpDexs:": 1, "frontendOpenOrders:": 1, "frontendOpenOrders:xyz": 1, "frontendOpenOrders:flx": 1}},
		{name: "a first DEX pair", pairs: currency.Pairs{perpetualPair}, exp: map[string]int{"frontendOpenOrders:": 1}},
		{name: "pairs of two DEXs", pairs: currency.Pairs{testBuilderPair, perpetualPair, testBuilderPair}, exp: map[string]int{"frontendOpenOrders:": 1, "frontendOpenOrders:xyz": 1}},
		{name: "an uncached pair", pairs: currency.Pairs{perpetualPair, currency.NewPair(currency.SOL, currency.USDC)}, exp: map[string]int{"perpDexs:": 1, "frontendOpenOrders:": 1, "frontendOpenOrders:xyz": 1, "frontendOpenOrders:flx": 1}},
	} {
		ex, requests := newCountingServerExchange(t, map[string]string{"perpDexs": `[null,{"name":"xyz"},{"name":"flx"}]`, "frontendOpenOrders": `[]`})
		ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
		_, err := ex.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Pairs: tc.pairs, Side: order.AnySide, Type: order.AnyType})
		require.NoErrorf(t, err, "GetActiveOrders must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, requests(), "GetActiveOrders should request only the open orders of the DEXs it needs for %s", tc.name)
	}

	// HIP-4 outcome orders, in the shape of live outcome orders, are skipped before any market lookup
	const outcomeOrders = `{"coin":"#90700","side":"A","limitPx":"0.26401","sz":"130.0","origSz":"130.0","oid":211,"timestamp":1791275926947,"orderType":"Limit","tif":"Alo"},` +
		`{"coin":"#90411","side":"B","limitPx":"0.65336","sz":"459.0","origSz":"459.0","oid":212,"timestamp":1791275926947,"orderType":"Limit","tif":"Alo"}`
	outcomes, requests := newCountingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[` + outcomeOrders + `,{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"1","oid":213,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"}]`,
	})
	outcomes.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	orders, err = outcomes.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must skip HIP-4 outcome orders")
	require.Len(t, orders, 1, "GetActiveOrders must return only the supported order")
	assert.Equal(t, "213", orders[0].OrderID, "GetActiveOrders should return the supported order")
	assert.Equal(t, map[string]int{"perpDexs:": 1, "frontendOpenOrders:": 1}, requests(), "GetActiveOrders should not look up the markets of HIP-4 outcome orders")
	cappedOutcomes := newTradingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`,
		"openOrders":         `[` + strings.Join(basicOrders[:maximumFrontendOpenOrders], ",") + `,` + outcomeOrders + `]`,
	}, nil)
	orders, err = cappedOutcomes.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must skip HIP-4 outcome orders beyond the frontendOpenOrders limit")
	assert.Len(t, orders, maximumFrontendOpenOrders, "GetActiveOrders should add no HIP-4 outcome order beyond the frontendOpenOrders limit")

	// Position TP/SL orders are listed without a size, as live, so they take the size of the position they close, read
	// once per DEX and only when such an order is listed
	const positionOrder = `{"coin":"BTC","side":"B","limitPx":"99000.0","sz":"0.0","origSz":"0.0","oid":14,"timestamp":1791275926947,"triggerCondition":"Price above 90000","isTrigger":true,"triggerPx":"90000.0","isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","tif":null}`
	const btcShortPositions = `{"assetPositions":[{"type":"oneWay","position":{"coin":"ETH","szi":"3"}},{"type":"oneWay","position":{"coin":"BTC","szi":"-0.25"}}]}`
	sized, requests := newCountingServerExchange(t, map[string]string{
		"perpDexs":               `[null,{"name":"xyz"}]`,
		"frontendOpenOrders:":    `[` + positionOrder + `,` + strings.Replace(positionOrder, `"oid":14`, `"oid":15`, 1) + `]`,
		"frontendOpenOrders:xyz": `[{"coin":"xyz:XYZ100","side":"A","limitPx":"27000.0","sz":"0.0","origSz":"0.0","oid":16,"timestamp":1791275926947,"isTrigger":true,"triggerPx":"30000.0","isPositionTpsl":true,"reduceOnly":true,"orderType":"Take Profit Market","tif":null}]`,
		"clearinghouseState:":    btcShortPositions,
		"clearinghouseState:xyz": `{"assetPositions":[]}`,
	})
	sized.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	orders, err = sized.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must not error for position TP/SL orders")
	positionTPSL := order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 99000, Amount: 0.25, TriggerPrice: 90000, RemainingAmount: 0.25, Exchange: sized.Name, OrderID: "14", Type: order.StopMarket, Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	secondPositionTPSL := positionTPSL
	secondPositionTPSL.OrderID = "15"
	closedPositionTPSL := order.Detail{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 27000, TriggerPrice: 30000, Exchange: sized.Name, OrderID: "16", Type: order.TakeProfitMarket, Side: order.Sell, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: testBuilderPair}
	assert.Equal(t, order.FilteredOrders{positionTPSL, secondPositionTPSL, closedPositionTPSL}, orders, "GetActiveOrders should size position TP/SL orders from their positions, leaving one whose position has closed unsized")
	expRequests := map[string]int{"perpDexs:": 1, "frontendOpenOrders:": 1, "frontendOpenOrders:xyz": 1, "clearinghouseState:": 1, "clearinghouseState:xyz": 1}
	assert.Equal(t, expRequests, requests(), "GetActiveOrders should read each DEX's positions once")
	_, err = sized.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for spot beside position TP/SL orders")
	expRequests["frontendOpenOrders:"]++
	assert.Equal(t, expRequests, requests(), "GetActiveOrders should not read positions for spot orders")
	cappedPosition := newTradingServerExchange(t, map[string]string{
		"frontendOpenOrders": `[` + strings.Join(frontendOrders, ",") + `]`,
		"openOrders":         `[` + strings.Join(basicOrders[:maximumFrontendOpenOrders], ",") + `,{"coin":"BTC","side":"B","limitPx":"99000.0","sz":"0.0","origSz":"0.0","oid":117,"timestamp":1791275926947}]`,
		"clearinghouseState": btcShortPositions,
	}, nil)
	orders, err = cappedPosition.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must not error for a position TP/SL order beyond the frontendOpenOrders limit")
	require.Len(t, orders, maximumFrontendOpenOrders+1, "GetActiveOrders must add the position TP/SL order beyond the frontendOpenOrders limit")
	exp = order.Detail{Price: 99000, Amount: 0.25, RemainingAmount: 0.25, Exchange: cappedPosition.Name, OrderID: "117", Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	assert.Equal(t, exp, orders[maximumFrontendOpenOrders], "GetActiveOrders should size an openOrders order without a size from its position")
	orders, err = newTradingServerExchange(t, map[string]string{"frontendOpenOrders": `[` + positionOrder + `]`}, nil).GetActiveOrders(t.Context(), perpetualRequest)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetActiveOrders should return a positions failure")
	require.Len(t, orders, 1, "GetActiveOrders must still return a position TP/SL order whose position cannot be read")
	assert.Zero(t, orders[0].Amount, "GetActiveOrders should leave a position TP/SL order unsized when its position cannot be read")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	orders, err = e.GetActiveOrders(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetActiveOrders must not error for perpetuals")
	spotOrders, err := e.GetActiveOrders(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetActiveOrders must not error for spot")
	if !mockTests {
		return
	}
	expOrders := order.FilteredOrders{
		{TimeInForce: order.GoodTillCancel, Price: 85000, Amount: 0.002, ExecutedAmount: 0.001, RemainingAmount: 0.001, Exchange: e.Name, OrderID: strconv.FormatUint(testOrderID, 10), ClientOrderID: testClientOrderID, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair},
		{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 99000, Amount: 0.01, TriggerPrice: 90000, RemainingAmount: 0.01, Exchange: e.Name, OrderID: "566563303543", Type: order.StopMarket, Side: order.Buy, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair},
		{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 27000, Amount: 0.05, TriggerPrice: 30000, RemainingAmount: 0.05, Exchange: e.Name, OrderID: "566563303539", Type: order.StopMarket, Side: order.Sell, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: testBuilderPair},
	}
	assert.Equal(t, expOrders, orders, "GetActiveOrders should convert every DEX's perpetual orders, sizing the position TP/SL from its position and skipping the outcome order")
	expOrders = order.FilteredOrders{
		{TimeInForce: order.PostOnly, Price: 95.5, Amount: 1.5, RemainingAmount: 1.5, Exchange: e.Name, OrderID: "566563303538", Type: order.Limit, Side: order.Sell, Status: order.Open, AssetType: asset.Spot, Date: placed, LastUpdated: placed, Pair: spotPair},
	}
	assert.Equal(t, expOrders, spotOrders, "GetActiveOrders should convert the spot orders")
}

func TestGetOrderInfoExecution(t *testing.T) {
	t.Parallel()
	// orderStatus reports no execution price, cost or fee, so a finished order that executed something takes them from the
	// account's fills between its placement and its final status
	const placedAt, finishedAt = 1791275926000, 1791275926500
	status := func(state, remaining string) string {
		return `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"89924.0","sz":"` + remaining + `","origSz":"0.5","oid":21,"timestamp":1791275926000,"orderType":"Market","tif":"FrontendMarket"},"status":"` + state + `","statusTimestamp":1791275926500}}`
	}
	fill := func(orderID, tradeID uint64, price, size, fee, feeToken string, at int64) string {
		return `{"coin":"BTC","px":"` + price + `","sz":"` + size + `","side":"B","time":` + strconv.FormatInt(at, 10) + `,"oid":` + strconv.FormatUint(orderID, 10) + `,"crossed":true,"fee":"` + fee + `","tid":` + strconv.FormatUint(tradeID, 10) + `,"feeToken":"` + feeToken + `"}`
	}
	first, second := fill(21, 1, "80000", "0.25", "8", "USDC", placedAt+100), fill(21, 2, "84000", "0.25", "8.5", "USDC", placedAt+400)
	other := fill(99, 3, "1", "1", "1", "USDC", placedAt+200)
	// The markets of the orders captured live in testdata/order_execution.json
	ethPair, fartcoinPair, hypePair := currency.NewPair(currency.ETH, currency.USDC), currency.NewPair(currency.NewCode("FARTCOIN"), currency.USDC), currency.NewPair(currency.HYPE, currency.USDC)
	capturedMappings := []pairMapping{
		{pair: ethPair, coin: "ETH", assetID: 1, sizeDecimals: 4, maxLeverage: 25},
		{pair: fartcoinPair, coin: "FARTCOIN", assetID: 165, sizeDecimals: 1, maxLeverage: 10},
		{pair: hypePair, coin: "HYPE", assetID: 159, sizeDecimals: 2, maxLeverage: 10},
	}
	newExecutionServer := func(t *testing.T, orderStatus string, pages map[uint64]string) (*Exchange, func() []InfoRequest) {
		t.Helper()
		var mu sync.Mutex
		var fillRequests []InfoRequest
		ex := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var body InfoRequest
			if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
				return
			}
			response := orderStatus
			if body.Type == "userFillsByTime" {
				mu.Lock()
				fillRequests = append(fillRequests, body)
				requests := len(fillRequests)
				mu.Unlock()
				// Capped, so a pager that stops advancing fails rather than requesting the same page until go test's timeout
				if requests > 5 {
					assert.Fail(t, "GetOrderInfo should not request more than five pages of fills")
					http.Error(w, "too many fills requests", http.StatusBadRequest)
					return
				}
				var ok bool
				if response, ok = pages[body.StartTime]; !ok {
					http.Error(w, "unexpected fills request", http.StatusBadRequest)
					return
				}
			} else if body.Type != "orderStatus" {
				http.Error(w, "unexpected info request "+body.Type, http.StatusBadRequest)
				return
			}
			_, err := w.Write([]byte(response))
			assert.NoError(t, err, "Writing the response should not error")
		}))
		ex.Name = t.Name()
		setTestCredentials(ex, &accounts.Credentials{Key: testAccountAddress, Secret: testPrivateKey})
		ex.setPairMappings(asset.PerpetualContract, append([]pairMapping{testPerpetualMapping, testBuilderMapping}, capturedMappings...))
		ex.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
		return ex, func() []InfoRequest {
			mu.Lock()
			defer mu.Unlock()
			return slices.Clone(fillRequests)
		}
	}

	ex, fillRequests := newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + first + `,` + other + `,` + first + `,` + second + `]`})
	result, err := ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a filled order")
	assert.Equal(t, 41000.0, result.Cost, "GetOrderInfo should add up the cost of the order's own fills, once each")
	assert.Equal(t, 82000.0, result.AverageExecutedPrice, "GetOrderInfo should report the order's average execution price")
	assert.Equal(t, 16.5, result.Fee, "GetOrderInfo should add up the fees of the order's fills")
	assert.Equal(t, currency.USDC, result.FeeAsset, "GetOrderInfo should report the fee token of the order's fills")
	assert.Equal(t, []InfoRequest{{Type: "userFillsByTime", User: testAccountAddress, StartTime: placedAt, EndTime: finishedAt}}, fillRequests(), "GetOrderInfo should read the fills between the order's placement and its final status")

	ex, _ = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + first + `,` + strings.Replace(second, `"feeToken":"USDC"`, `"feeToken":"HYPE"`, 1) + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for fills paying fees in different tokens")
	assert.Equal(t, 82000.0, result.AverageExecutedPrice, "GetOrderInfo should report the average price of fills paying fees in different tokens")
	assert.Zero(t, result.Fee, "GetOrderInfo should leave the fee unset when the fills paid it in different tokens")
	assert.True(t, result.FeeAsset.IsEmpty(), "GetOrderInfo should leave the fee token unset when the fills paid fees in different tokens")

	// A page short of the maximum holds every fill left in the range, so an empty page after it must not be read
	ex, _ = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + first + `]`, placedAt + 100: `[]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error when the fills fall short of the executed size")
	assert.Zero(t, result.Cost, "GetOrderInfo should leave the cost unset when the fills fall short of the executed size")
	assert.Zero(t, result.AverageExecutedPrice, "GetOrderInfo should leave the average price unset when the fills fall short of the executed size")
	assert.Zero(t, result.Fee, "GetOrderInfo should leave the fee unset when the fills fall short of the executed size")

	// Sizes are summed in floating point, where 0.10001 and 0.20003 do not add up to exactly 0.30004, and a resting
	// order's maker fill, at its limit price, earns a rebate
	limitStatus := func(size, limitPrice string) string {
		return strings.NewReplacer(`"origSz":"0.5"`, `"origSz":"`+size+`"`, `"limitPx":"89924.0"`, `"limitPx":"`+limitPrice+`"`, `"orderType":"Market","tif":"FrontendMarket"`, `"orderType":"Limit","tif":"Gtc"`).Replace(status("filled", "0.0"))
	}
	maker := func(tradeID uint64, price, size string) string {
		return strings.Replace(fill(21, tradeID, price, size, "-0.05", "USDC", placedAt+400), `"crossed":true`, `"crossed":false`, 1)
	}
	ex, _ = newExecutionServer(t, limitStatus("0.30004", "84001.0"), map[uint64]string{placedAt: `[` + fill(21, 4, "80001", "0.10001", "8.000123", "USDC", placedAt+100) + `,` + maker(5, "84001", "0.20003") + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for fills whose sizes do not sum exactly")
	assert.InDelta(t, 24803.62004, result.Cost, 1e-6, "GetOrderInfo should add up the cost of fills whose sizes sum to the executed size within half a size step")
	assert.InDelta(t, 24803.62004/0.30004, result.AverageExecutedPrice, 1e-6, "GetOrderInfo should divide the fills' cost by their size")
	assert.InDelta(t, 7.950123, result.Fee, 1e-9, "GetOrderInfo should net a maker rebate against the fees")
	ex, _ = newExecutionServer(t, limitStatus("0.25", "80000.0"), map[uint64]string{placedAt: `[` + maker(6, "80000", "0.25") + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for an order filled as a maker")
	assert.InDelta(t, -0.05, result.Fee, 1e-9, "GetOrderInfo should report fees that net to a rebate")

	// This market order is not reduce-only, so fills a size step short of its executed size are not all of its fills
	ex, _ = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + first + `,` + strings.Replace(second, `"sz":"0.25"`, `"sz":"0.24999"`, 1) + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for fills a size step short of the executed size")
	assert.Zero(t, result.Cost, "GetOrderInfo should leave the cost unset when the fills are a size step short of the executed size")

	ex, _ = newExecutionServer(t, status("filled", "0.0"), nil)
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must return the order's status though its fills cannot be read")
	assert.Equal(t, order.Filled, result.Status, "GetOrderInfo should return the order's status though its fills cannot be read")
	assert.Zero(t, result.AverageExecutedPrice, "GetOrderInfo should leave the average price unset when the fills cannot be read")

	// A full page continues from its last fill's time, and the fills repeated there are dropped by trade ID
	page := make([]string, 0, maximumUserFillsCount)
	for i := range maximumUserFillsCount - 1 {
		page = append(page, fill(99, uint64(1000+i), "1", "1", "1", "USDC", placedAt+50))
	}
	page = append(page, first)
	stuckAtEnd := make([]string, 0, maximumUserFillsCount)
	for i := range maximumUserFillsCount - 2 {
		stuckAtEnd = append(stuckAtEnd, fill(99, uint64(9000+i), "1", "1", "1", "USDC", placedAt))
	}
	stuckAtEnd = append(stuckAtEnd, strings.Replace(first, `"time":`+strconv.FormatInt(placedAt+100, 10), `"time":`+strconv.FormatInt(placedAt, 10), 1), strings.Replace(second, `"time":`+strconv.FormatInt(placedAt+400, 10), `"time":`+strconv.FormatInt(placedAt, 10), 1))
	ex, fillRequests = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + strings.Join(page, ",") + `]`, placedAt + 100: `[` + first + `,` + second + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for fills over two pages")
	assert.Equal(t, 41000.0, result.Cost, "GetOrderInfo should add up the order's fills across pages, once each")
	assert.Len(t, fillRequests(), 2, "GetOrderInfo should read the next page from the last fill of a full one")

	// A full page holding all of the order's fills is the last read
	complete := append(append(make([]string, 0, maximumUserFillsCount), page[:maximumUserFillsCount-2]...), first, second)
	ex, fillRequests = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + strings.Join(complete, ",") + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a full page holding all of the order's fills")
	assert.Equal(t, 82000.0, result.AverageExecutedPrice, "GetOrderInfo should use a full page holding all of the order's fills")
	assert.Len(t, fillRequests(), 1, "GetOrderInfo should stop reading once the order's fills sum to its executed size")

	// A full page of fills sharing the cursor's time cannot be read further, so the next page starts a millisecond on
	stuck := make([]string, 0, maximumUserFillsCount)
	for i := range maximumUserFillsCount {
		stuck = append(stuck, fill(99, uint64(5000+i), "1", "1", "1", "USDC", placedAt))
	}
	ex, fillRequests = newExecutionServer(t, status("filled", "0.0"), map[uint64]string{placedAt: `[` + strings.Join(stuck, ",") + `]`, placedAt + 1: `[` + first + `,` + second + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a full page of fills sharing one time")
	assert.Equal(t, 82000.0, result.AverageExecutedPrice, "GetOrderInfo should read the fills after a full page sharing one time")
	assert.Len(t, fillRequests(), 2, "GetOrderInfo should read on from a millisecond after a full page sharing the cursor's time")
	// The range ends at the order's final status, so for an order placed and filled in one millisecond, a full page of
	// that millisecond is the last, whether or not it holds all of the order's fills
	sameTime := strings.Replace(status("filled", "0.0"), `"statusTimestamp":1791275926500`, `"statusTimestamp":1791275926000`, 1)
	ex, fillRequests = newExecutionServer(t, sameTime, map[uint64]string{placedAt: `[` + strings.Join(stuckAtEnd, ",") + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a full page of the order's final millisecond")
	assert.Equal(t, 82000.0, result.AverageExecutedPrice, "GetOrderInfo should use the fills of the order's final millisecond")
	assert.Len(t, fillRequests(), 1, "GetOrderInfo should not read past the order's final status")
	shortAtEnd := append(slices.Clone(stuckAtEnd[:maximumUserFillsCount-1]), fill(99, 8999, "1", "1", "1", "USDC", placedAt))
	ex, fillRequests = newExecutionServer(t, sameTime, map[uint64]string{placedAt: `[` + strings.Join(shortAtEnd, ",") + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a full page of the order's final millisecond short of its fills")
	assert.Zero(t, result.AverageExecutedPrice, "GetOrderInfo should leave the average price unset when the order's final millisecond holds too few of its fills")
	assert.Len(t, fillRequests(), 1, "GetOrderInfo should not read past the order's final status when its fills fall short")

	// An order that executed nothing has no average price, which an empty page of fills must not make 0/0
	ex, _ = newExecutionServer(t, status("canceled", "0.5"), map[uint64]string{placedAt: `[]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a cancelled order that executed nothing")
	assert.Equal(t, order.Cancelled, result.Status, "GetOrderInfo should report an unexecuted order cancelled")
	assert.Zero(t, result.AverageExecutedPrice, "GetOrderInfo should leave an unexecuted order's average price unset")

	ex, _ = newExecutionServer(t, status("canceled", "0.25"), map[uint64]string{placedAt: `[` + first + `]`})
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not error for a partly filled cancelled order")
	assert.Equal(t, order.PartiallyFilledCancelled, result.Status, "GetOrderInfo should report the order partly filled and cancelled")
	assert.Equal(t, 80000.0, result.AverageExecutedPrice, "GetOrderInfo should report the average price of a partly filled cancelled order")

	// Hyperliquid cuts a reduce-only order down to the position it closes without a fill, as it cut order 577148510333
	// from 30000 to the 20000 it filled, so such an order's fills can fall short of its original size less its remaining
	// size and still be all of them; one that executed something then takes its executed and remaining sizes as its size
	reduceOnly := func(state, remaining string) string {
		return strings.Replace(status(state, remaining), `"orderType":"Market","tif":"FrontendMarket"`, `"reduceOnly":true,"orderType":"Limit","tif":"Gtc"`, 1)
	}
	read := func(start, end uint64) InfoRequest {
		return InfoRequest{Type: "userFillsByTime", User: testAccountAddress, StartTime: start, EndTime: end}
	}
	earlier, kept := read(1, placedAt-1), `[`+fill(99, 7, "80100", "0.01", "0.36045", "USDC", placedAt-1000)+`]`
	third := fill(21, 8, "86000", "0.25", "9", "USDC", placedAt+450)
	reduceOnlySameTime := strings.Replace(reduceOnly("filled", "0.0"), `"statusTimestamp":1791275926500`, `"statusTimestamp":1791275926000`, 1)
	firstAtPlacement := strings.Replace(first, `"time":`+strconv.FormatInt(placedAt+100, 10), `"time":`+strconv.FormatInt(placedAt, 10), 1)
	for _, tc := range []struct {
		name, orderStatus                                    string
		pages                                                map[uint64]string
		amount, executed, remaining, cost, averagePrice, fee float64
		feeAsset                                             currency.Code
		status                                               order.Status
		requests                                             []InfoRequest
	}{
		{name: "filled", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `,` + other + `]`, 1: kept}, amount: 0.25, executed: 0.25, cost: 20000, averagePrice: 80000, fee: 8, feeAsset: currency.USDC, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "filled to a size step", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + fill(21, 9, "80000", "0.12345", "4.444", "USDC", placedAt+100) + `]`, 1: kept}, amount: 0.12345, executed: 0.12345, cost: 9876, averagePrice: 80000, fee: 4.444, feeAsset: currency.USDC, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "filled in its millisecond", orderStatus: reduceOnlySameTime, pages: map[uint64]string{placedAt: `[` + firstAtPlacement + `]`, 1: kept}, amount: 0.25, executed: 0.25, cost: 20000, averagePrice: 80000, fee: 8, feeAsset: currency.USDC, status: order.Filled, requests: []InfoRequest{read(placedAt, placedAt), earlier}},
		// The order store keeps the executed size it holds when a lookup reports none, so such an order keeps its size
		{name: "cancelled unfilled", orderStatus: reduceOnly("reduceOnlyCanceled", "0.3"), pages: map[uint64]string{placedAt: `[` + other + `]`, 1: kept}, amount: 0.5, remaining: 0.3, status: order.Cancelled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "cancelled partly filled", orderStatus: reduceOnly("canceled", "0.1"), pages: map[uint64]string{placedAt: `[` + first + `]`, 1: kept}, amount: 0.35, executed: 0.25, remaining: 0.1, cost: 20000, averagePrice: 80000, fee: 8, feeAsset: currency.USDC, status: order.PartiallyFilledCancelled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "fills over two pages", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + strings.Join(page, ",") + `]`, placedAt + 100: `[` + first + `]`, 1: kept}, amount: 0.25, executed: 0.25, cost: 20000, averagePrice: 80000, fee: 8, feeAsset: currency.USDC, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), read(placedAt+100, finishedAt), earlier}},
		// Hyperliquid drops an account's oldest fills first, so without an earlier fill the order's may have been dropped
		{name: "earlier fills dropped", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `]`, 1: `[]`}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "earlier fills unread", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `]`}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), earlier}},
		{name: "millisecond passed over", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + strings.Join(stuck, ",") + `]`, placedAt + 1: `[` + first + `]`, 1: kept}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt), read(placedAt+1, finishedAt)}},
		{name: "final millisecond unread", orderStatus: reduceOnlySameTime, pages: map[uint64]string{placedAt: `[` + strings.Join(shortAtEnd, ",") + `]`, 1: kept}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, placedAt)}},
		// Fills beyond the executed size are not the fills of an order cut down without one
		{name: "fills exceed executed size", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `,` + second + `,` + third + `]`, 1: kept}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt)}},
		{name: "fills match executed size", orderStatus: reduceOnly("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `,` + second + `]`, 1: kept}, amount: 0.5, executed: 0.5, cost: 41000, averagePrice: 82000, fee: 16.5, feeAsset: currency.USDC, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt)}},
		// Hyperliquid cuts only reduce-only orders down without a fill
		{name: "not reduce-only", orderStatus: status("filled", "0.0"), pages: map[uint64]string{placedAt: `[` + first + `]`, 1: kept}, amount: 0.5, executed: 0.5, status: order.Filled, requests: []InfoRequest{read(placedAt, finishedAt)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server, sent := newExecutionServer(t, tc.orderStatus, tc.pages)
			got, err := server.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
			require.NoError(t, err, "GetOrderInfo must not error for a reduce-only order")
			assert.Equal(t, tc.amount, got.Amount, "GetOrderInfo should report the order's size")
			assert.Equal(t, tc.executed, got.ExecutedAmount, "GetOrderInfo should report the order's executed size")
			assert.Equal(t, tc.remaining, got.RemainingAmount, "GetOrderInfo should keep the order's remaining size")
			assert.Equal(t, tc.cost, got.Cost, "GetOrderInfo should report the cost of the order's fills")
			assert.Equal(t, tc.averagePrice, got.AverageExecutedPrice, "GetOrderInfo should report the average price of the order's fills")
			assert.Equal(t, tc.fee, got.Fee, "GetOrderInfo should report the fees of the order's fills")
			assert.Equal(t, tc.feeAsset, got.FeeAsset, "GetOrderInfo should report the fee token of the order's fills")
			assert.Equal(t, tc.status, got.Status, "GetOrderInfo should report the order's status")
			assert.Equal(t, tc.requests, sent(), "GetOrderInfo should check that earlier fills are kept, after reading the order's, only for a cut reduce-only order")

			// The engine merges the lookup into the order it holds, which can carry the size cut as executed, as a lookup
			// that could not show its fills complete reports it
			pages := maps.Clone(tc.pages)
			delete(pages, 1)
			unproven, _ := newExecutionServer(t, tc.orderStatus, pages)
			held, err := unproven.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
			require.NoError(t, err, "GetOrderInfo must not error for a reduce-only order whose earlier fills cannot be read")
			merged := mergeOrderUpdates(t, *held, *got)
			assert.InDelta(t, merged.RemainingAmount, merged.Amount-merged.ExecutedAmount, 1e-9, "The merged order's size should be its executed and remaining sizes")
		})
	}

	// Live replies: position TP/SL order 585000471462 triggered and filled within one millisecond, and Hyperliquid cut
	// reduce-only orders 577148510333 and 574372010549 down to their positions; each witness reply keeps the first of its
	// 2000 fills
	raw, err := os.ReadFile("testdata/order_execution.json")
	require.NoError(t, err, "Reading the order execution captures must not error")
	var captures map[string]struct {
		OrderStatus, Fills, Witness json.RawMessage
	}
	require.NoError(t, json.Unmarshal(raw, &captures), "Decoding the order execution captures must not error")
	for _, tc := range []struct {
		orderID                                              string
		pair                                                 currency.Pair
		placedAt, finishedAt                                 uint64
		amount, executed, remaining, cost, averagePrice, fee float64
		feeAsset                                             currency.Code
		status                                               order.Status
		witness                                              bool
	}{
		{orderID: "585000471462", pair: ethPair, placedAt: 1792906755936, finishedAt: 1792906755936, amount: 19.7784, executed: 19.7784, cost: 46548.00531, averagePrice: 2353.476788314525, fee: 20.946597, feeAsset: currency.USDC, status: order.Filled},
		{orderID: "577148510333", pair: fartcoinPair, placedAt: 1792192466897, finishedAt: 1792192552969, amount: 20000, executed: 20000, cost: 3086.4, averagePrice: 0.15432, fee: 1.004687, feeAsset: currency.USDC, status: order.Filled, witness: true},
		{orderID: "574372010549", pair: hypePair, placedAt: 1791994300968, finishedAt: 1791994306173, amount: 505.62, remaining: 319.27, status: order.Cancelled, witness: true},
	} {
		t.Run(tc.orderID, func(t *testing.T) {
			t.Parallel()
			capture, ok := captures[tc.orderID]
			require.True(t, ok, "The captures must hold the order")
			pages := map[uint64]string{tc.placedAt: string(capture.Fills)}
			requests := []InfoRequest{read(tc.placedAt, tc.finishedAt)}
			if tc.witness {
				pages[1] = string(capture.Witness)
				requests = append(requests, read(1, tc.placedAt-1))
			}
			server, sent := newExecutionServer(t, string(capture.OrderStatus), pages)
			got, err := server.GetOrderInfo(t.Context(), tc.orderID, tc.pair, asset.PerpetualContract)
			require.NoError(t, err, "GetOrderInfo must not error for a captured order")
			assert.Equal(t, tc.amount, got.Amount, "GetOrderInfo should report the order's size")
			assert.Equal(t, tc.executed, got.ExecutedAmount, "GetOrderInfo should report the order's executed size")
			assert.Equal(t, tc.remaining, got.RemainingAmount, "GetOrderInfo should report the order's remaining size")
			assert.InDelta(t, tc.cost, got.Cost, 1e-6, "GetOrderInfo should report the cost of the order's fills")
			assert.InDelta(t, tc.averagePrice, got.AverageExecutedPrice, 1e-9, "GetOrderInfo should report the average price of the order's fills")
			assert.InDelta(t, tc.fee, got.Fee, 1e-9, "GetOrderInfo should report the fees of the order's fills")
			assert.Equal(t, tc.feeAsset, got.FeeAsset, "GetOrderInfo should report the fee token of the order's fills")
			assert.Equal(t, tc.status, got.Status, "GetOrderInfo should report the order's status")
			assert.Equal(t, requests, sent(), "GetOrderInfo should read the order's fills, and whether earlier fills are kept only when they fall short")
		})
	}

	// An open order is still trading, so its fills are not read
	ex, fillRequests = newExecutionServer(t, status("open", "0.25"), nil)
	result, err = ex.GetOrderInfo(t.Context(), "21", perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "GetOrderInfo must not read fills for an open order")
	assert.Zero(t, result.AverageExecutedPrice, "GetOrderInfo should leave an open order's average price unset")
	assert.Empty(t, fillRequests(), "GetOrderInfo should not read fills for an open order")
}

func TestGetOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, order.ErrGetOrdersRequestIsNil, "GetOrderHistory must reject a nil request")
	_, err = e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options, Side: order.AnySide, Type: order.AnyType})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOrderHistory must reject an unsupported asset")
	perpetualRequest := &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.AnyType}
	unconfigured := new(Exchange)
	unconfigured.SetDefaults()
	_, err = unconfigured.GetOrderHistory(t.Context(), perpetualRequest)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetOrderHistory must require an account address")
	_, err = newUnavailableServerExchange(t).GetOrderHistory(t.Context(), perpetualRequest)
	require.ErrorIs(t, err, request.ErrBadStatus, "GetOrderHistory must return a history failure")

	mixed := newTradingServerExchange(t, map[string]string{
		"historicalOrders": `[{"order":{"coin":"MISSING","side":"B","limitPx":"1","sz":"1","origSz":"1","oid":7,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1791275930650},{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":8,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1791275930650}]`,
	}, nil)
	orders, err := mixed.GetOrderHistory(t.Context(), perpetualRequest)
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetOrderHistory should report an order without a market")
	require.Len(t, orders, 1, "GetOrderHistory must return the convertible orders")
	assert.Equal(t, order.Filled, orders[0].Status, "GetOrderHistory should convert the order's status")

	// Hyperliquid lists an order once per status change, newest first, as order 566563324876 is in the live captures
	repeated := newTradingServerExchange(t, map[string]string{
		"historicalOrders": `[{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"1","oid":9,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},"status":"canceled","statusTimestamp":1791275930650},` +
			`{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"1","oid":9,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1791275930036}]`,
	}, nil)
	orders, err = repeated.GetOrderHistory(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetOrderHistory must not error for an order listed once per status change")
	placed := time.UnixMilli(1791275926947).UTC()
	exp := order.FilteredOrders{{TimeInForce: order.GoodTillCancel, Price: 100, Amount: 1, RemainingAmount: 1, Exchange: repeated.Name, OrderID: "9", Type: order.Limit, Side: order.Buy, Status: order.Cancelled, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: time.UnixMilli(1791275930650).UTC(), Pair: perpetualPair}}
	assert.Equal(t, exp, orders, "GetOrderHistory should return an order once, with its latest status")

	// A trigger order's changes as live for stop market order 570654892796 of 0xd80adf681abe30a3d644db1daae9abaa2d25a895,
	// on BTC: once it triggered and filled, it was listed without its trigger flag and price and with the trigger time
	triggered := newTradingServerExchange(t, map[string]string{
		"historicalOrders": `[{"order":{"coin":"BTC","side":"A","limitPx":"4.7932","sz":"0.0","oid":570654892796,"timestamp":1791654299637,"triggerCondition":"Triggered","isTrigger":false,"triggerPx":"0.0","children":[],"isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","origSz":"4026.5","tif":"Gtc","cloid":null},"status":"filled","statusTimestamp":1791654299637},` +
			`{"order":{"coin":"BTC","side":"A","limitPx":"4.7931","sz":"0.0","oid":570654892796,"timestamp":1791651887088,"triggerCondition":"Price below 5.3257","isTrigger":true,"triggerPx":"5.3257","children":[],"isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","origSz":"0.0","tif":null,"cloid":null},"status":"triggered","statusTimestamp":1791654299637},` +
			`{"order":{"coin":"BTC","side":"A","limitPx":"4.7931","sz":"0.0","oid":570654892796,"timestamp":1791651887088,"triggerCondition":"Price below 5.3257","isTrigger":true,"triggerPx":"5.3257","children":[],"isPositionTpsl":true,"reduceOnly":true,"orderType":"Stop Market","origSz":"0.0","tif":null,"cloid":null},"status":"open","statusTimestamp":1791651887088}]`,
	}, nil)
	orders, err = triggered.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Side: order.AnySide, Type: order.StopMarket})
	require.NoError(t, err, "GetOrderHistory must not error for a triggered stop market order")
	exp = order.FilteredOrders{{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 4.7932, Amount: 4026.5, ExecutedAmount: 4026.5, TriggerPrice: 5.3257, Exchange: triggered.Name, OrderID: "570654892796", Type: order.StopMarket, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualContract, Date: time.UnixMilli(1791651887088).UTC(), LastUpdated: time.UnixMilli(1791654299637).UTC(), Pair: perpetualPair}}
	assert.Equal(t, exp, orders, "GetOrderHistory should return a triggered order by its type, with its trigger price and placement time")

	delisted := newInfoServerExchange(t, map[string]string{
		"meta":             `{"universe":[{"name":"BTC","szDecimals":5},{"name":"MATIC","szDecimals":1,"maxLeverage":20,"isDelisted":true}]}`,
		"spotMeta":         `{"universe":[],"tokens":[{"name":"USDC","index":0}]}`,
		"historicalOrders": `[{"order":{"coin":"MATIC","side":"A","limitPx":"0.38","sz":"0.0","origSz":"25.0","oid":10,"timestamp":1791275910000,"orderType":"Limit","tif":"Ioc"},"status":"filled","statusTimestamp":1791275910000}]`,
	}, nil)
	orders, err = delisted.GetOrderHistory(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetOrderHistory must not error for an order of a delisted market")
	maticFilled := time.UnixMilli(1791275910000).UTC()
	exp = order.FilteredOrders{{TimeInForce: order.ImmediateOrCancel, Price: 0.38, Amount: 25, ExecutedAmount: 25, Exchange: delisted.Name, OrderID: "10", Type: order.Limit, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualContract, Date: maticFilled, LastUpdated: maticFilled, Pair: testDelistedMapping.pair}}
	assert.Equal(t, exp, orders, "GetOrderHistory should convert an order of a delisted market")

	outcomes, requests := newCountingServerExchange(t, map[string]string{
		"historicalOrders": `[{"order":{"coin":"#90700","side":"A","limitPx":"0.26401","sz":"0.0","origSz":"130.0","oid":11,"timestamp":1791275926947,"orderType":"Limit","tif":"Alo"},"status":"filled","statusTimestamp":1791275930650},` +
			`{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":8,"timestamp":1791275926947,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1791275930650}]`,
	})
	outcomes.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	orders, err = outcomes.GetOrderHistory(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetOrderHistory must skip HIP-4 outcome orders")
	require.Len(t, orders, 1, "GetOrderHistory must return only the supported order")
	assert.Equal(t, "8", orders[0].OrderID, "GetOrderHistory should return the supported order")
	assert.Equal(t, map[string]int{"historicalOrders:": 1}, requests(), "GetOrderHistory should not look up the markets of HIP-4 outcome orders")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	orders, err = e.GetOrderHistory(t.Context(), perpetualRequest)
	require.NoError(t, err, "GetOrderHistory must not error for perpetuals")
	spotOrders, err := e.GetOrderHistory(t.Context(), &order.MultiOrderRequest{AssetType: asset.Spot, Side: order.AnySide, Type: order.AnyType})
	require.NoError(t, err, "GetOrderHistory must not error for spot")
	if !mockTests {
		return
	}
	exp = order.FilteredOrders{
		{TimeInForce: order.ImmediateOrCancel, Price: 86500, Amount: 0.01, ExecutedAmount: 0.01, Exchange: e.Name, OrderID: "566563303540", Type: order.Limit, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: time.UnixMilli(1791275930650).UTC(), Pair: perpetualPair},
		{TimeInForce: order.GoodTillCancel, Price: 85000, Amount: 0.002, ExecutedAmount: 0.001, RemainingAmount: 0.001, Exchange: e.Name, OrderID: strconv.FormatUint(testOrderID, 10), ClientOrderID: testClientOrderID, Type: order.Limit, Side: order.Buy, Status: order.PartiallyFilled, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair},
		{TimeInForce: order.GoodTillCancel, ReduceOnly: true, Price: 76500, Amount: 0.01, ExecutedAmount: 0.01, TriggerPrice: 85000, Exchange: e.Name, OrderID: "566563303546", Type: order.StopMarket, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualContract, Date: time.UnixMilli(1791275900000).UTC(), LastUpdated: time.UnixMilli(1791275920000).UTC(), Pair: perpetualPair},
		{TimeInForce: order.ImmediateOrCancel, Price: 0.38, Amount: 25, ExecutedAmount: 25, Exchange: e.Name, OrderID: "566563303545", Type: order.Limit, Side: order.Sell, Status: order.Filled, AssetType: asset.PerpetualContract, Date: time.UnixMilli(1791275910000).UTC(), LastUpdated: time.UnixMilli(1791275910000).UTC(), Pair: testDelistedMapping.pair},
	}
	assert.Equal(t, exp, orders, "GetOrderHistory should convert the perpetual orders, the triggered stop order with its trigger price and placement time and the delisted market's order, skipping the outcome order")
	exp = order.FilteredOrders{
		{TimeInForce: order.GoodTillCancel, Price: 90, Amount: 2, RemainingAmount: 2, Exchange: e.Name, OrderID: "566563303541", Type: order.Limit, Side: order.Buy, Status: order.Cancelled, AssetType: asset.Spot, Date: placed, LastUpdated: time.UnixMilli(1791275931000).UTC(), Pair: spotPair},
	}
	assert.Equal(t, exp, spotOrders, "GetOrderHistory should convert the spot order once, from its cancellation")
}

func TestLatestOrderStatuses(t *testing.T) {
	t.Parallel()
	assert.Empty(t, latestOrderStatuses(nil), "latestOrderStatuses should return nothing for no status changes")
	statusChange := func(orderID uint64, status string, statusTimestamp int64) HistoricalOrderResponse {
		return HistoricalOrderResponse{Order: FrontendOpenOrderResponse{OrderID: orderID, Timestamp: milli(1791275926947)}, Status: status, StatusTimestamp: milli(statusTimestamp)}
	}
	statusChanges := []HistoricalOrderResponse{
		statusChange(1, "canceled", 1791275930650),
		statusChange(2, "open", 1791275930000),
		statusChange(1, "open", 1791275930036),
		statusChange(3, "open", 1791275931000),
		statusChange(3, "filled", 1791275931000),
		statusChange(2, "filled", 1791275932000),
	}
	exp := []HistoricalOrderResponse{
		statusChange(1, "canceled", 1791275930650),
		statusChange(2, "filled", 1791275932000),
		statusChange(3, "filled", 1791275931000),
	}
	assert.Equal(t, exp, latestOrderStatuses(statusChanges), "latestOrderStatuses should keep each order's latest status in first-listed order")

	for _, statuses := range [][]string{
		{"triggered", "filled", "open"},
		{"triggered", "open", "filled"},
		{"filled", "triggered", "open"},
		{"filled", "open", "triggered"},
		{"open", "triggered", "filled"},
		{"open", "filled", "triggered"},
	} {
		changes := make([]HistoricalOrderResponse, len(statuses))
		for i, status := range statuses {
			changes[i] = statusChange(4, status, 1791275933000)
		}
		assert.Equalf(t, []HistoricalOrderResponse{statusChange(4, "filled", 1791275933000)}, latestOrderStatuses(changes), "latestOrderStatuses should keep the final change of %v at one time", statuses)
	}
	for _, statuses := range [][]string{{"triggered", "open"}, {"open", "triggered"}} {
		changes := []HistoricalOrderResponse{statusChange(5, statuses[0], 1791275934000), statusChange(5, statuses[1], 1791275934000)}
		assert.Equalf(t, []HistoricalOrderResponse{statusChange(5, "triggered", 1791275934000)}, latestOrderStatuses(changes), "latestOrderStatuses should keep the triggered change of %v at one time", statuses)
	}

	// Once triggered, an order's changes give the trigger time as its timestamp and no trigger price, as live
	placed, triggeredAt := milli(1791651887088), milli(1791654299637)
	triggerChange := func(status string, statusTimestamp, timestamp types.Time, triggerPrice types.Number) HistoricalOrderResponse {
		return HistoricalOrderResponse{Order: FrontendOpenOrderResponse{OrderID: 6, Timestamp: timestamp, TriggerPrice: triggerPrice}, Status: status, StatusTimestamp: statusTimestamp}
	}
	filled := triggerChange("filled", triggeredAt, triggeredAt, 0)
	triggered := triggerChange("triggered", triggeredAt, placed, 5.3257)
	opened := triggerChange("open", placed, placed, 5.3257)
	for _, changes := range [][]HistoricalOrderResponse{{filled, triggered, opened}, {opened, triggered, filled}, {triggered, filled}} {
		assert.Equal(t, []HistoricalOrderResponse{triggerChange("filled", triggeredAt, placed, 5.3257)}, latestOrderStatuses(changes), "latestOrderStatuses should keep the final change with the trigger price and placement time of the earlier ones")
	}
}

func TestOrderStatusRank(t *testing.T) {
	t.Parallel()
	assert.Zero(t, orderStatusRank("open"), "orderStatusRank should rank open first")
	assert.Equal(t, 1, orderStatusRank("triggered"), "orderStatusRank should rank triggered after open")
	for _, status := range []string{
		"filled", "canceled", "scheduledCancel", "marginCanceled", "vaultWithdrawalCanceled", "openInterestCapCanceled",
		"selfTradeCanceled", "reduceOnlyCanceled", "siblingFilledCanceled", "delistedCanceled", "liquidatedCanceled",
		"rejected", "tickRejected", "minTradeNtlRejected", "perpMarginRejected", "reduceOnlyRejected", "badAloPxRejected",
		"iocCancelRejected", "badTriggerPxRejected", "marketOrderNoLiquidityRejected",
		"positionIncreaseAtOpenInterestCapRejected", "positionFlipAtOpenInterestCapRejected",
		"tooAggressiveAtOpenInterestCapRejected", "openInterestIncreaseRejected", "insufficientSpotBalanceRejected",
		"oracleRejected", "perpMaxPositionRejected",
	} {
		assert.Equalf(t, 2, orderStatusRank(status), "orderStatusRank should rank the final status %s last", status)
	}
}

func TestOpenOrderDEXes(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, nil, nil)
	_, err := ex.openOrderDEXes(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "openOrderDEXes must reject an unsupported asset")
	dexes, err := ex.openOrderDEXes(t.Context(), asset.Spot)
	require.NoError(t, err, "openOrderDEXes must not error for spot")
	assert.Equal(t, []string{""}, dexes, "openOrderDEXes should return the first DEX, which lists spot orders, for spot")
	dexes, err = ex.openOrderDEXes(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "openOrderDEXes must not error for perpetuals")
	assert.Equal(t, []string{"", "xyz"}, dexes, "openOrderDEXes should return every DEX for perpetuals")
	_, err = newUnavailableServerExchange(t).openOrderDEXes(t.Context(), asset.PerpetualContract)
	assert.ErrorIs(t, err, request.ErrBadStatus, "openOrderDEXes should return a registry failure")
}

func TestActiveOrderDEXes(t *testing.T) {
	t.Parallel()
	ex, requests := newCountingServerExchange(t, map[string]string{"perpDexs": `[null,{"name":"xyz"},{"name":"flx"}]`})
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, testBuilderMapping})
	_, err := ex.activeOrderDEXes(t.Context(), &order.MultiOrderRequest{AssetType: asset.Options})
	require.ErrorIs(t, err, asset.ErrNotSupported, "activeOrderDEXes must reject an unsupported asset")
	for _, tc := range []struct {
		name     string
		filter   *order.MultiOrderRequest
		exp      []string
		registry int
	}{
		{name: "spot", filter: &order.MultiOrderRequest{AssetType: asset.Spot, Pairs: currency.Pairs{spotPair}}, exp: []string{""}},
		{name: "pairs of two DEXs", filter: &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Pairs: currency.Pairs{testBuilderPair, perpetualPair, testBuilderPair}}, exp: []string{"xyz", ""}},
		{name: "no pairs", filter: &order.MultiOrderRequest{AssetType: asset.PerpetualContract}, exp: []string{"", "xyz", "flx"}, registry: 1},
		{name: "an uncached pair", filter: &order.MultiOrderRequest{AssetType: asset.PerpetualContract, Pairs: currency.Pairs{perpetualPair, currency.NewPair(currency.SOL, currency.USDC)}}, exp: []string{"", "xyz", "flx"}, registry: 1},
	} {
		fetched := requests()["perpDexs:"]
		dexes, err := ex.activeOrderDEXes(t.Context(), tc.filter)
		require.NoErrorf(t, err, "activeOrderDEXes must not error for %s", tc.name)
		assert.Equalf(t, tc.exp, dexes, "activeOrderDEXes should return the DEXs of %s", tc.name)
		assert.Equalf(t, tc.registry, requests()["perpDexs:"]-fetched, "activeOrderDEXes should read the DEX registry only when it needs every DEX for %s", tc.name)
	}
	_, err = newUnavailableServerExchange(t).activeOrderDEXes(t.Context(), &order.MultiOrderRequest{AssetType: asset.PerpetualContract})
	assert.ErrorIs(t, err, request.ErrBadStatus, "activeOrderDEXes should return a registry failure")
}

func TestSizePositionOrders(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, map[string]string{
		"clearinghouseState:xyz": `{"assetPositions":[{"type":"oneWay","position":{"coin":"xyz:XYZ100","szi":"-0.25"}},{"type":"oneWay","position":{"coin":"xyz:TSLA","szi":"3"}}]}`,
	}, nil)
	orders := []order.Detail{{OrderID: "1"}, {OrderID: "2"}, {OrderID: "3"}}
	require.NoError(t, ex.sizePositionOrders(t.Context(), testAccountAddress, "xyz", orders, map[int]string{0: "xyz:XYZ100", 2: "xyz:GOLD"}), "sizePositionOrders must not error")
	exp := []order.Detail{{OrderID: "1", Amount: 0.25, RemainingAmount: 0.25}, {OrderID: "2"}, {OrderID: "3"}}
	assert.Equal(t, exp, orders, "sizePositionOrders should size each position TP/SL order from the size of its position, leaving one whose position has closed unsized")
	err := newUnavailableServerExchange(t).sizePositionOrders(t.Context(), testAccountAddress, "", orders, map[int]string{0: "BTC"})
	assert.ErrorIs(t, err, request.ErrBadStatus, "sizePositionOrders should return a positions failure")
}

func TestConvertOpenOrder(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, nil, nil)
	_, err := ex.convertOpenOrder(t.Context(), &BasicOrderResponse{Coin: "MISSING", Side: "B"})
	assert.ErrorIs(t, err, errPairMappingNotFound, "convertOpenOrder should reject an order without a market")
	detail, err := ex.convertOpenOrder(t.Context(), &BasicOrderResponse{Coin: "BTC", Side: "X", ReduceOnly: true})
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "convertOpenOrder should reject an order with an unknown side")
	assert.Zero(t, detail, "convertOpenOrder should return no order when it cannot convert one")
	detail, err = ex.convertOpenOrder(t.Context(), &BasicOrderResponse{Coin: "@107", Side: "A", LimitPrice: 95.5, Size: 1, OrderID: 9, Timestamp: milli(1791275926947), OriginalSize: 1.5, ClientOrderID: testClientOrderID})
	require.NoError(t, err, "convertOpenOrder must not error")
	placed := time.UnixMilli(1791275926947).UTC()
	exp := order.Detail{Price: 95.5, Amount: 1.5, ExecutedAmount: 0.5, RemainingAmount: 1, Exchange: ex.Name, OrderID: "9", ClientOrderID: testClientOrderID, Side: order.Sell, Status: order.PartiallyFilled, AssetType: asset.Spot, Date: placed, LastUpdated: placed, Pair: spotPair}
	assert.Equal(t, exp, detail, "convertOpenOrder should convert the order without its order type or time in force")
	detail, err = ex.convertOpenOrder(t.Context(), &BasicOrderResponse{Coin: "BTC", Side: "A", LimitPrice: 100, Size: 1, OrderID: 10, Timestamp: milli(1791275926947), OriginalSize: 1, ReduceOnly: true})
	require.NoError(t, err, "convertOpenOrder must not error for a reduce-only order")
	exp = order.Detail{ReduceOnly: true, Price: 100, Amount: 1, RemainingAmount: 1, Exchange: ex.Name, OrderID: "10", Side: order.Sell, Status: order.Open, AssetType: asset.PerpetualContract, Date: placed, LastUpdated: placed, Pair: perpetualPair}
	assert.Equal(t, exp, detail, "convertOpenOrder should keep a reduce-only order's flag")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	t.Parallel()
	_, err := e.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "WithdrawCryptocurrencyFunds must reject a nil request")
	newRequest := func() *withdraw.Request {
		return &withdraw.Request{Exchange: e.Name, Currency: currency.USDC, Amount: 25, Type: withdraw.Crypto, InternalTransfer: true, Crypto: withdraw.CryptoRequest{Address: testWithdrawalAddress}}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*withdraw.Request)
		err    error
	}{
		{name: "a withdrawal off Hyperliquid", mutate: func(r *withdraw.Request) { r.InternalTransfer = false }, err: common.ErrFunctionNotSupported},
		{name: "a currency other than USDC", mutate: func(r *withdraw.Request) { r.Currency = currency.BTC }, err: errTransferCurrencyInvalid},
		{name: "an address tag", mutate: func(r *withdraw.Request) { r.Crypto.AddressTag = "tag" }, err: errWithdrawalAddressTag},
		{name: "a fee", mutate: func(r *withdraw.Request) { r.Crypto.FeeAmount = 1 }, err: errWithdrawalFeeInput},
		{name: "an invalid address", mutate: func(r *withdraw.Request) { r.Crypto.Address = "invalid" }, err: errInvalidAddress},
	} {
		withdrawal := newRequest()
		tc.mutate(withdrawal)
		_, err := e.WithdrawCryptocurrencyFunds(t.Context(), withdrawal)
		assert.ErrorIsf(t, err, tc.err, "WithdrawCryptocurrencyFunds should reject %s", tc.name)
	}
	_, err = new(Exchange).WithdrawCryptocurrencyFunds(t.Context(), newRequest())
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "WithdrawCryptocurrencyFunds must require credentials")
	_, err = newUnavailableServerExchange(t).WithdrawCryptocurrencyFunds(t.Context(), newRequest())
	require.ErrorIs(t, err, request.ErrBadStatus, "WithdrawCryptocurrencyFunds must return an action failure")

	withdrawal := newRequest()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveUnlessAccountSigner(t)
		skipLiveWithoutTransferDestination(t)
		withdrawal.Amount, withdrawal.Crypto.Address = 1, liveTransferDestination
	}
	ex := newUserSignedTestExchange(t)
	result, err := ex.WithdrawCryptocurrencyFunds(t.Context(), withdrawal)
	require.NoError(t, err, "WithdrawCryptocurrencyFunds must not error for an internal transfer")
	if mockTests {
		exp := &withdraw.ExchangeResponse{Name: ex.Name, ID: strconv.FormatUint(mockUserSignedNonce, 10), Status: "submitted"}
		assert.Equal(t, exp, result, "WithdrawCryptocurrencyFunds should return the transfer's nonce")
		return
	}
	assert.Equal(t, "submitted", result.Status, "WithdrawCryptocurrencyFunds should submit the internal transfer")
	assert.NotEmpty(t, result.ID, "WithdrawCryptocurrencyFunds should return the transfer's nonce")
}

func TestGetFeeByType(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFeeByType must reject a nil fee builder")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.InternationalBankWithdrawalFee, Pair: perpetualPair})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetFeeByType must reject an unsupported fee type")
	_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFeeByType must require a pair")
	for _, notional := range [][2]float64{{-1, 1}, {1, -1}, {math.NaN(), 1}, {1, math.NaN()}, {math.Inf(1), 1}, {1, math.Inf(1)}} {
		_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: perpetualPair, PurchasePrice: notional[0], Amount: notional[1]})
		assert.ErrorIsf(t, err, order.ErrAmountIsInvalid, "GetFeeByType should reject price %v and amount %v", notional[0], notional[1])
	}

	// The mock responses list the builder market xyz:XYZ100 in growth mode with a deployer fee scale of 1, which the
	// docs' table charges at 0.1 + 0.1 of the normal fee; a scale of 2 outside growth mode charges 2 + 2 of it. As live,
	// USDH quotes HYPE-USDH, so USDH-USDC is a pair between two spot quote assets, which pays a fifth of the normal fee
	growthBuilder := testBuilderMapping
	scaledBuilder := pairMapping{pair: currency.NewPair(currency.NewCode("xyz:TSLA"), currency.USDC), coin: "xyz:TSLA", dex: "xyz", assetID: 110001, sizeDecimals: 3, maxLeverage: 20, deployerFeeScale: 2}
	usdh := currency.NewCode("USDH")
	stableMapping := pairMapping{pair: currency.NewPair(usdh, currency.USDC), coin: "@230", assetID: 10230, sizeDecimals: 2}
	usdhQuotedMapping := pairMapping{pair: currency.NewPair(currency.HYPE, usdh), coin: "@232", assetID: 10232, sizeDecimals: 2}
	offline := new(Exchange)
	require.NoError(t, testexch.Setup(offline), "Setup must not error")
	offline.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, growthBuilder, scaledBuilder})
	offline.setPairMappings(asset.Spot, []pairMapping{testSpotMapping, stableMapping, usdhQuotedMapping})
	for _, tc := range []struct {
		pair    currency.Pair
		isMaker bool
		exp     float64
	}{
		{pair: perpetualPair, exp: 100 * 2 * perpetualTakerBaseFeeRate},
		{pair: perpetualPair, isMaker: true, exp: 100 * 2 * perpetualMakerBaseFeeRate},
		{pair: spotPair, exp: 100 * 2 * spotTakerBaseFeeRate},
		{pair: spotPair, isMaker: true, exp: 100 * 2 * spotMakerBaseFeeRate},
		{pair: growthBuilder.pair, exp: 100 * 2 * perpetualTakerBaseFeeRate * 0.2},
		{pair: growthBuilder.pair, isMaker: true, exp: 100 * 2 * perpetualMakerBaseFeeRate * 0.2},
		{pair: scaledBuilder.pair, exp: 100 * 2 * perpetualTakerBaseFeeRate * 4},
		{pair: stableMapping.pair, exp: 100 * 2 * spotTakerBaseFeeRate * 0.2},
		{pair: stableMapping.pair, isMaker: true, exp: 100 * 2 * spotMakerBaseFeeRate * 0.2},
		{pair: usdhQuotedMapping.pair, exp: 100 * 2 * spotTakerBaseFeeRate},
	} {
		builder := &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: tc.pair, IsMaker: tc.isMaker, PurchasePrice: 100, Amount: 2}
		fee, err := offline.GetFeeByType(t.Context(), builder)
		require.NoErrorf(t, err, "GetFeeByType must not error for %s maker %t without credentials", tc.pair, tc.isMaker)
		assert.InDeltaf(t, tc.exp, fee, 1e-12, "GetFeeByType should use the base %s rate for maker %t", tc.pair, tc.isMaker)
		assert.Equalf(t, exchange.OfflineTradeFee, builder.FeeType, "GetFeeByType should estimate offline without credentials for %s", tc.pair)
	}

	// ETH-USDC is enabled for neither asset, so only the cached mappings can identify its fee schedule
	ethPair := currency.NewPair(currency.ETH, currency.USDC)
	ethMapping := pairMapping{pair: ethPair, coin: "ETH", assetID: 1, sizeDecimals: 4, maxLeverage: 25}
	unmapped := newInfoServerExchange(t, nil, nil)
	_, err = unmapped.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: ethPair})
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetFeeByType should reject a pair of neither asset")
	unmapped.setPairMappings(asset.Spot, []pairMapping{{pair: ethPair, coin: "@1"}})
	unmapped.setPairMappings(asset.PerpetualContract, []pairMapping{ethMapping})
	_, err = unmapped.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: ethPair})
	assert.ErrorIs(t, err, errAmbiguousCoinMapping, "GetFeeByType should reject a pair traded by both assets")
	unmapped.setPairMappings(asset.Spot, nil)
	fee, err := unmapped.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: ethPair, PurchasePrice: 100, Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error for a mapped pair that is not enabled")
	assert.InDelta(t, 100*2*perpetualTakerBaseFeeRate, fee, 1e-12, "GetFeeByType should identify the asset of a mapped pair")
	unlisted := newInfoServerExchange(t, map[string]string{"meta": `{"universe":[]}`, "spotMeta": `{"universe":[],"tokens":[]}`}, nil)
	for _, pair := range []currency.Pair{perpetualPair, spotPair} {
		_, err = unlisted.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: pair})
		assert.ErrorIsf(t, err, errPairMappingNotFound, "GetFeeByType should report an enabled pair without a market for %s", pair)
	}

	// Spot quote assets come from spot metadata, shaped as live: USDC quotes every market and USDH quotes HYPE-USDH, so
	// USDH-USDC is between two spot quote assets while HYPE-USDH is not. HYPE-USDC, being enabled, fetches the metadata,
	// whose cached markets then identify the other pairs
	quoted := newInfoServerExchange(t, map[string]string{"spotMeta": `{"universe":[` +
		`{"tokens":[150,0],"name":"@107","index":107},{"tokens":[360,0],"name":"@230","index":230},{"tokens":[150,360],"name":"@232","index":232}],` +
		`"tokens":[{"name":"USDC","szDecimals":8,"weiDecimals":8,"index":0},{"name":"HYPE","szDecimals":2,"weiDecimals":8,"index":150},` +
		`{"name":"USDH","szDecimals":2,"weiDecimals":8,"index":360}]}`}, nil)
	for _, tc := range []struct {
		pair currency.Pair
		exp  float64
	}{
		{pair: spotPair, exp: 100 * 2 * spotTakerBaseFeeRate},
		{pair: stableMapping.pair, exp: 100 * 2 * spotTakerBaseFeeRate * 0.2},
		{pair: usdhQuotedMapping.pair, exp: 100 * 2 * spotTakerBaseFeeRate},
	} {
		quotedFee, err := quoted.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: tc.pair, PurchasePrice: 100, Amount: 2})
		require.NoErrorf(t, err, "GetFeeByType must not error for %s from fetched spot metadata", tc.pair)
		assert.InDeltaf(t, tc.exp, quotedFee, 1e-12, "GetFeeByType should identify whether %s is between two spot quote assets from fetched spot metadata", tc.pair)
	}

	online := newInfoServerExchange(t, map[string]string{"userFees": `{"userCrossRate":"0.000315","userAddRate":"-0.00001","userSpotCrossRate":"0.00049","userSpotAddRate":"0.00028","activeReferralDiscount":"0.04"}`}, nil)
	online.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping, growthBuilder})
	online.setPairMappings(asset.Spot, []pairMapping{testSpotMapping, stableMapping, usdhQuotedMapping})
	for _, tc := range []struct {
		pair    currency.Pair
		isMaker bool
		rate    float64
	}{
		{pair: perpetualPair, rate: 0.000315 * 0.96},
		{pair: perpetualPair, isMaker: true, rate: -0.00001},
		{pair: spotPair, rate: 0.00049 * 0.96},
		{pair: spotPair, isMaker: true, rate: 0.00028 * 0.96},
		{pair: growthBuilder.pair, rate: 0.000315 * 0.2 * 0.96},
		{pair: growthBuilder.pair, isMaker: true, rate: -0.00001 * 0.1},
		{pair: stableMapping.pair, rate: 0.00049 * 0.2 * 0.96},
		{pair: stableMapping.pair, isMaker: true, rate: 0.00028 * 0.2 * 0.96},
	} {
		accountFee, err := online.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: tc.pair, IsMaker: tc.isMaker, PurchasePrice: 100, Amount: 2})
		require.NoErrorf(t, err, "GetFeeByType must not error for %s maker %t", tc.pair, tc.isMaker)
		assert.InDeltaf(t, 100*2*tc.rate, accountFee, 1e-12, "GetFeeByType should scale the account's %s rate for maker %t as the fee formula does, discounting fees but not rebates", tc.pair, tc.isMaker)
	}
	for _, pair := range []currency.Pair{perpetualPair, spotPair} {
		_, err = newUnavailableServerExchange(t).GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: pair})
		assert.ErrorIsf(t, err, request.ErrBadStatus, "GetFeeByType should return a %s metadata failure", pair)
	}
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	failed.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	for _, pair := range []currency.Pair{perpetualPair, spotPair} {
		_, err = failed.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: pair})
		assert.ErrorIsf(t, err, request.ErrBadStatus, "GetFeeByType should return a %s fee failure", pair)
	}
	malformedAddress := newInfoServerExchange(t, nil, nil)
	malformedAddress.setPairMappings(asset.Spot, []pairMapping{testSpotMapping})
	setTestCredentials(malformedAddress, &accounts.Credentials{Key: "invalid", Secret: testPrivateKey})
	_, err = malformedAddress.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: spotPair})
	assert.ErrorIs(t, err, errInvalidAddress, "GetFeeByType should reject a configured account address that is not an address")

	if !mockTests {
		// Live spot metadata lists markets quoted in USDH, so USDH-USDC is between two spot quote assets; estimating
		// HYPE-USDC first caches every spot market, as USDH-USDC is not enabled
		_, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: spotPair})
		require.NoError(t, err, "GetFeeByType must not error for HYPE-USDC")
		fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.OfflineTradeFee, Pair: stableMapping.pair, PurchasePrice: 100, Amount: 2})
		require.NoError(t, err, "GetFeeByType must not error for USDH-USDC")
		assert.InDelta(t, 100*2*spotTakerBaseFeeRate*0.2, fee, 1e-12, "GetFeeByType should identify USDH-USDC as a pair between two spot quote assets")
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	fee, err = e.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: perpetualPair, PurchasePrice: 100, Amount: 2})
	require.NoError(t, err, "GetFeeByType must not error")
	if mockTests {
		// The mock exchange skips the credential check, so it estimates offline at the base taker rate
		assert.InDelta(t, 100*2*perpetualTakerBaseFeeRate, fee, 1e-12, "GetFeeByType should return the base taker fee")
	} else {
		assert.Positive(t, fee, "GetFeeByType should return the fee")
	}
}

func TestTradeFeeRate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		rate             float64
		referralDiscount float64
		isMaker          bool
		stablePair       bool
		scale            float64
		growthMode       bool
		exp              float64
	}{
		// The deployer fee table of the HIP-3 deployer actions docs, for a normal fee of 1
		{rate: 1, exp: 1},
		{rate: 1, scale: 0.5, exp: 1.5},
		{rate: 1, scale: 1, exp: 2},
		{rate: 1, scale: 3, exp: 6},
		{rate: 1, growthMode: true, exp: 0.1},
		{rate: 1, scale: 0.5, growthMode: true, exp: 0.15},
		{rate: 1, scale: 1, growthMode: true, exp: 0.2},
		{rate: 1, scale: 3.01, growthMode: true, exp: 0.602},
		{rate: 1, scale: 9.99, growthMode: true, exp: 1.998},
		{rate: 1, isMaker: true, scale: 3, growthMode: true, exp: 0.6},
		// Growth mode cuts a maker rebate too, which the deployer fee scale leaves whole
		{rate: -0.00001, isMaker: true, scale: 3, growthMode: true, exp: -0.000001},
		{rate: -0.00001, isMaker: true, scale: 3, exp: -0.00001},
		{scale: 3, growthMode: true},
		{isMaker: true, scale: 3, growthMode: true},
		// A pair between two spot quote assets pays a fifth of any rate, rebates included
		{rate: 0.0007, stablePair: true, exp: 0.00014},
		{rate: 0.0004, isMaker: true, stablePair: true, exp: 0.00008},
		{rate: -0.00002, isMaker: true, stablePair: true, exp: -0.000004},
		// The referral discount cuts fees but not maker rebates
		{rate: 0.00045, referralDiscount: 0.04, exp: 0.000432},
		{rate: 0.00015, referralDiscount: 0.04, isMaker: true, exp: 0.000144},
		{rate: -0.00001, referralDiscount: 0.04, isMaker: true, exp: -0.00001},
		{rate: 0.0007, referralDiscount: 0.04, stablePair: true, exp: 0.0001344},
		{rate: 1, referralDiscount: 0.04, scale: 1, growthMode: true, exp: 0.192},
	} {
		result := tradeFeeRate(tc.rate, tc.referralDiscount, tc.isMaker, tc.stablePair, &pairMapping{deployerFeeScale: tc.scale, growthMode: tc.growthMode})
		assert.InDeltaf(t, tc.exp, result, 1e-12, "tradeFeeRate should scale rate %v with referral discount %v, maker %t, stable pair %t, scale %v and growth mode %t as the fee formula does", tc.rate, tc.referralDiscount, tc.isMaker, tc.stablePair, tc.scale, tc.growthMode)
	}
}

func TestIsSpotQuoteAsset(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	assert.False(t, ex.isSpotQuoteAsset(currency.USDC), "isSpotQuoteAsset should not identify a quote asset without cached spot markets")
	// As live, USDC quotes HYPE and USDH, USDH quotes HYPE, and HYPE quotes only a builder DEX's perpetuals
	usdh := currency.NewCode("USDH")
	ex.setPairMappings(asset.Spot, []pairMapping{
		testSpotMapping,
		{pair: currency.NewPair(usdh, currency.USDC), coin: "@230", assetID: 10230, sizeDecimals: 2},
		{pair: currency.NewPair(currency.HYPE, usdh), coin: "@232", assetID: 10232, sizeDecimals: 2},
	})
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.HYPE), coin: "xyz:XYZ100", dex: "xyz", assetID: 110000, sizeDecimals: 4}})
	for token, exp := range map[currency.Code]bool{currency.USDC: true, usdh: true, currency.HYPE: false, currency.NewCode("PURR"): false} {
		assert.Equalf(t, exp, ex.isSpotQuoteAsset(token), "isSpotQuoteAsset should identify whether %s quotes a cached spot market", token)
	}
}

func TestValidateAPICredentials(t *testing.T) {
	t.Parallel()
	require.ErrorIs(t, e.ValidateAPICredentials(t.Context(), asset.Options), asset.ErrNotSupported, "ValidateAPICredentials must reject an unsupported asset")
	require.ErrorIs(t, new(Exchange).ValidateAPICredentials(t.Context(), asset.Empty), request.ErrAuthRequestFailed, "ValidateAPICredentials must require credentials")
	missing := newInfoServerExchange(t, map[string]string{"userRole": `{"role":"missing"}`}, nil)
	err := missing.ValidateAPICredentials(t.Context(), asset.Spot)
	assert.ErrorIs(t, err, errConfiguredAccountMissing, "ValidateAPICredentials should report a missing account")
	assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "ValidateAPICredentials should report a missing account as an authentication failure")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	assert.NoError(t, e.ValidateAPICredentials(t.Context(), asset.PerpetualContract), "ValidateAPICredentials should not error for the configured account")
}

func TestSetLeverage(t *testing.T) {
	t.Parallel()
	ethPair := currency.NewPair(currency.ETH, currency.USDC)
	isolatedPair := currency.NewPair(currency.NewCode("HPOS"), currency.USDC)
	var action string
	ex := newTradingServerExchange(t, map[string]string{
		"meta:": `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40},{"name":"ETH"},{"name":"OLD","isDelisted":true},{"name":"HPOS","maxLeverage":3,"onlyIsolated":true,"marginMode":"strictIsolated"}]}`,
	}, captureAction(t, &action, `{"status":"ok","response":{"type":"default"}}`))
	for _, tc := range []struct {
		a          asset.Item
		pair       currency.Pair
		marginType margin.Type
		amount     float64
		err        error
	}{
		{a: asset.Spot, pair: spotPair, marginType: margin.Multi, amount: 1, err: asset.ErrNotSupported},
		{a: asset.PerpetualContract, pair: currency.NewPair(currency.SOL, currency.USDC), marginType: margin.Multi, amount: 1, err: errPairMappingNotFound},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: 0, err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: -1, err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: 1.5, err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: math.NaN(), err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: math.Inf(1), err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.Multi, amount: 41, err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: ethPair, marginType: margin.Multi, amount: 1, err: errInvalidLeverage},
		{a: asset.PerpetualContract, pair: perpetualPair, marginType: margin.NoMargin, amount: 1, err: margin.ErrMarginTypeUnsupported},
		{a: asset.PerpetualContract, pair: isolatedPair, marginType: margin.Multi, amount: 1, err: errCrossMarginUnavailable},
	} {
		err := ex.SetLeverage(t.Context(), tc.a, tc.pair, tc.marginType, tc.amount, order.UnknownSide)
		assert.ErrorIsf(t, err, tc.err, "SetLeverage should reject %s %s %s leverage %v", tc.a, tc.pair, tc.marginType, tc.amount)
	}

	for _, tc := range []struct {
		pair       currency.Pair
		marginType margin.Type
		amount     float64
		exp        string
	}{
		{pair: perpetualPair, marginType: margin.Unset, amount: 10, exp: `{"type":"updateLeverage","asset":0,"isCross":true,"leverage":10}`},
		{pair: perpetualPair, marginType: margin.Multi, amount: 20, exp: `{"type":"updateLeverage","asset":0,"isCross":true,"leverage":20}`},
		{pair: perpetualPair, marginType: margin.Isolated, amount: 40, exp: `{"type":"updateLeverage","asset":0,"isCross":false,"leverage":40}`},
		{pair: isolatedPair, marginType: margin.Isolated, amount: 3, exp: `{"type":"updateLeverage","asset":3,"isCross":false,"leverage":3}`},
		{pair: testBuilderPair, marginType: margin.Multi, amount: 30, exp: `{"type":"updateLeverage","asset":110000,"isCross":true,"leverage":30}`},
	} {
		require.NoErrorf(t, ex.SetLeverage(t.Context(), asset.PerpetualContract, tc.pair, tc.marginType, tc.amount, order.UnknownSide), "SetLeverage must not error for %s %s", tc.pair, tc.marginType)
		assert.JSONEqf(t, tc.exp, action, "SetLeverage should send the leverage of %s %s", tc.pair, tc.marginType)
	}
	err := newTradingServerExchange(t, nil, respondWith(`{"status":"err","response":"leverage update failed"}`)).SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Multi, 1, order.UnknownSide)
	assert.ErrorIs(t, err, errActionResponse, "SetLeverage should return a rejected action")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		skipLiveWithoutPrivateKey(t)
	}
	// Setting the account's current margin mode and leverage back leaves a live account unchanged
	address, err := e.getWatchAddress(t.Context())
	require.NoError(t, err, "getWatchAddress must not error")
	current, err := e.GetActiveAssetData(t.Context(), address, testPerpetualMapping.coin)
	require.NoError(t, err, "GetActiveAssetData must not error")
	marginType := margin.Isolated
	if strings.EqualFold(current.Leverage.Type, "cross") {
		marginType = margin.Multi
	}
	assert.NoError(t, e.SetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, marginType, float64(current.Leverage.Value), order.UnknownSide), "SetLeverage should set the account's current margin mode and leverage")
}

func TestGetLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetLeverage(t.Context(), asset.Spot, spotPair, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLeverage must reject spot")
	_, err = e.GetLeverage(t.Context(), asset.PerpetualContract, currency.EMPTYPAIR, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLeverage must require a pair")
	unconfigured := new(Exchange)
	unconfigured.SetDefaults()
	unconfigured.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = unconfigured.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, exchange.ErrCredentialsAreEmpty, "GetLeverage must require an account address")

	for _, tc := range []struct {
		leverage   string
		marginType margin.Type
		exp        float64
		err        error
	}{
		{leverage: `{"type":"cross","value":20}`, marginType: margin.Unset, exp: 20},
		{leverage: `{"type":"cross","value":20}`, marginType: margin.Multi, exp: 20},
		{leverage: `{"type":"cross","value":20}`, marginType: margin.Isolated, err: margin.ErrMarginTypeUnsupported},
		{leverage: `{"type":"isolated","value":5,"rawUsd":"-95.06"}`, marginType: margin.Isolated, exp: 5},
		{leverage: `{"type":"isolated","value":5,"rawUsd":"-95.06"}`, marginType: margin.Multi, err: margin.ErrMarginTypeUnsupported},
		{leverage: `{"type":"portfolio","value":5}`, marginType: margin.Unset, err: margin.ErrMarginTypeUnsupported},
		{leverage: `{"type":"cross","value":0}`, marginType: margin.Unset, err: errInvalidLeverage},
	} {
		ex := newTradingServerExchange(t, map[string]string{"activeAssetData": `{"user":"` + testAccountAddress + `","coin":"BTC","leverage":` + tc.leverage + `}`}, nil)
		value, err := ex.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, tc.marginType, order.UnknownSide)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "GetLeverage should reject leverage %s as %s", tc.leverage, tc.marginType)
			continue
		}
		require.NoErrorf(t, err, "GetLeverage must not error for leverage %s as %s", tc.leverage, tc.marginType)
		assert.Equalf(t, tc.exp, value, "GetLeverage should return leverage %s as %s", tc.leverage, tc.marginType)
	}
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Unset, order.UnknownSide)
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLeverage should return an asset data failure")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	value, err := e.GetLeverage(t.Context(), asset.PerpetualContract, perpetualPair, margin.Unset, order.UnknownSide)
	require.NoError(t, err, "GetLeverage must not error")
	if mockTests {
		assert.Equal(t, 20.0, value, "GetLeverage should return the account's leverage")
	} else {
		assert.Positive(t, value, "GetLeverage should return the account's leverage")
	}
}

func TestGetLatestFundingRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLatestFundingRates must reject a nil request")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetLatestFundingRates must reject spot")
	_, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "GetLatestFundingRates must reject predicted rates")

	rates, err := e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, Pair: perpetualPair})
	require.NoError(t, err, "GetLatestFundingRates must not error for a pair")
	require.Len(t, rates, 1, "GetLatestFundingRates must return the pair's rate")
	checked := rates[0].TimeChecked
	assert.WithinDuration(t, time.Now(), checked, time.Minute, "GetLatestFundingRates should time the check when it is made")
	exp := fundingrate.LatestRateResponse{
		Exchange:       e.Name,
		Asset:          asset.PerpetualContract,
		Pair:           perpetualPair,
		LatestRate:     fundingrate.Rate{Time: checked.Truncate(time.Hour), Rate: rates[0].LatestRate.Rate},
		TimeOfNextRate: checked.Truncate(time.Hour).Add(time.Hour),
		TimeChecked:    checked,
	}
	if mockTests {
		exp.LatestRate.Rate = decimal.MustFromFloat(0.0000125)
	}
	assert.Equal(t, exp, rates[0], "GetLatestFundingRates should return the current hour's rate")

	rates, err = e.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.NoError(t, err, "GetLatestFundingRates must not error for every pair")
	if mockTests {
		require.NotEmpty(t, rates, "GetLatestFundingRates must return every market's rate")
		checked = rates[0].TimeChecked
		expRates := make([]fundingrate.LatestRateResponse, len(testPerpetualMappings))
		for i, rate := range []float64{0.0000125, 0.0000125, 0.0000125, 0.00000625, 0.00000625, 0.0000125} {
			expRates[i] = fundingrate.LatestRateResponse{
				Exchange:       e.Name,
				Asset:          asset.PerpetualContract,
				Pair:           testPerpetualMappings[i].pair,
				LatestRate:     fundingrate.Rate{Time: checked.Truncate(time.Hour), Rate: decimal.MustFromFloat(rate)},
				TimeOfNextRate: checked.Truncate(time.Hour).Add(time.Hour),
				TimeChecked:    checked,
			}
		}
		assert.Equal(t, expRates, rates, "GetLatestFundingRates should return every DEX's current rates")
	} else {
		assert.NotEmpty(t, rates, "GetLatestFundingRates should return every market's rate")
	}

	empty := newInfoServerExchange(t, map[string]string{"meta": `{"universe":[]}`}, nil)
	_, err = empty.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	assert.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "GetLatestFundingRates should report no markets")
	_, err = empty.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, Pair: perpetualPair})
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetLatestFundingRates should report a pair without a market")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLatestFundingRates should return a context failure")
	_, err = newUnavailableServerExchange(t).GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetLatestFundingRates should return a metadata failure")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	t.Parallel()
	start, end := getTime()
	_, err := e.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalFundingRates must reject a nil request")
	for _, tc := range []struct {
		arg fundingrate.HistoricalRatesRequest
		err error
	}{
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.Spot}, err: asset.ErrNotSupported},
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract}, err: currency.ErrCurrencyPairEmpty},
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, IncludePredictedRate: true}, err: common.ErrFunctionNotSupported},
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, IncludePayments: true}, err: common.ErrFunctionNotSupported},
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: end, EndDate: start}, err: common.ErrStartAfterEnd},
		{arg: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: start, EndDate: end, PaymentCurrency: currency.BTC}, err: asset.ErrNotSupported},
	} {
		_, err := e.GetHistoricalFundingRates(t.Context(), &tc.arg)
		assert.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates should reject %+v", tc.arg)
	}

	result, err := e.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: start, EndDate: end, PaymentCurrency: currency.USDC})
	require.NoError(t, err, "GetHistoricalFundingRates must not error")
	if mockTests {
		rates := []fundingrate.Rate{
			{Time: time.UnixMilli(1791255600057).UTC(), Rate: decimal.MustFromFloat(0.0000125)},
			{Time: time.UnixMilli(1791259200041).UTC(), Rate: decimal.MustFromFloat(-0.0000032)},
		}
		exp := &fundingrate.HistoricalRates{
			Exchange:        e.Name,
			Asset:           asset.PerpetualContract,
			Pair:            perpetualPair,
			StartDate:       start,
			EndDate:         end,
			LatestRate:      rates[1],
			FundingRates:    rates,
			PaymentCurrency: currency.USDC,
		}
		assert.Equal(t, exp, result, "GetHistoricalFundingRates should return every hourly rate in the range")
	} else {
		assert.NotEmpty(t, result.FundingRates, "GetHistoricalFundingRates should return rates")
	}

	pageStart := time.UnixMilli(mockStartTime).Add(-(maximumFundingHistoryCount + 2) * time.Hour)
	pageEnd := pageStart.Add((maximumFundingHistoryCount + 1) * time.Hour)
	var pages atomic.Int32
	paged := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		count := 2
		if pages.Add(1) == 1 {
			count = maximumFundingHistoryCount
		}
		records := make([]string, 0, count)
		for i := range uint64(count) {
			records = append(records, fmt.Sprintf(`{"coin":"BTC","fundingRate":"0.0001","premium":"0.0","time":%d}`, body.StartTime+i*uint64(time.Hour/time.Millisecond)))
		}
		_, err := w.Write([]byte("[" + strings.Join(records, ",") + "]"))
		assert.NoError(t, err, "Writing the funding page should not error")
	}))
	paged.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	result, err = paged.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: pageStart, EndDate: pageEnd})
	require.NoError(t, err, "GetHistoricalFundingRates must not error across pages")
	assert.Len(t, result.FundingRates, maximumFundingHistoryCount+2, "GetHistoricalFundingRates should return every page's rates")
	assert.Equal(t, int32(2), pages.Load(), "GetHistoricalFundingRates should request the next page after a full page")

	// A full page ending a millisecond before the end date leaves a request for the end date alone, whose bounds are
	// both inclusive, so it must be sent rather than rejected
	var requests []InfoRequest
	boundary := newTestServerExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body InfoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decoding the info request should not error") {
			return
		}
		requests = append(requests, body)
		// Funding settles hourly from pageStart, and a page holds the records of the range up to the page size
		var records []string
		for recordTime := unixMilli(pageStart); recordTime <= body.EndTime && len(records) < maximumFundingHistoryCount; recordTime += uint64(time.Hour / time.Millisecond) {
			if recordTime >= body.StartTime {
				records = append(records, fmt.Sprintf(`{"coin":"BTC","fundingRate":"0.0001","premium":"0.0","time":%d}`, recordTime))
			}
		}
		_, err := w.Write([]byte("[" + strings.Join(records, ",") + "]"))
		assert.NoError(t, err, "Writing the funding page should not error")
	}))
	boundary.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	boundaryEnd := pageStart.Add((maximumFundingHistoryCount-1)*time.Hour + time.Millisecond)
	result, err = boundary.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: pageStart, EndDate: boundaryEnd})
	require.NoError(t, err, "GetHistoricalFundingRates must not error when a full page ends a millisecond before the end date")
	assert.Len(t, result.FundingRates, maximumFundingHistoryCount, "GetHistoricalFundingRates should return the full page's rates")
	require.Len(t, requests, 2, "GetHistoricalFundingRates must request the end date after the full page")
	assert.Equal(t, unixMilli(boundaryEnd), requests[1].StartTime, "GetHistoricalFundingRates should start the last request at the end date")
	assert.Equal(t, unixMilli(boundaryEnd), requests[1].EndTime, "GetHistoricalFundingRates should end the last request at the end date")

	builder := newTradingServerExchange(t, map[string]string{"fundingHistory": `[{"coin":"xyz:XYZ100","fundingRate":"0.0002","premium":"0.0001","time":` + strconv.FormatInt(pageStart.UnixMilli(), 10) + `}]`}, nil)
	builder.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.HYPE), coin: "xyz:XYZ100", dex: "xyz"}})
	builderRequest := &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.HYPE), StartDate: pageStart, EndDate: pageEnd}
	result, err = builder.GetHistoricalFundingRates(t.Context(), builderRequest)
	require.NoError(t, err, "GetHistoricalFundingRates must not error for a builder DEX")
	assert.Equal(t, currency.HYPE, result.PaymentCurrency, "GetHistoricalFundingRates should pay funding in the DEX's collateral")
	builderRequest.PaymentCurrency = currency.USDC
	_, err = builder.GetHistoricalFundingRates(t.Context(), builderRequest)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "GetHistoricalFundingRates should reject a payment currency other than the DEX's collateral")

	record := `{"coin":"BTC","fundingRate":"0.0001","time":` + strconv.FormatInt(pageStart.UnixMilli(), 10) + `}`
	for _, tc := range []struct {
		name    string
		history string
		err     error
	}{
		{name: "empty history", history: `[]`, err: fundingrate.ErrNoFundingRatesFound},
		{name: "oversized page", history: `[` + strings.Repeat(record+",", maximumFundingHistoryCount) + record + `]`, err: errUnexpectedResponseLength},
		{name: "page of another coin", history: `[{"coin":"ETH","fundingRate":"0.1","time":` + strconv.FormatInt(pageStart.UnixMilli(), 10) + `}]`, err: errMalformedResponse},
		{name: "repeated time", history: `[` + record + `,` + record + `]`, err: errMalformedResponse},
		{name: "record outside the range", history: `[{"coin":"BTC","fundingRate":"0.1","time":` + strconv.FormatInt(pageEnd.Add(time.Hour).UnixMilli(), 10) + `}]`, err: errMalformedResponse},
	} {
		ex := newTradingServerExchange(t, map[string]string{"fundingHistory": tc.history}, nil)
		_, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: pageStart, EndDate: pageEnd})
		assert.ErrorIsf(t, err, tc.err, "GetHistoricalFundingRates should reject an %s", tc.name)
	}
	_, err = newInfoServerExchange(t, map[string]string{"meta": `{"universe":[]}`}, nil).GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: pageStart, EndDate: pageEnd})
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetHistoricalFundingRates should report a pair without a market")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: perpetualPair, StartDate: pageStart, EndDate: pageEnd})
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetHistoricalFundingRates should return a history failure")
}

func TestGetOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: spotPair.Base.Item, Quote: spotPair.Quote.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "GetOpenInterest must reject spot")

	result, err := e.GetOpenInterest(t.Context(), key.PairAsset{Base: perpetualPair.Base.Item, Quote: perpetualPair.Quote.Item, Asset: asset.PerpetualContract})
	require.NoError(t, err, "GetOpenInterest must not error for a pair")
	require.Len(t, result, 1, "GetOpenInterest must return the pair's open interest")
	if mockTests {
		exp := []futures.OpenInterest{{Key: key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, perpetualPair), OpenInterest: 39310.43958}}
		assert.Equal(t, exp, result, "GetOpenInterest should return the pair's open interest")
	} else {
		assert.Positive(t, result[0].OpenInterest, "GetOpenInterest should return the pair's open interest")
	}
	result, err = e.GetOpenInterest(t.Context())
	require.NoError(t, err, "GetOpenInterest must not error for every pair")
	if mockTests {
		exp := make([]futures.OpenInterest, len(testPerpetualMappings))
		for i, openInterest := range []float64{39310.43958, 1184127.1915999996, 1250, 7148.95, 133465.102, 1520.5} {
			exp[i] = futures.OpenInterest{Key: key.NewExchangeAssetPair(e.Name, asset.PerpetualContract, testPerpetualMappings[i].pair), OpenInterest: openInterest}
		}
		assert.Equal(t, exp, result, "GetOpenInterest should return every DEX's active markets")
	} else {
		assert.NotEmpty(t, result, "GetOpenInterest should return every market's open interest")
	}

	empty := newInfoServerExchange(t, map[string]string{"meta": `{"universe":[]}`}, nil)
	result, err = empty.GetOpenInterest(t.Context())
	require.NoError(t, err, "GetOpenInterest must not error without markets")
	assert.Empty(t, result, "GetOpenInterest should return nothing without markets")
	_, err = empty.GetOpenInterest(t.Context(), key.PairAsset{Base: perpetualPair.Base.Item, Quote: perpetualPair.Quote.Item, Asset: asset.PerpetualContract})
	assert.ErrorIs(t, err, errPairMappingNotFound, "GetOpenInterest should report a pair without a market")
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, err = failed.GetOpenInterest(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenInterest should return a context failure")
	_, err = newUnavailableServerExchange(t).GetOpenInterest(t.Context())
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetOpenInterest should return a metadata failure")
}

func TestBuildOrderRequest(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, map[string]string{"allMids": `{"BTC":"100","@107":"10"}`, "allMids:xyz": `{"xyz:XYZ100":"50"}`}, nil)
	for _, tc := range []struct {
		name         string
		pair         currency.Pair
		a            asset.Item
		orderType    order.Type
		side         order.Side
		timeInForce  order.TimeInForce
		amount       float64
		price        float64
		triggerPrice float64
		slippage     float64
		reduceOnly   bool
		exp          OrderRequest
	}{
		{
			name: "limit", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.Limit, side: order.Buy, timeInForce: order.GoodTillCancel, amount: 0.12345, price: 100.5, reduceOnly: true,
			exp: OrderRequest{Asset: 0, IsBuy: true, Price: 100.5, Size: 0.12345, ReduceOnly: true, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "market buy", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.Market, side: order.Buy, amount: 0.1, slippage: 0.01,
			exp: OrderRequest{IsBuy: true, Price: 101, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "market sell", pair: spotPair, a: asset.Spot, orderType: order.Market, side: order.Sell, amount: 1, slippage: 0.01,
			exp: OrderRequest{Asset: 10107, Price: 9.9, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "market buy between ticks", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.Market, side: order.Buy, amount: 0.1, slippage: 0.0123,
			exp: OrderRequest{IsBuy: true, Price: 101.2, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "market sell between ticks", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.Market, side: order.Sell, amount: 0.1, slippage: 0.0123,
			exp: OrderRequest{Price: 98.8, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "builder DEX market", pair: testBuilderPair, a: asset.PerpetualContract, orderType: order.Market, side: order.Buy, amount: 1, slippage: 0.01,
			exp: OrderRequest{Asset: 110000, IsBuy: true, Price: 50.5, Size: 1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID},
		},
		{
			name: "stop market", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.StopMarket, side: order.Sell, amount: 0.1, triggerPrice: 90, slippage: 0.1, reduceOnly: true,
			exp: OrderRequest{Price: 81, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 90, TakeProfitStopLoss: TriggerStopLoss}, ClientOrderID: testClientOrderID},
		},
		{
			name: "stop market between ticks", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.StopMarket, side: order.Sell, amount: 0.1, triggerPrice: 90, slippage: 0.0123, reduceOnly: true,
			exp: OrderRequest{Price: 88.9, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 90, TakeProfitStopLoss: TriggerStopLoss}, ClientOrderID: testClientOrderID},
		},
		{
			name: "stop market that may open a position", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.StopMarket, side: order.Buy, amount: 0.1, triggerPrice: 110, slippage: 0.1,
			exp: OrderRequest{IsBuy: true, Price: 121, Size: 0.1, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 110, TakeProfitStopLoss: TriggerStopLoss}, ClientOrderID: testClientOrderID},
		},
		{
			name: "take profit limit", pair: perpetualPair, a: asset.PerpetualContract, orderType: order.TakeProfit, side: order.Buy, amount: 0.1, price: 111, triggerPrice: 110, reduceOnly: true,
			exp: OrderRequest{IsBuy: true, Price: 111, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{TriggerPrice: 110, TakeProfitStopLoss: TriggerTakeProfit}, ClientOrderID: testClientOrderID},
		},
	} {
		result, _, err := ex.buildOrderRequest(t.Context(), tc.pair, tc.a, tc.orderType, tc.side, tc.timeInForce, tc.amount, tc.price, tc.triggerPrice, tc.slippage, tc.reduceOnly, strings.ToUpper(testClientOrderID[:2])+testClientOrderID[2:])
		require.NoErrorf(t, err, "buildOrderRequest must not error for a %s order", tc.name)
		assert.Equalf(t, tc.exp, result, "buildOrderRequest should build the %s order", tc.name)
	}

	// An unavailable server fails every request and no market is cached, so these errors prove the parameters are
	// checked before the market lookup
	unavailable := newUnavailableServerExchange(t)
	for _, tc := range []struct {
		name      string
		a         asset.Item
		orderType order.Type
		clientID  string
		err       error
	}{
		{name: "invalid client order ID", a: asset.PerpetualContract, orderType: order.Limit, clientID: "invalid", err: errClientOrderIDInvalid},
		{name: "spot trigger", a: asset.Spot, orderType: order.StopLimit, err: errTriggerOrderPerpetualOnly},
		{name: "unsupported type", a: asset.PerpetualContract, orderType: order.TrailingStop, err: order.ErrTypeIsInvalid},
	} {
		_, _, err := unavailable.buildOrderRequest(t.Context(), perpetualPair, tc.a, tc.orderType, order.Buy, order.UnknownTIF, 1, 1, 1, 0, false, tc.clientID)
		assert.ErrorIsf(t, err, tc.err, "buildOrderRequest should reject a %s before looking up the market", tc.name)
	}

	for _, tc := range []struct {
		name         string
		pair         currency.Pair
		orderType    order.Type
		amount       float64
		price        float64
		triggerPrice float64
		slippage     float64
		err          error
	}{
		{name: "pair without a market", pair: currency.NewPair(currency.SOL, currency.USDC), orderType: order.Limit, amount: 1, price: 1, err: errPairMappingNotFound},
		{name: "size beyond the market's precision", pair: perpetualPair, orderType: order.Limit, amount: 0.000001, price: 1, err: errSizePrecision},
		{name: "price beyond the market's precision", pair: perpetualPair, orderType: order.Limit, amount: 1, price: 100.55, err: errPricePrecision},
		{name: "limit without a price", pair: perpetualPair, orderType: order.Limit, amount: 1, err: errInvalidMarketPrice},
		{name: "trigger price beyond the market's precision", pair: perpetualPair, orderType: order.StopMarket, amount: 1, triggerPrice: 100.55, slippage: 0.1, err: errPricePrecision},
		{name: "unbounded trigger market price", pair: perpetualPair, orderType: order.StopMarket, amount: 1, triggerPrice: math.MaxFloat64, slippage: 0.9, err: errInvalidMarketPrice},
		{name: "trigger limit without price", pair: perpetualPair, orderType: order.StopLimit, amount: 1, triggerPrice: 100, err: errInvalidMarketPrice},
	} {
		_, _, err := ex.buildOrderRequest(t.Context(), tc.pair, asset.PerpetualContract, tc.orderType, order.Buy, order.UnknownTIF, tc.amount, tc.price, tc.triggerPrice, tc.slippage, false, "")
		assert.ErrorIsf(t, err, tc.err, "buildOrderRequest should reject a %s", tc.name)
	}

	for _, mids := range []string{`{}`, `{"BTC":"0"}`} {
		_, _, err := newTradingServerExchange(t, map[string]string{"allMids": mids}, nil).buildOrderRequest(t.Context(), perpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
		assert.ErrorIsf(t, err, errMarketMidPriceNotFound, "buildOrderRequest should reject mid prices %s", mids)
	}
	failed := newUnavailableServerExchange(t)
	failed.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	_, _, err := failed.buildOrderRequest(t.Context(), perpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	assert.ErrorIs(t, err, request.ErrBadStatus, "buildOrderRequest should return a mid price failure")
	delisted := newInfoServerExchange(t, map[string]string{"meta": `{"universe":[{"name":"BTC","szDecimals":5},{"name":"MATIC","szDecimals":1,"isDelisted":true}]}`}, nil)
	_, _, err = delisted.buildOrderRequest(t.Context(), testDelistedMapping.pair, asset.PerpetualContract, order.Limit, order.Buy, order.GoodTillCancel, 10, 0.4, 0, 0, false, "")
	assert.ErrorIs(t, err, errMarketDelisted, "buildOrderRequest should refuse an order on a delisted market")
	unpriceable := newTradingServerExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, nil)
	unpriceable.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: perpetualPair, coin: "BTC", sizeDecimals: 7}})
	_, _, err = unpriceable.buildOrderRequest(t.Context(), perpetualPair, asset.PerpetualContract, order.Market, order.Buy, order.UnknownTIF, 1, 0, 0, 0.01, false, "")
	assert.ErrorIs(t, err, errSizePrecision, "buildOrderRequest should reject a market whose size decimals leave no price decimals")
}

func TestValidateOrderParameters(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name         string
		a            asset.Item
		orderType    order.Type
		timeInForce  order.TimeInForce
		price        float64
		triggerPrice float64
		slippage     float64
		clientID     string
		err          error
	}{
		{name: "limit", a: asset.Spot, orderType: order.Limit, timeInForce: order.PostOnly, price: 1, clientID: testClientOrderID},
		{name: "market", a: asset.Spot, orderType: order.Market, slippage: 0.5},
		{name: "IOC market", a: asset.Spot, orderType: order.Market, timeInForce: order.ImmediateOrCancel, slippage: 0.5},
		{name: "FOK market", a: asset.Spot, orderType: order.Market, timeInForce: order.FillOrKill, slippage: 0.5, err: order.ErrUnsupportedTimeInForce},
		{name: "GTC market", a: asset.Spot, orderType: order.Market, timeInForce: order.GoodTillCancel, slippage: 0.5, err: order.ErrUnsupportedTimeInForce},
		{name: "post-only market", a: asset.Spot, orderType: order.Market, timeInForce: order.PostOnly, slippage: 0.5, err: order.ErrUnsupportedTimeInForce},
		{name: "trigger market", a: asset.PerpetualContract, orderType: order.StopMarket, triggerPrice: 1, slippage: 0.5},
		{name: "trigger market with a limit price", a: asset.PerpetualContract, orderType: order.TakeProfitMarket, price: 2, triggerPrice: 1},
		{name: "trigger limit", a: asset.PerpetualContract, orderType: order.StopLimit, price: 2, triggerPrice: 1},
		{name: "GTC trigger limit", a: asset.PerpetualContract, orderType: order.TakeProfit, timeInForce: order.GoodTillCancel, price: 2, triggerPrice: 1},
		{name: "IOC trigger market", a: asset.PerpetualContract, orderType: order.StopMarket, timeInForce: order.ImmediateOrCancel, triggerPrice: 1, slippage: 0.5, err: order.ErrUnsupportedTimeInForce},
		{name: "FOK trigger limit", a: asset.PerpetualContract, orderType: order.StopLimit, timeInForce: order.FillOrKill, price: 2, triggerPrice: 1, err: order.ErrUnsupportedTimeInForce},
		{name: "post-only trigger limit", a: asset.PerpetualContract, orderType: order.TakeProfit, timeInForce: order.PostOnly, price: 2, triggerPrice: 1, err: order.ErrUnsupportedTimeInForce},
		{name: "invalid client order ID", a: asset.Spot, orderType: order.Limit, price: 1, clientID: "invalid", err: errClientOrderIDInvalid},
		{name: "negative price", a: asset.Spot, orderType: order.Limit, price: -1, err: errInvalidMarketPrice},
		{name: "NaN price", a: asset.Spot, orderType: order.Limit, price: math.NaN(), err: errInvalidMarketPrice},
		{name: "infinite price", a: asset.Spot, orderType: order.Limit, price: math.Inf(1), err: errInvalidMarketPrice},
		{name: "limit trigger price", a: asset.Spot, orderType: order.Limit, price: 1, triggerPrice: 2, err: errRiskManagementUnsupported},
		{name: "unsupported time in force", a: asset.Spot, orderType: order.Limit, timeInForce: order.FillOrKill, price: 1, err: order.ErrUnsupportedTimeInForce},
		{name: "market trigger price", a: asset.Spot, orderType: order.Market, triggerPrice: 2, slippage: 0.5, err: errRiskManagementUnsupported},
		{name: "zero slippage", a: asset.Spot, orderType: order.Market, err: errSlippageTolerance},
		{name: "whole slippage", a: asset.Spot, orderType: order.Market, slippage: 1, err: errSlippageTolerance},
		{name: "NaN slippage", a: asset.Spot, orderType: order.Market, slippage: math.NaN(), err: errSlippageTolerance},
		{name: "spot trigger", a: asset.Spot, orderType: order.StopMarket, triggerPrice: 1, slippage: 0.5, err: errTriggerOrderPerpetualOnly},
		{name: "NaN trigger price", a: asset.PerpetualContract, orderType: order.StopLimit, price: 1, triggerPrice: math.NaN(), err: errInvalidMarketPrice},
		{name: "missing trigger price", a: asset.PerpetualContract, orderType: order.StopLimit, price: 1, err: errTriggerPriceRequired},
		{name: "trigger market without slippage", a: asset.PerpetualContract, orderType: order.Stop, triggerPrice: 1, err: errSlippageTolerance},
		{name: "unsupported type", a: asset.PerpetualContract, orderType: order.TrailingStop, price: 1, err: order.ErrTypeIsInvalid},
	} {
		err := validateOrderParameters(tc.a, tc.orderType, tc.timeInForce, tc.price, tc.triggerPrice, tc.slippage, tc.clientID)
		assert.ErrorIsf(t, err, tc.err, "validateOrderParameters should return the expected error for a %s", tc.name)
	}
}

func TestIsOutcomeCoin(t *testing.T) {
	t.Parallel()
	for coin, exp := range map[string]bool{
		"#85440":     true,
		"BTC":        false,
		"@107":       false,
		"xyz:XYZ100": false,
	} {
		assert.Equalf(t, exp, isOutcomeCoin(coin), "isOutcomeCoin should report whether %s is an outcome market", coin)
	}
}

func TestIsMarketTriggerOrder(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[order.Type]bool{
		order.Stop:             true,
		order.StopMarket:       true,
		order.TakeProfitMarket: true,
		order.StopLimit:        false,
		order.TakeProfit:       false,
		order.Market:           false,
	} {
		assert.Equalf(t, exp, isMarketTriggerOrder(orderType), "isMarketTriggerOrder should report whether a %s executes at market", orderType)
	}
}

func TestBuildOrderRequests(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, nil, nil)
	submit := &order.Submit{
		Exchange:      ex.Name,
		Type:          order.Limit,
		Side:          order.Buy,
		Pair:          perpetualPair,
		AssetType:     asset.PerpetualContract,
		TimeInForce:   order.GoodTillCancel,
		Amount:        0.1,
		Price:         100,
		ClientOrderID: testClientOrderID,
	}
	parent := OrderRequest{IsBuy: true, Price: 100, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID}
	orders, mapping, grouping, err := ex.buildOrderRequests(t.Context(), submit)
	require.NoError(t, err, "buildOrderRequests must not error")
	assert.Equal(t, []OrderRequest{parent}, orders, "buildOrderRequests should build the order")
	assert.Equal(t, testPerpetualMapping, mapping, "buildOrderRequests should return the market's mapping")
	assert.Equal(t, GroupingNone, grouping, "buildOrderRequests should not group an order without children")

	grouped := *submit
	grouped.RiskManagementModes = order.RiskManagementModes{
		Mode:       GroupingNormalTPSL,
		TakeProfit: order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110, OrderType: order.Market},
		StopLoss:   order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90, LimitPrice: 89, OrderType: order.StopLimit},
	}
	orders, _, grouping, err = ex.buildOrderRequests(t.Context(), &grouped)
	require.NoError(t, err, "buildOrderRequests must not error with children")
	exp := []OrderRequest{
		parent,
		{Price: 99, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 110, TakeProfitStopLoss: TriggerTakeProfit}},
		{Price: 89, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{TriggerPrice: 90, TakeProfitStopLoss: TriggerStopLoss}},
	}
	assert.Equal(t, exp, orders, "buildOrderRequests should add reduce-only children on the opposite side, bounding a market child by the default slippage")
	assert.Equal(t, GroupingNormalTPSL, grouping, "buildOrderRequests should group children with their parent")

	short := grouped
	short.Side = order.Sell
	short.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90, LimitPrice: 91, OrderType: order.TakeProfit}
	short.RiskManagementModes.StopLoss = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110, OrderType: order.Stop}
	orders, _, _, err = ex.buildOrderRequests(t.Context(), &short)
	require.NoError(t, err, "buildOrderRequests must not error for a short parent")
	exp = []OrderRequest{
		{Price: 100, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceGTC}, ClientOrderID: testClientOrderID},
		{IsBuy: true, Price: 91, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{TriggerPrice: 90, TakeProfitStopLoss: TriggerTakeProfit}},
		{IsBuy: true, Price: 121, Size: 0.1, ReduceOnly: true, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 110, TakeProfitStopLoss: TriggerStopLoss}},
	}
	assert.Equal(t, exp, orders, "buildOrderRequests should add buying children to a short parent")

	market := *submit
	market.Type, market.TimeInForce, market.Price = order.Market, order.UnknownTIF, 0
	orders, _, _, err = newTradingServerExchange(t, map[string]string{"allMids": `{"BTC":"100"}`}, nil).buildOrderRequests(t.Context(), &market)
	require.NoError(t, err, "buildOrderRequests must not error for a market order without a slippage tolerance")
	exp = []OrderRequest{{IsBuy: true, Price: 105, Size: 0.1, Limit: &LimitOrderType{TimeInForce: TimeInForceIOC}, ClientOrderID: testClientOrderID}}
	assert.Equal(t, exp, orders, "buildOrderRequests should bound a market order without a slippage tolerance by the default")
	stopMarket := *submit
	stopMarket.Type, stopMarket.Side, stopMarket.TimeInForce, stopMarket.Price, stopMarket.TriggerPrice, stopMarket.TriggerPriceType = order.StopMarket, order.Sell, order.UnknownTIF, 0, 90, order.MarkPrice
	orders, _, _, err = ex.buildOrderRequests(t.Context(), &stopMarket)
	require.NoError(t, err, "buildOrderRequests must not error for a market trigger order without a limit price or slippage tolerance")
	exp = []OrderRequest{{Price: 81, Size: 0.1, Trigger: &TriggerOrderType{IsMarket: true, TriggerPrice: 90, TakeProfitStopLoss: TriggerStopLoss}, ClientOrderID: testClientOrderID}}
	assert.Equal(t, exp, orders, "buildOrderRequests should bound a market trigger order without a limit price or slippage tolerance by the trigger default")

	// An unavailable server fails every request and no market is cached, so these errors prove the submission is
	// checked before the parent's market lookup
	unavailable := newUnavailableServerExchange(t)
	for _, tc := range []struct {
		name   string
		mutate func(*order.Submit)
		err    error
	}{
		{name: "trigger on the last price", mutate: func(s *order.Submit) {
			s.Type, s.TriggerPrice, s.TriggerPriceType, s.SlippageTolerance = order.StopMarket, 90, order.LastPrice, 0.1
		}, err: errRiskManagementUnsupported},
		{name: "spot children", mutate: func(s *order.Submit) {
			s.Pair, s.AssetType = spotPair, asset.Spot
			s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, Price: 11}
		}, err: errRiskManagementUnsupported},
		{name: "child without a trigger price", mutate: func(s *order.Submit) { s.RiskManagementModes.TakeProfit.Enabled = true }, err: errTriggerPriceRequired},
		{name: "child on the index price", mutate: func(s *order.Submit) {
			s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, Price: 110, TriggerPriceType: order.IndexPrice}
		}, err: errRiskManagementUnsupported},
		{name: "whole child slippage", mutate: func(s *order.Submit) {
			s.SlippageTolerance = 1
			s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110}
		}, err: errSlippageTolerance},
	} {
		invalid := *submit
		tc.mutate(&invalid)
		orders, _, _, err := unavailable.buildOrderRequests(t.Context(), &invalid)
		assert.ErrorIsf(t, err, tc.err, "buildOrderRequests should reject a %s before any request", tc.name)
		assert.Nilf(t, orders, "buildOrderRequests should not build a %s", tc.name)
	}

	for _, tc := range []struct {
		name   string
		mutate func(*order.Submit)
		err    error
	}{
		{name: "invalid parent", mutate: func(s *order.Submit) { s.Amount = 0.000001 }, err: errSizePrecision},
		{name: "child price beyond the market's precision", mutate: func(s *order.Submit) {
			s.RiskManagementModes.TakeProfit = order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110, LimitPrice: 100.55, OrderType: order.Limit}
		}, err: errPricePrecision},
	} {
		invalid := *submit
		tc.mutate(&invalid)
		_, _, _, err := ex.buildOrderRequests(t.Context(), &invalid)
		assert.ErrorIsf(t, err, tc.err, "buildOrderRequests should reject a %s", tc.name)
	}
}

func TestRiskManagementChildren(t *testing.T) {
	t.Parallel()
	submit := &order.Submit{Type: order.Limit, Side: order.Buy, Pair: perpetualPair, AssetType: asset.PerpetualContract, Amount: 0.1, Price: 100}
	children, err := riskManagementChildren(submit)
	require.NoError(t, err, "riskManagementChildren must not error without children")
	assert.Empty(t, children, "riskManagementChildren should return no children when none are enabled")

	takeProfit := order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 110}
	stopLoss := order.RiskManagement{Enabled: true, TriggerPriceType: order.MarkPrice, Price: 90, LimitPrice: 89, OrderType: order.Limit}
	submit.RiskManagementModes = order.RiskManagementModes{Mode: GroupingNormalTPSL, TakeProfit: takeProfit, StopLoss: stopLoss}
	children, err = riskManagementChildren(submit)
	require.NoError(t, err, "riskManagementChildren must not error")
	exp := []riskManagementChild{
		{settings: takeProfit, takeProfit: true, orderType: order.TakeProfitMarket, slippage: defaultTriggerSlippage},
		{settings: stopLoss, orderType: order.StopLimit},
	}
	assert.Equal(t, exp, children, "riskManagementChildren should return the take profit first, bounding its market order by the default slippage")
	submit.SlippageTolerance = 0.05
	submit.RiskManagementModes.StopLoss = order.RiskManagement{}
	children, err = riskManagementChildren(submit)
	require.NoError(t, err, "riskManagementChildren must not error with a slippage tolerance")
	exp = []riskManagementChild{{settings: takeProfit, takeProfit: true, orderType: order.TakeProfitMarket, slippage: 0.05}}
	assert.Equal(t, exp, children, "riskManagementChildren should bound a market child by the submission's slippage tolerance")

	for _, tc := range []struct {
		name   string
		mutate func(*order.Submit)
		err    error
	}{
		{name: "spot parent", mutate: func(s *order.Submit) { s.AssetType = asset.Spot }, err: errRiskManagementUnsupported},
		{name: "stop entry", mutate: func(s *order.Submit) { s.RiskManagementModes.StopEntry.Enabled = true }, err: errRiskManagementUnsupported},
		{name: "position grouping", mutate: func(s *order.Submit) { s.RiskManagementModes.Mode = GroupingPositionTPSL }, err: errRiskManagementUnsupported},
		{name: "child without a trigger price", mutate: func(s *order.Submit) { s.RiskManagementModes.TakeProfit.Price = 0 }, err: errTriggerPriceRequired},
		{name: "child on the last price", mutate: func(s *order.Submit) { s.RiskManagementModes.TakeProfit.TriggerPriceType = order.LastPrice }, err: errRiskManagementUnsupported},
		{name: "stop take profit", mutate: func(s *order.Submit) { s.RiskManagementModes.TakeProfit.OrderType = order.Stop }, err: errRiskManagementUnsupported},
		{name: "NaN child limit price", mutate: func(s *order.Submit) { s.RiskManagementModes.TakeProfit.LimitPrice = math.NaN() }, err: errInvalidMarketPrice},
		{name: "whole child slippage", mutate: func(s *order.Submit) { s.SlippageTolerance = 1 }, err: errSlippageTolerance},
	} {
		invalid := *submit
		tc.mutate(&invalid)
		_, err := riskManagementChildren(&invalid)
		assert.ErrorIsf(t, err, tc.err, "riskManagementChildren should reject a %s", tc.name)
	}
}

func TestOrderSlippage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		price     float64
		slippage  float64
		exp       float64
	}{
		{orderType: order.Market, slippage: 0.2, exp: 0.2},
		{orderType: order.Market, slippage: -0.1, exp: -0.1},
		{orderType: order.Market, exp: defaultMarketSlippage},
		{orderType: order.StopMarket, exp: defaultTriggerSlippage},
		{orderType: order.TakeProfitMarket, slippage: 0.2, exp: 0.2},
		{orderType: order.Stop, price: 80},
		{orderType: order.StopLimit},
		{orderType: order.Limit, price: 100},
	} {
		assert.Equalf(t, tc.exp, orderSlippage(tc.orderType, tc.price, tc.slippage), "orderSlippage should return the slippage tolerance of a %s priced %v with %v", tc.orderType, tc.price, tc.slippage)
	}
}

func TestRiskManagementOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType  order.Type
		takeProfit bool
		exp        order.Type
		err        error
	}{
		{orderType: order.UnknownType, takeProfit: true, exp: order.TakeProfitMarket},
		{orderType: order.Market, takeProfit: true, exp: order.TakeProfitMarket},
		{orderType: order.TakeProfitMarket, takeProfit: true, exp: order.TakeProfitMarket},
		{orderType: order.Limit, takeProfit: true, exp: order.TakeProfit},
		{orderType: order.TakeProfit, takeProfit: true, exp: order.TakeProfit},
		{orderType: order.Stop, takeProfit: true, err: errRiskManagementUnsupported},
		{orderType: order.UnknownType, exp: order.StopMarket},
		{orderType: order.Market, exp: order.StopMarket},
		{orderType: order.Stop, exp: order.StopMarket},
		{orderType: order.StopMarket, exp: order.StopMarket},
		{orderType: order.Limit, exp: order.StopLimit},
		{orderType: order.StopLimit, exp: order.StopLimit},
		{orderType: order.TakeProfit, err: errRiskManagementUnsupported},
	} {
		result, err := riskManagementOrderType(tc.orderType, tc.takeProfit)
		require.ErrorIsf(t, err, tc.err, "riskManagementOrderType must return the expected error for %s take profit %t", tc.orderType, tc.takeProfit)
		assert.Equalf(t, tc.exp, result, "riskManagementOrderType should map %s take profit %t", tc.orderType, tc.takeProfit)
	}
}

func TestConvertOrder(t *testing.T) {
	t.Parallel()
	ex := newTradingServerExchange(t, nil, nil)
	source := testOpenOrder()
	result, err := ex.convertOrder(t.Context(), &source, "open", time.UnixMilli(1791275930650))
	require.NoError(t, err, "convertOrder must not error")
	exp := order.Detail{
		TimeInForce:     order.GoodTillCancel,
		Price:           85000,
		Amount:          0.002,
		ExecutedAmount:  0.001,
		RemainingAmount: 0.001,
		Exchange:        ex.Name,
		OrderID:         strconv.FormatUint(testOrderID, 10),
		ClientOrderID:   testClientOrderID,
		Type:            order.Limit,
		Side:            order.Buy,
		Status:          order.PartiallyFilled,
		AssetType:       asset.PerpetualContract,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     time.UnixMilli(1791275930650).UTC(),
		Pair:            perpetualPair,
	}
	assert.Equal(t, exp, result, "convertOrder should convert the order")
	source.Coin = "MISSING"
	_, err = ex.convertOrder(t.Context(), &source, "open", time.Time{})
	assert.ErrorIs(t, err, errPairMappingNotFound, "convertOrder should report an order without a market")
}

func TestConvertOrderFromMapping(t *testing.T) {
	t.Parallel()
	child := testOpenOrder().Children[0]
	result, err := e.convertOrderFromMapping(&child, "triggered", time.Time{}, &testPerpetualMapping, asset.PerpetualContract)
	require.NoError(t, err, "convertOrderFromMapping must not error")
	exp := order.Detail{
		TimeInForce:     order.GoodTillCancel,
		ReduceOnly:      true,
		Price:           90000,
		Amount:          0.001,
		TriggerPrice:    90000,
		RemainingAmount: 0.001,
		Exchange:        e.Name,
		OrderID:         "566563303536",
		Type:            order.TakeProfitMarket,
		Side:            order.Sell,
		Status:          order.Open,
		AssetType:       asset.PerpetualContract,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     time.UnixMilli(1791275926947).UTC(),
		Pair:            perpetualPair,
	}
	assert.Equal(t, exp, result, "convertOrderFromMapping should convert a triggered order as open, as a triggered limit order rests")

	// Once triggered and filled, Hyperliquid clears the order's trigger flag and price but keeps its type, as live
	filled := child
	filled.Size, filled.IsTrigger, filled.TriggerPrice, filled.TriggerCondition, filled.TimeInForce = 0, false, 0, "Triggered", TimeInForceGTC
	result, err = e.convertOrderFromMapping(&filled, "filled", time.UnixMilli(1791275930650), &testPerpetualMapping, asset.PerpetualContract)
	require.NoError(t, err, "convertOrderFromMapping must not error for a filled trigger order")
	exp = order.Detail{
		TimeInForce:    order.GoodTillCancel,
		ReduceOnly:     true,
		Price:          90000,
		Amount:         0.001,
		ExecutedAmount: 0.001,
		Exchange:       e.Name,
		OrderID:        "566563303536",
		Type:           order.TakeProfitMarket,
		Side:           order.Sell,
		Status:         order.Filled,
		AssetType:      asset.PerpetualContract,
		Date:           time.UnixMilli(1791275926947).UTC(),
		LastUpdated:    time.UnixMilli(1791275930650).UTC(),
		Pair:           perpetualPair,
	}
	assert.Equal(t, exp, result, "convertOrderFromMapping should classify a filled trigger order by its type, whatever its trigger flag")

	for _, tc := range []struct {
		mutate func(*FrontendOpenOrderResponse)
		status string
		err    error
	}{
		{mutate: func(o *FrontendOpenOrderResponse) { o.OrderType = "unknown" }, status: "open", err: order.ErrTypeIsInvalid},
		{mutate: func(o *FrontendOpenOrderResponse) { o.TimeInForce = "bad" }, status: "open", err: order.ErrInvalidTimeInForce},
		{mutate: func(o *FrontendOpenOrderResponse) { o.Side = "X" }, status: "open", err: order.ErrSideIsInvalid},
		{mutate: func(*FrontendOpenOrderResponse) {}, status: "unknown", err: errUnsupportedOrderStatus},
	} {
		invalid := testOpenOrder()
		tc.mutate(&invalid)
		_, err := e.convertOrderFromMapping(&invalid, tc.status, time.Time{}, &testPerpetualMapping, asset.PerpetualContract)
		assert.ErrorIsf(t, err, tc.err, "convertOrderFromMapping should reject %+v with status %s", invalid, tc.status)
	}
}

func TestConvertWebsocketOrder(t *testing.T) {
	t.Parallel()
	ex := new(Exchange)
	ex.Name = t.Name()
	update := &WebsocketOrder{Order: testOpenOrder().BasicOrderResponse, Status: "canceled", StatusTimestamp: milli(1791275930650)}
	_, err := ex.convertWsOrder(update)
	require.ErrorIs(t, err, errPairMappingNotFound, "convertWsOrder must report an order without a cached market")
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{testPerpetualMapping})
	result, err := ex.convertWsOrder(update)
	require.NoError(t, err, "convertWsOrder must not error")
	exp := order.Detail{
		Price:           85000,
		Amount:          0.002,
		ExecutedAmount:  0.001,
		RemainingAmount: 0.001,
		Exchange:        ex.Name,
		OrderID:         strconv.FormatUint(testOrderID, 10),
		ClientOrderID:   testClientOrderID,
		Side:            order.Buy,
		Status:          order.PartiallyFilledCancelled,
		AssetType:       asset.PerpetualContract,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     time.UnixMilli(1791275930650).UTC(),
		Pair:            perpetualPair,
	}
	assert.Equal(t, exp, result, "convertWsOrder should convert the update without an order type or time in force")
}

func TestConvertBasicOrder(t *testing.T) {
	t.Parallel()
	source := BasicOrderResponse{Coin: "BTC", Side: "A", LimitPrice: 100, Size: 1, OrderID: 7, Timestamp: milli(1791275926947)}
	result, err := e.convertBasicOrder(&source, "filled", time.Time{}, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "convertBasicOrder must not error")
	exp := order.Detail{
		Price:           100,
		Amount:          1,
		RemainingAmount: 1,
		Exchange:        e.Name,
		OrderID:         "7",
		Side:            order.Sell,
		Status:          order.Filled,
		AssetType:       asset.PerpetualContract,
		Date:            time.UnixMilli(1791275926947).UTC(),
		LastUpdated:     time.UnixMilli(1791275926947).UTC(),
		Pair:            perpetualPair,
	}
	assert.Equal(t, exp, result, "convertBasicOrder should use the remaining size as the amount without an original size, and the placement time as the update time")
	source.OriginalSize = 2
	for status, exp := range map[string]order.Status{
		"open":           order.PartiallyFilled,
		"canceled":       order.PartiallyFilledCancelled,
		"marginCanceled": order.PartiallyFilledCancelled,
		"filled":         order.PartiallyFilledCancelled,
		"rejected":       order.Rejected,
	} {
		result, err := e.convertBasicOrder(&source, status, time.Time{}, perpetualPair, asset.PerpetualContract)
		require.NoErrorf(t, err, "convertBasicOrder must not error for status %s", status)
		assert.Equalf(t, exp, result.Status, "convertBasicOrder should report a %s order with an executed size as %s", status, exp)
	}
	source.Size = 0
	fullyFilled, err := e.convertBasicOrder(&source, "filled", time.Time{}, perpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "convertBasicOrder must not error for a fully filled order")
	assert.Equal(t, order.Filled, fullyFilled.Status, "convertBasicOrder should report an order that executed its whole size as filled")
	source.Size = 1
	source.OriginalSize = 0
	source.Side = "X"
	_, err = e.convertBasicOrder(&source, "open", time.Time{}, perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "convertBasicOrder should reject an invalid side")
	source.Side = "B"
	_, err = e.convertBasicOrder(&source, "unknown", time.Time{}, perpetualPair, asset.PerpetualContract)
	assert.ErrorIs(t, err, errUnsupportedOrderStatus, "convertBasicOrder should reject an unknown status")
}

func TestPerpetualTickerPrice(t *testing.T) {
	t.Parallel()
	market := &PerpetualAssetContext{Funding: 0.0001, OpenInterest: 10, PreviousDayPrice: 99, DayNotionalVolume: 1000, OraclePrice: 100, MarkPrice: 101, MidPrice: 100.5, DayBaseVolume: 10}
	result := e.perpetualTickerPrice(perpetualPair, market)
	exp := &ticker.Price{Last: 100.5, Open: 99, BaseVolume: 10, QuoteVolume: 1000, OpenInterest: 10, MarkPrice: 101, IndexPrice: 100, Pair: perpetualPair, ExchangeName: e.Name, AssetType: asset.PerpetualContract, LastUpdated: result.LastUpdated}
	assert.Equal(t, exp, result, "perpetualTickerPrice should use the mid price as the last price")
	market.MidPrice = 0
	assert.Equal(t, 101.0, e.perpetualTickerPrice(perpetualPair, market).Last, "perpetualTickerPrice should use the mark price without a mid price")
}

func TestSpotTickerPrice(t *testing.T) {
	t.Parallel()
	market := &SpotAssetContext{PreviousDayPrice: 9, DayNotionalVolume: 100, MarkPrice: 10, MidPrice: 9.5, Coin: "@107", DayBaseVolume: 10}
	result := e.spotTickerPrice(spotPair, market)
	exp := &ticker.Price{Last: 9.5, Open: 9, BaseVolume: 10, QuoteVolume: 100, MarkPrice: 10, Pair: spotPair, ExchangeName: e.Name, AssetType: asset.Spot, LastUpdated: result.LastUpdated}
	assert.Equal(t, exp, result, "spotTickerPrice should use the mid price as the last price")
	market.MidPrice = 0
	assert.Equal(t, 10.0, e.spotTickerPrice(spotPair, market).Last, "spotTickerPrice should use the mark price without a mid price")
}

func TestParseSide(t *testing.T) {
	t.Parallel()
	for side, exp := range map[string]order.Side{"A": order.Sell, "B": order.Buy} {
		result, err := parseSide(side)
		require.NoErrorf(t, err, "parseSide must not error for %s", side)
		assert.Equalf(t, exp, result, "parseSide should parse %s", side)
	}
	_, err := parseSide("b")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "parseSide should reject an unknown side")
}

func TestParseOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"open":                    order.Open,
		"filled":                  order.Filled,
		"triggered":               order.Open,
		"canceled":                order.Cancelled,
		"scheduledCancel":         order.Cancelled,
		"marginCanceled":          order.Cancelled,
		"liquidatedCanceled":      order.Cancelled,
		"rejected":                order.Rejected,
		"perpMarginRejected":      order.Rejected,
		"perpMaxPositionRejected": order.Rejected,
	} {
		result, err := parseOrderStatus(status)
		require.NoErrorf(t, err, "parseOrderStatus must not error for %s", status)
		assert.Equalf(t, exp, result, "parseOrderStatus should parse %s", status)
	}
	_, err := parseOrderStatus("unknown")
	assert.ErrorIs(t, err, errUnsupportedOrderStatus, "parseOrderStatus should reject an unknown status")
}

func TestExecutionStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		status              order.Status
		executed, remaining float64
		exp                 order.Status
	}{
		{status: order.Open, remaining: 0.5, exp: order.Open},
		{status: order.Open, executed: 0.25, remaining: 0.25, exp: order.PartiallyFilled},
		{status: order.Cancelled, remaining: 0.5, exp: order.Cancelled},
		{status: order.Cancelled, executed: 0.25, remaining: 0.25, exp: order.PartiallyFilledCancelled},
		{status: order.Cancelled, executed: 0.25, exp: order.PartiallyFilledCancelled},
		{status: order.Filled, executed: 0.5, exp: order.Filled},
		// A GTC order whose rest is under the minimum order value ends filled with that rest as its size
		{status: order.Filled, executed: 0.49998, remaining: 0.00002, exp: order.PartiallyFilledCancelled},
		{status: order.Rejected, executed: 0.25, remaining: 0.25, exp: order.Rejected},
	} {
		assert.Equalf(t, tc.exp, executionStatus(tc.status, tc.executed, tc.remaining), "executionStatus should return %s for a %s order with %v executed and %v left", tc.exp, tc.status, tc.executed, tc.remaining)
	}
}

func TestParseOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[string]order.Type{
		"Limit":              order.Limit,
		"Market":             order.Market,
		"Stop Limit":         order.StopLimit,
		"Stop Market":        order.StopMarket,
		"Stop":               order.Stop,
		"Take Profit Limit":  order.TakeProfit,
		"Take Profit Market": order.TakeProfitMarket,
		"Take Limit":         order.TakeProfit,
		"Take Market":        order.TakeProfitMarket,
	} {
		result, err := parseOrderType(orderType)
		require.NoErrorf(t, err, "parseOrderType must not error for %s", orderType)
		assert.Equalf(t, exp, result, "parseOrderType should parse %s", orderType)
	}
	_, err := parseOrderType("unknown")
	assert.ErrorIs(t, err, order.ErrTypeIsInvalid, "parseOrderType should reject an unknown type")
}

func TestParseTimeInForce(t *testing.T) {
	t.Parallel()
	for timeInForce, exp := range map[string]order.TimeInForce{
		"":                  order.GoodTillCancel,
		"Gtc":               order.GoodTillCancel,
		"Alo":               order.PostOnly,
		"Ioc":               order.ImmediateOrCancel,
		"FrontendMarket":    order.ImmediateOrCancel,
		"LiquidationMarket": order.ImmediateOrCancel,
	} {
		result, err := parseTimeInForce(timeInForce)
		require.NoErrorf(t, err, "parseTimeInForce must not error for %q", timeInForce)
		assert.Equalf(t, exp, result, "parseTimeInForce should parse %q", timeInForce)
	}
	_, err := parseTimeInForce("bad")
	assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "parseTimeInForce should reject an unknown time in force")
}

func TestFormatOrderTimeInForce(t *testing.T) {
	t.Parallel()
	for timeInForce, exp := range map[order.TimeInForce]string{
		order.UnknownTIF:                      TimeInForceGTC,
		order.GoodTillCancel:                  TimeInForceGTC,
		order.PostOnly:                        TimeInForceALO,
		order.GoodTillCancel | order.PostOnly: TimeInForceALO,
		order.ImmediateOrCancel:               TimeInForceIOC,
	} {
		result, err := formatOrderTimeInForce(timeInForce)
		require.NoErrorf(t, err, "formatOrderTimeInForce must not error for %s", timeInForce)
		assert.Equalf(t, exp, result, "formatOrderTimeInForce should format %s", timeInForce)
	}
	_, err := formatOrderTimeInForce(order.FillOrKill)
	assert.ErrorIs(t, err, order.ErrUnsupportedTimeInForce, "formatOrderTimeInForce should reject fill or kill")
}

func TestValidateOrderSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		size     float64
		decimals uint64
		err      error
	}{
		{size: 2},
		{size: 1.23, decimals: 2},
		{size: 0.00000001, decimals: 8},
		{size: 0, decimals: 2, err: errSizePrecision},
		{size: -1, decimals: 2, err: errSizePrecision},
		{size: math.NaN(), decimals: 2, err: errSizePrecision},
		{size: math.Inf(1), decimals: 2, err: errSizePrecision},
		{size: 1, decimals: 9, err: errSizePrecision},
		{size: 1.234, decimals: 2, err: errSizePrecision},
	} {
		assert.ErrorIsf(t, validateOrderSize(tc.size, tc.decimals), tc.err, "validateOrderSize should return the expected error for %v with %d decimals", tc.size, tc.decimals)
	}
}

func TestDeriveFilledOrderState(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		requested, filled float64
		decimals          uint64
		timeInForce       string
		status            order.Status
		remaining         float64
		err               error
	}{
		{requested: 0.1, filled: 0.1, decimals: 5, timeInForce: TimeInForceGTC, status: order.Filled},
		{requested: 0.1, filled: 0.04, decimals: 5, timeInForce: TimeInForceGTC, status: order.PartiallyFilledCancelled, remaining: 0.06},
		{requested: 0.1, filled: 0.04, decimals: 5, timeInForce: TimeInForceIOC, status: order.PartiallyFilledCancelled, remaining: 0.06},
		{requested: 0.1, filled: 0.04, decimals: 5, timeInForce: TimeInForceALO, err: errActionStatusMalformed},
		{requested: 0.1, filled: 0.04, decimals: 5, err: errActionStatusMalformed},
		{requested: 0, filled: 0.04, decimals: 5, timeInForce: TimeInForceGTC, err: errInvalidFilledSize},
		{requested: 0.1, filled: 0.0400001, decimals: 5, timeInForce: TimeInForceGTC, err: errInvalidFilledSize},
		{requested: 0.1, filled: 0.04, decimals: 9, timeInForce: TimeInForceGTC, err: errInvalidFilledSize},
		{requested: 0.1, filled: 0.11, decimals: 5, timeInForce: TimeInForceGTC, err: errInvalidFilledSize},
	} {
		status, remaining, err := deriveFilledOrderState(tc.requested, tc.filled, tc.decimals, tc.timeInForce)
		require.ErrorIsf(t, err, tc.err, "deriveFilledOrderState must return the expected error for %v of %v %q", tc.filled, tc.requested, tc.timeInForce)
		assert.Equalf(t, tc.status, status, "deriveFilledOrderState should derive the status of %v of %v %q", tc.filled, tc.requested, tc.timeInForce)
		assert.InDeltaf(t, tc.remaining, remaining, 1e-12, "deriveFilledOrderState should derive the remainder of %v of %v %q", tc.filled, tc.requested, tc.timeInForce)
	}
}

func TestPriceDecimalBase(t *testing.T) {
	t.Parallel()
	for a, exp := range map[asset.Item]uint64{asset.Spot: spotPriceDecimalBase, asset.PerpetualContract: perpetualPriceDecimalBase} {
		result, err := priceDecimalBase(a, 2)
		require.NoErrorf(t, err, "priceDecimalBase must not error for %s", a)
		assert.Equalf(t, exp, result, "priceDecimalBase should return the %s decimal base", a)
	}
	_, err := priceDecimalBase(asset.Options, 0)
	assert.ErrorIs(t, err, asset.ErrNotSupported, "priceDecimalBase should reject an unsupported asset")
	_, err = priceDecimalBase(asset.PerpetualContract, 7)
	assert.ErrorIs(t, err, errSizePrecision, "priceDecimalBase should reject size decimals above the decimal base")
}

func TestValidateLimitPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		price    float64
		a        asset.Item
		decimals uint64
		err      error
	}{
		{price: 1234.5, a: asset.PerpetualContract, decimals: 5},
		{price: 0.001234, a: asset.Spot, decimals: 2},
		{price: 123456789, a: asset.PerpetualContract, decimals: 5},
		{a: asset.Spot, err: errInvalidMarketPrice},
		{price: math.NaN(), a: asset.Spot, err: errInvalidMarketPrice},
		{price: math.Inf(1), a: asset.Spot, err: errInvalidMarketPrice},
		{price: 1, a: asset.Options, err: asset.ErrNotSupported},
		{price: 1, a: asset.PerpetualContract, decimals: 7, err: errSizePrecision},
		{price: 100.55, a: asset.PerpetualContract, decimals: 5, err: errPricePrecision},
		{price: 12.3456, a: asset.Spot, decimals: 2, err: errPricePrecision},
		{price: 0.000000001, a: asset.Spot, err: errWireNumberRounding},
	} {
		assert.ErrorIsf(t, validateLimitPrice(tc.price, tc.a, tc.decimals), tc.err, "validateLimitPrice should return the expected error for %v %s with %d size decimals", tc.price, tc.a, tc.decimals)
	}
}

func TestRoundMarketPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		price    float64
		side     order.Side
		a        asset.Item
		decimals uint64
		exp      float64
		err      error
	}{
		{price: 123.456789, side: order.Buy, a: asset.Spot, decimals: 2, exp: 123.45},
		{price: 123.456789, side: order.Sell, a: asset.Spot, decimals: 2, exp: 123.46},
		{price: 123.456789, side: order.Buy, a: asset.PerpetualContract, decimals: 5, exp: 123.4},
		{price: 123.456789, side: order.Sell, a: asset.PerpetualContract, decimals: 5, exp: 123.5},
		{price: 12345.67, side: order.Buy, a: asset.Spot, exp: 12345},
		{price: 12345.67, side: order.Sell, a: asset.Spot, exp: 12346},
		{price: 123456.7, side: order.Buy, a: asset.Spot, exp: 123456},
		{price: 123456.7, side: order.Sell, a: asset.Spot, exp: 123457},
		{price: 9.99995, side: order.Buy, a: asset.Spot, decimals: 2, exp: 9.9999},
		{price: 9.99995, side: order.Sell, a: asset.Spot, decimals: 2, exp: 10},
		{price: 0.000012345678, side: order.Buy, a: asset.Spot, exp: 0.00001234},
		{price: 0.000012345678, side: order.Sell, a: asset.Spot, exp: 0.00001235},
		{price: math.Nextafter(105, 0), side: order.Buy, a: asset.PerpetualContract, decimals: 5, exp: 105},
		{price: math.Nextafter(99, 100), side: order.Sell, a: asset.PerpetualContract, decimals: 5, exp: 99},
		{price: 99.99999, side: order.Buy, a: asset.PerpetualContract, decimals: 5, exp: 99.9},
		{price: 99.00001, side: order.Sell, a: asset.PerpetualContract, decimals: 5, exp: 99.1},
		{price: math.Nextafter(1e15, 0), side: order.Sell, a: asset.Spot, exp: 1e15},
		{price: 1000000000000.75, side: order.Buy, a: asset.Spot, exp: 1000000000000},
		{price: 602.9999999999999, side: order.Buy, a: asset.Spot, decimals: 2, exp: 603},
		{price: 0.009999999999999992, side: order.Buy, a: asset.Spot, exp: 0.0099999},
		{price: 1000000000000.25, side: order.Sell, a: asset.Spot, exp: 1000000000001},
		{price: 0.4, side: order.Sell, a: asset.PerpetualContract, decimals: 6, exp: 1},
		{price: 0.4, side: order.Buy, a: asset.PerpetualContract, decimals: 6, err: errInvalidMarketPrice},
		{side: order.Buy, a: asset.Spot, err: errInvalidMarketPrice},
		{price: math.NaN(), side: order.Buy, a: asset.Spot, err: errInvalidMarketPrice},
		{price: math.Inf(1), side: order.Sell, a: asset.Spot, err: errInvalidMarketPrice},
		{price: 1, side: order.Buy, a: asset.Options, err: asset.ErrNotSupported},
		{price: 1, side: order.Buy, a: asset.PerpetualContract, decimals: 7, err: errSizePrecision},
	} {
		result, err := roundMarketPrice(tc.price, tc.side, tc.a, tc.decimals)
		require.ErrorIsf(t, err, tc.err, "roundMarketPrice must return the expected error for a %s at %v %s with %d size decimals", tc.side, tc.price, tc.a, tc.decimals)
		assert.Equalf(t, tc.exp, result, "roundMarketPrice should round a %s at %v %s with %d size decimals", tc.side, tc.price, tc.a, tc.decimals)
	}

	// Each rounded price must be valid, within its bound, and the valid price nearest the bound on the bound's side
	random := rand.New(rand.NewPCG(2303, 1)) //nolint:gosec // A fixed sequence of test prices needs no cryptographic randomness
	for range 20000 {
		a, decimalBase := asset.PerpetualContract, uint64(perpetualPriceDecimalBase)
		if random.IntN(2) == 0 {
			a, decimalBase = asset.Spot, spotPriceDecimalBase
		}
		sizeDecimals := random.Uint64N(decimalBase + 1)
		side := order.Buy
		if random.IntN(2) == 0 {
			side = order.Sell
		}
		// Rounding each product first keeps an architecture that fuses multiplies and adds to the same prices
		bound := slippagePrice(math.Pow(10, float64(random.Float64()*20)-6), float64(random.Float64()*0.99)+0.001, side)
		// roundMarketPrice absorbs four units in the last place of the bound's tick count, at most eight of the bound's own
		slack := 8 * (math.Nextafter(bound, math.Inf(1)) - bound)
		result, err := roundMarketPrice(bound, side, a, sizeDecimals)
		if side == order.Buy && errors.Is(err, errInvalidMarketPrice) {
			require.Greaterf(t, math.Pow10(-int(decimalBase-sizeDecimals)), bound, "roundMarketPrice must only refuse a buy bounded below the smallest price, got %v %s with %d size decimals", bound, a, sizeDecimals)
			continue
		}
		require.NoErrorf(t, err, "roundMarketPrice must not error for a %s at %v %s with %d size decimals", side, bound, a, sizeDecimals)
		require.NoErrorf(t, validateLimitPrice(result, a, sizeDecimals), "roundMarketPrice must return a valid price for a %s at %v %s with %d size decimals, got %v", side, bound, a, sizeDecimals, result)
		beyond := nextValidPrice(t, result, side.IsLong(), a, sizeDecimals)
		if side == order.Buy {
			require.LessOrEqualf(t, result, bound+slack, "roundMarketPrice must not raise a buy above %v %s with %d size decimals", bound, a, sizeDecimals)
			require.Greaterf(t, beyond, bound-slack, "roundMarketPrice must round a buy at %v %s with %d size decimals to the highest valid price within it, got %v", bound, a, sizeDecimals, result)
		} else {
			require.GreaterOrEqualf(t, result, bound-slack, "roundMarketPrice must not lower a sell below %v %s with %d size decimals", bound, a, sizeDecimals)
			require.Lessf(t, beyond, bound+slack, "roundMarketPrice must round a sell at %v %s with %d size decimals to the lowest valid price within it, got %v", bound, a, sizeDecimals, result)
		}
	}
}

// nextValidPrice returns the valid price next to a valid price, above it or below it, found as the smallest power of ten
// step that validateLimitPrice accepts, or zero when no valid price lies below it
func nextValidPrice(t *testing.T, price float64, above bool, a asset.Item, sizeDecimals uint64) float64 {
	t.Helper()
	for exponent := -8; exponent <= 20; exponent++ {
		next := price - math.Pow10(exponent)
		if above {
			next = price + math.Pow10(exponent)
		}
		if next <= 0 {
			return 0
		}
		if next != price && validateLimitPrice(next, a, sizeDecimals) == nil {
			return next
		}
	}
	require.FailNowf(t, "no valid price found", "nextValidPrice must find a valid price next to %v", price)
	return 0
}

func TestSlippagePrice(t *testing.T) {
	t.Parallel()
	assert.InDelta(t, 101.0, slippagePrice(100, 0.01, order.Buy), 1e-12, "slippagePrice should raise a buy's price")
	assert.InDelta(t, 99.0, slippagePrice(100, 0.01, order.Sell), 1e-12, "slippagePrice should lower a sell's price")
}
