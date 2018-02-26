package translation

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/pair"
)

func TestGetTranslation(t *testing.T) {
	currencyPair := pair.NewCurrencyPair("BTC", "USD")
	expected := pair.CurrencyItem("XBT")
	actual, err := GetTranslation(currencyPair.FirstCurrency)
	if err != nil {
		t.Error("GetTranslation: failed to retrieve translation for BTC")
	}

	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	currencyPair.FirstCurrency = "NEO"
	_, err = GetTranslation(currencyPair.FirstCurrency)
	if err == nil {
		t.Error("GetTranslation: no error on non translatable currency")
	}
}

func TestHasTranslation(t *testing.T) {
	currencyPair := pair.NewCurrencyPair("BTC", "USD")
	expected := true
	actual := HasTranslation(currencyPair.FirstCurrency)
	if expected != actual {
		t.Error("HasTranslation: translation result was different to expected result")
	}

	currencyPair.FirstCurrency = "NEO"
	expected = false
	actual = HasTranslation(currencyPair.FirstCurrency)
	if expected != actual {
		t.Error("HasTranslation: translation result was different to expected result")
	}
}
