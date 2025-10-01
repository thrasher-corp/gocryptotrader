//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
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

	e.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	e.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	err := populateTradablePairs()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	err := e.UpdateTradablePairs(context.Background())
	if err != nil {
		return err
	}
	tradablePairs, err := e.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	} else if len(tradablePairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	spotTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Spot)
	if err != nil {
		return err
	}
	tradablePairs, err = e.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	} else if len(tradablePairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	futuresTradablePair, err = e.FormatExchangeCurrency(tradablePairs[0], asset.Futures)
	return err
}
