package currency

import (
	"testing"

	"github.com/stretchr/testify/require"
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

func TestNewTranslations(t *testing.T) {
	t.Parallel()
	translations := NewTranslations(map[Code]Code{
		XBT:  BTC,
		XETH: ETH,
		XDG:  DOGE,
		USDM: USD,
	})
	require.NotNil(t, translations)

	if !translations.Translate(XBT).Equal(BTC) {
		t.Error("NewTranslations: translation failed")
	}

	if !translations.Translate(LTC).Equal(LTC) {
		t.Error("NewTranslations: translation failed")
	}
}

func TestFindMatchingPairsBetween(t *testing.T) {
	t.Parallel()
	ltcusd := NewPair(LTC, USD)

	spotPairs := Pairs{
		NewPair(BTC, USD),
		NewPair(ETH, USD),
		NewPair(ETH, BTC),
		ltcusd,
	}

	futuresPairs := Pairs{
		NewPair(XBT, USDM),
		NewPair(XETH, USDM),
		NewPair(XETH, BTCM),
		ltcusd,             // exact match
		NewPair(XRP, USDM), // no match
	}

	matchingPairs := FindMatchingPairsBetween(PairsWithTranslation{spotPairs, nil}, PairsWithTranslation{futuresPairs, nil})
	require.Len(t, matchingPairs, 1)

	if !matchingPairs[ltcusd].Equal(ltcusd) {
		t.Error("FindMatchingPairsBetween: matching pair not found")
	}

	translations := NewTranslations(map[Code]Code{
		XBT:  BTC,
		XETH: ETH,
		XDG:  DOGE,
		USDM: USD,
		BTCM: BTC,
	})

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, nil}, PairsWithTranslation{futuresPairs, translations})
	require.Len(t, matchingPairs, 4)

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, translations}, PairsWithTranslation{futuresPairs, translations})
	require.Len(t, matchingPairs, 4)

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, translations}, PairsWithTranslation{futuresPairs, nil})
	require.Len(t, matchingPairs, 1)
}

func BenchmarkFindMatchingPairsBetween(b *testing.B) {
	ltcusd := NewPair(LTC, USD)

	spotPairs := Pairs{
		NewPair(BTC, USD),
		NewPair(ETH, USD),
		NewPair(ETH, BTC),
		ltcusd,
	}

	futuresPairs := Pairs{
		NewPair(XBT, USDM),
		NewPair(XETH, USDM),
		NewPair(XETH, BTCM),
		ltcusd,             // exact match
		NewPair(XRP, USDM), // no match
	}

	translations := NewTranslations(map[Code]Code{
		XBT:  BTC,
		XETH: ETH,
		XDG:  DOGE,
		USDM: USD,
		BTCM: BTC,
	})

	for i := 0; i < b.N; i++ {
		_ = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, translations}, PairsWithTranslation{futuresPairs, translations})
	}
}
