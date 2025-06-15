package exchangerates

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var e ExchangeRates

const (
	apiKey      = ""
	apiKeyLevel = apiKeyFree // Adjust this if your API key level is different
)

func TestMain(t *testing.M) {
	err := e.Setup(base.Settings{
		Name:      "ExchangeRates",
		APIKey:    apiKey,
		APIKeyLvl: apiKeyLevel,
	})
	if err != nil && !(errors.Is(err, errAPIKeyNotSet)) {
		log.Fatal(err)
	}
	os.Exit(t.Run())
}

func isAPIKeySet() bool {
	return e.APIKey != ""
}

func TestGetSymbols(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	r, err := e.GetSymbols()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) == 0 {
		t.Error("expected rates map greater than 0")
	}
}

func TestGetLatestRates(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	result, err := e.GetLatestRates("", "")
	if err != nil {
		t.Fatal(err)
	}

	if result.Base != "EUR" {
		t.Fatalf("unexpected result. Base currency should be EUR")
	}

	if result.Rates["EUR"] != 1 {
		t.Fatalf("unexpected result. EUR value should be 1")
	}

	if len(result.Rates) <= 1 {
		t.Fatalf("unexpected result. Rates map should be 1")
	}

	if e.APIKeyLvl <= apiKeyFree {
		_, err = e.GetLatestRates("USD", "")
		assert.ErrorIs(t, err, errCannotSetBaseCurrencyOnFreePlan)
	}

	result, err = e.GetLatestRates("EUR", "AUD")
	if err != nil {
		t.Fatalf("failed to GetLatestRates. Err: %s", err)
	}

	if result.Base != "EUR" {
		t.Fatalf("unexpected result. Base currency should be EUR")
	}

	if len(result.Rates) != 1 {
		t.Fatalf("unexpected result. Rates len should be 1")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	_, err := e.GetHistoricalRates(time.Time{}, "EUR", []string{"AUD"})
	if err == nil {
		t.Fatalf("invalid date should throw an error")
	}

	if e.APIKeyLvl <= apiKeyFree {
		_, err = e.GetHistoricalRates(time.Now(), "USD", []string{"AUD"})
		assert.ErrorIs(t, err, errCannotSetBaseCurrencyOnFreePlan)
	}

	_, err = e.GetHistoricalRates(time.Now(), "EUR", []string{"AUD,USD"})
	if err != nil {
		t.Error(err)
	}
}

func TestConvertCurrency(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	if e.APIKeyLvl <= apiKeyFree {
		_, err := e.ConvertCurrency("USD", "AUD", 1000, time.Time{})
		assert.ErrorIs(t, err, errAPIKeyLevelRestrictedAccess)

		return
	}

	_, err := e.ConvertCurrency("", "AUD", 1000, time.Time{})
	if err == nil {
		t.Errorf("no from currency should throw an error")
	}

	_, err = e.ConvertCurrency("USD", "AUD", 1000, time.Now())
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimeSeriesRates(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	if e.APIKeyLvl <= apiKeyFree {
		_, err := e.GetTimeSeriesRates(time.Time{}, time.Time{}, "EUR", []string{"EUR,USD"})
		assert.ErrorIs(t, err, errAPIKeyLevelRestrictedAccess)

		return
	}

	_, err := e.GetTimeSeriesRates(time.Time{}, time.Time{}, "USD", []string{"EUR", "USD"})
	require.ErrorIs(t, err, errStartEndDatesInvalid)

	tmNow := time.Now()
	_, err = e.GetTimeSeriesRates(tmNow.AddDate(0, 1, 0), tmNow, "USD", []string{"EUR", "USD"})
	require.ErrorIs(t, err, errStartAfterEnd)

	_, err = e.GetTimeSeriesRates(tmNow.AddDate(0, -1, 0), tmNow, "EUR", []string{"AUD,USD"})
	if err != nil {
		t.Error(err)
	}
}

func TestGetFluctuation(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	if e.APIKeyLvl <= apiKeyFree {
		_, err := e.GetFluctuations(time.Time{}, time.Time{}, "EUR", "")
		assert.ErrorIs(t, err, errAPIKeyLevelRestrictedAccess)

		return
	}

	tmNow := time.Now()
	_, err := e.GetFluctuations(tmNow.AddDate(0, -1, 0), tmNow, "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
}

func TestCleanCurrencies(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	result := e.cleanCurrencies("EUR", "EUR,AUD")
	if result != "AUD" {
		t.Fatalf("AUD should be the only symbol")
	}

	if e.cleanCurrencies("EUR", "RUR") != "RUB" {
		t.Fatalf("unexpected result. RUB should be the only symbol")
	}

	if e.cleanCurrencies("EUR", "AUD,BLA") != "AUD" {
		t.Fatalf("AUD should be the only symbol")
	}
}

func TestGetRates(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	if _, err := e.GetRates("EUR", ""); err != nil {
		t.Fatalf("failed to GetRates. Err: %s", err)
	}
}

func TestGetSupportedCurrencies(t *testing.T) {
	if !isAPIKeySet() {
		t.Skip("API key not set, skipping test")
	}

	r, err := e.GetSupportedCurrencies()
	if err != nil {
		t.Fatal(err)
	}
	if len(r) == 0 {
		t.Error("expected greater than zero supported symbols")
	}
}
