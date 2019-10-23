package openexchangerates

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

// please set apikey for due diligence testing NOTE testing uses your allocated
// API request quota
const (
	apikey = ""
	apilvl = 2
)

var o OXR

var initialSetup bool

func setup() {
	o.Setup(base.Settings{
		Name:    "OpenExchangeRates",
		Enabled: true,
	})
	initialSetup = true
}

func TestGetRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetRates("USD", "AUD")
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetLatest(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetLatest("USD", "AUD", false, false)
	if err == nil {
		t.Error("GetLatest() Expected error")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetHistoricalRates("2017-12-01", "USD", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetCurrencies(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetCurrencies(true, true, true)
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetTimeSeries(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetTimeSeries("USD", "2017-12-01", "2017-12-02", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetTimeSeries() Expected error")
	}
}

func TestConvertCurrency(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.ConvertCurrency(1337, "USD", "AUD")
	if err == nil {
		t.Error("ConvertCurrency() Expected error")
	}
}

func TestGetOHLC(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetOHLC("2017-07-17T08:30:00Z", "1m", "USD", []string{"AUD"}, false)
	if err == nil {
		t.Error("GetOHLC() Expected error")
	}
}

func TestGetUsageStats(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := o.GetUsageStats(false)
	if err == nil {
		t.Error("GetUsageStats() Expected error")
	}
}
