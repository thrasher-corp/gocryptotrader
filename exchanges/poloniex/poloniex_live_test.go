//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Poloniex load config error", err)
	}
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		log.Fatal("Poloniex Setup() init error", err)
	}
	poloniexConfig.API.AuthenticatedSupport = true
	poloniexConfig.API.Credentials.Key = apiKey
	poloniexConfig.API.Credentials.Secret = apiSecret
	p.SetDefaults()
	p.Websocket = sharedtestvalues.NewTestWebsocket()
	err = p.Setup(poloniexConfig)
	if err != nil {
		log.Fatal("Poloniex setup error", err)
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
	p.HTTPRecording = true
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
