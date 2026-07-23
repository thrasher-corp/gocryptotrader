package openexchangerates

import (
	"log"
	"os"
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

func TestMain(m *testing.M) {
	err := o.Setup(base.Settings{
		Name:      "OpenExchangeRates",
		Enabled:   true,
		APIKey:    apikey,
		APIKeyLvl: apilvl,
	})
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestGetRates(t *testing.T) {
	t.Parallel()
	_, err := o.GetRates("USD", "AUD")
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetLatest(t *testing.T) {
	t.Parallel()
	_, err := o.GetLatest("USD", "AUD", false, false)
	if err == nil {
		t.Error("GetLatest() Expected error")
	}
}

func TestGetHistoricalRates(t *testing.T) {
	t.Parallel()
	_, err := o.GetHistoricalRates("2017-12-01", "USD", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetRates() Expected error")
	}
}

func TestGetCurrencies(t *testing.T) {
	t.Parallel()
	_, err := o.GetCurrencies(true, true, true)
	if err != nil {
		t.Error("GetCurrencies() error", err)
	}
}

func TestGetTimeSeries(t *testing.T) {
	t.Parallel()
	_, err := o.GetTimeSeries("USD", "2017-12-01", "2017-12-02", []string{"CNH", "AUD", "ANG"}, false, false)
	if err == nil {
		t.Error("GetTimeSeries() Expected error")
	}
}

func TestConvertCurrency(t *testing.T) {
	t.Parallel()
	_, err := o.ConvertCurrency(1337, "USD", "AUD")
	if err == nil {
		t.Error("ConvertCurrency() Expected error")
	}
}

func TestGetOHLC(t *testing.T) {
	t.Parallel()
	_, err := o.GetOHLC("2017-07-17T08:30:00Z", "1m", "USD", []string{"AUD"}, false)
	if err == nil {
		t.Error("GetOHLC() Expected error")
	}
}

func TestGetUsageStats(t *testing.T) {
	t.Parallel()
	if _, err := o.GetUsageStats(false); err == nil {
		t.Error("GetUsageStats() Expected error")
	}
}
