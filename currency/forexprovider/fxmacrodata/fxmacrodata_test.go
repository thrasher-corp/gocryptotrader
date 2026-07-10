package fxmacrodata

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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
	var requestCount int
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Query().Get("api_key") != "test-key" {
			t.Errorf("expected api_key query auth")
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

	rates, err := provider.GetRates("USD", "AUD, EUR, XYZ, AUD, usd")
	require.NoError(t, err, "GetRates must not error")
	assert.Equal(t, 1.5, rates["USDAUD"], "USDAUD should match mocked latest rate")
	assert.Equal(t, 0.9, rates["USDEUR"], "USDEUR should match mocked latest rate")
	assert.NotContains(t, rates, "USDXYZ", "unsupported currency should not be requested")
	assert.Len(t, rates, 2, "GetRates should return only unique supported targets")
	assert.Equal(t, 2, requestCount, "GetRates should request each unique supported target once")
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

func TestReadEndpointHelpers(t *testing.T) {
	seen := make([]string, 0)
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer closeServer()

	values := url.Values{"limit": []string{"1"}}
	helpers := []struct {
		name string
		fn   func() (map[string]any, error)
	}{
		{"DataCatalogue", func() (map[string]any, error) { return provider.DataCatalogue("usd") }},
		{"Announcements", func() (map[string]any, error) { return provider.Announcements("usd", "cpi", values) }},
		{"LatestAnnouncements", func() (map[string]any, error) { return provider.LatestAnnouncements("usd", values) }},
		{"AnnouncementChanges", func() (map[string]any, error) { return provider.AnnouncementChanges(values) }},
		{"Calendar", func() (map[string]any, error) { return provider.Calendar("usd", values) }},
		{"Predictions", func() (map[string]any, error) { return provider.Predictions("usd", "cpi", values) }},
		{"COT", func() (map[string]any, error) { return provider.COT("jpy", values) }},
		{"Commodity", func() (map[string]any, error) { return provider.Commodity("brent", values) }},
		{"CommoditiesLatest", func() (map[string]any, error) { return provider.CommoditiesLatest(values) }},
		{"Curves", func() (map[string]any, error) { return provider.Curves("usd", values) }},
		{"CurveProxies", func() (map[string]any, error) { return provider.CurveProxies("usd", values) }},
		{"ForwardCurves", func() (map[string]any, error) { return provider.ForwardCurves("usd", values) }},
		{"RateDifferentials", func() (map[string]any, error) { return provider.RateDifferentials("eur", "usd", values) }},
		{"ForwardDifferentials", func() (map[string]any, error) { return provider.ForwardDifferentials("eur", "usd", values) }},
		{"MarketSessions", func() (map[string]any, error) { return provider.MarketSessions(values) }},
		{"RiskSentiment", func() (map[string]any, error) { return provider.RiskSentiment(values) }},
		{"News", func() (map[string]any, error) { return provider.News("usd", values) }},
		{"PressReleases", func() (map[string]any, error) { return provider.PressReleases("usd", values) }},
	}
	for _, tc := range helpers {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.fn()
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
		assert.Equal(t, "test-key", r.URL.Query().Get("api_key"), "GraphQL should pass query auth")
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

func TestSetupRequiresAPIKey(t *testing.T) {
	var provider FXMacroData
	err := provider.Setup(base.Settings{Name: "FXMacroData"})
	assert.ErrorIs(t, err, errAPIKeyNotSet, "Setup should require API key")

	err = provider.SendHTTPRequest("data_catalogue/usd", nil, &map[string]any{})
	assert.ErrorIs(t, err, errAPIKeyNotSet, "SendHTTPRequest should require API key")
}
