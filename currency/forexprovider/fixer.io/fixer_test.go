package fixer

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

// Please set API key and apikey subscription level for correct due diligence
// testing - NOTE please be aware tests will diminish your monthly API calls

var f Fixer

var isSetup bool

func setup(t *testing.T) {
	t.Helper()
	if !isSetup {
		err := f.Setup(base.Settings{})
		if err != nil {
			t.Fatal("Setup error", err)
		}
		isSetup = true
	}
}

func TestGetRates(t *testing.T) {
	setup(t)
	_, err := f.GetRates("EUR", "AUD")
	if err == nil {
		t.Error("fixer GetRates() Expected error")
	}
}

func TestGetLatestRates(t *testing.T) {
	setup(t)
	_, err := f.GetLatestRates("EUR", "AUD")
	if err == nil {
		t.Error("fixer GetLatestRates() Expected error")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	setup(t)
	_, err := f.GetHistoricalRates("2013-12-24", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetHistoricalRates() Expected error")
	}
}

func TestConvertCurrency(t *testing.T) {
	setup(t)
	_, err := f.ConvertCurrency("AUD", "EUR", "", 1337)
	if err == nil {
		t.Error("fixer ConvertCurrency() Expected error")
	}
}

func TestGetTimeSeriesData(t *testing.T) {
	setup(t)
	_, err := f.GetTimeSeriesData("2013-12-24", "2013-12-25", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetTimeSeriesData() Expected error")
	}
}

func TestGetFluctuationData(t *testing.T) {
	setup(t)
	_, err := f.GetFluctuationData("2013-12-24", "2013-12-25", "EUR", []string{"AUD,KRW"})
	if err == nil {
		t.Error("fixer GetFluctuationData() Expected error")
	}
}
