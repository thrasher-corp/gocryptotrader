package allratestoday

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func newTestProvider(t *testing.T, apiKey string, handler http.Handler) *AllRatesToday {
	t.Helper()
	server := httptest.NewTestServer(t, handler)
	t.Cleanup(server.Close)
	a := &AllRatesToday{}
	require.NoError(t, a.Setup(base.Settings{
		Name:            "AllRatesToday",
		Enabled:         true,
		APIKey:          apiKey,
		PrimaryProvider: true,
	}), "Setup must not error")
	require.NoError(t, a.Requester.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	a.APIURL = server.URL + "/api/v1/"
	return a
}

func TestGetRates(t *testing.T) {
	t.Parallel()
	a := newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/rates", r.URL.Path, "request path should be the rates endpoint")
		assert.Equal(t, "Bearer placeholder", r.Header.Get("Authorization"), "request should use Bearer authorisation")
		assert.Equal(t, "USD", r.URL.Query().Get("source"), "source should be the base currency")
		assert.Equal(t, "EUR,GBP", r.URL.Query().Get("target"), "target should list the requested symbols")
		_, _ = w.Write([]byte(`[{"source":"USD","target":"EUR","rate":0.85,"time":"2026-10-22T12:00:00Z"},{"source":"USD","target":"GBP","rate":0.75,"time":"2026-10-22T12:00:00Z"}]`))
	}))

	rates, err := a.GetRates("usd", "EUR, GBP")
	require.NoError(t, err, "GetRates must not error")
	assert.Equal(t, map[string]float64{"USDEUR": 0.85, "USDGBP": 0.75}, rates, "GetRates should return rates keyed by base and target")
}

func TestGetRatesAllTargets(t *testing.T) {
	t.Parallel()
	a := newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.False(t, r.URL.Query().Has("target"), "target should be omitted when no symbols are requested")
		_, _ = w.Write([]byte(`[{"source":"EUR","target":"USD","rate":1.17,"time":"2026-10-22T12:00:00Z"}]`))
	}))

	rates, err := a.GetRates("EUR", "")
	require.NoError(t, err, "GetRates must not error")
	assert.Equal(t, 1.17, rates["EURUSD"], "EURUSD should match the mocked rate")
}

func TestGetRatesErrors(t *testing.T) {
	t.Parallel()
	a := newTestProvider(t, "", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a request should not be sent without an API key")
		http.NotFound(w, r)
	}))
	_, err := a.GetRates("USD", "EUR")
	assert.ErrorIs(t, err, errAPIKeyNotSet, "GetRates should error when the API key is not set")

	a.APIKey = "placeholder"
	_, err = a.GetRates(" ", "EUR")
	assert.ErrorIs(t, err, errEmptyBase, "GetRates should error on an empty base currency")

	a = newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	}))
	_, err = a.GetRates("USD", "EUR")
	assert.ErrorIs(t, err, errNoRatesReturn, "GetRates should error when no rates are returned")

	a = newTestProvider(t, "invalid", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"Invalid API key"}`))
	}))
	_, err = a.GetRates("USD", "EUR")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetRates should error on a non-200 response")
}

func TestGetSupportedCurrencies(t *testing.T) {
	t.Parallel()
	var a AllRatesToday
	currencies, err := a.GetSupportedCurrencies()
	require.NoError(t, err, "GetSupportedCurrencies must not error")
	assert.Contains(t, currencies, "USD", "supported currencies should include USD")
	assert.Contains(t, currencies, "EUR", "supported currencies should include EUR")
}
