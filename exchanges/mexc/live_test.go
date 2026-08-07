//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package mexc

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(&accounts.Credentials{Key: apiKey, Secret: apiSecret})
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if err := populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	if err := e.setEnabledPairs(spotTradablePair); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

// populateTradablePairs pins the pair the live tests run against. It used to take whichever pair
// happened to be first in the enabled set, which varies between runs: the candle tests ask for a
// fixed historic window, and an illiquid pair has no candles in it, so they failed at random.
func populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	if len(tradablePairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	pair := tradablePairs[0]
	if btc := currency.NewBTCUSDT(); tradablePairs.Contains(btc, true) {
		pair = btc
	}
	spotTradablePair, err = e.FormatExchangeCurrency(pair, asset.Spot)
	return err
}
