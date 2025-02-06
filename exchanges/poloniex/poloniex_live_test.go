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
	p = new(Poloniex)
	if err := testexch.Setup(p); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		p.API.AuthenticatedSupport = true
		p.API.AuthenticatedWebsocketSupport = true
		p.SetCredentials(apiKey, apiSecret, "", "", "", "")
		p.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	p.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	p.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	err = p.Websocket.Enable()
	if err != nil {
		log.Fatal(err)
	}
	err = populateTradablePairs()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func populateTradablePairs() error {
	err := p.UpdateTradablePairs(context.Background(), false)
	if err != nil {
		return err
	}
	tradablePairs, err := p.GetEnabledPairs(asset.Spot)
	if err != nil {
		return err
	} else if len(tradablePairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	spotTradablePair = tradablePairs[0]
	tradablePairs, err = p.GetEnabledPairs(asset.Futures)
	if err != nil {
		return err
	} else if len(tradablePairs) == 0 {
		return currency.ErrCurrencyPairsEmpty
	}
	futuresTradablePair = tradablePairs[0]
	return nil
}
