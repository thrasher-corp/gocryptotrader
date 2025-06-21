//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

import (
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

	b = new(Binance)
	if err := testexch.Setup(b); err != nil {
		log.Fatal(err)
	}

	if err := testexch.MockHTTPInstance(b); err != nil {
		log.Fatal(err)
	}

<<<<<<< HEAD
	b.setupOrderbookManager()
	if err := b.populateTradablePairs(); err != nil {
=======
	ctx := context.Background()
	b.setupOrderbookManager(ctx)
	if err := b.UpdateTradablePairs(ctx, true); err != nil {
>>>>>>> f21a18fa67af04e4858903251e3caa0725402a02
		log.Fatal(err)
	}
	if mockTests {
		optionsTradablePair = currency.Pair{Base: currency.NewCode("ETH"), Quote: currency.NewCode("240927-3800-P"), Delimiter: currency.DashDelimiter}
		usdtmTradablePair = currency.NewPair(currency.NewCode("BTC"), currency.NewCode("USDT"))
	}
	assetToTradablePairMap = map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.Options:             optionsTradablePair,
		asset.USDTMarginedFutures: usdtmTradablePair,
		asset.CoinMarginedFutures: coinmTradablePair,
		asset.Margin:              spotTradablePair,
	}
	setupWs()
	os.Exit(m.Run())
}
