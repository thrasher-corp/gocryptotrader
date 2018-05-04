package openexchangerates

import (
	"testing"
)

// please set apikey for due diligence testing NOTE testing uses your allocated
// API request quota
const (
	apikey = ""
	apilvl = 2
)

var o OXR

func TestGetRates(t *testing.T) {
	_, err := o.GetRates("USD", "AUD")
	if err == nil {
		t.Error("test failed - GetRates() error", err)
	}
}

func TestGetLatest(t *testing.T) {
	_, err := o.GetLatest("USD", "AUD", false, false)
	if err == nil {
		t.Error("test failed - GetLatest() error", err)
	}
}

func TestGetHistoricalRates(t *testing.T) {
	_, err := o.GetHistoricalRates("2017-12-01", "USD", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("test failed - GetRates() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := o.GetCurrencies(true, true, true)
	if err != nil {
		t.Error("test failed - GetCurrencies() error", err)
	}
}

func TestGetTimeSeries(t *testing.T) {
	_, err := o.GetTimeSeries("USD", "2017-12-01", "2017-12-02", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("test failed - GetTimeSeries() error", err)
	}
}

func TestConvertCurrency(t *testing.T) {
	_, err := o.ConvertCurrency(1337, "USD", "AUD")
	if err == nil {
		t.Error("test failed - ConvertCurrency() error", err)
	}
}

func TestGetOHLC(t *testing.T) {
	_, err := o.GetOHLC("2017-07-17T08:30:00Z", "1m", "USD", []string{"AUD"}, false)
	if err == nil {
		t.Error("test failed - GetOHLC() error", err)
	}
}

func TestGetUsageStats(t *testing.T) {
	_, err := o.GetUsageStats(false)
	if err == nil {
		t.Error("test failed - GetUsageStats() error", err)
	}
}
