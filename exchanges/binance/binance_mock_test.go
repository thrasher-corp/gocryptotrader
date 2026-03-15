//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

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
	if useTestNet {
		log.Fatal("cannot use testnet with mock tests")
	}

	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Binance Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("Binance MockHTTPInstance error: %s", err)
	}
	ctx := context.Background()
	e.setupOrderbookManager(ctx)
	if err := e.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	spotTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	marginTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	usdtmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	coinmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USD_PERP"))
	optionsTradablePair = currency.Pair{Base: currency.NewCode("BTC"), Quote: currency.NewCode("260327-100000-C"), Delimiter: currency.DashDelimiter}

	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, false); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, true); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.Margin, []currency.Pair{marginTradablePair}, false); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.Margin, []currency.Pair{marginTradablePair}, true); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, []currency.Pair{usdtmTradablePair}, false); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.USDTMarginedFutures, []currency.Pair{usdtmTradablePair}, true); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, []currency.Pair{coinmTradablePair}, false); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.CoinMarginedFutures, []currency.Pair{coinmTradablePair}, true); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.Options, []currency.Pair{optionsTradablePair}, false); err != nil {
		log.Fatal(err)
	}
	if err := e.CurrencyPairs.StorePairs(asset.Options, []currency.Pair{optionsTradablePair}, true); err != nil {
		log.Fatal(err)
	}

	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
		asset.Margin:              spotTradablePair,
	}
	os.Exit(m.Run())
}
