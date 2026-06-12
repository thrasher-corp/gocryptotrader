package coinmarketcap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Please set API keys to test endpoint
const (
	apikey              = ""
	apiAccountPlanLevel = ""
)

func skipIfLiveCredentialsUnavailable(t *testing.T, c *Coinmarketcap, minAllowable uint8) {
	t.Helper()
	switch {
	case apiAccountPlanLevel != "" && apikey != "":
		if err := c.CheckAccountPlan(minAllowable); err != nil {
			t.Skip("CoinMarketCap account plan not allowed for function, please review or upgrade plan to test")
		}
		return
	default:
		t.Skip("CoinMarketCap API key or account plan not set")
	}
}

func newConfiguredClient(t *testing.T) *Coinmarketcap {
	t.Helper()

	c := &Coinmarketcap{}
	c.SetDefaults()
	plan := apiAccountPlanLevel
	if plan == "" {
		plan = "basic"
	}
	cfg := Settings{
		APIKey:      apikey,
		AccountPlan: plan,
		Enabled:     true,
	}
	err := c.Setup(cfg)
	require.NoError(t, err)
	return c
}

func newSyntheticClient(t *testing.T, responses map[string]string) (client *Coinmarketcap, closeFn func()) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp, ok := responses[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"status":{"error_code":404,"error_message":"not found"}}`))
			return
		}
		_, _ = w.Write([]byte(resp))
	}))
	c := &Coinmarketcap{}
	c.SetDefaults()
	c.APIUrl = server.URL
	c.APIkey = "test"
	c.Plan = Enterprise
	return c, server.Close
}

func TestSetDefaults(t *testing.T) {
	t.Parallel()
	var c Coinmarketcap
	c.SetDefaults()
	assert.Equal(t, "CoinMarketCap", c.Name, "SetDefaults should name")
	assert.Equal(t, baseURL, c.APIUrl, "SetDefaults should populate url with default")
	assert.Empty(t, c.APIkey, "SetDefaults should not populate API key")
	assert.NotNil(t, c.Requester, c.APIUrl, "SetDefaults should populate requester")
}

func TestSetup(t *testing.T) {
	t.Parallel()
	var c Coinmarketcap
	c.SetDefaults()
	cfg := Settings{}
	cfg.APIKey = apikey
	cfg.AccountPlan = apiAccountPlanLevel
	cfg.Enabled = true
	if cfg.AccountPlan == "" {
		cfg.AccountPlan = "basic"
	}

	err := c.Setup(cfg)
	require.NoError(t, err)
}

func TestCheckAccountPlan(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name      string
		plan      uint8
		min       uint8
		expectErr bool
	}{
		{name: "basic allows basic", plan: Basic, min: Basic},
		{name: "basic blocks hobbyist", plan: Basic, min: Hobbyist, expectErr: true},
		{name: "startup allows hobbyist", plan: Startup, min: Hobbyist},
		{name: "startup blocks standard", plan: Startup, min: Standard, expectErr: true},
		{name: "enterprise allows professional", plan: Enterprise, min: Professional},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := &Coinmarketcap{Plan: tc.plan}
			err := c.CheckAccountPlan(tc.min)
			if tc.expectErr {
				assert.ErrorIs(t, err, errFunctionUseNotAllowed, "CheckAccountPlan should return expected error")
				return
			}
			assert.NoError(t, err, "CheckAccountPlan should not error")
		})
	}
}

func TestGetCryptocurrencyInfo(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyInfo(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyIDMap(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	data, err := c.GetCryptocurrencyIDMap()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestGetCryptocurrencyHistoricalListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetCryptocurrencyHistoricalListings()
	assert.Error(t, err)
}

func TestGetCryptocurrencyLatestListing(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyLatestListing(0, 0)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyLatestMarketPairs(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyOHLCHistorical(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetCryptocurrencyOHLCLatest(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Startup)
	_, err := c.GetCryptocurrencyOHLCLatest(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetCryptocurrencyLatestQuotes(1)
	assert.NoError(t, err)
}

func TestGetCryptocurrencyHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Startup)
	_, err := c.GetExchangeInfo(1)
	assert.NoError(t, err)
}

func TestGetExchangeMap(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Startup)
	_, err := c.GetExchangeMap(0, 0)
	assert.NoError(t, err)
}

func TestGetExchangeHistoricalListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetExchangeHistoricalListings()
	// TODO: update this once the feature above is implemented
	assert.ErrorIs(t, err, errEndpointNotAvailable, "GetExchangeHistoricalListings should return expected error")
}

func TestGetExchangeLatestListings(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	_, err := c.GetExchangeLatestListings()
	// TODO: update this once the feature above is implemented
	assert.ErrorIs(t, err, errEndpointNotAvailable, "GetExchangeLatestListings should return expected error")
}

func TestGetExchangeLatestMarketPairs(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetExchangeLatestMarketPairs(1, 0, 0)
	assert.NoError(t, err)
}

func TestGetExchangeLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetExchangeLatestQuotes(1)
	assert.NoError(t, err)
}

func TestGetExchangeHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetGlobalMeticLatestQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Basic)
	_, err := c.GetGlobalMeticLatestQuotes()
	assert.NoError(t, err)
}

func TestGetGlobalMeticHistoricalQuotes(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Standard)
	_, err := c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
	assert.NoError(t, err)
}

func TestGetPriceConversion(t *testing.T) {
	t.Parallel()
	c := newConfiguredClient(t)
	skipIfLiveCredentialsUnavailable(t, c, Hobbyist)
	_, err := c.GetPriceConversion(0, 1, time.Now())
	assert.NoError(t, err)
}

func TestSetAccountPlan(t *testing.T) {
	t.Parallel()
	var c Coinmarketcap
	accPlans := []string{"basic", "startup", "hobbyist", "standard", "professional", "enterprise"}
	for _, plan := range accPlans {
		err := c.SetAccountPlan(plan)
		assert.NoError(t, err)

		switch plan {
		case "basic":
			assert.Equal(t, Basic, c.Plan)
		case "startup":
			assert.Equal(t, Startup, c.Plan)
		case "hobbyist":
			assert.Equal(t, Hobbyist, c.Plan)
		case "standard":
			assert.Equal(t, Standard, c.Plan)
		case "professional":
			assert.Equal(t, Professional, c.Plan)
		case "enterprise":
			assert.Equal(t, Enterprise, c.Plan)
		}
	}
}

func TestNewFromSettingsAndSetupDisabled(t *testing.T) {
	t.Parallel()
	cfg := Settings{Enabled: true, AccountPlan: "basic", APIKey: "x"}
	client, err := NewFromSettings(cfg)
	require.NoError(t, err)
	assert.True(t, client.Enabled)
	assert.Equal(t, Basic, client.Plan)

	var disabled Coinmarketcap
	disabled.SetDefaults()
	err = disabled.Setup(Settings{Enabled: false})
	require.NoError(t, err)
	assert.False(t, disabled.Enabled)
}

func TestQuoteMapUnmarshal(t *testing.T) {
	t.Parallel()
	var qm QuoteMap
	err := qm.UnmarshalJSON([]byte(`{"USD":{"price":1.23},"BTC":{"price":0.1}}`))
	require.NoError(t, err)
	assert.Equal(t, 1.23, qm["USD"].Price)
	assert.Equal(t, 0.1, qm["BTC"].Price)

	err = qm.UnmarshalJSON([]byte(`[{"USD":{"price":2.34}},{"ETH":{"price":3.45}}]`))
	require.NoError(t, err)
	assert.Equal(t, 2.34, qm["USD"].Price)
	assert.Equal(t, 3.45, qm["ETH"].Price)
}

func TestAPIErrorCodeUnmarshal(t *testing.T) {
	t.Parallel()
	var code APIErrorCode
	err := code.UnmarshalJSON([]byte(`123`))
	require.NoError(t, err)
	assert.Equal(t, APIErrorCode(123), code)

	err = code.UnmarshalJSON([]byte(`"456"`))
	require.NoError(t, err)
	assert.Equal(t, APIErrorCode(456), code)

	err = code.UnmarshalJSON([]byte(`"bad"`))
	assert.Error(t, err)
}

func TestCoinmarketcapEndpointSuccessSynthetic(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		path    string
		payload string
		invoke  func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", "/v2/cryptocurrency/info", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", "/v1/cryptocurrency/map", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", "/v3/cryptocurrency/listings/latest", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestMarketPairs", "/v2/cryptocurrency/market-pairs/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestMarketPairs(1, 1, 2); return err }},
		{"GetCryptocurrencyOHLCHistorical", "/v2/cryptocurrency/ohlcv/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetCryptocurrencyOHLCLatest", "/v2/cryptocurrency/ohlcv/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyOHLCLatest(1); return err }},
		{"GetCryptocurrencyLatestQuotes", "/v3/cryptocurrency/quotes/latest", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetCryptocurrencyHistoricalQuotes", "/v3/cryptocurrency/quotes/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetExchangeInfo", "/v1/exchange/info", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeInfo(1); return err }},
		{"GetExchangeMap", "/v1/exchange/map", `{"data":[],"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeMap(1, 2); return err }},
		{"GetExchangeLatestMarketPairs", "/v1/exchange/market-pairs/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestMarketPairs(1, 1, 2); return err }},
		{"GetExchangeLatestQuotes", "/v1/exchange/quotes/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestQuotes(1); return err }},
		{"GetExchangeHistoricalQuotes", "/v1/exchange/quotes/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetExchangeHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetGlobalMeticLatestQuotes", "/v1/global-metrics/quotes/latest", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetGlobalMeticHistoricalQuotes", "/v1/global-metrics/quotes/historical", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error {
			_, err := c.GetGlobalMeticHistoricalQuotes(time.Now().Add(-time.Hour), time.Now())
			return err
		}},
		{"GetPriceConversion", "/v2/tools/price-conversion", `{"data":{},"status":{"error_code":0,"error_message":""}}`, func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Now()); return err }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{tc.path: tc.payload})
			defer done()
			err := tc.invoke(client)
			require.NoErrorf(t, err, "%s must not error", tc.name)
		})
	}
}

func TestCoinmarketcapEndpointStatusErrorSynthetic(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name    string
		path    string
		payload string
		invoke  func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", "/v2/cryptocurrency/info", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", "/v1/cryptocurrency/map", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", "/v3/cryptocurrency/listings/latest", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestQuotes", "/v3/cryptocurrency/quotes/latest", `{"data":[],"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetGlobalMeticLatestQuotes", "/v1/global-metrics/quotes/latest", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetPriceConversion", "/v2/tools/price-conversion", `{"data":{},"status":{"error_code":1001,"error_message":"boom"}}`, func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Now()); return err }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{tc.path: tc.payload})
			defer done()
			err := tc.invoke(client)
			assert.ErrorIs(t, err, errAPIResponse, "endpoint should return expected error")
			assert.ErrorContains(t, err, "boom", "endpoint should include API error message")
		})
	}
}

func TestCoinmarketcapEndpointRequestFailureSynthetic(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		invoke func(*Coinmarketcap) error
	}{
		{"GetCryptocurrencyInfo", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyInfo(1); return err }},
		{"GetCryptocurrencyIDMap", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyIDMap(); return err }},
		{"GetCryptocurrencyLatestListing", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestListing(1, 2); return err }},
		{"GetCryptocurrencyLatestMarketPairs", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestMarketPairs(1, 1, 2); return err }},
		{"GetCryptocurrencyOHLCHistorical", func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyOHLCHistorical(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetCryptocurrencyOHLCLatest", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyOHLCLatest(1); return err }},
		{"GetCryptocurrencyLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetCryptocurrencyLatestQuotes(1); return err }},
		{"GetCryptocurrencyHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetCryptocurrencyHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetExchangeInfo", func(c *Coinmarketcap) error { _, err := c.GetExchangeInfo(1); return err }},
		{"GetExchangeMap", func(c *Coinmarketcap) error { _, err := c.GetExchangeMap(1, 2); return err }},
		{"GetExchangeLatestMarketPairs", func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestMarketPairs(1, 1, 2); return err }},
		{"GetExchangeLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetExchangeLatestQuotes(1); return err }},
		{"GetExchangeHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetExchangeHistoricalQuotes(1, time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetGlobalMeticLatestQuotes", func(c *Coinmarketcap) error { _, err := c.GetGlobalMeticLatestQuotes(); return err }},
		{"GetGlobalMeticHistoricalQuotes", func(c *Coinmarketcap) error {
			_, err := c.GetGlobalMeticHistoricalQuotes(time.Now().Add(-time.Hour), time.Time{})
			return err
		}},
		{"GetPriceConversion", func(c *Coinmarketcap) error { _, err := c.GetPriceConversion(1, 1, time.Time{}); return err }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client, done := newSyntheticClient(t, map[string]string{})
			defer done()
			err := tc.invoke(client)
			assert.Error(t, err)
		})
	}
}

func TestCoinmarketcapAccountPlanGatesSynthetic(t *testing.T) {
	t.Parallel()
	c, done := newSyntheticClient(t, map[string]string{})
	defer done()

	c.Plan = Basic

	_, err := c.GetCryptocurrencyLatestMarketPairs(1, 0, 0)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetCryptocurrencyLatestMarketPairs should return expected error")
	_, err = c.GetCryptocurrencyOHLCHistorical(1, time.Now(), time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetCryptocurrencyOHLCHistorical should return expected error")
	_, err = c.GetCryptocurrencyOHLCLatest(1)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetCryptocurrencyOHLCLatest should return expected error")
	_, err = c.GetCryptocurrencyHistoricalQuotes(1, time.Now(), time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetCryptocurrencyHistoricalQuotes should return expected error")
	_, err = c.GetExchangeInfo(1)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetExchangeInfo should return expected error")
	_, err = c.GetExchangeMap(1, 1)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetExchangeMap should return expected error")
	_, err = c.GetExchangeLatestMarketPairs(1, 1, 1)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetExchangeLatestMarketPairs should return expected error")
	_, err = c.GetExchangeLatestQuotes(1)
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetExchangeLatestQuotes should return expected error")
	_, err = c.GetExchangeHistoricalQuotes(1, time.Now(), time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetExchangeHistoricalQuotes should return expected error")
	_, err = c.GetGlobalMeticHistoricalQuotes(time.Now(), time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetGlobalMeticHistoricalQuotes should return expected error")
	_, err = c.GetPriceConversion(1, 1, time.Now())
	assert.ErrorIs(t, err, errFunctionUseNotAllowed, "GetPriceConversion should return expected error")
}
