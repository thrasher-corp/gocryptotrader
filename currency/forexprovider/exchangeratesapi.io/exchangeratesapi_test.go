package exchangerates

import "testing"

var e ExchangeRates

func TestGetLatestRates(t *testing.T) {
	e.Verbose = true
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
}

func TestGetRates(t *testing.T) {
	_, err := e.GetRates("USD", "AUD")
	if err != nil {
		t.Fatalf("failed to GetRates. Err: %s", err)
	}
}
