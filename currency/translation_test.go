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
	translationsTest := NewTranslations(map[Code]Code{
		XBT:  BTC,
		XETH: ETH,
		XDG:  DOGE,
		USDM: USD,
	})
	require.NotNil(t, translations)

	if !translationsTest.Translate(XBT).Equal(BTC) {
		t.Error("NewTranslations: translation failed")
	}

	if !translationsTest.Translate(LTC).Equal(LTC) {
		t.Error("NewTranslations: translation failed")
	}
}

func TestFindMatchingPairsBetween(t *testing.T) {
	t.Parallel()
	ltcusd := NewPair(LTC, USD)

	spotPairs := Pairs{
		NewPair(BTC, USD).Format(PairFormat{Delimiter: "DELIMITER"}),
		NewPair(ETH, USD),
		NewPair(ETH, BTC).Format(PairFormat{Delimiter: "DELIMITER"}),
		ltcusd,
	}

	futuresPairs := Pairs{
		NewPair(XBT, USDM),
		NewPair(XETH, USDM).Format(PairFormat{Delimiter: "DELIMITER"}),
		NewPair(XETH, BTCM),
		ltcusd.Format(PairFormat{Delimiter: "DELIMITER"}), // exact match
		NewPair(XRP, USDM), // no match
	}

	matchingPairs := FindMatchingPairsBetween(PairsWithTranslation{spotPairs, nil}, PairsWithTranslation{futuresPairs, nil})
	require.Len(t, matchingPairs, 1)

	if !matchingPairs[ltcusd].Equal(ltcusd) {
		t.Error("FindMatchingPairsBetween: matching pair not found")
	}

	translationsTest := NewTranslations(map[Code]Code{
		XBT:  BTC,
		XETH: ETH,
		XDG:  DOGE,
		USDM: USD,
		BTCM: BTC,
	})

	expected := map[keyPair]Pair{
		NewPair(BTC, USD).KeyPair(): NewPair(XBT, USDM),
		NewPair(ETH, USD).KeyPair(): NewPair(XETH, USDM),
		NewPair(ETH, BTC).KeyPair(): NewPair(XETH, BTCM),
		ltcusd.KeyPair():            ltcusd,
	}

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, nil}, PairsWithTranslation{futuresPairs, translationsTest})
	require.Len(t, matchingPairs, 4)

	for k, v := range matchingPairs {
		if !expected[k.KeyPair()].Equal(v) {
			t.Error("FindMatchingPairsBetween: translation failed")
		}
	}

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, translationsTest}, PairsWithTranslation{futuresPairs, translationsTest})
	require.Len(t, matchingPairs, 4)

	for k, v := range matchingPairs {
		if !expected[k.KeyPair()].Equal(v) {
			t.Error("FindMatchingPairsBetween: translation failed")
		}
	}

	expected = map[keyPair]Pair{
		ltcusd.KeyPair(): ltcusd,
	}

	matchingPairs = FindMatchingPairsBetween(PairsWithTranslation{spotPairs, translationsTest}, PairsWithTranslation{futuresPairs, nil})
	require.Len(t, matchingPairs, 1)

	for k, v := range matchingPairs {
		if !expected[k.KeyPair()].Equal(v) {
			t.Error("FindMatchingPairsBetween: translation failed")
		}
	}
}

func (p Pair) KeyPair() keyPair {
	return keyPair{Base: p.Base.Item, Quote: p.Quote.Item}
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
