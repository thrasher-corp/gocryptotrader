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
	if err := e.enablePairs(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func (e *Exchange) enablePairs() error {
	var err error
	enabledAssetPair = make(map[asset.Item]currency.Pair, 7)
	enabledAssetPair[asset.Spot], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.Margin], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.CrossMargin], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.USDTMarginedFutures], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.CoinMarginedFutures], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.DeliveryFutures], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT_20260213", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	enabledAssetPair[asset.Options], err = e.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT-20221028-26000-C", "_"), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	// store the pairs into the enabled pairs
	for a, p := range enabledAssetPair {
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, false); err != nil {
			return err
		}
		if err := e.CurrencyPairs.StorePairs(a, []currency.Pair{p}, true); err != nil {
			return err
		}
	}
	return nil
}
