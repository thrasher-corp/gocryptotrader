//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	e = testInstance()

	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		log.Fatalf("Bybit unable to UpdateTradablePairs: %s", err)
	}

	setEnabledPair := func(assetType asset.Item, pair currency.Pair) {
		okay, err := e.IsPairEnabled(pair, assetType)
		if !okay || err != nil {
			err = e.CurrencyPairs.EnablePair(assetType, pair)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	spotTradablePair = currency.Pair{Base: currency.BTC, Quote: currency.USDT}
	usdtMarginedTradablePair = currency.Pair{Base: currency.NewCode("10000LADYS"), Quote: currency.USDT}
	usdcMarginedTradablePair = currency.Pair{Base: currency.ETH, Quote: currency.PERP}
	inverseTradablePair = currency.Pair{Base: currency.ADA, Quote: currency.USD}
	optionsTradablePair = currency.Pair{Base: currency.BTC, Delimiter: currency.DashDelimiter, Quote: currency.NewCode("26NOV24-92000-C")}

	setEnabledPair(asset.Spot, spotTradablePair)
	setEnabledPair(asset.USDTMarginedFutures, usdtMarginedTradablePair)
	setEnabledPair(asset.USDCMarginedFutures, usdcMarginedTradablePair)
	setEnabledPair(asset.CoinMarginedFutures, inverseTradablePair)
	setEnabledPair(asset.Options, optionsTradablePair)

	os.Exit(m.Run())
}

func testInstance() *Exchange {
	b := new(Exchange)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Bybit Setup error: %s", err)
	}

	b.SetCredentials("mock", "tester", "", "", "", "") // Hack for UpdateAccountBalances test

	if err := testexch.MockHTTPInstance(b); err != nil {
		log.Fatalf("Bybit MockHTTPInstance error: %s", err)
	}

	return b
}
