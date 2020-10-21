package main

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

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

	if badPair.String() != "BTC---TEST DELIMITER---usdt" {
		t.Fatal("incorrect disrupted pair")
	}
}
