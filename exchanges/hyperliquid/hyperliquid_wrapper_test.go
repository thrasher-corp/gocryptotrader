package hyperliquid

import (
	"context"
	"fmt"
	"math"
	"net/http"
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
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fundingrate"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/portfolio/withdraw"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	testPerpetualPair = currency.NewPairWithDelimiter("BTC", "USDC", "-")
	testSpotPair      = currency.NewPairWithDelimiter("HYPE", "USDC", "-")
)

func setTestPair(t *testing.T, ex *Exchange, a asset.Item, pair currency.Pair, coin string) {
	t.Helper()
	ex.setPairMappings(a, []pairMapping{{pair: pair, coin: coin}})
	require.NoError(t, ex.UpdatePairs(currency.Pairs{pair}, a, false), "Updating available test pairs must not error")
	require.NoError(t, ex.UpdatePairs(currency.Pairs{pair}, a, true), "Updating enabled test pairs must not error")
}

func TestLogDefaultError(t *testing.T) {
	assert.NotPanics(t, func() { logDefaultError(nil) }, "Logging a nil default error should not panic")
	assert.NotPanics(t, func() { logDefaultError(assert.AnError) }, "Logging a default error should not panic")
}

func TestSetDefaults(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	assert.Equal(t, "Hyperliquid", ex.Name, "Exchange name should be configured")
	assert.True(t, ex.Enabled, "Exchange should be enabled by default")
	assert.True(t, ex.Features.Supports.REST, "REST should be supported")
	assert.True(t, ex.Features.Supports.Websocket, "Websocket should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.AuthenticatedEndpoints, "Authenticated REST endpoints should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.SubmitOrder, "REST order submission should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.TradeFee, "Account-specific trade fees should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.CryptoWithdrawal, "USDC bridge withdrawals should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.DepositHistory, "Account deposit history should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.WithdrawalHistory, "Bridge withdrawal history should be supported")
	assert.True(t, ex.HasAssetTypeAccountSegregation(), "Separate spot and perpetual balance pools should report asset type account segregation")
	assert.Equal(t, exchange.AutoWithdrawCryptoWithSetup, ex.Features.Supports.WithdrawPermissions, "Withdrawal permissions should require master-wallet setup")
	assert.True(t, ex.Features.Supports.WebsocketCapabilities.AuthenticatedEndpoints, "Address-scoped websocket feeds should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.TickerBatching, "Ticker batching should be supported")
	assert.True(t, ex.Features.Supports.RESTCapabilities.AutoPairUpdates, "Automatic pair updates should be supported")
	assert.True(t, ex.Features.Supports.FuturesCapabilities.FundingRates, "Perpetual funding rates should be supported")
	assert.True(t, ex.Features.Supports.FuturesCapabilities.Leverage, "Perpetual leverage readback should be supported")
	assert.True(t, ex.Features.Supports.FuturesCapabilities.OpenInterest.Supported, "Perpetual open interest should be supported")
	assert.Equal(t, uint64(maximumCandleCount), ex.Features.Enabled.Kline.GlobalResultLimit, "Candle result limit should match Hyperliquid retention")
	assert.NotNil(t, ex.Requester, "REST requester should be initialised")
	assert.NotNil(t, ex.Websocket, "Websocket manager should be initialised")
	assert.NotNil(t, ex.pairMappings, "Pair mappings should be initialised")
	assert.NotNil(t, ex.pairMappingMisses, "Pair mapping miss cache should be initialised")
	spotURL, err := ex.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "Getting the default spot URL must not error")
	futuresURL, err := ex.API.Endpoints.GetURL(exchange.RestFutures)
	require.NoError(t, err, "Getting the default futures URL must not error")
	assert.Equal(t, hyperliquidAPIURL, spotURL, "Spot URL should use the Hyperliquid API")
	assert.Equal(t, spotURL, futuresURL, "Spot and futures should share the Hyperliquid API URL")
	websocketURL, err := ex.API.Endpoints.GetURL(exchange.WebsocketSpot)
	require.NoError(t, err, "Getting the default websocket URL must not error")
	assert.Equal(t, hyperliquidWebsocketURL, websocketURL, "Websocket URL should use the Hyperliquid stream")
	for _, a := range []asset.Item{asset.Spot, asset.PerpetualContract} {
		format, err := ex.GetPairFormat(a, true)
		require.NoError(t, err, "Getting the request pair format must not error")
		assert.True(t, format.Uppercase, "Request pair format should be uppercase")
		assert.Equal(t, currency.DashDelimiter, format.Delimiter, "Request pair format should use a dash")
	}
	require.NoError(t, ex.Shutdown(), "Shutting down the default exchange must not error")
}

func TestConfigSetup(t *testing.T) {
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Loading Hyperliquid from configtest must not error")
	assert.True(t, ex.SupportsAsset(asset.Spot), "Configtest should enable spot markets")
	assert.True(t, ex.SupportsAsset(asset.PerpetualContract), "Configtest should enable perpetual markets")
	require.NoError(t, ex.Shutdown(), "Shutting down the configured exchange must not error")
}

func TestSetPairMappings(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	mappings := []pairMapping{{pair: testSpotPair, coin: "@107"}}
	ex.setPairMappings(asset.Spot, mappings)
	mappings[0].coin = "changed"
	coin, err := ex.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting a cloned pair mapping must not error")
	assert.Equal(t, "@107", coin, "Stored pair mapping in an initialised map should not alias the source slice")

	ex.pairMappings = nil
	mappings[0].coin = "@108"
	ex.setPairMappings(asset.Spot, mappings)
	mappings[0].coin = "changed again"
	coin, err = ex.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting a cloned pair mapping from a newly initialised map must not error")
	assert.Equal(t, "@108", coin, "Stored pair mapping in a newly initialised map should not alias the source slice")
	require.NoError(t, ex.Shutdown(), "Shutting down the exchange must not error")
}

func TestLookupPairMapping(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, ok := ex.lookupPairMapping(testSpotPair, asset.Spot)
	assert.False(t, ok, "Missing pair mapping should not be found")

	expected := pairMapping{pair: testSpotPair, coin: "@107"}
	ex.setPairMappings(asset.Spot, []pairMapping{expected})
	result, ok := ex.lookupPairMapping(testSpotPair, asset.Spot)
	require.True(t, ok, "Cached pair mapping must be found")
	assert.Equal(t, expected, result, "Cached pair mapping should match")
	require.NoError(t, ex.Shutdown(), "Shutting down the lookup exchange must not error")
}

func TestFetchPairMapping(t *testing.T) {
	cached := new(Exchange)
	cached.SetDefaults()
	expected := pairMapping{pair: testPerpetualPair, coin: "BTC"}
	cached.setPairMappings(asset.PerpetualContract, []pairMapping{expected})
	result, err := cached.fetchPairMapping(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Fetching a concurrently cached pair mapping must not error")
	assert.Equal(t, expected, result, "Concurrently cached pair mapping should be returned")
	require.NoError(t, cached.Shutdown(), "Shutting down the cached exchange must not error")

	var fetches atomic.Int32
	refreshed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "Decoding pair metadata request should not error") {
			return
		}
		fetches.Add(1)
		switch payload.Type {
		case infoTypePerpetualDEXs:
			_, err := w.Write([]byte(`[null]`))
			assert.NoError(t, err, "Writing perpetual DEX registry should not error")
		case infoTypeMetadata:
			_, err := w.Write([]byte(perpetualMetadataJSON))
			assert.NoError(t, err, "Writing perpetual metadata should not error")
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	refreshed.setPairMappings(asset.PerpetualContract, nil)
	result, err = refreshed.fetchPairMapping(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Refreshing a pair mapping must not error")
	assert.Equal(t, "BTC", result.coin, "Refreshed pair mapping should be returned")
	refreshed.pairMappingMisses = nil
	missingPair := currency.NewPair(currency.ETH, currency.USDC)
	_, err = refreshed.fetchPairMapping(t.Context(), missingPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing refreshed pair mapping must return the expected error")
	fetchCount := fetches.Load()
	_, err = refreshed.fetchPairMapping(t.Context(), missingPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "Cached missing pair mapping must return the expected error")
	assert.Equal(t, fetchCount, fetches.Load(), "Cached missing pair mapping should not refetch metadata")
	cacheKey := "pair:" + asset.PerpetualContract.String() + ":" + strings.ToLower(missingPair.String())
	refreshed.pairMappingMisses[cacheKey] = time.Now().Add(-time.Second)
	_, err = refreshed.fetchPairMapping(t.Context(), missingPair, asset.PerpetualContract)
	require.ErrorIs(t, err, errPairMappingNotFound, "Expired missing pair mapping must return the expected error")
	assert.Greater(t, fetches.Load(), fetchCount, "Expired missing pair mapping should refresh metadata")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = failed.fetchPairMapping(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Pair mapping refresh failure must be returned")
}

func TestLookupPairMappingByCoin(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, _, err := ex.lookupPairMappingByCoin("BTC")
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing coin mapping must return the expected error")

	expected := pairMapping{pair: testPerpetualPair, coin: "BTC"}
	ex.setPairMappings(asset.PerpetualContract, []pairMapping{expected})
	result, a, err := ex.lookupPairMappingByCoin("BTC")
	require.NoError(t, err, "Unique coin mapping must not error")
	assert.Equal(t, expected, result, "Unique coin mapping should match")
	assert.Equal(t, asset.PerpetualContract, a, "Unique coin mapping should return its asset")

	ex.setPairMappings(asset.Spot, []pairMapping{{pair: testSpotPair, coin: "BTC"}})
	_, _, err = ex.lookupPairMappingByCoin("BTC")
	require.ErrorIs(t, err, errAmbiguousCoinMapping, "Ambiguous coin mapping must return the expected error")
	require.NoError(t, ex.Shutdown(), "Shutting down the coin lookup exchange must not error")
}

func TestFetchPairMappingByCoin(t *testing.T) {
	cached := new(Exchange)
	cached.SetDefaults()
	expected := pairMapping{pair: testPerpetualPair, coin: "BTC"}
	cached.setPairMappings(asset.PerpetualContract, []pairMapping{expected})
	result, a, err := cached.fetchPairMappingByCoin(t.Context(), "BTC")
	require.NoError(t, err, "Fetching a concurrently cached coin mapping must not error")
	assert.Equal(t, expected, result, "Concurrently cached coin mapping should be returned")
	assert.Equal(t, asset.PerpetualContract, a, "Concurrently cached coin mapping should retain its asset")

	cached.setPairMappings(asset.Spot, []pairMapping{{pair: testSpotPair, coin: "BTC"}})
	_, _, err = cached.fetchPairMappingByCoin(t.Context(), "BTC")
	require.ErrorIs(t, err, errAmbiguousCoinMapping, "Concurrently cached ambiguous mapping must return the expected error")
	require.NoError(t, cached.Shutdown(), "Shutting down the cached coin exchange must not error")

	unsupported := new(Exchange)
	_, _, err = unsupported.fetchPairMappingByCoin(t.Context(), "BTC")
	require.ErrorIs(t, err, errPairMappingNotFound, "Exchange without supported assets must return a missing coin mapping")

	var fetches atomic.Int32
	refreshed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&payload), "Decoding coin metadata request should not error") {
			return
		}
		fetches.Add(1)
		switch payload.Type {
		case infoTypePerpetualDEXs:
			_, err := w.Write([]byte(`[null]`))
			assert.NoError(t, err, "Writing perpetual DEX registry should not error")
		case "meta":
			_, err := w.Write([]byte(perpetualMetadataJSON))
			assert.NoError(t, err, "Writing perpetual metadata should not error")
		case "spotMeta":
			_, err := w.Write([]byte(spotMetadataJSON))
			assert.NoError(t, err, "Writing spot metadata should not error")
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
		}
	}))
	refreshed.setPairMappings(asset.Spot, nil)
	refreshed.setPairMappings(asset.PerpetualContract, nil)
	result, a, err = refreshed.fetchPairMappingByCoin(t.Context(), "BTC")
	require.NoError(t, err, "Refreshing a coin mapping must not error")
	assert.Equal(t, "BTC", result.coin, "Refreshed coin mapping should be returned")
	assert.Equal(t, asset.PerpetualContract, a, "Refreshed coin mapping should return its asset")
	_, _, err = refreshed.fetchPairMappingByCoin(t.Context(), "MISSING")
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing refreshed coin mapping must return the expected error")
	fetchCount := fetches.Load()
	_, _, err = refreshed.fetchPairMappingByCoin(t.Context(), "MISSING")
	require.ErrorIs(t, err, errPairMappingNotFound, "Cached missing coin mapping must return the expected error")
	assert.Equal(t, fetchCount, fetches.Load(), "Cached missing coin mapping should not refetch metadata")
	refreshed.pairMappingMisses["coin:missing"] = time.Now().Add(-time.Second)
	_, _, err = refreshed.fetchPairMappingByCoin(t.Context(), "MISSING")
	require.ErrorIs(t, err, errPairMappingNotFound, "Expired missing coin mapping must return the expected error")
	assert.Greater(t, fetches.Load(), fetchCount, "Expired missing coin mapping should refresh metadata")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, _, err = failed.fetchPairMappingByCoin(t.Context(), "BTC")
	require.Error(t, err, "Coin mapping refresh failure must be returned")
}

func TestGetPairMappingByCoin(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"meta": perpetualMetadataJSON, "spotMeta": spotMetadataJSON})
	_, _, err := ex.getPairMappingByCoin(t.Context(), "")
	require.ErrorIs(t, err, errCoinRequired, "Empty API coin must return the expected error")

	ex.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC"}})
	result, a, err := ex.getPairMappingByCoin(t.Context(), "BTC")
	require.NoError(t, err, "Getting a cached coin mapping must not error")
	assert.Equal(t, testPerpetualPair, result.pair, "Cached coin mapping should be returned")
	assert.Equal(t, asset.PerpetualContract, a, "Cached coin mapping should return its asset")

	ex.setPairMappings(asset.Spot, []pairMapping{{pair: testSpotPair, coin: "BTC"}})
	_, _, err = ex.getPairMappingByCoin(t.Context(), "BTC")
	require.ErrorIs(t, err, errAmbiguousCoinMapping, "Getting an ambiguous cached coin mapping must return the expected error")

	ex.setPairMappings(asset.Spot, nil)
	ex.setPairMappings(asset.PerpetualContract, nil)
	result, a, err = ex.getPairMappingByCoin(t.Context(), "@107")
	require.NoError(t, err, "Getting a refreshed coin mapping must not error")
	assert.True(t, result.pair.Equal(testSpotPair), "Refreshed coin mapping should be returned")
	assert.Equal(t, asset.Spot, a, "Refreshed coin mapping should return its asset")
}

func TestGetCoin(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"spotMeta": spotMetadataJSON})
	_, err := ex.getCoin(t.Context(), testSpotPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported asset must return the expected error")
	_, err = ex.getCoin(t.Context(), currency.EMPTYPAIR, asset.Spot)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "Empty pair must return the expected error")

	ex.setPairMappings(asset.Spot, []pairMapping{{pair: testSpotPair, coin: "@cached"}})
	coin, err := ex.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting a cached coin mapping must not error")
	assert.Equal(t, "@cached", coin, "Cached coin mapping should be returned")

	ex.setPairMappings(asset.Spot, nil)
	coin, err = ex.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Refreshing a coin mapping must not error")
	assert.Equal(t, "@107", coin, "Refreshed coin mapping should be returned")

	_, err = ex.getCoin(t.Context(), currency.NewPairWithDelimiter("MISSING", "USDC", "-"), asset.Spot)
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing refreshed mapping must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.Error(t, err, "Refreshing a mapping from a failing server must error")
}

func TestFetchTradablePairs(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		"meta":     `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40,"onlyIsolated":true},{"name":"OLD","isDelisted":true}]}`,
		"spotMeta": `{"universe":[{"tokens":[150,0],"name":"@107"}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`,
	})
	_, err := ex.FetchTradablePairs(t.Context(), asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported asset must return the expected error")
	_, err = new(Exchange).FetchTradablePairs(t.Context(), asset.Spot)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unconfigured spot support must return the expected error")

	perpetualPairs, err := ex.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Fetching perpetual pairs must not error")
	require.Len(t, perpetualPairs, 1, "Delisted perpetual pairs must be excluded")
	assert.True(t, perpetualPairs[0].Equal(testPerpetualPair), "Perpetual pair should use USDC as quote")
	coin, err := ex.getCoin(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting the fetched perpetual mapping must not error")
	assert.Equal(t, "BTC", coin, "Perpetual mapping should use the API coin")
	mapping, err := ex.getPairMapping(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting fetched perpetual metadata must not error")
	assert.Equal(t, uint64(40), mapping.maxLeverage, "Perpetual mapping should retain the market leverage limit")
	assert.True(t, mapping.onlyIsolated, "Perpetual mapping should retain the isolated-only restriction")

	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding HIP-3 discovery request should not error") {
			return
		}
		var response string
		switch request.Type {
		case infoTypePerpetualDEXs:
			response = `[null,{"name":"xyz"},{"name":"flx"}]`
		case "meta":
			switch request.DEX {
			case "":
				response = `{"universe":[{"name":"BTC","szDecimals":5}]}`
			case testBuilderDEXName:
				response = `{"universe":[{"name":"xyz:XYZ100","szDecimals":2},{"name":"xyz:OLD","isDelisted":true}]}`
			case "flx":
				response = `{"universe":[{"name":"flx:TSLA","szDecimals":3}]}`
			default:
				http.Error(w, "unexpected DEX", http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing HIP-3 discovery response should not error")
	}))
	hip3Pairs, err := hip3.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Fetching default and HIP-3 markets must not error")
	require.Len(t, hip3Pairs, 3, "FetchTradablePairs must retain active markets from every perpetual DEX")
	for _, tc := range []struct {
		pair    currency.Pair
		coin    string
		dex     string
		assetID uint64
	}{
		{pair: testPerpetualPair, coin: "BTC", assetID: 0},
		{pair: currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC), coin: "xyz:XYZ100", dex: testBuilderDEXName, assetID: 110000},
		{pair: currency.NewPair(currency.NewCode("flx:TSLA"), currency.USDC), coin: "flx:TSLA", dex: "flx", assetID: 120000},
	} {
		mapping, err := hip3.getPairMapping(t.Context(), tc.pair, asset.PerpetualContract)
		require.NoError(t, err, "Getting discovered DEX mapping must not error")
		assert.Equal(t, tc.coin, mapping.coin, "DEX mapping should retain the API coin")
		assert.Equal(t, tc.dex, mapping.dex, "DEX mapping should retain its DEX scope")
		assert.Equal(t, tc.assetID, mapping.assetID, "DEX mapping should apply the official asset-ID offset")
	}

	variableCollateral := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding variable-collateral discovery request should not error") {
			return
		}
		var response string
		switch request.Type {
		case infoTypePerpetualDEXs:
			response = `[null,{"name":"xyz"}]`
		case infoTypeMetadata:
			if request.DEX == testBuilderDEXName {
				response = `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100","szDecimals":2}]}`
			} else {
				response = `{"collateralToken":0,"universe":[{"name":"BTC","szDecimals":5}]}`
			}
		case "spotMeta":
			response = `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`
		default:
			http.Error(w, "unexpected request", http.StatusBadRequest)
			return
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing variable-collateral discovery response should not error")
	}))
	variablePairs, err := variableCollateral.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Fetching a HIP-3 market with non-USDC collateral must not error")
	assert.Contains(t, variablePairs, currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.NewCode("HYPE")), "HIP-3 pair quote should use the DEX collateral token")

	for _, tc := range []struct {
		name         string
		registry     string
		builderMeta  string
		secondMeta   string
		spotMetadata string
		spotFailure  bool
		expectedIs   error
	}{
		{
			name:        "spot metadata failure",
			registry:    `[null,{"name":"xyz"}]`,
			builderMeta: `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			spotFailure: true,
		},
		{
			name:         "duplicate collateral token",
			registry:     `[null,{"name":"xyz"}]`,
			builderMeta:  `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			spotMetadata: `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"HYPE2","index":150}]}`,
			expectedIs:   errUnexpectedResponseLength,
		},
		{
			name:         "blank collateral token",
			registry:     `[null,{"name":"xyz"}]`,
			builderMeta:  `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			spotMetadata: `{"tokens":[{"name":"USDC","index":0},{"name":" ","index":150}]}`,
			expectedIs:   errSpotTokenNotFound,
		},
		{
			name:         "invalid canonical collateral",
			registry:     `[null,{"name":"xyz"}]`,
			builderMeta:  `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			spotMetadata: `{"tokens":[{"name":"USDT","index":0},{"name":"HYPE","index":150}]}`,
			expectedIs:   errSpotTokenNotFound,
		},
		{
			name:         "missing collateral token",
			registry:     `[null,{"name":"xyz"}]`,
			builderMeta:  `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			spotMetadata: `{"tokens":[{"name":"USDC","index":0}]}`,
			expectedIs:   errSpotTokenNotFound,
		},
		{
			name:         "later DEX missing collateral token",
			registry:     `[null,{"name":"xyz"},{"name":"flx"}]`,
			builderMeta:  `{"collateralToken":150,"universe":[{"name":"xyz:XYZ100"}]}`,
			secondMeta:   `{"collateralToken":151,"universe":[{"name":"flx:TSLA"}]}`,
			spotMetadata: `{"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150}]}`,
			expectedIs:   errSpotTokenNotFound,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request infoRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding invalid collateral discovery request should not error") {
					return
				}
				switch request.Type {
				case infoTypePerpetualDEXs:
					_, err := w.Write([]byte(tc.registry))
					assert.NoError(t, err, "Writing invalid collateral registry should not error")
				case infoTypeMetadata:
					response := `{"collateralToken":0,"universe":[{"name":"BTC"}]}`
					switch request.DEX {
					case testBuilderDEXName:
						response = tc.builderMeta
					case "flx":
						response = tc.secondMeta
					}
					_, err := w.Write([]byte(response))
					assert.NoError(t, err, "Writing invalid collateral metadata should not error")
				case "spotMeta":
					if tc.spotFailure {
						http.Error(w, "spot metadata unavailable", http.StatusServiceUnavailable)
						return
					}
					_, err := w.Write([]byte(tc.spotMetadata))
					assert.NoError(t, err, "Writing invalid spot metadata should not error")
				default:
					http.Error(w, "unexpected request", http.StatusBadRequest)
				}
			}))
			_, err := invalid.FetchTradablePairs(t.Context(), asset.PerpetualContract)
			require.Error(t, err, "Invalid collateral discovery must return an error")
			if tc.expectedIs != nil {
				require.ErrorIs(t, err, tc.expectedIs, "Invalid collateral discovery must return the expected error")
			}
		})
	}

	spotPairs, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "Fetching spot pairs must not error")
	require.Len(t, spotPairs, 1, "Spot metadata must produce one pair")
	assert.True(t, spotPairs[0].Equal(testSpotPair), "Spot pair should use token metadata names")
	coin, err = ex.getCoin(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting the fetched spot mapping must not error")
	assert.Equal(t, "@107", coin, "Spot mapping should retain the API market identifier")

	duplicatePerpetual := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[{"name":"BTC"},{"name":"BTC"},{"name":"ETH"}]}`})
	perpetualPairs, err = duplicatePerpetual.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Fetching perpetual markets with an ambiguous display pair must not error")
	require.Len(t, perpetualPairs, 1, "Ambiguous perpetual display pairs must be skipped")
	assert.True(t, perpetualPairs[0].Equal(currency.NewPair(currency.ETH, currency.USDC)), "Unambiguous perpetual pairs should be retained")

	duplicateSpot := newStaticInfoExchange(t, map[string]string{
		"spotMeta": `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@108","index":108},{"tokens":[152,0],"name":"@109","index":109}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"HYPE","index":151},{"name":"PURR","index":152}]}`,
	})
	spotPairs, err = duplicateSpot.FetchTradablePairs(t.Context(), asset.Spot)
	require.NoError(t, err, "Fetching spot markets with an ambiguous display pair must not error")
	require.Len(t, spotPairs, 1, "Ambiguous spot display pairs must be skipped")
	assert.True(t, spotPairs[0].Equal(currency.NewPair(currency.NewCode("PURR"), currency.USDC)), "Unambiguous spot pairs should be retained")
	coin, err = duplicateSpot.getCoin(t.Context(), spotPairs[0], asset.Spot)
	require.NoError(t, err, "Getting an unambiguous spot mapping must not error")
	assert.Equal(t, "@109", coin, "Unambiguous spot mapping should retain the API market identifier")

	invalidPerpetual := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[{"name":"BAD COIN"},{"name":"ETH"}]}`})
	perpetualPairs, err = invalidPerpetual.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.NoError(t, err, "Fetching a perpetual market with an invalid currency name must not error")
	require.Len(t, perpetualPairs, 1, "Invalid perpetual names must be skipped")
	assert.True(t, perpetualPairs[0].Equal(currency.NewPair(currency.ETH, currency.USDC)), "Valid perpetual markets should remain available")

	for _, tc := range []struct {
		name       string
		registry   string
		metadata   string
		expectedIs error
	}{
		{name: "missing builder entry", registry: `[null,null]`, metadata: perpetualMetadataJSON, expectedIs: errInvalidPerpetualDEX},
		{name: "blank builder name", registry: `[null,{"name":" "}]`, metadata: perpetualMetadataJSON, expectedIs: errInvalidPerpetualDEX},
		{name: "duplicate builder name", registry: `[null,{"name":"xyz"},{"name":"xyz"}]`, metadata: `{"universe":[{"name":"xyz:XYZ100"}]}`, expectedIs: errInvalidPerpetualDEX},
		{name: "unscoped builder market", registry: `[null,{"name":"xyz"}]`, metadata: `{"universe":[{"name":"XYZ100"}]}`, expectedIs: errInvalidPerpetualDEX},
		{
			name:       "too many builder markets",
			registry:   `[null,{"name":"xyz"}]`,
			metadata:   `{"universe":[` + strings.Repeat(`{"name":"xyz:X"},`, builderPerpetualDEXAssetStride) + `{"name":"xyz:X"}]}`,
			expectedIs: errInvalidPerpetualDEX,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request infoRequest
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding invalid DEX request should not error") {
					return
				}
				response := tc.metadata
				if request.Type == infoTypePerpetualDEXs {
					response = tc.registry
				}
				_, err := w.Write([]byte(response))
				assert.NoError(t, err, "Writing invalid DEX response should not error")
			}))
			_, err := invalid.FetchTradablePairs(t.Context(), asset.PerpetualContract)
			require.ErrorIs(t, err, tc.expectedIs, "Invalid DEX discovery must return the expected error")
		})
	}

	metadataFailure := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding failing DEX request should not error") {
			return
		}
		if request.Type == infoTypePerpetualDEXs {
			_, err := w.Write([]byte(`[null,{"name":"xyz"}]`))
			assert.NoError(t, err, "Writing perpetual DEX registry should not error")
			return
		}
		http.Error(w, "metadata unavailable", http.StatusServiceUnavailable)
	}))
	_, err = metadataFailure.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.Error(t, err, "Builder metadata failure must be returned")

	for _, tc := range []struct {
		name     string
		response string
	}{
		{
			name:     "invalid token count",
			response: `{"universe":[{"tokens":[1],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"PURR","index":2}]}`,
		},
		{
			name:     "missing base token",
			response: `{"universe":[{"tokens":[1,0],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"PURR","index":2}]}`,
		},
		{
			name:     "missing quote token",
			response: `{"universe":[{"tokens":[1,9],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"TOKEN","index":1},{"name":"PURR","index":2}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalidExchange := newStaticInfoExchange(t, map[string]string{"spotMeta": tc.response})
			pairs, err := invalidExchange.FetchTradablePairs(t.Context(), asset.Spot)
			require.NoError(t, err, "Fetching structurally invalid spot metadata must not error")
			require.Len(t, pairs, 1, "Structurally invalid spot markets must be skipped")
			assert.True(t, pairs[0].Equal(currency.NewPair(currency.NewCode("PURR"), currency.USDC)), "Valid spot markets should remain available")
		})
	}
	for _, tc := range []struct {
		name     string
		response string
	}{
		{
			name:     "empty token name",
			response: `{"universe":[{"tokens":[1,0],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"","index":1},{"name":"PURR","index":2}]}`,
		},
		{
			name:     "invalid token spacing",
			response: `{"universe":[{"tokens":[1,0],"name":"@1","index":1},{"tokens":[2,0],"name":"@2","index":2}],"tokens":[{"name":"USDC","index":0},{"name":"BAD COIN","index":1},{"name":"PURR","index":2}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalidExchange := newStaticInfoExchange(t, map[string]string{"spotMeta": tc.response})
			pairs, err := invalidExchange.FetchTradablePairs(t.Context(), asset.Spot)
			require.NoError(t, err, "Fetching spot metadata with an invalid token name must not error")
			require.Len(t, pairs, 1, "Invalid spot token names must be skipped")
			assert.True(t, pairs[0].Equal(currency.NewPair(currency.NewCode("PURR"), currency.USDC)), "Valid spot markets should remain available")
		})
	}

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.FetchTradablePairs(t.Context(), asset.Spot)
	require.Error(t, err, "Fetching spot pairs from a failing server must error")
	_, err = errorExchange.FetchTradablePairs(t.Context(), asset.PerpetualContract)
	require.Error(t, err, "Fetching perpetual pairs from a failing server must error")
}

func TestFetchTradablePairsRejectsDuplicateSpotIdentities(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response string
	}{
		{
			name:     "token index",
			response: `{"universe":[{"tokens":[150,0],"name":"@107","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":150}]}`,
		},
		{
			name:     "market index",
			response: `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@108","index":107}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":151}]}`,
		},
		{
			name:     "market name",
			response: `{"universe":[{"tokens":[150,0],"name":"@107","index":107},{"tokens":[151,0],"name":"@107","index":108}],"tokens":[{"name":"USDC","index":0},{"name":"HYPE","index":150},{"name":"PURR","index":151}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newStaticInfoExchange(t, map[string]string{"spotMeta": tc.response})
			_, err := ex.FetchTradablePairs(t.Context(), asset.Spot)
			require.ErrorIs(t, err, errUnexpectedResponseLength, "Duplicate spot metadata identity must return the expected error")
			assert.Empty(t, ex.pairMappings[asset.Spot], "Rejected spot metadata should not install pair mappings")
		})
	}
}

func TestGetPerpetualDEXNames(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		infoTypePerpetualDEXs: `[null,{"name":" xyz "},{"name":"hyna"}]`,
	})
	names, err := ex.getPerpetualDEXNames(t.Context())
	require.NoError(t, err, "Getting valid perpetual DEX names must not error")
	assert.Equal(t, []string{"", "xyz", "hyna"}, names, "Perpetual DEX names should retain registry order and normalise whitespace")

	for _, tc := range []struct {
		name       string
		registry   string
		expectedIs error
	}{
		{name: "empty", registry: `[]`, expectedIs: errUnexpectedResponseLength},
		{name: "non-default first entry", registry: `[{"name":"default"}]`, expectedIs: errUnexpectedResponseLength},
		{name: "missing builder entry", registry: `[null,null]`, expectedIs: errInvalidPerpetualDEX},
		{name: "blank builder name", registry: `[null,{"name":" "}]`, expectedIs: errInvalidPerpetualDEX},
		{name: "duplicate builder name", registry: `[null,{"name":"xyz"},{"name":" xyz "}]`, expectedIs: errInvalidPerpetualDEX},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := newStaticInfoExchange(t, map[string]string{infoTypePerpetualDEXs: tc.registry})
			_, err := invalid.getPerpetualDEXNames(t.Context())
			require.ErrorIs(t, err, tc.expectedIs, "Invalid perpetual DEX registry must return the expected error")
		})
	}

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = failed.getPerpetualDEXNames(t.Context())
	require.Error(t, err, "Perpetual DEX registry HTTP failure must be returned")
}

func TestUpdateTradablePairs(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"meta": perpetualMetadataJSON, "spotMeta": spotMetadataJSON})
	require.NoError(t, ex.UpdateTradablePairs(t.Context()), "Updating tradable pairs must not error")
	spot, err := ex.GetAvailablePairs(asset.Spot)
	require.NoError(t, err, "Getting updated spot pairs must not error")
	assert.Contains(t, spot, testSpotPair, "Updated spot pairs should contain HYPE-USDC")
	perpetual, err := ex.GetAvailablePairs(asset.PerpetualContract)
	require.NoError(t, err, "Getting updated perpetual pairs must not error")
	assert.Contains(t, perpetual, testPerpetualPair, "Updated perpetual pairs should contain BTC-USDC")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	require.Error(t, errorExchange.UpdateTradablePairs(t.Context()), "Updating tradable pairs from a failing server must error")

	emptyResponseExchange := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`, "spotMeta": `{"universe":[],"tokens":[]}`})
	require.Error(t, emptyResponseExchange.UpdateTradablePairs(t.Context()), "Updating tradable pairs from empty metadata must error")

	noAssetExchange := newStaticInfoExchange(t, map[string]string{})
	noAssetExchange.CurrencyPairs.Pairs = make(map[asset.Item]*currency.PairStore)
	require.Error(t, noAssetExchange.UpdateTradablePairs(t.Context()), "Updating tradable pairs without configured assets must error")
}

func TestUpdatePerpetualTickers(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.PerpetualContract), "Updating perpetual tickers must not error")
	price, err := ticker.GetTicker(ex.Name, testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting the updated perpetual ticker must not error")
	assert.Equal(t, 100.5, price.Last, "Perpetual ticker should use the current midpoint")
	assert.Equal(t, 10.0, price.Volume, "Perpetual ticker should include base volume")
	assert.Equal(t, 1000.0, price.QuoteVolume, "Perpetual ticker should include quote volume")
	assert.Equal(t, 99.0, price.Open, "Perpetual ticker should include the previous-day price")
	assert.Zero(t, price.Close, "Perpetual ticker should not label a midpoint or mark price as the close")
	assert.Zero(t, price.Bid, "Perpetual ticker should not treat impact prices as the best bid")
	assert.Zero(t, price.Ask, "Perpetual ticker should not treat impact prices as the best ask")
	assert.Equal(t, 101.0, price.MarkPrice, "Perpetual mark price should be populated")
	assert.Equal(t, 100.0, price.IndexPrice, "Perpetual index price should be populated")
	assert.WithinDuration(t, time.Now().UTC(), price.LastUpdated, time.Second, "Perpetual ticker update time should be current")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	var requestedDEXes []string
	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding scoped ticker request should not error") {
			return
		}
		requestedDEXes = append(requestedDEXes, request.DEX)
		response := perpetualContextsJSON
		if request.DEX == "xyz" {
			response = `[{"universe":[{"name":"xyz:XYZ100"}]},[{"openInterest":"20","prevDayPx":"49","dayNtlVlm":"2000","oraclePx":"50","markPx":"51","midPx":"50.5","dayBaseVlm":"40"}]]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing scoped ticker response should not error")
	}))
	hip3.setPairMappings(asset.PerpetualContract, []pairMapping{
		{pair: testPerpetualPair, coin: "BTC"},
		{pair: hip3Pair, coin: "xyz:XYZ100", dex: "xyz", assetID: 110000},
	})
	require.NoError(t, hip3.UpdatePairs(currency.Pairs{testPerpetualPair, hip3Pair}, asset.PerpetualContract, false), "Updating scoped ticker pairs must not error")
	require.NoError(t, hip3.UpdatePairs(currency.Pairs{testPerpetualPair, hip3Pair}, asset.PerpetualContract, true), "Enabling scoped ticker pairs must not error")
	require.NoError(t, hip3.UpdateTickers(t.Context(), asset.PerpetualContract), "Updating default and HIP-3 tickers must not error")
	assert.Equal(t, []string{"", "xyz"}, requestedDEXes, "Ticker batch should query each required DEX once")
	hip3Price, err := ticker.GetTicker(hip3.Name, hip3Pair, asset.PerpetualContract)
	require.NoError(t, err, "Getting updated HIP-3 ticker must not error")
	assert.Equal(t, 50.5, hip3Price.Last, "HIP-3 ticker should use its DEX-scoped midpoint")
	assert.Equal(t, 20.0, hip3Price.OpenInterest, "HIP-3 ticker should use its DEX-scoped open interest")

	fallbackJSON := `[` + perpetualMetadataJSON + `,[{"openInterest":"10","prevDayPx":"99","dayNtlVlm":"1000","oraclePx":"100","markPx":"101","midPx":"0","dayBaseVlm":"10"}]]`
	fallbackExchange := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": fallbackJSON})
	setTestPair(t, fallbackExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	require.NoError(t, fallbackExchange.UpdateTickers(t.Context(), asset.PerpetualContract), "Updating a perpetual ticker without a midpoint must not error")
	price, err = ticker.GetTicker(fallbackExchange.Name, testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting the fallback perpetual ticker must not error")
	assert.Equal(t, 101.0, price.Last, "Perpetual ticker should fall back to the mark price when the midpoint is unavailable")

	lengthExchange := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": `[` + perpetualMetadataJSON + `,[]]`})
	setTestPair(t, lengthExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	require.ErrorIs(t, lengthExchange.UpdateTickers(t.Context(), asset.PerpetualContract), errUnexpectedResponseLength, "Mismatched perpetual contexts must return the expected error")

	missingExchange := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, missingExchange, asset.PerpetualContract, testPerpetualPair, "ETH")
	require.ErrorIs(t, missingExchange.UpdateTickers(t.Context(), asset.PerpetualContract), errAssetContextNotFound, "Missing perpetual context must return the expected error")

	processExchange := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, processExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	processExchange.Name = ""
	require.ErrorIs(t, processExchange.UpdateTickers(t.Context(), asset.PerpetualContract), common.ErrExchangeNameNotSet, "Invalid perpetual ticker must return its processing error")

	mappingExchange := newStaticInfoExchange(t, map[string]string{
		"metaAndAssetCtxs": perpetualContextsJSON,
		"meta":             `{"universe":[{"name":"ETH"}]}`,
	})
	setTestPair(t, mappingExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	mappingExchange.setPairMappings(asset.PerpetualContract, nil)
	require.ErrorIs(t, mappingExchange.UpdateTickers(t.Context(), asset.PerpetualContract), errPairMappingNotFound, "Missing perpetual pair mapping must return the expected error")

	mixedExchange := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	missingPair := currency.NewPair(currency.ETH, currency.USDC)
	mixedExchange.setPairMappings(asset.PerpetualContract, []pairMapping{
		{pair: testPerpetualPair, coin: "BTC"},
		{pair: missingPair, coin: "ETH"},
	})
	require.NoError(t, mixedExchange.UpdatePairs(currency.Pairs{testPerpetualPair, missingPair}, asset.PerpetualContract, false), "Updating mixed available pairs must not error")
	require.NoError(t, mixedExchange.UpdatePairs(currency.Pairs{testPerpetualPair, missingPair}, asset.PerpetualContract, true), "Updating mixed enabled pairs must not error")
	require.ErrorIs(t, mixedExchange.UpdateTickers(t.Context(), asset.PerpetualContract), errAssetContextNotFound, "Mixed ticker update must report the missing context")
	price, err = ticker.GetTicker(mixedExchange.Name, testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting a valid ticker from a partial batch must not error")
	assert.Equal(t, 100.5, price.Last, "Partial batch should retain valid ticker updates")
}

func TestUpdateSpotTickers(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": spotContextsJSON})
	setTestPair(t, ex, asset.Spot, testSpotPair, "@107")
	require.NoError(t, ex.UpdateTickers(t.Context(), asset.Spot), "Updating spot tickers must not error")
	price, err := ticker.GetTicker(ex.Name, testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting the updated spot ticker must not error")
	assert.Equal(t, 9.5, price.Last, "Spot ticker should use the current midpoint")
	assert.Equal(t, 10.0, price.Volume, "Spot ticker should include base volume")
	assert.Equal(t, 100.0, price.QuoteVolume, "Spot ticker should include quote volume")
	assert.Equal(t, 9.0, price.Open, "Spot ticker should include the previous-day price")
	assert.Zero(t, price.Close, "Spot ticker should not label a midpoint or mark price as the close")
	assert.Equal(t, 10.0, price.MarkPrice, "Spot mark price should be populated")
	assert.WithinDuration(t, time.Now().UTC(), price.LastUpdated, time.Second, "Spot ticker update time should be current")

	fallbackJSON := `[` + spotMetadataJSON + `,[{"prevDayPx":"9","dayNtlVlm":"100","markPx":"10","midPx":"0","coin":"@107","dayBaseVlm":"10"}]]`
	fallbackExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": fallbackJSON})
	setTestPair(t, fallbackExchange, asset.Spot, testSpotPair, "@107")
	require.NoError(t, fallbackExchange.UpdateTickers(t.Context(), asset.Spot), "Updating a spot ticker without a midpoint must not error")
	price, err = ticker.GetTicker(fallbackExchange.Name, testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting the fallback spot ticker must not error")
	assert.Equal(t, 10.0, price.Last, "Spot ticker should fall back to the mark price when the midpoint is unavailable")

	positionalJSON := `[` + spotMetadataJSON + `,[{"prevDayPx":"9","dayNtlVlm":"100","markPx":"10","midPx":"9.5","dayBaseVlm":"10"}]]`
	positionalExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": positionalJSON})
	setTestPair(t, positionalExchange, asset.Spot, testSpotPair, "@107")
	require.NoError(t, positionalExchange.UpdateTickers(t.Context(), asset.Spot), "Updating aligned positional spot contexts must not error")
	price, err = ticker.GetTicker(positionalExchange.Name, testSpotPair, asset.Spot)
	require.NoError(t, err, "Getting the positional spot ticker must not error")
	assert.Equal(t, 9.5, price.Last, "Aligned positional spot context should populate the midpoint")

	mixedContextJSON := `[{"universe":[{"name":"@1"},{"name":"@2"}],"tokens":[]},[{"coin":"@2"},{}]]`
	mixedContextExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": mixedContextJSON})
	require.ErrorIs(t, mixedContextExchange.UpdateTickers(t.Context(), asset.Spot), errAssetContextNotFound, "Mixed explicit and positional spot contexts must fail closed")

	unmappedContextJSON := `[{"universe":[],"tokens":[]},[{"markPx":"10"}]]`
	unmappedContextExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": unmappedContextJSON})
	require.ErrorIs(t, unmappedContextExchange.UpdateTickers(t.Context(), asset.Spot), errAssetContextNotFound, "Unidentified spot context must return the expected error")

	missingExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": spotContextsJSON})
	setTestPair(t, missingExchange, asset.Spot, testSpotPair, "@missing")
	require.ErrorIs(t, missingExchange.UpdateTickers(t.Context(), asset.Spot), errAssetContextNotFound, "Missing spot context must return the expected error")

	processExchange := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": spotContextsJSON})
	setTestPair(t, processExchange, asset.Spot, testSpotPair, "@107")
	processExchange.Name = ""
	require.ErrorIs(t, processExchange.UpdateTickers(t.Context(), asset.Spot), common.ErrExchangeNameNotSet, "Invalid spot ticker must return its processing error")

	mappingExchange := newStaticInfoExchange(t, map[string]string{
		"spotMetaAndAssetCtxs": spotContextsJSON,
		"spotMeta":             `{"universe":[],"tokens":[]}`,
	})
	setTestPair(t, mappingExchange, asset.Spot, testSpotPair, "@107")
	mappingExchange.setPairMappings(asset.Spot, nil)
	require.ErrorIs(t, mappingExchange.UpdateTickers(t.Context(), asset.Spot), errPairMappingNotFound, "Missing spot pair mapping must return the expected error")
}

func TestUpdateTickersErrors(t *testing.T) {
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	require.ErrorIs(t, ex.UpdateTickers(t.Context(), asset.Options), asset.ErrNotSupported, "Unsupported ticker asset must return the expected error")
	require.ErrorIs(t, new(Exchange).UpdateTickers(t.Context(), asset.Spot), asset.ErrNotSupported, "Unconfigured ticker asset must return the expected error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, false), "Disabling the spot asset must not error")
	require.Error(t, ex.UpdateTickers(t.Context(), asset.Spot), "Updating tickers for a disabled asset must error")
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Spot, true), "Re-enabling the spot asset must not error")
	setTestPair(t, ex, asset.Spot, testSpotPair, "@107")
	require.Error(t, ex.UpdateTickers(t.Context(), asset.Spot), "Updating spot tickers from a failing server must error")
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	require.Error(t, ex.UpdateTickers(t.Context(), asset.PerpetualContract), "Updating perpetual tickers from a failing server must error")
}

func TestUpdateTicker(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"spotMetaAndAssetCtxs": spotContextsJSON})
	setTestPair(t, ex, asset.Spot, testSpotPair, "@107")
	price, err := ex.UpdateTicker(t.Context(), testSpotPair, asset.Spot)
	require.NoError(t, err, "Updating one ticker must not error")
	assert.Equal(t, 10.0, price.MarkPrice, "Updated ticker should be returned from the cache")
	assert.Equal(t, 9.5, price.Last, "Updated ticker should include the current midpoint")
	_, err = ex.UpdateTicker(t.Context(), testPerpetualPair, asset.Spot)
	require.ErrorIs(t, err, ticker.ErrTickerNotFound, "Updating a non-enabled pair must return the cache lookup error")
	_, err = ex.UpdateTicker(t.Context(), testSpotPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Updating one unsupported ticker must return the expected error")
}

func TestUpdateOrderbook(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"l2Book": bookJSON})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	book, err := ex.UpdateOrderbook(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Updating an L2 orderbook must not error")
	require.Len(t, book.Bids, 1, "Updated orderbook must contain one bid")
	require.Len(t, book.Asks, 1, "Updated orderbook must contain one ask")
	assert.Equal(t, 100.0, book.Bids[0].Price, "Bid price should be converted")
	assert.Equal(t, 2.0, book.Bids[0].Amount, "Bid amount should be converted")
	assert.Equal(t, 101.0, book.Asks[0].Price, "Ask price should be converted")
	assert.Equal(t, 3.0, book.Asks[0].Amount, "Ask amount should be converted")

	_, err = ex.UpdateOrderbook(t.Context(), testPerpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Updating an unsupported orderbook must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, errorExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = errorExchange.UpdateOrderbook(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Updating an orderbook from a failing server must error")

	invalidBook := `{"coin":"BTC","levels":[[{"px":"100","sz":"-2","n":1}],[{"px":"101","sz":"3","n":2}]],"time":1700000000000}`
	processExchange := newStaticInfoExchange(t, map[string]string{"l2Book": invalidBook})
	setTestPair(t, processExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = processExchange.UpdateOrderbook(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Processing an invalid orderbook must error")
}

func TestGetRecentTrades(t *testing.T) {
	response := `[{"coin":"BTC","side":"B","px":"101","sz":"3","time":1700000001000,"tid":8},{"coin":"BTC","side":"A","px":"100","sz":"2","time":1700000000000,"tid":7}]`
	ex := newStaticInfoExchange(t, map[string]string{"recentTrades": response})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	trades, err := ex.GetRecentTrades(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting converted recent trades must not error")
	require.Len(t, trades, 2, "Converted recent trades must contain both results")
	assert.Equal(t, order.Sell, trades[0].Side, "Ask-side trade should convert to sell")
	assert.Equal(t, order.Buy, trades[1].Side, "Bid-side trade should convert to buy")
	assert.Equal(t, "7", trades[0].TID, "Recent trades should be sorted chronologically")
	assert.Equal(t, "8", trades[1].TID, "Trade ID should be converted to text")

	_, err = ex.GetRecentTrades(t.Context(), testPerpetualPair, asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Getting trades for an unsupported asset must return the expected error")

	invalidExchange := newStaticInfoExchange(t, map[string]string{"recentTrades": `[{"coin":"BTC","side":"X","px":"100","sz":"2","time":1700000000000,"tid":7}]`})
	setTestPair(t, invalidExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = invalidExchange.GetRecentTrades(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "Invalid trade side must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, errorExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = errorExchange.GetRecentTrades(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.Error(t, err, "Getting trades from a failing server must error")
}

func TestGetHistoricCandles(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Minute).Add(-2 * time.Minute)
	end := start.Add(2 * time.Minute)
	response := fmt.Sprintf(`[{"t":%d,"T":%d,"s":"BTC","i":"1m","o":"90","c":"91","h":"92","l":"89","v":"1","n":1},{"t":%d,"T":%d,"s":"BTC","i":"1m","o":"100","c":"101","h":"102","l":"99","v":"5","n":3},{"t":%d,"T":%d,"s":"BTC","i":"1m","o":"110","c":"111","h":"112","l":"109","v":"2","n":1}]`,
		start.Add(-time.Minute).UnixMilli(), start.Add(-time.Millisecond).UnixMilli(),
		start.UnixMilli(), start.Add(time.Minute-time.Millisecond).UnixMilli(),
		end.Add(time.Minute).UnixMilli(), end.Add(2*time.Minute-time.Millisecond).UnixMilli())
	ex := newStaticInfoExchange(t, map[string]string{"candleSnapshot": response})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	item, err := ex.GetHistoricCandles(t.Context(), testPerpetualPair, asset.PerpetualContract, kline.OneMin, start, end)
	require.NoError(t, err, "Getting historic candles must not error")
	require.NotEmpty(t, item.Candles, "Historic candle result must contain data")
	assert.Equal(t, start, item.Candles[0].Time, "Candles before the requested range should be filtered")
	assert.Equal(t, 100.0, item.Candles[0].Open, "Candle open should be converted")
	_, err = ex.GetHistoricCandles(t.Context(), testPerpetualPair, asset.PerpetualContract, kline.OneMin, start.Add(-(maximumCandleCount+1)*time.Minute), start)
	require.ErrorIs(t, err, kline.ErrRequestExceedsExchangeLimits, "Candle ranges outside Hyperliquid retention must fail closed")

	_, err = ex.GetHistoricCandles(t.Context(), currency.EMPTYPAIR, asset.PerpetualContract, kline.OneMin, start, end)
	require.Error(t, err, "Getting candles with an empty pair must error")
	_, err = ex.GetHistoricCandles(t.Context(), testPerpetualPair, asset.Options, kline.OneMin, start, end)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Getting candles for an unsupported asset must return the expected error")

	emptyExchange := newStaticInfoExchange(t, map[string]string{"candleSnapshot": `[]`})
	setTestPair(t, emptyExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = emptyExchange.GetHistoricCandles(t.Context(), testPerpetualPair, asset.PerpetualContract, kline.OneMin, start, end)
	require.ErrorIs(t, err, kline.ErrNoTimeSeriesDataToConvert, "Empty candle response must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, errorExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = errorExchange.GetHistoricCandles(t.Context(), testPerpetualPair, asset.PerpetualContract, kline.OneMin, start, end)
	require.Error(t, err, "Getting candles from a failing server must error")

	mappingExchange := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`})
	setTestPair(t, mappingExchange, asset.PerpetualContract, testPerpetualPair, "BTC")
	mappingExchange.setPairMappings(asset.PerpetualContract, nil)
	_, err = mappingExchange.GetHistoricCandles(t.Context(), testPerpetualPair, asset.PerpetualContract, kline.OneMin, start, end)
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing candle pair mapping must return the expected error")
}

func TestGetFeeByType(t *testing.T) {
	feesJSON := `{"userCrossRate":"0.0003","userAddRate":"0.0001","userSpotCrossRate":"0.0005","userSpotAddRate":"0.0002"}`
	ex := newStaticInfoExchange(t, map[string]string{"userFees": feesJSON})
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")

	_, err := ex.GetFeeByType(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil fee builder must return the expected error")
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.BankFee, Pair: testPerpetualPair})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "Unsupported fee type must return the expected error")
	_, err = ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "Empty fee pair must return the expected error")

	for _, tc := range []struct {
		name   string
		price  float64
		amount float64
	}{
		{name: "negative price", price: -1, amount: 1},
		{name: "negative amount", price: 1, amount: -1},
		{name: "nan price", price: math.NaN(), amount: 1},
		{name: "nan amount", price: 1, amount: math.NaN()},
		{name: "infinite price", price: math.Inf(1), amount: 1},
		{name: "infinite amount", price: 1, amount: math.Inf(1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
				FeeType:       exchange.CryptocurrencyTradeFee,
				Pair:          testPerpetualPair,
				PurchasePrice: tc.price,
				Amount:        tc.amount,
			})
			require.ErrorIs(t, err, order.ErrAmountIsInvalid, "Invalid fee notional must return the expected error")
		})
	}

	taker, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Getting perpetual taker fee must not error")
	assert.InDelta(t, 0.06, taker, 1e-12, "Perpetual taker fee should use the effective account rate")
	maker, err := ex.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testPerpetualPair,
		IsMaker:       true,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Getting perpetual maker fee must not error")
	assert.InDelta(t, 0.03, maker, 1e-12, "Offline perpetual maker fee should use the published base rate")

	spot := newStaticInfoExchange(t, map[string]string{"userFees": feesJSON})
	setTestCredentials(spot, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, spot, asset.Spot, testSpotPair, "@107")
	spotFee, err := spot.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          testSpotPair,
		IsMaker:       true,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Getting spot maker fee must not error")
	assert.InDelta(t, 0.04, spotFee, 1e-12, "Spot maker fee should use the effective account rate")

	missing := newStaticInfoExchange(t, nil)
	setTestCredentials(missing, &accounts.Credentials{Key: officialSigningAddress})
	_, err = missing.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testPerpetualPair})
	require.ErrorIs(t, err, errPairMappingNotFound, "Unmapped fee pair must return the expected error")

	ambiguous := newStaticInfoExchange(t, nil)
	setTestCredentials(ambiguous, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, ambiguous, asset.Spot, testPerpetualPair, "@1")
	setTestPair(t, ambiguous, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = ambiguous.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testPerpetualPair})
	require.ErrorIs(t, err, errAmbiguousCoinMapping, "Asset-ambiguous fee pair must fail closed")

	dualListedSpot := newStaticInfoExchange(t, nil)
	setTestPair(t, dualListedSpot, asset.Spot, testPerpetualPair, "@1")
	dualListedSpot.setPairMappings(asset.PerpetualContract, []pairMapping{{pair: testPerpetualPair, coin: "BTC"}})
	dualListedFee, err := dualListedSpot.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Spot-enabled dual-listed fee pair must not be ambiguous")
	assert.InDelta(t, 0.14, dualListedFee, 1e-12, "Spot-enabled dual-listed fee pair should use the spot taker rate")

	dualListedPerpetual := newStaticInfoExchange(t, nil)
	setTestPair(t, dualListedPerpetual, asset.PerpetualContract, testPerpetualPair, "BTC")
	dualListedPerpetual.setPairMappings(asset.Spot, []pairMapping{{pair: testPerpetualPair, coin: "@1"}})
	dualListedFee, err = dualListedPerpetual.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Perpetual-enabled dual-listed fee pair must not be ambiguous")
	assert.InDelta(t, 0.09, dualListedFee, 1e-12, "Perpetual-enabled dual-listed fee pair should use the perpetual taker rate")

	offlinePerpetual := newStaticInfoExchange(t, nil)
	setTestPair(t, offlinePerpetual, asset.PerpetualContract, testPerpetualPair, "BTC")
	offlineFee, err := offlinePerpetual.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Offline perpetual fee must not require credentials or network access")
	assert.InDelta(t, 0.09, offlineFee, 1e-12, "Offline perpetual taker fee should use the published base rate")

	credentiallessBuilder := &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	}
	offlineFee, err = offlinePerpetual.GetFeeByType(t.Context(), credentiallessBuilder)
	require.NoError(t, err, "Credentialless cryptocurrency fee must downgrade to an offline estimate")
	assert.Equal(t, exchange.OfflineTradeFee, credentiallessBuilder.FeeType, "Credentialless fee request should be classified as offline")
	assert.InDelta(t, 0.09, offlineFee, 1e-12, "Credentialless perpetual taker fee should use the published base rate")

	offlineSpot := newStaticInfoExchange(t, nil)
	setTestPair(t, offlineSpot, asset.Spot, testSpotPair, "@107")
	offlineFee, err = offlineSpot.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testSpotPair,
		IsMaker:       true,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Offline spot fee must not require credentials or network access")
	assert.InDelta(t, 0.08, offlineFee, 1e-12, "Offline spot maker fee should use the published base rate")
	offlineFee, err = offlineSpot.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.OfflineTradeFee,
		Pair:          testSpotPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.NoError(t, err, "Offline spot taker fee must not require credentials or network access")
	assert.InDelta(t, 0.14, offlineFee, 1e-12, "Offline spot taker fee should use the published base rate")

	for _, tc := range []struct {
		name     string
		asset    asset.Item
		pair     currency.Pair
		expected float64
	}{
		{name: "perpetual", asset: asset.PerpetualContract, pair: testPerpetualPair, expected: 0.09},
		{name: "spot", asset: asset.Spot, pair: testSpotPair, expected: 0.14},
	} {
		t.Run("cold configured "+tc.name, func(t *testing.T) {
			cold := newStaticInfoExchange(t, nil)
			require.NoError(t, cold.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, false), "Updating cold available fee pair must not error")
			require.NoError(t, cold.UpdatePairs(currency.Pairs{tc.pair}, tc.asset, true), "Updating cold enabled fee pair must not error")
			fee, err := cold.GetFeeByType(t.Context(), &exchange.FeeBuilder{
				FeeType:       exchange.OfflineTradeFee,
				Pair:          tc.pair,
				PurchasePrice: 100,
				Amount:        2,
			})
			require.NoError(t, err, "Cold configured offline fee must not require discovered API mappings")
			assert.InDelta(t, tc.expected, fee, 1e-12, "Cold configured offline fee should use its asset base rate")
		})
	}

	invalidAddress := newStaticInfoExchange(t, nil)
	setTestCredentials(invalidAddress, &accounts.Credentials{Key: "not-an-address", Secret: officialSigningTestKey})
	setTestPair(t, invalidAddress, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = invalidAddress.GetFeeByType(t.Context(), &exchange.FeeBuilder{
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          testPerpetualPair,
		PurchasePrice: 100,
		Amount:        2,
	})
	require.ErrorIs(t, err, errInvalidAddress, "Online fee lookup with an invalid account address must fail closed")

	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, failed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = failed.GetFeeByType(t.Context(), &exchange.FeeBuilder{FeeType: exchange.CryptocurrencyTradeFee, Pair: testPerpetualPair})
	require.Error(t, err, "Fee lookup HTTP failure must be returned")
}

func TestGetLeverage(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		"activeAssetData": `{"user":"` + officialSigningAddress + `","coin":"BTC","leverage":{"type":"cross","value":20}}`,
	})
	setTestCredentials(ex, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")

	_, err := ex.GetLeverage(t.Context(), asset.Spot, testSpotPair, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Spot leverage must return the expected error")
	_, err = ex.GetLeverage(t.Context(), asset.PerpetualContract, currency.EMPTYPAIR, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "Empty leverage pair must return the expected error")
	value, err := ex.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, order.UnknownSide)
	require.NoError(t, err, "Getting cross leverage without a mode filter must not error")
	assert.Equal(t, 20.0, value, "Cross leverage value should be returned")
	value, err = ex.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, order.Buy)
	require.NoError(t, err, "Getting cross leverage with a cross filter must not error")
	assert.Equal(t, 20.0, value, "Filtered cross leverage value should be returned")
	_, err = ex.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Isolated, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "Cross leverage queried as isolated must fail closed")

	isolated := newStaticInfoExchange(t, map[string]string{
		"activeAssetData": `{"leverage":{"type":"isolated","value":5}}`,
	})
	setTestCredentials(isolated, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, isolated, asset.PerpetualContract, testPerpetualPair, "BTC")
	value, err = isolated.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Isolated, order.UnknownSide)
	require.NoError(t, err, "Getting isolated leverage with an isolated filter must not error")
	assert.Equal(t, 5.0, value, "Isolated leverage value should be returned")
	_, err = isolated.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Multi, order.UnknownSide)
	require.ErrorIs(t, err, margin.ErrMarginTypeUnsupported, "Isolated leverage queried as cross must fail closed")

	for _, raw := range []string{
		`{"leverage":{"type":"portfolio","value":5}}`,
		`{"leverage":{"type":"cross","value":0}}`,
	} {
		invalid := newStaticInfoExchange(t, map[string]string{"activeAssetData": raw})
		setTestCredentials(invalid, &accounts.Credentials{Key: officialSigningAddress})
		setTestPair(t, invalid, asset.PerpetualContract, testPerpetualPair, "BTC")
		_, err = invalid.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, order.UnknownSide)
		require.Error(t, err, "Invalid leverage response must error")
	}

	missingMapping := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`})
	setTestCredentials(missingMapping, &accounts.Credentials{Key: officialSigningAddress})
	_, err = missingMapping.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, order.UnknownSide)
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing leverage pair mapping must return the expected error")
	missingAddress := newStaticInfoExchange(t, nil)
	setTestPair(t, missingAddress, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = missingAddress.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, order.UnknownSide)
	require.Error(t, err, "Leverage lookup without an account address must error")
	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failed, &accounts.Credentials{Key: officialSigningAddress})
	setTestPair(t, failed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = failed.GetLeverage(t.Context(), asset.PerpetualContract, testPerpetualPair, margin.Unset, order.UnknownSide)
	require.Error(t, err, "Leverage HTTP failure must be returned")
}

func TestGetLatestFundingRates(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err := ex.GetLatestFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil latest funding request must return the expected error")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Spot latest funding request must return the expected error")
	_, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, IncludePredictedRate: true})
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "Predicted funding request must return the expected error")

	rates, err := ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair})
	require.NoError(t, err, "Getting one latest funding rate must not error")
	require.Len(t, rates, 1, "GetLatestFundingRates must return one requested market")
	assert.Equal(t, "0.0001", rates[0].LatestRate.Rate.String(), "Latest funding rate should be converted exactly")
	assert.Equal(t, testPerpetualPair, rates[0].Pair, "Latest funding pair should be returned")
	assert.Equal(t, time.Hour, rates[0].TimeOfNextRate.Sub(rates[0].LatestRate.Time), "Latest funding window should be hourly")
	rates, err = ex.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.NoError(t, err, "Getting all latest funding rates must not error")
	require.Len(t, rates, 1, "GetLatestFundingRates must include the configured market")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	var requestedDEXes []string
	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding scoped funding request should not error") {
			return
		}
		requestedDEXes = append(requestedDEXes, request.DEX)
		response := perpetualContextsJSON
		if request.DEX == "xyz" {
			response = `[{"universe":[{"name":"xyz:XYZ100"}]},[{"funding":"0.0002"}]]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing scoped funding response should not error")
	}))
	hip3.setPairMappings(asset.PerpetualContract, []pairMapping{
		{pair: testPerpetualPair, coin: "BTC"},
		{pair: hip3Pair, coin: "xyz:XYZ100", dex: "xyz", assetID: 110000},
	})
	rates, err = hip3.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.NoError(t, err, "Getting default and HIP-3 latest funding rates must not error")
	require.Len(t, rates, 2, "GetLatestFundingRates must include both scoped DEXes")
	assert.Equal(t, []string{"", "xyz"}, requestedDEXes, "Latest funding should query each required DEX once")
	assert.Equal(t, "0.0002", rates[1].LatestRate.Rate.String(), "HIP-3 funding should use its scoped context")

	cold := newStaticInfoExchange(t, map[string]string{
		"meta":             perpetualMetadataJSON,
		"metaAndAssetCtxs": perpetualContextsJSON,
	})
	rates, err = cold.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.NoError(t, err, "Cold latest funding lookup must discover active markets")
	require.Len(t, rates, 1, "Cold latest funding lookup must include the discovered market")

	empty := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[],"collateralToken":0}`})
	_, err = empty.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "No active markets must return the expected funding error")
	length := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": `[` + perpetualMetadataJSON + `,[]]`})
	setTestPair(t, length, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = length.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Mismatched funding contexts must return the expected error")
	missing := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, missing, asset.PerpetualContract, testPerpetualPair, "ETH")
	_, err = missing.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.ErrorIs(t, err, errAssetContextNotFound, "Missing latest funding context must return the expected error")
	missingMapping := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`})
	_, err = missingMapping.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair})
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing latest funding mapping must return the expected error")
	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, failed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = failed.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.Error(t, err, "Latest funding HTTP failure must be returned")
	coldFailure := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = coldFailure.GetLatestFundingRates(t.Context(), &fundingrate.LatestRateRequest{Asset: asset.PerpetualContract})
	require.Error(t, err, "Cold latest funding discovery failure must be returned")
}

func TestGetHistoricalFundingRates(t *testing.T) {
	start := time.Now().UTC().Truncate(time.Hour).Add(-600 * time.Hour)
	end := start.Add(501 * time.Hour)
	var requests atomic.Int64
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding historical funding request should not error") {
			return
		}
		page := requests.Add(1)
		var response strings.Builder
		response.WriteByte('[')
		count := 1
		firstTime := start.Add(500 * time.Hour)
		if page == 1 {
			count = maximumFundingHistoryCount
			firstTime = start
		}
		for i := range count {
			if i != 0 {
				response.WriteByte(',')
			}
			_, _ = fmt.Fprintf(&response, `{"coin":"BTC","fundingRate":"0.0001","premium":"0","time":%d}`, firstTime.Add(time.Duration(i)*time.Hour).UnixMilli())
		}
		response.WriteByte(']')
		_, err := w.Write([]byte(response.String()))
		assert.NoError(t, err, "Writing historical funding response should not error")
	}))
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")

	_, err := ex.GetHistoricalFundingRates(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil historical funding request must return the expected error")
	for _, tc := range []struct {
		name       string
		request    fundingrate.HistoricalRatesRequest
		expectedIs error
	}{
		{name: "unsupported asset", request: fundingrate.HistoricalRatesRequest{Asset: asset.Spot}, expectedIs: asset.ErrNotSupported},
		{name: "empty pair", request: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract}, expectedIs: currency.ErrCurrencyPairEmpty},
		{name: "predicted", request: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair, IncludePredictedRate: true}, expectedIs: common.ErrFunctionNotSupported},
		{name: "payments", request: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair, IncludePayments: true}, expectedIs: common.ErrFunctionNotSupported},
		{name: "unsupported payment currency", request: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair, PaymentCurrency: currency.BTC, StartDate: start, EndDate: end}, expectedIs: asset.ErrNotSupported},
		{name: "invalid range", request: fundingrate.HistoricalRatesRequest{Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: end, EndDate: start}, expectedIs: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ex.GetHistoricalFundingRates(t.Context(), &tc.request)
			require.ErrorIs(t, err, tc.expectedIs, "Invalid historical funding request must return the expected error")
		})
	}

	result, err := ex.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.PerpetualContract,
		Pair:            testPerpetualPair,
		PaymentCurrency: currency.USDC,
		StartDate:       start,
		EndDate:         end,
	})
	require.NoError(t, err, "Getting paginated historical funding must not error")
	require.Len(t, result.FundingRates, 501, "Historical funding must include both pages")
	assert.Equal(t, int64(2), requests.Load(), "Historical funding should request two pages")
	assert.Equal(t, result.FundingRates[500], result.LatestRate, "Historical funding should expose its latest rate")
	assert.Equal(t, currency.USDC, result.PaymentCurrency, "Historical funding should report USDC payments")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.NewCode("HYPE"))
	hip3 := newStaticInfoExchange(t, map[string]string{
		"fundingHistory": `[{"coin":"xyz:XYZ100","fundingRate":"0.0002","time":` + strconv.FormatInt(start.UnixMilli(), 10) + `}]`,
	})
	setTestPair(t, hip3, asset.PerpetualContract, hip3Pair, "xyz:XYZ100")
	hip3Result, err := hip3.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.PerpetualContract,
		Pair:            hip3Pair,
		PaymentCurrency: currency.NewCode("HYPE"),
		StartDate:       start,
		EndDate:         end,
	})
	require.NoError(t, err, "Getting non-USDC HIP-3 historical funding must not error")
	assert.Equal(t, currency.NewCode("HYPE"), hip3Result.PaymentCurrency, "HIP-3 funding should report its DEX collateral currency")
	_, err = hip3.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset:           asset.PerpetualContract,
		Pair:            hip3Pair,
		PaymentCurrency: currency.USDC,
		StartDate:       start,
		EndDate:         end,
	})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Mismatched HIP-3 funding currency must fail closed")

	empty := newStaticInfoExchange(t, map[string]string{"fundingHistory": `[]`})
	setTestPair(t, empty, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = empty.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: start, EndDate: end,
	})
	require.ErrorIs(t, err, fundingrate.ErrNoFundingRatesFound, "Empty historical funding must return the expected error")
	record := `{"coin":"BTC","fundingRate":"0.0001","premium":"0","time":` + strconv.FormatInt(start.UnixMilli(), 10) + `}`
	oversized := newStaticInfoExchange(t, map[string]string{
		"fundingHistory": `[` + strings.Repeat(record+",", maximumFundingHistoryCount) + record + `]`,
	})
	setTestPair(t, oversized, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = oversized.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: start, EndDate: end,
	})
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Oversized historical funding page must fail closed")
	malformed := newStaticInfoExchange(t, map[string]string{
		"fundingHistory": `[{"coin":"ETH","fundingRate":"0.1","time":` + strconv.FormatInt(start.UnixMilli(), 10) + `}]`,
	})
	setTestPair(t, malformed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = malformed.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: start, EndDate: end,
	})
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Mismatched historical funding coin must fail closed")
	missingMapping := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`})
	_, err = missingMapping.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: start, EndDate: end,
	})
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing historical funding mapping must return the expected error")
	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, failed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = failed.GetHistoricalFundingRates(t.Context(), &fundingrate.HistoricalRatesRequest{
		Asset: asset.PerpetualContract, Pair: testPerpetualPair, StartDate: start, EndDate: end,
	})
	require.Error(t, err, "Historical funding HTTP failure must be returned")
}

func TestGetOpenInterest(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	requested := key.PairAsset{Base: testPerpetualPair.Base.Item, Quote: testPerpetualPair.Quote.Item, Asset: asset.PerpetualContract}
	result, err := ex.GetOpenInterest(t.Context(), requested)
	require.NoError(t, err, "Getting requested open interest must not error")
	require.Len(t, result, 1, "GetOpenInterest must contain one requested market")
	assert.Equal(t, 10.0, result[0].OpenInterest, "Open interest should be converted")
	assert.True(t, result[0].Key.MatchesPairAsset(testPerpetualPair, asset.PerpetualContract), "Open-interest key should identify the market")
	result, err = ex.GetOpenInterest(t.Context())
	require.NoError(t, err, "Getting all open interest must not error")
	require.Len(t, result, 1, "GetOpenInterest must include the configured market")

	hip3Pair := currency.NewPair(currency.NewCode("xyz:XYZ100"), currency.USDC)
	var requestedDEXes []string
	hip3 := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding scoped open-interest request should not error") {
			return
		}
		requestedDEXes = append(requestedDEXes, request.DEX)
		response := perpetualContextsJSON
		if request.DEX == "xyz" {
			response = `[{"universe":[{"name":"xyz:XYZ100"}]},[{"openInterest":"20"}]]`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing scoped open-interest response should not error")
	}))
	hip3.setPairMappings(asset.PerpetualContract, []pairMapping{
		{pair: testPerpetualPair, coin: "BTC"},
		{pair: hip3Pair, coin: "xyz:XYZ100", dex: "xyz", assetID: 110000},
	})
	result, err = hip3.GetOpenInterest(t.Context())
	require.NoError(t, err, "Getting default and HIP-3 open interest must not error")
	require.Len(t, result, 2, "GetOpenInterest must include both scoped DEXes")
	assert.Equal(t, []string{"", "xyz"}, requestedDEXes, "Open interest should query each required DEX once")
	assert.Equal(t, 20.0, result[1].OpenInterest, "HIP-3 open interest should use its scoped context")

	_, err = ex.GetOpenInterest(t.Context(), key.PairAsset{Base: testSpotPair.Base.Item, Quote: testSpotPair.Quote.Item, Asset: asset.Spot})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Spot open interest must return the expected error")
	missingMapping := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[]}`})
	_, err = missingMapping.GetOpenInterest(t.Context(), requested)
	require.ErrorIs(t, err, errPairMappingNotFound, "Missing open-interest mapping must return the expected error")
	length := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": `[` + perpetualMetadataJSON + `,[]]`})
	setTestPair(t, length, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = length.GetOpenInterest(t.Context())
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Mismatched open-interest contexts must return the expected error")
	missing := newStaticInfoExchange(t, map[string]string{"metaAndAssetCtxs": perpetualContextsJSON})
	setTestPair(t, missing, asset.PerpetualContract, testPerpetualPair, "ETH")
	_, err = missing.GetOpenInterest(t.Context())
	require.ErrorIs(t, err, errAssetContextNotFound, "Missing open-interest context must return the expected error")
	cold := newStaticInfoExchange(t, map[string]string{
		"meta":             perpetualMetadataJSON,
		"metaAndAssetCtxs": perpetualContextsJSON,
	})
	result, err = cold.GetOpenInterest(t.Context())
	require.NoError(t, err, "Cold open-interest lookup must discover active markets")
	require.Len(t, result, 1, "Cold open-interest lookup must include the discovered market")
	empty := newStaticInfoExchange(t, map[string]string{"meta": `{"universe":[],"collateralToken":0}`})
	result, err = empty.GetOpenInterest(t.Context())
	require.NoError(t, err, "Getting open interest without active markets must not error")
	assert.Empty(t, result, "Open interest without active markets should be empty")
	failed := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestPair(t, failed, asset.PerpetualContract, testPerpetualPair, "BTC")
	_, err = failed.GetOpenInterest(t.Context())
	require.Error(t, err, "Open-interest HTTP failure must be returned")
	coldFailure := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = coldFailure.GetOpenInterest(t.Context())
	require.Error(t, err, "Cold open-interest discovery failure must be returned")
}

func TestGetUserNonFundingLedgerUpdatesPaginated(t *testing.T) {
	start := time.UnixMilli(1700000000000).UTC()
	end := start.Add(time.Minute)
	var calls atomic.Int32
	var requestedStarts []int64
	success := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding paginated ledger request should not error") {
			return
		}
		requestedStarts = append(requestedStarts, request.StartTime)
		call := calls.Add(1)
		count := maximumUserLedgerHistoryCount
		if call > 1 {
			count = 1
		}
		updates := make([]map[string]any, count)
		for i := range updates {
			updates[i] = map[string]any{
				"time": request.StartTime + int64(i),
				"hash": fmt.Sprintf("0x%d-%d", call, i),
				"delta": map[string]any{
					"type": "deposit",
					"usdc": "1",
				},
			}
		}
		body, err := json.Marshal(updates)
		if !assert.NoError(t, err, "Encoding paginated ledger response should not error") {
			return
		}
		_, err = w.Write(body)
		assert.NoError(t, err, "Writing paginated ledger response should not error")
	}))
	result, err := success.getUserNonFundingLedgerUpdatesPaginated(t.Context(), officialSigningAddress, start, end)
	require.NoError(t, err, "Fetching multiple ledger pages must not error")
	assert.Len(t, result, maximumUserLedgerHistoryCount+1, "Every ledger page should be retained")
	assert.Equal(t, int32(2), calls.Load(), "A full ledger page should advance the cursor")
	require.Len(t, requestedStarts, 2, "Ledger fixture must receive two requests")
	assert.Equal(t, start.Add((maximumUserLedgerHistoryCount-1)*time.Millisecond).UnixMilli(), requestedStarts[1], "Ledger pagination should reuse the inclusive terminal timestamp")

	var duplicateCalls atomic.Int32
	exactDuplicate := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding duplicate ledger request should not error") {
			return
		}
		call := duplicateCalls.Add(1)
		count := maximumUserLedgerHistoryCount
		if call > 1 {
			count = 1
		}
		updates := make([]map[string]any, count)
		for i := range updates {
			recordTime := start.Add(time.Duration(i) * time.Millisecond).UnixMilli()
			hash := fmt.Sprintf("0x%d", i)
			if call > 1 {
				recordTime = request.StartTime
				hash = fmt.Sprintf("0x%d", maximumUserLedgerHistoryCount-1)
			}
			updates[i] = map[string]any{
				"time":  recordTime,
				"hash":  hash,
				"delta": map[string]any{"type": "deposit", "usdc": "1"},
			}
		}
		body, err := json.Marshal(updates)
		if !assert.NoError(t, err, "Encoding duplicate ledger response should not error") {
			return
		}
		_, err = w.Write(body)
		assert.NoError(t, err, "Writing duplicate ledger response should not error")
	}))
	result, err = exactDuplicate.getUserNonFundingLedgerUpdatesPaginated(t.Context(), officialSigningAddress, start, end)
	require.NoError(t, err, "Overlapping exact terminal ledger record must not error")
	assert.Len(t, result, maximumUserLedgerHistoryCount, "Overlapping exact terminal ledger record should be deduplicated")
	assert.Equal(t, int32(2), duplicateCalls.Load(), "Exact-terminal deduplication should still request the next inclusive page")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = failing.getUserNonFundingLedgerUpdatesPaginated(t.Context(), officialSigningAddress, start, end)
	require.Error(t, err, "Ledger page request failure must be returned")

	for _, tc := range []struct {
		name  string
		times []int64
		count int
	}{
		{name: "oversized page", count: maximumUserLedgerHistoryCount + 1},
		{name: "before cursor", times: []int64{start.Add(-time.Millisecond).UnixMilli()}},
		{name: "after end", times: []int64{end.Add(time.Millisecond).UnixMilli()}},
		{name: "decreasing", times: []int64{start.Add(2 * time.Millisecond).UnixMilli(), start.Add(time.Millisecond).UnixMilli()}},
		{name: "cursor does not advance", count: maximumUserLedgerHistoryCount, times: []int64{start.UnixMilli()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				count := tc.count
				if count == 0 {
					count = len(tc.times)
				}
				updates := make([]map[string]any, count)
				for i := range updates {
					recordTime := start.UnixMilli()
					if len(tc.times) > 1 {
						recordTime = tc.times[i]
					} else if len(tc.times) == 1 {
						recordTime = tc.times[0]
					}
					updates[i] = map[string]any{
						"time":  recordTime,
						"hash":  fmt.Sprintf("0x%d", i),
						"delta": map[string]any{"type": "deposit", "usdc": "1"},
					}
				}
				body, err := json.Marshal(updates)
				if !assert.NoError(t, err, "Encoding invalid ledger fixture should not error") {
					return
				}
				_, err = w.Write(body)
				assert.NoError(t, err, "Writing invalid ledger fixture should not error")
			}))
			_, err := invalid.getUserNonFundingLedgerUpdatesPaginated(t.Context(), officialSigningAddress, start, end)
			require.ErrorIs(t, err, errUnexpectedResponseLength, "Malformed ledger page must return the expected error")
		})
	}
}

func TestConvertUserLedgerUpdate(t *testing.T) {
	ex := new(Exchange)
	ex.Name = "HyperliquidTest"
	_, err := ex.convertUserLedgerUpdate(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil ledger update must return the expected error")
	_, err = ex.convertUserLedgerUpdate(&UserLedgerUpdate{})
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Empty ledger type must return the expected error")

	recordTime := time.UnixMilli(1700000000000).UTC()
	for _, tc := range []struct {
		name                string
		delta               UserLedgerDelta
		expectedCurrency    string
		expectedAmount      float64
		expectedDescription string
	}{
		{
			name:                "deposit",
			delta:               UserLedgerDelta{Type: "deposit", USDC: 10},
			expectedCurrency:    "USDC",
			expectedAmount:      10,
			expectedDescription: "deposit",
		},
		{
			name:                "spot transfer",
			delta:               UserLedgerDelta{Type: "spotTransfer", Token: "HYPE:0x96", Amount: 2},
			expectedCurrency:    "HYPE",
			expectedAmount:      2,
			expectedDescription: "spotTransfer",
		},
		{
			name:                "spot genesis without token",
			delta:               UserLedgerDelta{Type: "spotGenesis", Amount: 3},
			expectedCurrency:    "USDC",
			expectedAmount:      3,
			expectedDescription: "spotGenesis",
		},
		{
			name: "send asset",
			delta: UserLedgerDelta{
				Type: "send", Token: "USDC", Amount: 4, SourceDEX: "spot", DestinationDEX: "xyz",
			},
			expectedCurrency:    "USDC",
			expectedAmount:      4,
			expectedDescription: "send: spot to xyz",
		},
		{
			name:                "rewards claim",
			delta:               UserLedgerDelta{Type: "rewardsClaim", Amount: 5},
			expectedCurrency:    "USDC",
			expectedAmount:      5,
			expectedDescription: "rewardsClaim",
		},
		{
			name:                "vault withdrawal",
			delta:               UserLedgerDelta{Type: "vaultWithdraw", NetWithdrawnUSD: 6},
			expectedCurrency:    "USDC",
			expectedAmount:      6,
			expectedDescription: "vaultWithdraw",
		},
		{
			name:                "perpetual to spot",
			delta:               UserLedgerDelta{Type: "accountClassTransfer", USDC: 7},
			expectedCurrency:    "USDC",
			expectedAmount:      7,
			expectedDescription: "perpetual to spot",
		},
		{
			name:                "spot to perpetual",
			delta:               UserLedgerDelta{Type: "accountClassTransfer", USDC: 8, ToPerp: true},
			expectedCurrency:    "USDC",
			expectedAmount:      8,
			expectedDescription: "spot to perpetual",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.delta.Fee = 0.5
			tc.delta.User = officialSigningAddress
			tc.delta.Destination = testOtherAddress
			result, err := ex.convertUserLedgerUpdate(&UserLedgerUpdate{
				Delta: tc.delta,
				Hash:  "0xhash",
				Time:  types.Time(recordTime),
			})
			require.NoError(t, err, "Converting a valid ledger update must not error")
			assert.Equal(t, "HyperliquidTest", result.ExchangeName, "Exchange name should be retained")
			assert.Equal(t, "processed", result.Status, "Ledger status should identify a processed L1 update")
			assert.Equal(t, tc.expectedCurrency, result.Currency, "Ledger currency should match")
			assert.Equal(t, tc.expectedAmount, result.Amount, "Ledger amount should match")
			assert.Equal(t, 0.5, result.Fee, "Ledger fee should match")
			assert.Equal(t, tc.expectedDescription, result.Description, "Ledger description should match")
			assert.Equal(t, officialSigningAddress, result.CryptoFromAddress, "Ledger source should match")
			assert.Equal(t, testOtherAddress, result.CryptoToAddress, "Ledger destination should match")
			assert.Equal(t, "0xhash", result.TransferID, "Ledger transfer ID should use the L1 hash")
			assert.Equal(t, recordTime, result.Timestamp, "Ledger timestamp should match")
		})
	}
}

func TestGetAccountFundingHistory(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.GetAccountFundingHistory(t.Context())
	require.Error(t, err, "Account history without credentials must error")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failing, &accounts.Credentials{Key: officialSigningAddress})
	_, err = failing.GetAccountFundingHistory(t.Context())
	require.Error(t, err, "Account ledger request failure must be returned")

	invalid := newTransferTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		"userNonFundingLedgerUpdates": `[{"time":1700000000000,"hash":"0x1","delta":{"type":""}}]`,
	}, nil)
	_, err = invalid.GetAccountFundingHistory(t.Context())
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Malformed account ledger update must return the expected error")

	success := newTransferTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		"userNonFundingLedgerUpdates": `[
			{"time":1700000000001,"hash":"0x1","delta":{"type":"deposit","usdc":"1"}},
			{"time":1700000000002,"hash":"0x2","delta":{"type":"internalTransfer","usdc":"2","user":"` + officialSigningAddress + `","destination":"` + testOtherAddress + `"}}
		]`,
	}, nil)
	result, err := success.GetAccountFundingHistory(t.Context())
	require.NoError(t, err, "Getting account ledger history must not error")
	require.Len(t, result, 2, "GetAccountFundingHistory must convert every account ledger update")
	assert.Equal(t, "0x1", result[0].TransferID, "Account ledger history should be sorted oldest first")
	assert.Equal(t, "0x2", result[1].TransferID, "Later account ledger update should sort last")
}

func TestGetWithdrawalsHistory(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	result, err := ex.GetWithdrawalsHistory(t.Context(), currency.BTC, asset.PerpetualContract)
	require.NoError(t, err, "Filtering withdrawal history by an unsupported currency must not error")
	assert.Empty(t, result, "GetWithdrawalsHistory should return no records for an unsupported currency")
	_, err = ex.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.Spot)
	require.Error(t, err, "Withdrawal history for an asset-agnostic USDC filter without credentials must error")
	_, err = ex.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.PerpetualContract)
	require.Error(t, err, "Withdrawal history without credentials must error")

	failing := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(failing, &accounts.Credentials{Key: officialSigningAddress})
	_, err = failing.GetWithdrawalsHistory(t.Context(), currency.USDC, asset.PerpetualContract)
	require.Error(t, err, "Withdrawal ledger request failure must be returned")

	success := newTransferTestExchange(t, &accounts.Credentials{Key: officialSigningAddress}, map[string]string{
		"userNonFundingLedgerUpdates": `[
			{"time":1700000000000,"hash":"0xdeposit","delta":{"type":"deposit","usdc":"3"}},
			{"time":1700000000001,"hash":"0xwithdraw","delta":{"type":"withdraw","usdc":"-2","fee":"-1","nonce":7}}
		]`,
	}, nil)
	success.Config.UseSandbox = true
	result, err = success.GetWithdrawalsHistory(t.Context(), currency.EMPTYCODE, asset.Empty)
	require.NoError(t, err, "Getting bridge withdrawal history must not error")
	require.Len(t, result, 1, "GetWithdrawalsHistory must return only bridge withdrawals")
	assert.Equal(t, "0xwithdraw", result[0].TransferID, "Withdrawal ID should use the L1 hash")
	assert.Equal(t, 2.0, result[0].Amount, "Withdrawal amount should be normalised positive")
	assert.Equal(t, 1.0, result[0].Fee, "Withdrawal fee should be normalised positive")
	assert.Equal(t, "USDC", result[0].Currency, "Withdrawal currency should be USDC")
	assert.Equal(t, "Arbitrum Sepolia", result[0].CryptoChain, "Sandbox withdrawal history should identify Arbitrum Sepolia")
}

func TestGetAvailableTransferChains(t *testing.T) {
	mainnet := new(Exchange)
	mainnet.Config = new(config.Exchange)
	_, err := mainnet.GetAvailableTransferChains(t.Context(), currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "Empty transfer currency must return the expected error")
	_, err = mainnet.GetAvailableTransferChains(t.Context(), currency.BTC)
	require.ErrorIs(t, err, errTransferCurrencyInvalid, "Unsupported transfer currency must return the expected error")
	chains, err := mainnet.GetAvailableTransferChains(t.Context(), currency.USDC)
	require.NoError(t, err, "Getting mainnet transfer chains must not error")
	assert.Equal(t, []string{"Arbitrum"}, chains, "Mainnet transfer chain should be Arbitrum")

	sandbox := new(Exchange)
	sandbox.Config = &config.Exchange{UseSandbox: true}
	chains, err = sandbox.GetAvailableTransferChains(t.Context(), currency.USDC)
	require.NoError(t, err, "Getting sandbox transfer chains must not error")
	assert.Equal(t, []string{"Arbitrum Sepolia"}, chains, "Sandbox transfer chain should be Arbitrum Sepolia")
}

func TestWithdrawCryptocurrencyFunds(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	_, err := ex.WithdrawCryptocurrencyFunds(t.Context(), nil)
	require.ErrorIs(t, err, withdraw.ErrRequestCannotBeNil, "Nil withdrawal must return the expected error")

	request := withdraw.Request{
		Exchange: "Hyperliquid",
		Currency: currency.USDC,
		Amount:   2,
		Type:     withdraw.Crypto,
		Crypto:   withdraw.CryptoRequest{Address: testOtherAddress},
	}
	invalidCurrency := request
	invalidCurrency.Currency = currency.BTC
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &invalidCurrency)
	require.ErrorIs(t, err, errTransferCurrencyInvalid, "Unsupported withdrawal currency must return the expected error")
	addressTag := request
	addressTag.Crypto.AddressTag = "tag"
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &addressTag)
	require.ErrorIs(t, err, errWithdrawalAddressTag, "Withdrawal address tag must be rejected")
	fee := request
	fee.Crypto.FeeAmount = 1
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &fee)
	require.ErrorIs(t, err, errWithdrawalFeeInput, "Caller-supplied withdrawal fee must be rejected")
	invalidChain := request
	invalidChain.Crypto.Chain = "Ethereum"
	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &invalidChain)
	require.ErrorIs(t, err, errBridgeChainInvalid, "Wrong bridge chain must be rejected")

	_, err = ex.WithdrawCryptocurrencyFunds(t.Context(), &request)
	require.Error(t, err, "Withdrawal without credentials must error")

	var captured signedActionRequest
	success := newTransferTestExchange(t, &accounts.Credentials{
		Key: officialSigningAddress, Secret: officialSigningTestKey,
	}, nil, &captured)
	request.Crypto.Chain = "arbitrum"
	response, err := success.WithdrawCryptocurrencyFunds(t.Context(), &request)
	require.NoError(t, err, "Submitting a bridge withdrawal must not error")
	assert.Equal(t, "Hyperliquid", response.Name, "Withdrawal response exchange should match")
	assert.NotEmpty(t, response.ID, "Withdrawal response should include the action nonce")
	assert.Equal(t, "submitted", response.Status, "Withdrawal response should identify submission")
	assert.Equal(t, "withdraw3", getCapturedAction(t, &captured)["type"], "External withdrawal should use the bridge action")

	request.InternalTransfer = true
	response, err = success.WithdrawCryptocurrencyFunds(t.Context(), &request)
	require.NoError(t, err, "Submitting an internal USDC send must not error")
	assert.NotEmpty(t, response.ID, "Internal-transfer response should include the action nonce")
	assert.Equal(t, "usdSend", getCapturedAction(t, &captured)["type"], "Internal withdrawal should use a Core USDC send")

	actionFailure := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	setTestCredentials(actionFailure, &accounts.Credentials{Key: officialSigningAddress, Secret: officialSigningTestKey})
	_, err = actionFailure.WithdrawCryptocurrencyFunds(t.Context(), &request)
	require.Error(t, err, "Signed withdrawal action failure must be returned")
}

func TestUnsupportedMethods(t *testing.T) {
	ex := new(Exchange)
	ctx := t.Context()
	assertUnsupported := func(err error) {
		t.Helper()
		require.ErrorIs(t, err, common.ErrFunctionNotSupported, "Public-only method must return the expected unsupported error")
	}

	_, err := ex.GetHistoricTrades(ctx, testSpotPair, asset.Spot, time.Time{}, time.Time{})
	assertUnsupported(err)
	_, err = ex.GetServerTime(ctx, asset.Spot)
	assertUnsupported(err)
	_, err = ex.GetDepositAddress(ctx, currency.USDC, "", "")
	assertUnsupported(err)
	_, err = ex.WithdrawFiatFunds(ctx, nil)
	assertUnsupported(err)
	_, err = ex.WithdrawFiatFundsToInternationalBank(ctx, nil)
	assertUnsupported(err)
	_, err = ex.GetHistoricCandlesExtended(ctx, testSpotPair, asset.Spot, kline.OneMin, time.Time{}, time.Time{})
	assertUnsupported(err)
	_, err = ex.GetFuturesContractDetails(ctx, asset.PerpetualContract)
	assertUnsupported(err)
	_, err = ex.GetCurrencyTradeURL(ctx, asset.Spot, testSpotPair)
	assertUnsupported(err)
	require.ErrorIs(t, ex.UpdateOrderExecutionLimits(ctx, asset.Spot), common.ErrNotYetImplemented, "Execution-limit bootstrap method must return the expected not-implemented error")
}

func TestUpdateOrderbookUsesValidationSetting(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"l2Book": bookJSON})
	setTestPair(t, ex, asset.PerpetualContract, testPerpetualPair, "BTC")
	ex.Name = "HyperliquidValidationDisabled"
	ex.ValidateOrderbook = false
	book, err := ex.UpdateOrderbook(t.Context(), testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Updating an orderbook with validation disabled must not error")
	assert.False(t, book.ValidateOrderbook, "Orderbook should inherit the exchange validation setting")

	cached, err := orderbook.Get(ex.Name, testPerpetualPair, asset.PerpetualContract)
	require.NoError(t, err, "Getting the cached orderbook must not error")
	assert.False(t, cached.ValidateOrderbook, "Cached orderbook should retain the exchange validation setting")
}

func TestUnsupportedMethodsIgnoreContext(t *testing.T) {
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	ex := new(Exchange)
	_, err := ex.GetServerTime(cancelled, asset.Spot)
	require.ErrorIs(t, err, common.ErrFunctionNotSupported, "Unsupported method must remain deterministic for a cancelled context")
}
