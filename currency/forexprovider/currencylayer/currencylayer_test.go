package currencylayer

import (
	"testing"
)

var c CurrencyLayer

// please set your API key here for due diligence testing NOTE be aware you will
// minimize your API calls using this test.
const (
	APIkey   = ""
	Apilevel = 3
)

func TestGetRates(t *testing.T) {
	_, err := c.GetRates("USD", "AUD")
	if err == nil {
		t.Error("test error - currencylayer GetRates() error", err)
	}
}

func TestGetSupportedCurrencies(t *testing.T) {
	_, err := c.GetSupportedCurrencies()
	if err == nil {
		t.Error("test error - currencylayer GetSupportedCurrencies() error", err)
	}
}

func TestGetliveData(t *testing.T) {
	_, err := c.GetliveData("AUD", "USD")
	if err == nil {
		t.Error("test error - currencylayer GetliveData() error", err)
	}
}

func TestGetHistoricalData(t *testing.T) {
	_, err := c.GetHistoricalData("2016-12-15", []string{"AUD"}, "USD")
	if err == nil {
		t.Error("test error - currencylayer GetHistoricalData() error", err)
	}
}

func TestConvert(t *testing.T) {
	_, err := c.Convert("USD", "AUD", "", 1)
	if err == nil {
		t.Error("test error - currencylayer Convert() error")
	}
}

func TestQueryTimeFrame(t *testing.T) {
	_, err := c.QueryTimeFrame("2010-12-0", "2010-12-5", "USD", []string{"AUD"})
	if err == nil {
		t.Error("test error - currencylayer QueryTimeFrame() error")
	}
}

func TestQueryCurrencyChange(t *testing.T) {
	_, err := c.QueryCurrencyChange("2010-12-0", "2010-12-5", "USD", []string{"AUD"})
	if err == nil {
		t.Error("test error - currencylayer QueryCurrencyChange() error")
	}
}
