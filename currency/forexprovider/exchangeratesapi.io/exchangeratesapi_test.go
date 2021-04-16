package exchangerates

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency/forexprovider/base"
)

var e ExchangeRates

var initialSetup bool

func setup() {
	e.Setup(base.Settings{
		Name:    "ExchangeRates",
		Enabled: true,
	})
	initialSetup = true
}

func TestGetLatestRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	result, err := e.GetLatestRates("USD", "")
	if err != nil {
		t.Fatalf("failed to GetLatestRates. Err: %s", err)
	}

	if result.Base != "USD" {
		t.Fatalf("unexepcted result. Base currency should be USD")
	}

	if result.Rates["USD"] != 1 {
		t.Fatalf("unexepcted result. USD value should be 1")
	}

	if len(result.Rates) <= 1 {
		t.Fatalf("unexepcted result. Rates map should be 1")
	}

	result, err = e.GetLatestRates("", "AUD")
	if err != nil {
		t.Fatalf("failed to GetLatestRates. Err: %s", err)
	}

	if result.Base != "EUR" {
		t.Fatalf("unexepcted result. Base currency should be EUR")
	}

	if len(result.Rates) != 1 {
		t.Fatalf("unexepcted result. Rates len should be 1")
	}
}

func TestCleanCurrencies(t *testing.T) {
	if !initialSetup {
		setup()
	}
	result := cleanCurrencies("USD", "USD,AUD")
	if result != "AUD" {
		t.Fatalf("unexpected result. AUD should be the only symbol")
	}

	result = cleanCurrencies("", "EUR,USD")
	if result != "USD" {
		t.Fatalf("unexpected result. USD should be the only symbol")
	}

	if cleanCurrencies("EUR", "RUR") != "RUB" {
		t.Fatalf("unexpected result. RUB should be the only symbol")
	}

	if cleanCurrencies("EUR", "AUD,BLA") != "AUD" {
		t.Fatalf("unexpected result. AUD should be the only symbol")
	}
}

func TestGetRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := e.GetRates("USD", "AUD")
	if err != nil {
		t.Fatalf("failed to GetRates. Err: %s", err)
	}
}

func TestGetHistoricalRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := e.GetHistoricalRates("-1", "USD", []string{"AUD"})
	if err == nil {
		t.Fatalf("unexpected result. Invalid date should throw an error")
	}

	_, err = e.GetHistoricalRates("2010-01-12", "USD", []string{"EUR,USD"})
	if err != nil {
		t.Fatalf("failed to GetHistoricalRates. Err: %s", err)
	}
}

func TestGetTimeSeriesRates(t *testing.T) {
	if !initialSetup {
		setup()
	}
	_, err := e.GetTimeSeriesRates("", "", "USD", []string{"EUR", "USD"})
	if err == nil {
		t.Fatal("unexpected result. Empty startDate endDate params should throw an error")
	}

	resp, err := e.GetTimeSeriesRates("2018-01-01", "2018-09-01", "USD", []string{"EUR,USD"})
	t.Log(resp)
	if err != nil {
		t.Fatalf("failed to TestGetTimeSeriesRates. Err: %s", err)
	}

	resp, err = e.GetTimeSeriesRates("-1", "-1", "USD", []string{"EUR,USD"})
	if err == nil {
		t.Fatal("unexpected result. Invalid date params should throw an error")
	}
}
