//go:build mock_test_off

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
	spotTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	marginTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	usdtmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	coinmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USD_PERP"))
	optionsTradablePair = currency.Pair{Base: currency.NewCode("BTC"), Quote: currency.NewCode("260327-100000-C"), Delimiter: currency.DashDelimiter}

	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
		asset.Margin:              spotTradablePair,
	}

	for assetType, pair := range assetToTradablePairMap {
		if err := e.CurrencyPairs.StorePairs(assetType, []currency.Pair{pair}, false); err != nil {
			log.Fatal(err)
		}
		if err := e.CurrencyPairs.StorePairs(assetType, []currency.Pair{pair}, true); err != nil {
			log.Fatal(err)
		}
	}
	os.Exit(m.Run())
}
