package currency

import (
	"testing"
)

func TestGetTranslation(t *testing.T) {
	currencyPair := NewPair(BTC, USD)
	expected := XBT
	actual := GetTranslation(currencyPair.Base)
	if !expected.Equal(actual) {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	currencyPair.Base = NEO
	actual = GetTranslation(currencyPair.Base)
	if !actual.Equal(currencyPair.Base) {
		t.Error("GetTranslation: no error on non translatable currency")
	}

	expected = BTC
	currencyPair.Base = XBT

	actual = GetTranslation(currencyPair.Base)
	if !expected.Equal(actual) {
		t.Error("GetTranslation: translation result was different to expected result")
	}

	// This test accentuates the issue of comparing code types as this will
	// not match for lower and upper differences and a key (*Item) needs to be
	// used.
	// Code{Item: 0xc000094140, Upper: true} != Code{Item: 0xc000094140, Upper: false}
	if actual = GetTranslation(NewCode("btc")); !XBT.Equal(actual) {
		t.Errorf("received: '%v', but expected: '%v'", actual, XBT)
	}
}
