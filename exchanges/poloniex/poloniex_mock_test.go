//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockVarLock sync.Mutex
var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}
	e.setAPICredential(apiKey, apiSecret)
	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("Poloniex MockHTTPInstance error: %s", err)
	}
	var err error
	spotTradablePair, err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	if err != nil {
		log.Fatal(err)
	}
	futuresTradablePair, err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT_PERP", ""), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	if err := e.setEnabledPairs(spotTradablePair, futuresTradablePair); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func (e *Exchange) setEnabledPairs(spotTradablePair, futuresTradablePair currency.Pair) error {
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair, currency.NewPairWithDelimiter("BTC", "ETH", "_")}, false); err != nil {
		return err
	}
	if err := e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair, currency.NewPairWithDelimiter("BTC", "ETH", "_")}, true); err != nil {
		return err
	}
	if err := e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, false); err != nil {
		return err
	}
	return e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, true)
}
