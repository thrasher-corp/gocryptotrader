package allratestoday

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
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

func TestGetRatesUnsupportedTargets(t *testing.T) {
	t.Parallel()
	a := newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "EUR,GBP", r.URL.Query().Get("target"), "target should drop unsupported currencies")
		_, _ = w.Write([]byte(`[{"source":"USD","target":"EUR","rate":0.891107,"time":"2026-10-22T23:07:01+0000"},{"source":"USD","target":"GBP","rate":0.75,"time":"2026-10-22T23:07:01+0000"}]`))
	}))

	rates, err := a.GetRates("USD", "RUR,EUR,TVD,GBP,ZWD,VEF")
	require.NoError(t, err, "GetRates must not error")
	assert.Equal(t, map[string]float64{"USDEUR": 0.891107, "USDGBP": 0.75}, rates, "GetRates should return rates for the supported targets")

	a = newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("a request should not be sent without a supported target")
		http.NotFound(w, r)
	}))
	_, err = a.GetRates("USD", "RUR,TVD,ZWD")
	assert.ErrorIs(t, err, errNoSupportedTargets, "GetRates should error when no requested target is supported")
}

func TestTimestampUnmarshalJSON(t *testing.T) {
	t.Parallel()
	want := time.Date(2026, 10, 22, 23, 7, 1, 0, time.UTC)
	for _, input := range []string{`"2026-10-22T23:07:01+0000"`, `"2026-10-22T23:07:01+00:00"`, `"2026-10-22T23:07:01Z"`, `"2026-10-23T01:07:01+0200"`} {
		var ts Timestamp
		require.NoErrorf(t, json.Unmarshal([]byte(input), &ts), "Unmarshal must not error for %s", input)
		assert.Truef(t, want.Equal(ts.Time()), "Unmarshal should decode %s", input)
	}

	var ts Timestamp
	require.NoError(t, json.Unmarshal([]byte(`null`), &ts), "Unmarshal must not error on null")
	assert.True(t, ts.Time().IsZero(), "null should leave a zero time")
	require.NoError(t, json.Unmarshal([]byte(`""`), &ts), "Unmarshal must not error on an empty string")
	assert.True(t, ts.Time().IsZero(), "an empty string should leave a zero time")

	assert.ErrorIs(t, json.Unmarshal([]byte(`"22/10/2026"`), &ts), errInvalidTimestamp, "Unmarshal should error on an unknown layout")
	assert.ErrorIs(t, json.Unmarshal([]byte(`1761174421`), &ts), errInvalidTimestamp, "Unmarshal should error on a non-string")
}

func TestTimestampMarshalJSON(t *testing.T) {
	t.Parallel()
	var resp RateResponse
	require.NoError(t, json.Unmarshal([]byte(`{"source":"USD","target":"EUR","rate":0.891107,"time":"2026-10-22T23:07:01+0000"}`), &resp), "Unmarshal must not error")
	data, err := json.Marshal(resp.Time)
	require.NoError(t, err, "Marshal must not error")
	assert.Equal(t, `"2026-10-22T23:07:01Z"`, string(data), "Marshal should emit RFC 3339")

	var roundTrip Timestamp
	require.NoError(t, json.Unmarshal(data, &roundTrip), "Unmarshal must not error on marshalled output")
	assert.True(t, resp.Time.Time().Equal(roundTrip.Time()), "a marshalled timestamp should round-trip")

	var offset Timestamp
	require.NoError(t, json.Unmarshal([]byte(`"2026-10-23T01:07:01+0200"`), &offset), "Unmarshal must not error on a non-zero offset")
	data, err = json.Marshal(offset)
	require.NoError(t, err, "Marshal must not error on a non-zero offset")
	var decoded time.Time
	require.NoError(t, json.Unmarshal(data, &decoded), "time.Time must decode the marshalled output")
	assert.True(t, offset.Time().Equal(decoded), "a marshalled timestamp should decode to the same instant")
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

func TestGetRatesSkipsUnusableRows(t *testing.T) {
	t.Parallel()
	a := newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"source":"USD","target":"EUR","rate":0.891107,"time":"2026-10-22T23:07:01+0000"},null,{},{"source":"USD","rate":0.75},{"target":"AUD","rate":1.52},{"source":"US","target":"EUR","rate":1},{"source":"USD","target":"EU","rate":1},{"source":"USDX","target":"EUR","rate":1},{"source":"USD","target":"EURO","rate":1},{"source":"USD","target":"GBP","rate":0},{"source":"USD","target":"JPY","rate":null},{"source":"USD","target":"CAD","rate":-0.73}]`))
	}))
	rates, err := a.GetRates("USD", "EUR,GBP,JPY,CAD")
	require.NoError(t, err, "GetRates must not error while a usable row remains")
	assert.Equal(t, map[string]float64{"USDEUR": 0.891107}, rates, "GetRates should skip rows without both codes or a positive rate")

	a = newTestProvider(t, "placeholder", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{},{"source":"USD","target":"EUR","rate":0}]`))
	}))
	_, err = a.GetRates("USD", "EUR")
	assert.ErrorIs(t, err, errNoRatesReturn, "GetRates should error when no row is usable")
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
