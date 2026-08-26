package fxmacrodata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

// Live test toggles. The unit and contract tests are hermetic and always run;
// the two smoke tests below reach the live FXMacroData API and are opt-in.
// Set these to true, or set the matching environment variable, to enable them.
//
//	testLive   / GCT_RUN_LIVE_TESTS               public endpoints, no key needed
//	testAuth   / GCT_RUN_FXMACRODATA_AUTH_TESTS   authenticated endpoints
//	testAPIKey / FXMACRODATA_API_KEY, FXMD_API_KEY  key for the authenticated test
var (
	testLive   = false
	testAuth   = false
	testAPIKey = ""
)

// liveTestsEnabled reports whether the public live smoke test should run.
func liveTestsEnabled() bool {
	return testLive || os.Getenv("GCT_RUN_LIVE_TESTS") == "true"
}

// authTestsEnabled reports whether the authenticated live smoke test should run.
func authTestsEnabled() bool {
	return testAuth || os.Getenv("GCT_RUN_FXMACRODATA_AUTH_TESTS") == "true"
}

// liveTestAPIKey returns the API key used by the authenticated smoke test,
// preferring the package variable over the environment.
func liveTestAPIKey() string {
	if testAPIKey != "" {
		return testAPIKey
	}
	if key := os.Getenv("FXMACRODATA_API_KEY"); key != "" {
		return key
	}
	return os.Getenv("FXMD_API_KEY")
}

func newTestProvider(t *testing.T, handler http.Handler) (provider *FXMacroData, closeServer func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	provider = &FXMacroData{}
	err := provider.Setup(base.Settings{
		Name:            "FXMacroData",
		Enabled:         true,
		APIKey:          "placeholder",
		PrimaryProvider: true,
	})
	if err != nil {
		server.Close()
		require.NoError(t, err, "Setup must not error")
	}
	provider.APIURL = server.URL + "/api/v1/"
	err = provider.Requester.DisableRateLimiter()
	require.NoError(t, err, "rate limiter must disable for local httptest provider")
	return provider, server.Close
}

func TestGetRates(t *testing.T) {
	var requestCount atomic.Int64
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if r.Header.Get("X-API-Key") != "placeholder" {
			t.Errorf("expected X-API-Key header auth")
			http.Error(w, "missing API key", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/api/v1/forex/usd/aud":
			_, _ = w.Write([]byte(`{"data":[{"val":1.5}]}`))
		case "/api/v1/forex/usd/eur":
			_, _ = w.Write([]byte(`{"data":[{"val":0.9}]}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer closeServer()

	rates, err := provider.GetRates(" USD ", " AUD, EUR ,XYZ, usd ")
	require.NoError(t, err, "GetRates must not error")
	assert.Equal(t, 1.5, rates["USDAUD"], "USDAUD should match mocked latest rate")
	assert.Equal(t, 0.9, rates["USDEUR"], "USDEUR should match mocked latest rate")
	assert.NotContains(t, rates, "USDXYZ", "unsupported currency should not be requested")
	assert.Len(t, rates, 2, "GetRates should return only unique supported targets")
	assert.Equal(t, int64(2), requestCount.Load(), "GetRates should request each unique supported target once")
}

func TestGetRatesDuplicateTarget(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("duplicate targets should not issue HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", "AUD,EUR,AUD")
	assert.ErrorIs(t, err, errDuplicateCurrency, "GetRates should reject duplicate target currencies")
	assert.Nil(t, rates, "rates should be nil when target currencies are duplicated")
}

func TestGetRatesEmptyTarget(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("empty targets should not issue HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", "AUD,,EUR")
	assert.ErrorIs(t, err, errEmptyCurrency, "GetRates should reject empty target currency segments")
	assert.Nil(t, rates, "rates should be nil when target currencies include an empty segment")
}

func TestGetRatesRejectsNoEffectiveTarget(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a base-only request should not issue an HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", " USD ")
	assert.ErrorIs(t, err, errNoTargetCurrencies, "GetRates should reject target lists that only contain the base currency")
	assert.Nil(t, rates, "rates should be nil when no target currencies remain")
}

func TestGetRatesDefaultsToSupportedTargets(t *testing.T) {
	var requestCount atomic.Int64
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		if !strings.HasPrefix(r.URL.Path, "/api/v1/forex/usd/") {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"val":1.0}]}`))
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", "")
	require.NoError(t, err, "GetRates must not error")
	supported, err := provider.GetSupportedCurrencies()
	require.NoError(t, err, "GetSupportedCurrencies must not error")
	assert.Len(t, rates, len(supported)-1, "GetRates should default to every supported target except base currency")
	assert.Equal(t, int64(len(supported)-1), requestCount.Load(), "GetRates should request each default target once")
}

func TestGetRatesUnsupportedTargetsOnly(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unsupported targets should not issue HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", "XYZ")
	assert.ErrorIs(t, err, errUnsupportedCurrency, "GetRates should reject unsupported target currencies when no rates are available")
	assert.Nil(t, rates, "rates should be nil when every target currency is unsupported")
}

func TestGetRatesPropagatesLatestRateError(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/forex/usd/aud", r.URL.Path, "GetRates should request the expected FX pair")
		_, _ = w.Write([]byte(`{"inflation":{"name":"Inflation (CPI)","unit":"%YoY","frequency":"Monthly","source":"BLS"}}`))
	}))
	defer closeServer()

	rates, err := provider.GetRates("USD", "AUD")
	assert.ErrorContains(t, err, "no FXMacroData rate returned", "GetRates should propagate latest rate lookup errors")
	assert.Nil(t, rates, "rates should be nil when latest rate lookup fails")
}

func TestGetRatesUnsupportedBase(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unsupported base should not issue HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	rates, err := provider.GetRates("MXN", "AUD")
	assert.ErrorIs(t, err, errUnsupportedCurrency, "GetRates should reject unsupported base currency")
	assert.Nil(t, rates, "rates should be nil when base currency is unsupported")
}

func TestGetLatestForexRateEmptyData(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer closeServer()

	rate, err := provider.GetLatestForexRate(context.Background(), "USD", "AUD")
	assert.ErrorContains(t, err, "no FXMacroData rate returned", "GetLatestForexRate should reject empty data")
	assert.Zero(t, rate, "rate should be zero when no data is returned")
}

func TestGetLatestForexRateHTTPError(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer closeServer()

	rate, err := provider.GetLatestForexRate(context.Background(), "USD", "AUD")
	assert.Error(t, err, "GetLatestForexRate should return HTTP errors")
	assert.Zero(t, rate, "rate should be zero when the request fails")
}

func TestPing(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-API-Key"), "public status requests should not include an API key")
		assert.Equal(t, "/api/v1/ping", r.URL.Path, "Ping should use the documented endpoint")
		_, _ = w.Write([]byte(`{"status":"ok","service":"fxmacrodata-api"}`))
	}))
	defer closeServer()

	ping, err := provider.Ping(t.Context())
	require.NoError(t, err, "Ping must not error")
	assert.Equal(t, "ok", ping.Status, "Ping should decode the status")
	assert.Equal(t, "fxmacrodata-api", ping.Service, "Ping should decode the service name")
}

func TestSetupAllowsPublicRequestsWithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-API-Key"), "public requests should not include an API key")
		assert.Equal(t, "/api/v1/data_catalogue/usd", r.URL.Path, "public request should use the requested endpoint")
		_, _ = w.Write([]byte(`{"inflation":{"name":"Inflation (CPI)","unit":"%YoY","frequency":"Monthly","source":"BLS"}}`))
	}))
	defer server.Close()

	provider := new(FXMacroData)
	require.NoError(t, provider.Setup(base.Settings{Name: "FXMacroData"}), "Setup allows API-key-free public use")
	assert.Equal(t, APIURL, provider.APIURL, "Setup should use the canonical FXMacroData API URL")
	provider.APIURL = server.URL + "/api/v1/"
	require.NoError(t, provider.Requester.DisableRateLimiter(), "rate limiter must disable for local httptest provider")

	_, err := provider.DataCatalogue(context.Background(), "usd")
	require.NoError(t, err, "public data catalogue request does not require an API key")
}

func TestPublicEndpointsLive(t *testing.T) {
	if !liveTestsEnabled() {
		t.Skip("set testLive = true or GCT_RUN_LIVE_TESTS=true to run the public FXMacroData smoke test")
	}

	provider := new(FXMacroData)
	require.NoError(t, provider.Setup(base.Settings{Name: "FXMacroData"}),
		"Setup must configure the public endpoint client")

	ping, err := provider.Ping(t.Context())
	require.NoError(t, err, "Ping must not error")
	assert.NotEmpty(t, ping.Status, "Ping should return a status")

	catalogue, err := provider.DataCatalogue(t.Context(), "usd")
	require.NoError(t, err, "DataCatalogue must not error")
	assert.NotNil(t, catalogue, "DataCatalogue should return a response")
}

func TestGetLatestForexRateHonoursCancellation(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a cancelled context should not issue an HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := provider.GetLatestForexRate(ctx, "USD", "AUD")
	assert.ErrorIs(t, err, context.Canceled, "GetLatestForexRate should return the caller cancellation")
}

func TestAuthenticatedEndpointsRequireAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("an API-key-required endpoint should fail before issuing an HTTP request")
		http.NotFound(w, r)
	}))
	defer server.Close()

	provider := new(FXMacroData)
	require.NoError(t, provider.Setup(base.Settings{Name: "FXMacroData"}))
	provider.APIURL = server.URL + "/api/v1/"
	require.NoError(t, provider.Requester.DisableRateLimiter())

	_, err := provider.GetLatestForexRate(context.Background(), "USD", "AUD")
	assert.ErrorIs(t, err, errAPIKeyNotConfigured, "forex requests should require a configured API key")
}
