package fxmacrodata

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

func newTestProvider(t *testing.T, handler http.Handler) (provider *FXMacroData, closeServer func()) {
	t.Helper()
	server := httptest.NewServer(handler)
	provider = &FXMacroData{}
	err := provider.Setup(base.Settings{
		Name:            "FXMacroData",
		Enabled:         true,
		APIKey:          "test-key",
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
		if r.Header.Get("X-API-Key") != "test-key" {
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
		_, _ = w.Write([]byte(`{"data":[]}`))
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

	rate, err := provider.GetLatestForexRate("USD", "AUD")
	assert.ErrorContains(t, err, "no FXMacroData rate returned", "GetLatestForexRate should reject empty data")
	assert.Zero(t, rate, "rate should be zero when no data is returned")
}

func TestGetLatestForexRateHTTPError(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	defer closeServer()

	rate, err := provider.GetLatestForexRate("USD", "AUD")
	assert.Error(t, err, "GetLatestForexRate should return HTTP errors")
	assert.Zero(t, rate, "rate should be zero when the request fails")
}

func TestServiceStatusEndpoints(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-API-Key"), "public status requests should not include an API key")
		switch r.URL.Path {
		case "/api/v1/health", "/api/v1/ping":
			_, _ = w.Write([]byte(`{"status":"ok","service":"fxmacrodata-api"}`))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer closeServer()

	health, err := provider.Health()
	require.NoError(t, err, "Health must not error")
	assert.Equal(t, "ok", health.Status, "Health should decode the service status")

	ping, err := provider.Ping()
	require.NoError(t, err, "Ping must not error")
	assert.Equal(t, "fxmacrodata-api", ping.Service, "Ping should decode the service name")
}

func TestReadEndpointHelpers(t *testing.T) {
	seen := make([]string, 0)
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer closeServer()

	values := url.Values{"limit": []string{"1"}}
	helpers := []struct {
		name string
		fn   func() error
	}{
		{"DataCatalogue", func() error { _, err := provider.DataCatalogue("usd"); return err }},
		{"Announcements", func() error { _, err := provider.Announcements("usd", "cpi", values); return err }},
		{"LatestAnnouncements", func() error { _, err := provider.LatestAnnouncements("usd", values); return err }},
		{"AnnouncementChanges", func() error { _, err := provider.AnnouncementChanges(values); return err }},
		{"Calendar", func() error { _, err := provider.Calendar("usd", values); return err }},
		{"Predictions", func() error { _, err := provider.Predictions("usd", "cpi", values); return err }},
		{"COT", func() error { _, err := provider.COT("jpy", values); return err }},
		{"Commodity", func() error { _, err := provider.Commodity("brent", values); return err }},
		{"CommoditiesLatest", func() error { _, err := provider.CommoditiesLatest(values); return err }},
		{"Curves", func() error { _, err := provider.Curves("usd", values); return err }},
		{"CurveProxies", func() error { _, err := provider.CurveProxies("usd", values); return err }},
		{"ForwardCurves", func() error { _, err := provider.ForwardCurves("usd", values); return err }},
		{"RateDifferentials", func() error { _, err := provider.RateDifferentials("eur", "usd", values); return err }},
		{"ForwardDifferentials", func() error { _, err := provider.ForwardDifferentials("eur", "usd", values); return err }},
		{"MarketSessions", func() error { _, err := provider.MarketSessions(values); return err }},
		{"RiskSentiment", func() error { _, err := provider.RiskSentiment(values); return err }},
		{"News", func() error { _, err := provider.News("usd", values); return err }},
		{"PressReleases", func() error { _, err := provider.PressReleases("usd", values); return err }},
	}
	for _, tc := range helpers {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.fn()
			require.NoErrorf(t, err, "%s must not error", tc.name)
		})
	}

	expected := []string{
		"/api/v1/data_catalogue/usd",
		"/api/v1/announcements/usd/cpi",
		"/api/v1/announcements/usd/latest",
		"/api/v1/announcements/changes",
		"/api/v1/calendar/usd",
		"/api/v1/predictions/usd/cpi",
		"/api/v1/cot/jpy",
		"/api/v1/commodities/brent",
		"/api/v1/commodities/latest",
		"/api/v1/curves/usd",
		"/api/v1/curve_proxies/usd",
		"/api/v1/forward_curves/usd",
		"/api/v1/rate_differentials/eur/usd",
		"/api/v1/forward_differentials/eur/usd",
		"/api/v1/market_sessions",
		"/api/v1/risk_sentiment",
		"/api/v1/news/usd",
		"/api/v1/press-releases/usd",
	}
	require.Len(t, seen, len(expected), "seen requests must match expected request count")
	for i := range expected {
		assert.Equal(t, expected[i], seen[i], "request path should match expected order")
	}
	assert.Empty(t, values.Get("api_key"), "SendHTTPRequest should not mutate caller query values")
}

func TestGraphQL(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method, "GraphQL should use POST")
		assert.Equal(t, "/api/v1/graphql", r.URL.Path, "GraphQL should use graphql endpoint")
		assert.Equal(t, "test-key", r.Header.Get("X-API-Key"), "GraphQL should pass header auth")
		assert.Empty(t, r.URL.Query().Get("api_key"), "GraphQL should not pass query auth")
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"), "GraphQL should send JSON content type")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("request body must be readable: %v", err)
			http.Error(w, "body read failed", http.StatusBadRequest)
			return
		}
		assert.JSONEq(t, `{"query":"{ viewer }"}`, string(body), "GraphQL should forward JSON payload")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer closeServer()

	var result map[string]bool
	err := provider.GraphQL(`{"query":"{ viewer }"}`, &result)
	require.NoError(t, err, "GraphQL must not error")
	assert.True(t, result["ok"], "GraphQL should decode response")
}

func TestSetupAllowsPublicRequestsWithoutAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-API-Key"), "public requests should not include an API key")
		assert.Equal(t, "/api/v1/data_catalogue/usd", r.URL.Path, "public request should use the requested endpoint")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	provider := new(FXMacroData)
	require.NoError(t, provider.Setup(base.Settings{Name: "FXMacroData"}), "Setup allows API-key-free public use")
	provider.APIURL = server.URL + "/api/v1/"
	require.NoError(t, provider.Requester.DisableRateLimiter(), "rate limiter must disable for local httptest provider")

	_, err := provider.DataCatalogue("usd")
	require.NoError(t, err, "public data catalogue request does not require an API key")
}

func TestSendHTTPRequestContextHonoursCancellation(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("a cancelled context should not issue an HTTP request")
		http.NotFound(w, r)
	}))
	defer closeServer()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := provider.SendHTTPRequestContext(ctx, "data_catalogue/usd", nil, new(DataCatalogueResponse))
	assert.ErrorIs(t, err, context.Canceled, "SendHTTPRequestContext should return the caller cancellation")
}
