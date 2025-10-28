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

	e.setAPICredential(apiKey, apiSecret)

	e.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	e.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	if err := e.populateTradablePairs(); err != nil {
		log.Fatal(err)
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
	if err = e.setEnabledPairs(spotTradablePair, futuresTradablePair); err != nil {
		log.Fatal(err)
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

func (e *Exchange) populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
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
