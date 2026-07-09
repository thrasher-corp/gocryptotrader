package fxmacrodata

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

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
		t.Fatal(err)
	}
	provider.APIURL = server.URL + "/api/v1/"
	return provider, server.Close
}

func TestGetRates(t *testing.T) {
	provider, closeServer := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	rates, err := provider.GetRates("USD", "AUD,EUR")
	if err != nil {
		t.Fatal(err)
	}
	if rates["USDAUD"] != 1.5 {
		t.Fatalf("expected USDAUD 1.5, got %f", rates["USDAUD"])
	}
	if rates["USDEUR"] != 0.9 {
		t.Fatalf("expected USDEUR 0.9, got %f", rates["USDEUR"])
	}
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
			if _, err := tc.fn(); err != nil {
				t.Fatal(err)
			}
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
	if len(seen) != len(expected) {
		t.Fatalf("expected %d requests, got %d", len(expected), len(seen))
	}
	for i := range expected {
		if seen[i] != expected[i] {
			t.Fatalf("expected path %s, got %s", expected[i], seen[i])
		}
	}
}
