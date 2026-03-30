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
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	if err := populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	var err error
	spotTradablePair, err = currency.NewPairFromString("BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}
	futuresTradablePair, err = currency.NewPairFromString("BTC_USDT")
	if err != nil {
		log.Fatal(err)
	}
	if err := e.setEnabledPairs(spotTradablePair, futuresTradablePair); err != nil {
		log.Fatal(err)
	}
	e.HTTPRecording = true
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	}
	spotTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	}
	futuresTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Futures)
	return err
}
