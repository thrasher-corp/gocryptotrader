package currency

import "testing"

func TestGetTranslation(t *testing.T) {
	currencyPair := NewCurrencyPair("BTC", "USD")
	expected := Code("XBT")
	actual, err := GetTranslation(currencyPair.Base)
	if err != nil {
		t.Error("GetTranslation: failed to retrieve translation for BTC")
	}

	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	currencyPair.Base = "NEO"
	_, err = GetTranslation(currencyPair.Base)
	if err == nil {
		t.Error("GetTranslation: no error on non translatable currency")
	}

	expected = "BTC"
	currencyPair.Base = "XBT"

	actual, err = GetTranslation(currencyPair.Base)
	if err != nil {
		t.Error("GetTranslation: failed to retrieve translation for BTC")
	}

	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}
}

func TestHasTranslation(t *testing.T) {
	currencyPair := NewCurrencyPair("BTC", "USD")
	expected := true
	actual := HasTranslation(currencyPair.Base)
	if expected != actual {
		t.Error("HasTranslation: translation result was different to expected result")
	}

	currencyPair.Base = "XBT"
	expected = true
	actual = HasTranslation(currencyPair.Base)
	if expected != actual {
		t.Error("HasTranslation: translation result was different to expected result")
	}

	currencyPair.Base = "NEO"
	expected = false
	actual = HasTranslation(currencyPair.Base)
	if expected != actual {
		t.Error("HasTranslation: translation result was different to expected result")
	}
}
