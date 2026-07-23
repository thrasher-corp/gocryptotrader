package currencylayer

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var c CurrencyLayer

// Either set your API key here or in env var for integration testing
var (
	apiKey      = ""
	apiKeyLevel = 0
)

func TestMain(m *testing.M) {
	if apiKey == "" {
		apiKey = os.Getenv("CURRENCYLAYER_APIKEY")
	}

	cfg := base.Settings{
		Name:      "CurrencyLayer",
		Enabled:   true,
		APIKeyLvl: apiKeyLevel,
		APIKey:    apiKey,
	}

	if err := c.Setup(cfg); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func skipIfNoAPIKey(tb testing.TB) {
	tb.Helper()
	if apiKey == "" {
		tb.Skip("No API Key configured. Set env var CURRENCYLAYER_APIKEY")
	}
}

func TestNoAPIKey(t *testing.T) {
	t.Parallel()
	n := c
	n.APIKey = ""
	_, err := n.GetSupportedCurrencies()
	assert.ErrorContains(t, err, "You have not supplied an API Access Key", "Should error APIKeyRequired")
}

func TestGetSupportedCurrencies(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	currs, err := c.GetSupportedCurrencies()
	if assert.NoError(t, err, "GetSupportedCurrencies should not error") {
		assert.Contains(t, currs, "AUD", "AUD is a valid currency")
		assert.Contains(t, currs, "USD", "USD is a valid currency") // Might fail in the near future
	}
}

func TestGetRates(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.GetRates("USD", "AUD")
	if assert.NoError(t, err, "GetRates should not error") {
		assert.Contains(t, r, "USDAUD", "Should find a USDAUD rate")
		assert.Positive(t, r["USDAUD"], "Rate should be positive")
	}
}

func TestGetliveData(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.GetliveData("EUR", "GBP")
	if assert.NoError(t, err, "GetliveData should not error") {
		assert.Contains(t, r, "GBPEUR", "Should find rate")
		assert.Positive(t, r["GBPEUR"], "Rate should be positive")
	}
}

func TestGetHistoricalData(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.GetHistoricalData("2022-09-26", []string{"USD"}, "EUR")
	if assert.NoError(t, err, "GetHistoricalData should not error") {
		assert.Contains(t, r, "EURUSD", "Should find rate")
		assert.Equal(t, 0.962232, r["EURUSD"], "Rate should be exactly correct")
	}
}

func TestConvert(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.Convert("USD", "AUD", "", 1)
	if assert.NoError(t, err, "Convert should not error") {
		assert.Positive(t, r, "Should get a positive rate")
	}
}

func TestQueryTimeFrame(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.QueryTimeFrame("2020-03-12", "2020-03-16", "USD", []string{"AUD"})
	if assert.NoError(t, err, "QueryTimeFrame should not error") {
		assert.Len(t, r, 5, "Should get correct number of days")
		a, ok := r["2020-03-16"].(map[string]any)
		assert.True(t, ok, "Has final date entry")
		assert.Equal(t, 1.6397, a["USDAUD"], "And it was a bad week")
	}
}

func TestQueryCurrencyChange(t *testing.T) {
	t.Parallel()
	skipIfNoAPIKey(t)
	r, err := c.QueryCurrencyChange("2030-03-12", "2030-03-16", "USD", []string{"AUD"})
	switch {
	case err != nil && strings.Contains(err.Error(), "insufficient API privileges, upgrade to basic to use this function"):
		t.Skip("Upgrade to Basic API plan to test Currency Change")
	case assert.NoError(t, err, "QueryCurrencyChange should not error"):
		assert.Contains(t, r, "USDAUD", "Should find change")
		assert.Positive(t, r["USDAUD"].Change, "Change should be positive")
		assert.Positive(t, r["USDAUD"].ChangePCT, "Change PCT should be positive")
	}
}
