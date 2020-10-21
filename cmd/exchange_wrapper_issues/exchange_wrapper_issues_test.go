package main

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestGetStrange(t *testing.T) {
	if getStrange("BTC").String() != "bTc" {
		t.Fatal("did not return correct strange code")
	}

	if getStrange("bTc").String() != "bTc" {
		t.Fatal("did not return correct strange code")
	}

	if getStrange("BtC").String() != "bTc" {
		t.Fatal("did not return correct strange code")
	}
}

func TestDisruptFormatting(t *testing.T) {
	_, err := distruptFormatting(currency.Pair{})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	_, err = distruptFormatting(currency.Pair{Base: currency.BTC})
	if err == nil {
		t.Fatal("error cannot be nil")
	}

	p := currency.NewPair(currency.BTC, currency.USDT)

	badPair, err := distruptFormatting(p)
	if err != nil {
		t.Fatal(err)
	}

	if badPair.String() != "bTc////&&&***uSdT" {
		t.Fatal("incorrect disrupted pair")
	}
}
