//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

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
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

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
	err = e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, false)
	if err != nil {
		log.Fatal(err)
	}
	err = e.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, true)
	if err != nil {
		log.Fatal(err)
	}
	err = e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, false)
	if err != nil {
		log.Fatal(err)
	}
	err = e.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, true)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
