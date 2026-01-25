//go:build !mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gateio

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
	}
	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("Poloniex MockHTTPInstance error: %s", err)
	}
	e.HTTPRecording = true
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
