package currency

import "testing"

func TestGetTranslation(t *testing.T) {
	currencyPair := NewPair(BTC, USD)
	expected := XBT
	actual, ok := GetTranslation(currencyPair.Base)
	if !ok {
		t.Error("GetTranslation: failed to retrieve translation for BTC")
	}

	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	currencyPair.Base = NEO
	_, ok = GetTranslation(currencyPair.Base)
	if ok {
		t.Error("GetTranslation: no error on non translatable currency")
	}

	expected = BTC
	currencyPair.Base = XBT

	actual, ok = GetTranslation(currencyPair.Base)
	if !ok {
		t.Error("GetTranslation: failed to retrieve translation for BTC")
	}

	if expected != actual {
		t.Error("GetTranslation: translation result was different to expected result")
	}
}
