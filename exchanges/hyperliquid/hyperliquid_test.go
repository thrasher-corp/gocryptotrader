package hyperliquid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

const (
	testBuilderDEXName    = "xyz"
	perpetualMetadataJSON = `{"universe":[{"name":"BTC","szDecimals":5,"maxLeverage":40,"marginTableId":56}],"collateralToken":0,"marginTables":[]}`
	spotMetadataJSON      = `{"universe":[{"tokens":[150,0],"name":"@107","index":107,"isCanonical":true}],"tokens":[{"name":"USDC","szDecimals":8,"weiDecimals":8,"index":0,"tokenId":"0x0","isCanonical":true,"evmContract":null,"fullName":"USD Coin","deployerTradingFeeShare":"0"},{"name":"HYPE","szDecimals":2,"weiDecimals":8,"index":150,"tokenId":"0x96","isCanonical":true,"evmContract":{"address":"0x1"},"fullName":"Hyperliquid","deployerTradingFeeShare":"0"}]}`
	perpetualContextsJSON = `[` + perpetualMetadataJSON + `,[{"funding":"0.0001","openInterest":"10","prevDayPx":"99","dayNtlVlm":"1000","premium":"0.001","oraclePx":"100","markPx":"101","midPx":"100.5","impactPxs":["100","101"],"dayBaseVlm":"10"}]]`
	spotContextsJSON      = `[` + spotMetadataJSON + `,[{"prevDayPx":"9","dayNtlVlm":"100","markPx":"10","midPx":"9.5","circulatingSupply":"1000","totalSupply":"1000","coin":"@107","dayBaseVlm":"10"}]]`
	bookJSON              = `{"coin":"BTC","levels":[[{"px":"100","sz":"2","n":1}],[{"px":"101","sz":"3","n":2}]],"time":1700000000000}`
	tradesJSON            = `[{"coin":"BTC","side":"A","px":"100","sz":"2","time":1700000000000,"hash":"0x1","tid":7,"users":["0x2"]}]`
	candlesJSON           = `[{"t":1700000000000,"T":1700000059999,"s":"BTC","i":"1m","o":"100","c":"101","h":"102","l":"99","v":"5","n":3}]`
)

func newHTTPTestExchange(t *testing.T, handler http.Handler) *Exchange {
	t.Helper()
	ex := new(Exchange)
	ex.SetDefaults()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "Getting the standard config must not error")
	require.NoError(t, ex.Setup(cfg), "Setting up the test exchange must not error")
	require.NoError(t, ex.DisableRateLimiter(), "Disabling the test rate limiter must not error")

	server := httptest.NewServer(handler)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), server.URL), "Setting the spot test URL must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), server.URL), "Setting the futures test URL must not error")
	t.Cleanup(func() {
		server.Close()
		assert.NoError(t, ex.Shutdown(), "Shutting down the test exchange should not error")
	})
	return ex
}

func newStaticInfoExchange(t *testing.T, responses map[string]string) *Exchange {
	t.Helper()
	return newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("request method should be POST: %s", r.Method)
			http.Error(w, "unexpected method", http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/info" {
			t.Errorf("request path should be /info: %s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		var payload infoRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decoding request body should not error: %v", err)
			http.Error(w, "invalid body", http.StatusBadRequest)
			return
		}
		response, ok := responses[payload.Type]
		if !ok && payload.Type == infoTypePerpetualDEXs {
			response = `[null]`
			ok = true
		}
		if !ok {
			t.Errorf("request type should have a configured response: %s", payload.Type)
			http.Error(w, "unexpected request type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(response)); err != nil {
			t.Errorf("writing response should not error: %v", err)
		}
	}))
}

func TestGetRESTEndpoint(t *testing.T) {
	for _, tc := range []struct {
		name     string
		asset    asset.Item
		expected exchange.URL
		errorIs  error
	}{
		{name: "spot", asset: asset.Spot, expected: exchange.RestSpot},
		{name: "perpetual", asset: asset.PerpetualContract, expected: exchange.RestFutures},
		{name: "unsupported", asset: asset.Options, errorIs: asset.ErrNotSupported},
	} {
		t.Run(tc.name, func(t *testing.T) {
			endpoint, err := getRESTEndpoint(tc.asset)
			require.ErrorIs(t, err, tc.errorIs, "Getting a REST endpoint must return the expected error")
			assert.Equal(t, tc.expected, endpoint, "REST endpoint should match the asset type")
		})
	}
}

func TestSetup(t *testing.T) {
	ex := new(Exchange)
	ex.SetDefaults()
	cfg, err := ex.GetStandardConfig()
	require.NoError(t, err, "Getting the standard config must not error")

	disabled := *cfg
	disabled.Enabled = false
	require.NoError(t, ex.Setup(&disabled), "Setting up a disabled exchange must not error")
	assert.False(t, ex.IsEnabled(), "Disabled exchange should remain disabled")

	cfg.Enabled = true
	cfg.API.AuthenticatedSupport = true
	cfg.API.AuthenticatedWebsocketSupport = true
	cfg.API.Credentials.Key = officialSigningAddress
	require.NoError(t, ex.Setup(cfg), "Setting up an enabled exchange must not error")
	assert.True(t, ex.IsEnabled(), "Enabled exchange should remain enabled")
	assert.True(t, ex.API.AuthenticatedSupport, "Configured authenticated REST support should remain enabled")
	assert.True(t, ex.API.AuthenticatedWebsocketSupport, "Configured account websocket support should remain enabled")
	credentials, err := ex.GetCredentials(t.Context())
	require.NoError(t, err, "Getting configured credentials must not error")
	assert.Equal(t, officialSigningAddress, credentials.Key, "Setup should load the configured watch address")
	assert.True(t, ex.isMainnetEnvironment(), "The default configuration should use the mainnet signing environment")

	sandbox := new(Exchange)
	sandbox.SetDefaults()
	sandboxConfig, err := sandbox.GetStandardConfig()
	require.NoError(t, err, "Getting a sandbox config must not error")
	sandboxConfig.UseSandbox = true
	require.NoError(t, sandbox.Setup(sandboxConfig), "Setting up the official sandbox must not error")
	assert.False(t, sandbox.isMainnetEnvironment(), "Sandbox configuration should use the testnet signing environment")
	for _, endpoint := range []struct {
		kind     exchange.URL
		expected string
	}{
		{kind: exchange.RestSpot, expected: hyperliquidTestnetAPIURL},
		{kind: exchange.RestFutures, expected: hyperliquidTestnetAPIURL},
		{kind: exchange.WebsocketSpot, expected: hyperliquidTestnetWebsocketURL},
	} {
		runningURL, err := sandbox.API.Endpoints.GetURL(endpoint.kind)
		require.NoError(t, err, "Getting an official sandbox endpoint must not error")
		assert.Equal(t, endpoint.expected, runningURL, "Sandbox should replace the matching production endpoint")
	}
	require.NoError(t, sandbox.Shutdown(), "Shutting down the sandbox exchange must not error")

	trailingSlashSandbox := new(Exchange)
	trailingSlashSandbox.SetDefaults()
	trailingSlashConfig, err := trailingSlashSandbox.GetStandardConfig()
	require.NoError(t, err, "Getting a trailing-slash sandbox config must not error")
	trailingSlashConfig.UseSandbox = true
	trailingSlashConfig.API.Endpoints = map[string]string{
		exchange.RestSpot.String():      hyperliquidAPIURL + "/",
		exchange.RestFutures.String():   hyperliquidAPIURL + "/",
		exchange.WebsocketSpot.String(): hyperliquidWebsocketURL + "/",
	}
	require.NoError(t, trailingSlashSandbox.Setup(trailingSlashConfig), "Setting up official production endpoints with trailing slashes must not error")
	for _, endpoint := range []struct {
		kind     exchange.URL
		expected string
	}{
		{kind: exchange.RestSpot, expected: hyperliquidTestnetAPIURL},
		{kind: exchange.RestFutures, expected: hyperliquidTestnetAPIURL},
		{kind: exchange.WebsocketSpot, expected: hyperliquidTestnetWebsocketURL},
	} {
		runningURL, err := trailingSlashSandbox.API.Endpoints.GetURL(endpoint.kind)
		require.NoError(t, err, "Getting a trailing-slash sandbox endpoint must not error")
		assert.Equal(t, endpoint.expected, runningURL, "Sandbox should normalise the production endpoint before selecting testnet")
	}
	require.NoError(t, trailingSlashSandbox.Shutdown(), "Shutting down the trailing-slash sandbox exchange must not error")

	customSandbox := new(Exchange)
	customSandbox.SetDefaults()
	customConfig, err := customSandbox.GetStandardConfig()
	require.NoError(t, err, "Getting a custom sandbox config must not error")
	customConfig.UseSandbox = true
	customConfig.API.Endpoints = map[string]string{
		exchange.RestSpot.String():      "https://hyperliquid-testnet.internal.example",
		exchange.RestFutures.String():   "https://hyperliquid-testnet.internal.example",
		exchange.WebsocketSpot.String(): "wss://hyperliquid-testnet.internal.example/ws",
	}
	require.NoError(t, customSandbox.Setup(customConfig), "Setting up a custom testnet gateway must not error")
	customRESTURL, err := customSandbox.API.Endpoints.GetURL(exchange.RestSpot)
	require.NoError(t, err, "Getting a custom REST endpoint must not error")
	assert.Equal(t, customConfig.API.Endpoints[exchange.RestSpot.String()], customRESTURL, "Sandbox setup should preserve a custom gateway")
	assert.False(t, customSandbox.isMainnetEnvironment(), "A custom gateway should use the explicitly configured sandbox environment")
	require.NoError(t, customSandbox.Shutdown(), "Shutting down the custom sandbox exchange must not error")

	mismatchedEnvironment := new(Exchange)
	mismatchedEnvironment.SetDefaults()
	mismatchedConfig, err := mismatchedEnvironment.GetStandardConfig()
	require.NoError(t, err, "Getting an environment-mismatch config must not error")
	mismatchedConfig.API.Endpoints = map[string]string{
		exchange.RestSpot.String():      hyperliquidTestnetAPIURL,
		exchange.RestFutures.String():   hyperliquidTestnetAPIURL,
		exchange.WebsocketSpot.String(): hyperliquidTestnetWebsocketURL,
	}
	require.ErrorIs(t, mismatchedEnvironment.Setup(mismatchedConfig), errEndpointEnvironment, "Official testnet endpoints without sandbox mode must fail closed")
	require.NoError(t, mismatchedEnvironment.Shutdown(), "Shutting down the environment-mismatch exchange must not error")

	missingEndpoints := new(Exchange)
	missingEndpoints.SetDefaults()
	missingConfig, err := missingEndpoints.GetStandardConfig()
	require.NoError(t, err, "Getting a missing-endpoint config must not error")
	missingConfig.UseSandbox = true
	missingEndpoints.API.Endpoints = missingEndpoints.NewEndpoints()
	require.Error(t, missingEndpoints.Setup(missingConfig), "Sandbox setup without endpoints must error")
	require.NoError(t, missingEndpoints.Shutdown(), "Shutting down the missing-endpoint exchange must not error")

	invalidEndpoint := new(Exchange)
	invalidEndpoint.SetDefaults()
	invalidEndpointConfig, err := invalidEndpoint.GetStandardConfig()
	require.NoError(t, err, "Getting an invalid-endpoint config must not error")
	invalidEndpointConfig.API.Endpoints = map[string]string{"invalid": "https://example.com"}
	require.Error(t, invalidEndpoint.Setup(invalidEndpointConfig), "Setup with an invalid configured endpoint must error")
	require.NoError(t, invalidEndpoint.Shutdown(), "Shutting down the invalid-endpoint exchange must not error")

	missingWebsocketEndpoint := new(Exchange)
	missingWebsocketEndpoint.SetDefaults()
	missingWebsocketConfig, err := missingWebsocketEndpoint.GetStandardConfig()
	require.NoError(t, err, "Getting a missing-websocket config must not error")
	missingWebsocketEndpoint.API.Endpoints = missingWebsocketEndpoint.NewEndpoints()
	require.Error(t, missingWebsocketEndpoint.Setup(missingWebsocketConfig), "Setup without a websocket endpoint must error")
	require.NoError(t, missingWebsocketEndpoint.Shutdown(), "Shutting down the missing-websocket exchange must not error")

	invalidTrafficTimeout := new(Exchange)
	invalidTrafficTimeout.SetDefaults()
	invalidTrafficConfig, err := invalidTrafficTimeout.GetStandardConfig()
	require.NoError(t, err, "Getting an invalid-traffic config must not error")
	invalidTrafficConfig.WebsocketTrafficTimeout = time.Millisecond
	require.Error(t, invalidTrafficTimeout.Setup(invalidTrafficConfig), "Setup with an invalid websocket traffic timeout must error")
	require.NoError(t, invalidTrafficTimeout.Shutdown(), "Shutting down the invalid-traffic exchange must not error")

	connectionFailure := new(Exchange)
	connectionFailure.SetDefaults()
	connectionFailureConfig, err := connectionFailure.GetStandardConfig()
	require.NoError(t, err, "Getting a connection-failure config must not error")
	connectionFailure.Websocket.TrafficAlert = nil
	require.Error(t, connectionFailure.Setup(connectionFailureConfig), "Setup with invalid websocket connection state must error")
	require.NoError(t, connectionFailure.Shutdown(), "Shutting down the connection-failure exchange must not error")

	require.Error(t, ex.Setup(nil), "Setting up with a nil config must error")
	require.NoError(t, ex.Shutdown(), "Shutting down the exchange must not error")
}

func TestSendHTTPRequest(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "Request method should be POST")
		assert.Equal(t, "/info", r.URL.Path, "Request path should target the info endpoint")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "Request content type should be JSON")
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding the request body should not error") {
			return
		}
		_, err := w.Write([]byte(`{"BTC":"100"}`))
		assert.NoError(t, err, "Writing the response should not error")
	}))
	var result map[string]string
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), "Sending a valid info request must not error")
	assert.Equal(t, "allMids", got.Type, "Request type should be serialised")
	assert.Equal(t, "100", result["BTC"], "Response should be decoded")

	uninitialised := new(Exchange)
	require.ErrorIs(t, uninitialised.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), common.ErrNilPointer, "Sending without endpoints must return the expected error")
	var nilExchange *Exchange
	require.ErrorIs(t, nilExchange.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), common.ErrNilPointer, "Sending with a nil exchange must return the expected error")
	nilEndpoints := new(Exchange)
	nilEndpoints.SetDefaults()
	nilEndpoints.API.Endpoints = nil
	require.ErrorIs(t, nilEndpoints.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), common.ErrNilPointer, "Sending without an endpoint store must return the expected error")
	require.NoError(t, nilEndpoints.Shutdown(), "Shutting down the nil-endpoints exchange must not error")
	missingEndpoint := new(Exchange)
	missingEndpoint.SetDefaults()
	missingEndpoint.API.Endpoints = missingEndpoint.NewEndpoints()
	require.Error(t, missingEndpoint.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), "Sending without a configured spot endpoint must error")
	require.NoError(t, missingEndpoint.Shutdown(), "Shutting down the missing-endpoint exchange must not error")
	require.Error(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpot, infoLightEPL, make(chan int), &result), "Sending an unsupported JSON value must error")

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	require.ErrorIs(t, ex.SendHTTPRequest(cancelled, exchange.RestSpot, infoLightEPL, &infoRequest{Type: "allMids"}, &result), context.Canceled, "Sending with a cancelled context must return its cancellation")
}

func TestGetMetadata(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{"spotMeta": spotMetadataJSON})
	futuresServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write([]byte(perpetualMetadataJSON))
		assert.NoError(t, err, "Writing the perpetual metadata response should not error")
	}))
	t.Cleanup(futuresServer.Close)
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestFutures.String(), futuresServer.URL), "Setting a distinct futures URL must not error")
	perpetual, err := ex.GetPerpetualMetadata(t.Context())
	require.NoError(t, err, "Getting perpetual metadata must not error")
	require.Len(t, perpetual.Universe, 1, "Perpetual metadata must contain one market")
	assert.Equal(t, "BTC", perpetual.Universe[0].Name, "Perpetual market name should be decoded")

	spot, err := ex.GetSpotMetadata(t.Context())
	require.NoError(t, err, "Getting spot metadata must not error")
	require.Len(t, spot.Universe, 1, "Spot metadata must contain one market")
	assert.Equal(t, "@107", spot.Universe[0].Name, "Spot market identifier should be decoded")

	nullExchange := newStaticInfoExchange(t, map[string]string{"meta": "null", "spotMeta": "null"})
	_, err = nullExchange.GetPerpetualMetadata(t.Context())
	require.ErrorIs(t, err, common.ErrNilPointer, "Null perpetual metadata must return the expected error")
	_, err = nullExchange.GetSpotMetadata(t.Context())
	require.ErrorIs(t, err, common.ErrNilPointer, "Null spot metadata must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetPerpetualMetadata(t.Context())
	require.Error(t, err, "Getting perpetual metadata from a failing server must error")
	_, err = errorExchange.GetSpotMetadata(t.Context())
	require.Error(t, err, "Getting spot metadata from a failing server must error")
}

func TestGetPerpetualMetadataForDEX(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding named DEX metadata request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"universe":[{"name":"xyz:XYZ100"}]}`))
		assert.NoError(t, err, "Writing named DEX metadata response should not error")
	}))
	result, err := ex.GetPerpetualMetadataForDEX(t.Context(), "xyz")
	require.NoError(t, err, "Getting named DEX metadata must not error")
	require.Len(t, result.Universe, 1, "GetPerpetualMetadataForDEX must decode one market")
	assert.Equal(t, "xyz", got.DEX, "Named DEX should be included in the metadata request")
}

func TestGetPerpetualDEXs(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		infoTypePerpetualDEXs: `[null,{"name":"xyz","fullName":"XYZ","deployer":"` + officialSigningAddress + `"}]`,
	})
	result, err := ex.GetPerpetualDEXs(t.Context())
	require.NoError(t, err, "Getting valid perpetual DEX registry must not error")
	require.Len(t, result, 2, "GetPerpetualDEXs must retain the default entry")
	require.NotNil(t, result[1], "Builder DEX registry entry must be decoded")
	assert.Equal(t, "xyz", result[1].Name, "Builder DEX name should be decoded")

	for _, raw := range []string{`[]`, `[{"name":"invalid-default"}]`} {
		invalid := newStaticInfoExchange(t, map[string]string{infoTypePerpetualDEXs: raw})
		_, err = invalid.GetPerpetualDEXs(t.Context())
		require.ErrorIs(t, err, errUnexpectedResponseLength, "Invalid perpetual DEX registry must return the expected error")
	}
	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetPerpetualDEXs(t.Context())
	require.Error(t, err, "Getting perpetual DEX registry from a failing server must error")
}

func TestGetMetadataAndAssetContexts(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		"metaAndAssetCtxs":     perpetualContextsJSON,
		"spotMetaAndAssetCtxs": spotContextsJSON,
	})
	perpetual, err := ex.GetPerpetualMetadataAndAssetContexts(t.Context())
	require.NoError(t, err, "Getting perpetual metadata and contexts must not error")
	require.Len(t, perpetual.AssetContexts, 1, "Perpetual response must contain one context")
	assert.Equal(t, 101.0, perpetual.AssetContexts[0].MarkPrice.Float64(), "Perpetual mark price should be decoded")

	spot, err := ex.GetSpotMetadataAndAssetContexts(t.Context())
	require.NoError(t, err, "Getting spot metadata and contexts must not error")
	require.Len(t, spot.AssetContexts, 1, "Spot response must contain one context")
	assert.Equal(t, "@107", spot.AssetContexts[0].Coin, "Spot context identifier should be decoded")

	lengthExchange := newStaticInfoExchange(t, map[string]string{
		"metaAndAssetCtxs":     `[]`,
		"spotMetaAndAssetCtxs": `[{}]`,
	})
	_, err = lengthExchange.GetPerpetualMetadataAndAssetContexts(t.Context())
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Unexpected perpetual response length must return the expected error")
	_, err = lengthExchange.GetSpotMetadataAndAssetContexts(t.Context())
	require.ErrorIs(t, err, errUnexpectedResponseLength, "Unexpected spot response length must return the expected error")

	decodeExchange := newStaticInfoExchange(t, map[string]string{
		"metaAndAssetCtxs":     `[false,false]`,
		"spotMetaAndAssetCtxs": `[false,false]`,
	})
	_, err = decodeExchange.GetPerpetualMetadataAndAssetContexts(t.Context())
	require.ErrorContains(t, err, "perpetual metadata", "Invalid perpetual metadata must return a decoding error")
	_, err = decodeExchange.GetSpotMetadataAndAssetContexts(t.Context())
	require.ErrorContains(t, err, "spot metadata", "Invalid spot metadata must return a decoding error")

	contextDecodeExchange := newStaticInfoExchange(t, map[string]string{
		"metaAndAssetCtxs":     `[{},false]`,
		"spotMetaAndAssetCtxs": `[{},false]`,
	})
	_, err = contextDecodeExchange.GetPerpetualMetadataAndAssetContexts(t.Context())
	require.ErrorContains(t, err, "perpetual asset contexts", "Invalid perpetual contexts must return a decoding error")
	_, err = contextDecodeExchange.GetSpotMetadataAndAssetContexts(t.Context())
	require.ErrorContains(t, err, "spot asset contexts", "Invalid spot contexts must return a decoding error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetPerpetualMetadataAndAssetContexts(t.Context())
	require.Error(t, err, "Getting perpetual contexts from a failing server must error")
	_, err = errorExchange.GetSpotMetadataAndAssetContexts(t.Context())
	require.Error(t, err, "Getting spot contexts from a failing server must error")
}

func TestGetPerpetualMetadataAndAssetContextsForDEX(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding DEX contexts request should not error") {
			return
		}
		_, err := w.Write([]byte(perpetualContextsJSON))
		assert.NoError(t, err, "Writing DEX contexts response should not error")
	}))
	result, err := ex.GetPerpetualMetadataAndAssetContextsForDEX(t.Context(), "xyz")
	require.NoError(t, err, "Getting named DEX contexts must not error")
	require.Len(t, result.AssetContexts, 1, "GetPerpetualMetadataAndAssetContextsForDEX must decode one context")
	assert.Equal(t, "xyz", got.DEX, "DEX should be included in the request")
}

func TestGetFundingHistory(t *testing.T) {
	start := time.UnixMilli(1700000000000).UTC()
	end := start.Add(time.Hour)
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding funding history request should not error") {
			return
		}
		_, err := w.Write([]byte(`[{"coin":"BTC","fundingRate":"0.0001","premium":"0.0002","time":1700000000000}]`))
		assert.NoError(t, err, "Writing funding history response should not error")
	}))
	_, err := ex.GetFundingHistory(t.Context(), " ", start, end)
	require.ErrorIs(t, err, errCoinRequired, "Blank funding coin must return the expected error")
	_, err = ex.GetFundingHistory(t.Context(), "BTC", end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "Invalid funding range must return the expected error")

	result, err := ex.GetFundingHistory(t.Context(), " BTC ", start, end)
	require.NoError(t, err, "Getting valid funding history must not error")
	require.Len(t, result, 1, "GetFundingHistory must decode one record")
	assert.Equal(t, "BTC", got.Coin, "Funding coin should be trimmed")
	assert.Equal(t, start.UnixMilli(), got.StartTime, "Funding start time should be serialised")
	assert.Equal(t, end.UnixMilli(), got.EndTime, "Funding end time should be serialised")
	assert.Equal(t, 0.0001, result[0].FundingRate.Float64(), "Funding rate should be decoded")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetFundingHistory(t.Context(), "BTC", start, end)
	require.Error(t, err, "Getting funding history from a failing server must error")
}

func TestGetUserNonFundingLedgerUpdates(t *testing.T) {
	start := time.UnixMilli(1700000000000).UTC()
	end := start.Add(time.Hour)
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding ledger-history request should not error") {
			return
		}
		_, err := w.Write([]byte(`[{"time":1700000000000,"hash":"0x1","delta":{"type":"withdraw","usdc":"2","fee":"1","nonce":7}}]`))
		assert.NoError(t, err, "Writing ledger-history response should not error")
	}))
	_, err := ex.GetUserNonFundingLedgerUpdates(t.Context(), "invalid", start, end)
	require.ErrorIs(t, err, errInvalidAddress, "Invalid ledger address must return the expected error")
	_, err = ex.GetUserNonFundingLedgerUpdates(t.Context(), officialSigningAddress, end, start)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "Invalid ledger range must return the expected error")

	result, err := ex.GetUserNonFundingLedgerUpdates(t.Context(), strings.ToUpper(officialSigningAddress), start, end)
	require.NoError(t, err, "Getting valid ledger history must not error")
	require.Len(t, result, 1, "GetUserNonFundingLedgerUpdates must decode one record")
	assert.Equal(t, "userNonFundingLedgerUpdates", got.Type, "Ledger request type should match")
	assert.Equal(t, officialSigningAddress, got.User, "Ledger address should be normalised")
	assert.Equal(t, start.UnixMilli(), got.StartTime, "Ledger start time should be serialised")
	assert.Equal(t, end.UnixMilli(), got.EndTime, "Ledger end time should be serialised")
	assert.Equal(t, "withdraw", result[0].Delta.Type, "Ledger delta type should be decoded")
	assert.Equal(t, 2.0, result[0].Delta.USDC.Float64(), "Ledger amount should be decoded")
	assert.Equal(t, uint64(7), result[0].Delta.Nonce, "Ledger nonce should be decoded")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetUserNonFundingLedgerUpdates(t.Context(), officialSigningAddress, start, end)
	require.Error(t, err, "Getting ledger history from a failing server must error")
}

func TestGetUserFees(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding user-fees request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"userCrossRate":"0.0003","userAddRate":"0.0001","userSpotCrossRate":"0.0005","userSpotAddRate":"0.0002"}`))
		assert.NoError(t, err, "Writing user-fees response should not error")
	}))
	_, err := ex.GetUserFees(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "Invalid fee address must return the expected error")
	result, err := ex.GetUserFees(t.Context(), strings.ToUpper(officialSigningAddress))
	require.NoError(t, err, "Getting valid user fees must not error")
	assert.Equal(t, officialSigningAddress, got.User, "Fee address should be normalised")
	assert.Equal(t, 0.0005, result.UserSpotCrossRate.Float64(), "Spot taker rate should be decoded")

	nullExchange := newStaticInfoExchange(t, map[string]string{"userFees": "null"})
	_, err = nullExchange.GetUserFees(t.Context(), officialSigningAddress)
	require.ErrorIs(t, err, common.ErrNilPointer, "Null user fees must return the expected error")
	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetUserFees(t.Context(), officialSigningAddress)
	require.Error(t, err, "Getting user fees from a failing server must error")
}

func TestGetActiveAssetData(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding active-asset request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"user":"` + officialSigningAddress + `","coin":"BTC","leverage":{"type":"cross","value":20}}`))
		assert.NoError(t, err, "Writing active-asset response should not error")
	}))
	_, err := ex.GetActiveAssetData(t.Context(), "invalid", "BTC")
	require.ErrorIs(t, err, errInvalidAddress, "Invalid active-asset address must return the expected error")
	_, err = ex.GetActiveAssetData(t.Context(), officialSigningAddress, " ")
	require.ErrorIs(t, err, errCoinRequired, "Blank active-asset coin must return the expected error")
	result, err := ex.GetActiveAssetData(t.Context(), officialSigningAddress, " BTC ")
	require.NoError(t, err, "Getting valid active-asset data must not error")
	assert.Equal(t, "BTC", got.Coin, "Active-asset coin should be trimmed")
	assert.Equal(t, 20.0, result.Leverage.Value.Float64(), "Active-asset leverage should be decoded")

	nullExchange := newStaticInfoExchange(t, map[string]string{"activeAssetData": "null"})
	_, err = nullExchange.GetActiveAssetData(t.Context(), officialSigningAddress, "BTC")
	require.ErrorIs(t, err, common.ErrNilPointer, "Null active-asset data must return the expected error")
	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetActiveAssetData(t.Context(), officialSigningAddress, "BTC")
	require.Error(t, err, "Getting active-asset data from a failing server must error")
}

func TestGetAllMids(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding the all-mids request should not error") {
			return
		}
		_, err := w.Write([]byte(`{"BTC":"100.5"}`))
		assert.NoError(t, err, "Writing the all-mids response should not error")
	}))
	mids, err := ex.GetAllMids(t.Context(), "test-dex")
	require.NoError(t, err, "Getting all mid prices must not error")
	assert.Equal(t, "test-dex", got.DEX, "DEX should be serialised")
	assert.Equal(t, 100.5, mids["BTC"].Float64(), "Mid price should be decoded")

	nullExchange := newStaticInfoExchange(t, map[string]string{"allMids": "null"})
	_, err = nullExchange.GetAllMids(t.Context(), "")
	require.ErrorIs(t, err, common.ErrNilPointer, "Null all-mids response must return the expected error")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetAllMids(t.Context(), "")
	require.Error(t, err, "Getting all mids from a failing server must error")
}

func TestGetL2Book(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding the L2 book request should not error") {
			return
		}
		_, err := w.Write([]byte(bookJSON))
		assert.NoError(t, err, "Writing the L2 book response should not error")
	}))
	_, err := ex.GetL2Book(t.Context(), asset.PerpetualContract, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil L2 book request must return the expected error")
	_, err = ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{})
	require.ErrorIs(t, err, errCoinRequired, "Empty L2 book coin must return the expected error")

	invalidSignificantFigures := uint64(1)
	_, err = ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC", SignificantFigures: &invalidSignificantFigures})
	require.ErrorIs(t, err, errInvalidSignificantFigures, "Invalid significant figures must return the expected error")

	mantissa := uint64(1)
	_, err = ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC", Mantissa: &mantissa})
	require.ErrorIs(t, err, errInvalidMantissa, "Mantissa without significant figures must return the expected error")
	fourSignificantFigures := uint64(4)
	_, err = ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC", SignificantFigures: &fourSignificantFigures, Mantissa: &mantissa})
	require.ErrorIs(t, err, errInvalidMantissa, "Mantissa with non-five significant figures must return the expected error")
	fiveSignificantFigures := uint64(5)
	invalidMantissa := uint64(3)
	_, err = ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC", SignificantFigures: &fiveSignificantFigures, Mantissa: &invalidMantissa})
	require.ErrorIs(t, err, errInvalidMantissa, "Invalid mantissa must return the expected error")

	_, err = ex.GetL2Book(t.Context(), asset.Options, &L2BookRequest{Coin: "BTC"})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported L2 book asset must return the expected error")
	book, err := ex.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC", SignificantFigures: &fiveSignificantFigures, Mantissa: &mantissa})
	require.NoError(t, err, "Getting a valid L2 book must not error")
	require.Len(t, book.Levels, 2, "L2 book must contain bid and ask sides")
	assert.Equal(t, "BTC", got.Coin, "Coin should be serialised")
	assert.Equal(t, fiveSignificantFigures, *got.NSigFigs, "Significant figures should be serialised")
	assert.Equal(t, mantissa, *got.Mantissa, "Mantissa should be serialised")

	nullExchange := newStaticInfoExchange(t, map[string]string{"l2Book": "null"})
	_, err = nullExchange.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC"})
	require.ErrorIs(t, err, common.ErrNilPointer, "Null L2 book must return the expected error")
	lengthExchange := newStaticInfoExchange(t, map[string]string{"l2Book": `{"coin":"BTC","levels":[],"time":1700000000000}`})
	_, err = lengthExchange.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC"})
	require.ErrorIs(t, err, errInvalidBookLevelCount, "Invalid L2 book side count must return the expected error")
	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetL2Book(t.Context(), asset.PerpetualContract, &L2BookRequest{Coin: "BTC"})
	require.Error(t, err, "Getting an L2 book from a failing server must error")
}

func TestGetRecentTradesForCoin(t *testing.T) {
	var got infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding the recent-trades request should not error") {
			return
		}
		_, err := w.Write([]byte(tradesJSON))
		assert.NoError(t, err, "Writing the recent-trades response should not error")
	}))
	_, err := ex.GetRecentTradesForCoin(t.Context(), " ", asset.PerpetualContract)
	require.ErrorIs(t, err, errCoinRequired, "Empty recent-trades coin must return the expected error")
	_, err = ex.GetRecentTradesForCoin(t.Context(), "BTC", asset.Options)
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported recent-trades asset must return the expected error")
	trades, err := ex.GetRecentTradesForCoin(t.Context(), "BTC", asset.PerpetualContract)
	require.NoError(t, err, "Getting recent trades must not error")
	require.Len(t, trades, 1, "Recent trades must contain one result")
	assert.Equal(t, uint64(7), trades[0].TradeID, "Trade ID should be decoded")
	assert.Equal(t, "BTC", got.Coin, "Coin should be serialised")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetRecentTradesForCoin(t.Context(), "BTC", asset.PerpetualContract)
	require.Error(t, err, "Getting recent trades from a failing server must error")
}

func TestGetCandles(t *testing.T) {
	start := time.UnixMilli(1700000000000).UTC()
	end := start.Add(time.Minute)
	var got struct {
		Type    string `json:"type"`
		Request struct {
			Coin      string `json:"coin"`
			Interval  string `json:"interval"`
			StartTime int64  `json:"startTime"`
			EndTime   int64  `json:"endTime"`
		} `json:"req"`
	}
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&got), "Decoding the candle request should not error") {
			return
		}
		_, err := w.Write([]byte(candlesJSON))
		assert.NoError(t, err, "Writing the candle response should not error")
	}))
	_, err := ex.GetCandles(t.Context(), asset.PerpetualContract, nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "Nil candle request must return the expected error")
	_, err = ex.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{})
	require.ErrorIs(t, err, errCoinRequired, "Empty candle coin must return the expected error")
	_, err = ex.GetCandles(t.Context(), asset.Options, &CandleRequest{Coin: "BTC", Interval: kline.OneMin})
	require.ErrorIs(t, err, asset.ErrNotSupported, "Unsupported candle asset must return the expected error")
	_, err = ex.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{Coin: "BTC", Interval: kline.Interval(42)})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "Unsupported candle interval must return the expected error")
	_, err = ex.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{Coin: "BTC", Interval: kline.OneMin, StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "Invalid candle times must return the expected error")
	for _, tc := range []struct {
		name      string
		startTime time.Time
		endTime   time.Time
		errorIs   error
	}{
		{name: "start unset", endTime: end, errorIs: common.ErrDateUnset},
		{name: "end unset", startTime: start, errorIs: common.ErrDateUnset},
		{name: "equal times", startTime: start, endTime: start, errorIs: common.ErrStartEqualsEnd},
		{name: "future range", startTime: time.Now().Add(time.Hour), endTime: time.Now().Add(2 * time.Hour), errorIs: common.ErrStartAfterTimeNow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ex.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{Coin: "BTC", Interval: kline.OneMin, StartTime: tc.startTime, EndTime: tc.endTime})
			require.ErrorIs(t, err, tc.errorIs, "Invalid candle time range must return the expected error")
		})
	}

	candles, err := ex.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{Coin: "BTC", Interval: kline.OneMin, StartTime: start, EndTime: end})
	require.NoError(t, err, "Getting valid candles must not error")
	require.Len(t, candles, 1, "Candle response must contain one result")
	assert.Equal(t, "candleSnapshot", got.Type, "Candle request type should be serialised")
	assert.Equal(t, "BTC", got.Request.Coin, "Candle coin should be serialised")
	assert.Equal(t, "1m", got.Request.Interval, "Candle interval should be serialised")
	assert.Equal(t, start.UnixMilli(), got.Request.StartTime, "Candle start time should be serialised")
	assert.Equal(t, end.Add(-time.Millisecond).UnixMilli(), got.Request.EndTime, "Candle end time should use an exclusive GCT range")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetCandles(t.Context(), asset.PerpetualContract, &CandleRequest{Coin: "BTC", Interval: kline.OneMin, StartTime: start, EndTime: end})
	require.Error(t, err, "Getting candles from a failing server must error")
}

func TestGetAccountInfoEndpoints(t *testing.T) {
	ex := newStaticInfoExchange(t, map[string]string{
		testUserRoleInfoType:     testUserRoleResponse,
		"vaultDetails":           `{"vaultAddress":"` + testVaultAddress + `","leader":"` + officialSigningAddress + `"}`,
		"spotClearinghouseState": `{"balances":[{"coin":"USDC","token":0,"total":"10","hold":"2","entryNtl":"1"}]}`,
		"clearinghouseState":     `{"marginSummary":{"accountValue":"12","totalMarginUsed":"3"},"withdrawable":"8","assetPositions":[]}`,
		"frontendOpenOrders":     `[{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"}]`,
		"historicalOrders":       `[{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"0","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"filled","statusTimestamp":1700000001000}]`,
		"orderStatus":            `{"status":"order","order":{"order":{"coin":"BTC","side":"B","limitPx":"100","sz":"1","origSz":"2","oid":7,"timestamp":1700000000000,"isTrigger":false,"reduceOnly":false,"orderType":"Limit","tif":"Gtc"},"status":"open","statusTimestamp":1700000001000}}`,
	})

	role, err := ex.GetUserRole(t.Context(), strings.ToUpper(officialSigningAddress))
	require.NoError(t, err, "Getting a valid user role must not error")
	assert.Equal(t, "user", role.Role, "User role should be decoded")

	vault, err := ex.GetVaultDetails(t.Context(), testVaultAddress, officialSigningAddress)
	require.NoError(t, err, "Getting valid vault details must not error")
	assert.Equal(t, officialSigningAddress, vault.Leader, "Vault leader should be decoded")
	_, err = ex.GetVaultDetails(t.Context(), testVaultAddress, "")
	require.NoError(t, err, "Getting vault details without optional user must not error")

	spotState, err := ex.GetSpotClearinghouseState(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Getting valid spot state must not error")
	require.Len(t, spotState.Balances, 1, "Spot state must contain one balance")
	assert.Equal(t, 10.0, spotState.Balances[0].Total.Float64(), "Spot balance should be decoded")

	perpetualState, err := ex.GetClearinghouseState(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Getting valid perpetual state must not error")
	assert.Equal(t, 12.0, perpetualState.MarginSummary.AccountValue.Float64(), "Perpetual account value should be decoded")
	_, err = ex.GetClearinghouseStateForDEX(t.Context(), officialSigningAddress, "xyz")
	require.NoError(t, err, "Getting named DEX perpetual state must not error")

	openOrders, err := ex.GetOpenOrdersForUser(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Getting valid open orders must not error")
	require.Len(t, openOrders, 1, "Open order response must be decoded")
	assert.Equal(t, uint64(7), openOrders[0].OrderID, "Open order ID should be decoded")
	openOrders, err = ex.GetOpenOrdersForUserForDEX(t.Context(), officialSigningAddress, "xyz")
	require.NoError(t, err, "Getting named DEX open orders must not error")
	require.Len(t, openOrders, 1, "GetOpenOrdersForUserForDEX must decode one order")

	history, err := ex.GetHistoricalOrdersForUser(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Getting valid order history must not error")
	require.Len(t, history, 1, "Order history response must be decoded")
	assert.Equal(t, "filled", history[0].Status, "Historical order status should be decoded")

	status, err := ex.GetOrderStatusForUser(t.Context(), officialSigningAddress, uint64(7))
	require.NoError(t, err, "Getting order status by numeric ID must not error")
	assert.Equal(t, "order", status.Status, "Order status response should be decoded")
	_, err = ex.GetOrderStatusForUser(t.Context(), officialSigningAddress, validClientOrderID)
	require.NoError(t, err, "Getting order status by client ID must not error")

	for _, call := range []func() error{
		func() error { _, err := ex.GetUserRole(t.Context(), "invalid"); return err },
		func() error { _, err := ex.GetVaultDetails(t.Context(), "invalid", ""); return err },
		func() error { _, err := ex.GetVaultDetails(t.Context(), testVaultAddress, "invalid"); return err },
		func() error { _, err := ex.GetSpotClearinghouseState(t.Context(), "invalid"); return err },
		func() error { _, err := ex.GetClearinghouseState(t.Context(), "invalid"); return err },
		func() error { _, err := ex.GetOpenOrdersForUser(t.Context(), "invalid"); return err },
		func() error { _, err := ex.GetHistoricalOrdersForUser(t.Context(), "invalid"); return err },
		func() error { _, err := ex.GetOrderStatusForUser(t.Context(), "invalid", uint64(7)); return err },
	} {
		require.ErrorIs(t, call(), errInvalidAddress, "Invalid address must return the expected error")
	}
	_, err = ex.GetOrderStatusForUser(t.Context(), officialSigningAddress, uint64(0))
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "Zero order ID must return the expected error")
	_, err = ex.GetOrderStatusForUser(t.Context(), officialSigningAddress, "invalid")
	require.ErrorIs(t, err, errClientOrderIDInvalid, "Invalid client order ID must return the expected error")
	_, err = ex.GetOrderStatusForUser(t.Context(), officialSigningAddress, int64(7))
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "Unsupported order ID type must return the expected error")

	nullExchange := newStaticInfoExchange(t, map[string]string{
		testUserRoleInfoType:     `null`,
		"vaultDetails":           `null`,
		"spotClearinghouseState": `null`,
		"clearinghouseState":     `null`,
		"frontendOpenOrders":     `[]`,
		"historicalOrders":       `[]`,
		"orderStatus":            `null`,
	})
	for _, call := range []func() error{
		func() error { _, err := nullExchange.GetUserRole(t.Context(), officialSigningAddress); return err },
		func() error { _, err := nullExchange.GetVaultDetails(t.Context(), testVaultAddress, ""); return err },
		func() error {
			_, err := nullExchange.GetSpotClearinghouseState(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := nullExchange.GetClearinghouseState(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := nullExchange.GetOrderStatusForUser(t.Context(), officialSigningAddress, uint64(7))
			return err
		},
	} {
		require.ErrorIs(t, call(), common.ErrNilPointer, "Null account response must return the expected error")
	}
	openOrders, err = nullExchange.GetOpenOrdersForUser(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Empty open-order response must not error")
	assert.Empty(t, openOrders, "Empty open-order response should be retained")
	history, err = nullExchange.GetHistoricalOrdersForUser(t.Context(), officialSigningAddress)
	require.NoError(t, err, "Empty historical-order response must not error")
	assert.Empty(t, history, "Empty historical-order response should be retained")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	for _, call := range []func() error{
		func() error { _, err := errorExchange.GetUserRole(t.Context(), officialSigningAddress); return err },
		func() error { _, err := errorExchange.GetVaultDetails(t.Context(), testVaultAddress, ""); return err },
		func() error {
			_, err := errorExchange.GetSpotClearinghouseState(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := errorExchange.GetClearinghouseState(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := errorExchange.GetOpenOrdersForUser(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := errorExchange.GetHistoricalOrdersForUser(t.Context(), officialSigningAddress)
			return err
		},
		func() error {
			_, err := errorExchange.GetOrderStatusForUser(t.Context(), officialSigningAddress, uint64(7))
			return err
		},
	} {
		require.Error(t, call(), "Account endpoint HTTP failure must be returned")
	}
}

func TestGetUserAbstraction(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode AccountAbstraction
	}{
		{name: "default", mode: AccountAbstractionDefault},
		{name: "disabled", mode: AccountAbstractionDisabled},
		{name: "legacy DEX abstraction", mode: AccountAbstractionDEX},
		{name: "unified", mode: AccountAbstractionUnified},
		{name: "portfolio margin", mode: AccountAbstractionPortfolio},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ex := newStaticInfoExchange(t, map[string]string{"userAbstraction": `"` + string(tc.mode) + `"`})
			result, err := ex.GetUserAbstraction(t.Context(), officialSigningAddress)
			require.NoError(t, err, "Getting a supported account abstraction mode must not error")
			assert.Equal(t, tc.mode, result, "Account abstraction mode should be decoded")
		})
	}

	ex := newStaticInfoExchange(t, map[string]string{"userAbstraction": `"futureMode"`})
	_, err := ex.GetUserAbstraction(t.Context(), "invalid")
	require.ErrorIs(t, err, errInvalidAddress, "Invalid abstraction address must return the expected error")
	_, err = ex.GetUserAbstraction(t.Context(), officialSigningAddress)
	require.ErrorIs(t, err, errAccountAbstractionInvalid, "Unknown abstraction mode must fail closed")

	errorExchange := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	_, err = errorExchange.GetUserAbstraction(t.Context(), officialSigningAddress)
	require.Error(t, err, "Account abstraction HTTP failure must be returned")
}

func TestDEXScopedAccountInfoRequests(t *testing.T) {
	var requests []infoRequest
	ex := newHTTPTestExchange(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request infoRequest
		if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request), "Decoding DEX-scoped account request should not error") {
			return
		}
		requests = append(requests, request)
		response := `[]`
		if request.Type == "clearinghouseState" {
			response = `{"assetPositions":[]}`
		}
		_, err := w.Write([]byte(response))
		assert.NoError(t, err, "Writing DEX-scoped account response should not error")
	}))
	_, err := ex.GetClearinghouseStateForDEX(t.Context(), officialSigningAddress, "xyz")
	require.NoError(t, err, "Getting DEX-scoped clearinghouse state must not error")
	_, err = ex.GetOpenOrdersForUserForDEX(t.Context(), officialSigningAddress, "xyz")
	require.NoError(t, err, "Getting DEX-scoped open orders must not error")
	require.Len(t, requests, 2, "DEX-scoped account requests must contain two entries")
	assert.Equal(t, "xyz", requests[0].DEX, "Clearinghouse state should include its DEX")
	assert.Equal(t, "xyz", requests[1].DEX, "Open orders should include their DEX")
}
