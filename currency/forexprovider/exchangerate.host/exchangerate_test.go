package exchangeratehost

import (
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var (
	e              ExchangeRateHost
	testCurrencies = "USD,EUR,CZK"
)

func TestMain(t *testing.M) {
	e.Setup(base.Settings{
		Name: "ExchangeRateHost",
	})
	os.Exit(t.Run())
}

func TestGetLatestRates(t *testing.T) {
	_, err := e.GetLatestRates("USD", testCurrencies, 1200, 2, "")
	if err != nil {
		t.Error(err)
	}
}

func TestConvertCurrency(t *testing.T) {
	_, err := e.ConvertCurrency("USD", "EUR", "", testCurrencies, "", time.Now(), 1200, 2)
	if err != nil {
		t.Error(err)
	}
}

func TestGetHistoricRates(t *testing.T) {
	_, err := e.GetHistoricalRates(time.Time{}, "AUD", testCurrencies, 1200, 2, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetTimeSeriesRates(t *testing.T) {
	_, err := e.GetTimeSeries(time.Time{}, time.Now(), "USD", testCurrencies, 1200, 2, "")
	if err == nil {
		t.Error("empty start time show throw an error")
	}
	tmNow := time.Now()
	_, err = e.GetTimeSeries(tmNow, tmNow, "USD", testCurrencies, 1200, 2, "")
	if err == nil {
		t.Error("equal times show throw an error")
	}
	tmStart := tmNow.AddDate(0, -3, 0)
	_, err = e.GetTimeSeries(tmStart, tmNow, "USD", testCurrencies, 1200, 2, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetFluctuationData(t *testing.T) {
	_, err := e.GetFluctuations(time.Time{}, time.Now(), "USD", testCurrencies, 1200, 2, "")
	if err == nil {
		t.Error("empty start time show throw an error")
	}
	tmNow := time.Now()
	_, err = e.GetFluctuations(tmNow, tmNow, "USD", testCurrencies, 1200, 2, "")
	if err == nil {
		t.Error("equal times show throw an error")
	}
	tmStart := tmNow.AddDate(0, -3, 0)
	_, err = e.GetFluctuations(tmStart, tmNow, "USD", testCurrencies, 1200, 2, "")
	if err != nil {
		t.Error(err)
	}
}

func TestGetSupportedSymbols(t *testing.T) {
	r, err := e.GetSupportedSymbols()
	if err != nil {
		t.Fatal(err)
	}
	_, ok := r.Symbols["AUD"]
	if !ok {
		t.Error("should contain AUD")
	}
}

func TestGetGetSupportedCurrencies(t *testing.T) {
	s, err := e.GetSupportedCurrencies()
	if err != nil {
		t.Fatal(err)
	}
	if len(s) == 0 {
		t.Error("supported currencies should be greater than 0")
	}
}

func TestGetRates(t *testing.T) {
	r, err := e.GetRates("USD", "")
	if err != nil {
		t.Fatal(err)
	}
	if rate := r["USDAUD"]; rate == 0 {
		t.Error("rate of USDAUD should be set")
	}
}
