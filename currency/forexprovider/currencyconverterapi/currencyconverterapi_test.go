package currencyconverter

import (
	"testing"
)

var c CurrencyConverter

func TestGetRates(t *testing.T) {
	result, err := c.GetRates("USD", "AUD")
	if err != nil {
		t.Error("Test Error. CurrencyConverter GetRates() error", err)
	}

	if len(result) != 1 {
		t.Fatal("Test error. Expected 2 rates")
	}

	result, err = c.GetRates("USD", "AUD,EUR")
	if err != nil {
		t.Error("Test Error. CurrencyConverter GetRates() error", err)
	}

	if len(result) != 2 {
		t.Fatal("Test error. Expected 2 rates")
	}

	result, err = c.GetRates("USD", "AUD,EUR,GBP")
	if err != nil {
		t.Error("Test Error. CurrencyConverter GetRates() error", err)
	}

	if len(result) != 3 {
		t.Fatal("Test error. Expected 3 rates")
	}

	result, err = c.GetRates("USD", "AUD,EUR,GBP,CNY")
	if err != nil {
		t.Error("Test Error. CurrencyConverter GetRates() error", err)
	}

	if len(result) != 4 {
		t.Fatal("Test error. Expected 4 rates")
	}
}
func TestConvertMany(t *testing.T) {
	currencies := []string{"USD_AUD", "USD_EUR"}
	_, err := c.ConvertMany(currencies)
	if err != nil {
		t.Fatal(err)
	}

	currencies = []string{"USD_AUD", "USD_EUR", "USD_GBP"}
	_, err = c.ConvertMany(currencies)
	if err == nil {
		t.Fatal("non error on supplying 3 or more currencies using the free API")
	}
}

func TestConvert(t *testing.T) {
	_, err := c.Convert("AUD", "USD")
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := c.GetCurrencies()
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetCountries(t *testing.T) {
	_, err := c.GetCountries()
	if err != nil {
		t.Fatal(err)
	}
}
