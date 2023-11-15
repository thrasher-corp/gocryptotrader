package exchangeratehost

import (
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var (
	e              ExchangeRateHost
	testCurrencies = "USD,EUR,CZK"
	apiKey         = "" // Option to set your API Key here if you can't use env vars
)

func TestMain(t *testing.M) {
	if apiKey == "" {
		apiKey = os.Getenv("APIKEY")
	}
	err := e.Setup(base.Settings{
		Name:   "ExchangeRateHost",
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(t.Run())
}

func skipIfNoAPIKey(tb testing.TB) {
	tb.Helper()
	if e.APIKey == "" {
		tb.Skip("No API Key configured. Set env var APIKEY")
	}
}

func TestGetLatestRates(t *testing.T) {
	skipIfNoAPIKey(t)
	r, err := e.GetLatestRates("USD", testCurrencies, 1200, 2, "")
	if assert.NoError(t, err, "GetLatestRates should not error") {
		assert.True(t, r.Success, "Should have Success getting rates")
		_ = r
		t.Error("Need to test r")
	}
}

func TestConvertCurrency(t *testing.T) {
	skipIfNoAPIKey(t)
	c, err := e.ConvertCurrency("USD", "EUR", "", testCurrencies, "", time.Now(), 1200, 2)
	assert.NoError(t, err, "ConvertCurrency should not error")
	_ = c
	t.Error("Need to test c")
}

func TestGetHistoricRates(t *testing.T) {
	skipIfNoAPIKey(t)
	r, err := e.GetHistoricalRates(time.Time{}, "AUD", testCurrencies, 1200, 2, "")
	assert.NoError(t, err, "GetHistoricalRates should not error")
	_ = r
	t.Error("Need to test")
}

func TestGetTimeSeriesRates(t *testing.T) {
	skipIfNoAPIKey(t)
	_, err := e.GetTimeSeries(time.Time{}, time.Now(), "USD", testCurrencies, 1200, 2, "")
	assert.ErrorIs(t, err, errors.New("x"), "GetTimeSeries should error with empty start time")
	n := time.Now()
	_, err = e.GetTimeSeries(n, n, "USD", testCurrencies, 1200, 2, "")
	assert.ErrorIs(t, err, errors.New("x"), "GetTimeSeries should error with equal start/end times")
	s, err := e.GetTimeSeries(n, n.AddDate(0, -3, 0), "USD", testCurrencies, 1200, 2, "")
	assert.NoError(t, err, "GetTimeSeries should not error")
	_ = s
	t.Error("Need to test")
}

func TestGetFluctuationData(t *testing.T) {
	skipIfNoAPIKey(t)
	_, err := e.GetFluctuations(time.Time{}, time.Now(), "USD", testCurrencies, 1200, 2, "")
	assert.ErrorIs(t, err, errors.New("x"), "GetFluctuations should error with empty start time")
	n := time.Now()
	_, err = e.GetFluctuations(n, n, "USD", testCurrencies, 1200, 2, "")
	assert.ErrorIs(t, err, errors.New("x"), "GetFluctuations should error with equal start/end times")
	f, err := e.GetFluctuations(n, n.AddDate(0, -3, 0), "USD", testCurrencies, 1200, 2, "")
	_ = f
	t.Error("Need to test")
}

func TestGetSupportedSymbols(t *testing.T) {
	skipIfNoAPIKey(t)
	r, err := e.GetSupportedSymbols()
	assert.NoError(t, err, "GetSupportedSymbols should not error")
	assert.Contains(t, r.Symbols, "AUD", "Should contain the currency AUD")
}

func TestGetGetSupportedCurrencies(t *testing.T) {
	skipIfNoAPIKey(t)
	s, err := e.GetSupportedCurrencies()
	assert.NoError(t, err, "GetSupportedSymbols should not error")
	assert.Positive(t, len(s), "Should return a list of currencies")
}

func TestGetRates(t *testing.T) {
	skipIfNoAPIKey(t)
	r, err := e.GetRates("USD", "")
	assert.NoError(t, err, "GetRates should not error")
	assert.Contains(t, r, "USDAUD", "Should contain USDAUD rate")
	assert.Positive(t, r["USDAUD"], "Should contain a positive USDAUD rate")
}
