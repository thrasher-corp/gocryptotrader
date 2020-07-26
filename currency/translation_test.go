package currency

import "testing"

func TestGetTranslation(t *testing.T) {
	currencyPair := NewPair(BTC, USD)
	expected := XBT
	actual := GetTranslation(currencyPair.Base)
	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	currencyPair.Base = NEO
	actual = GetTranslation(currencyPair.Base)
	if actual != currencyPair.Base {
		t.Error("GetTranslation: no error on non translatable currency")
	}

	expected = BTC
	currencyPair.Base = XBT

	actual = GetTranslation(currencyPair.Base)
	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}
}
