//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Binance load config error", err)
	}
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Fatal("Binance Setup() init error", err)
	}
	binanceConfig.API.AuthenticatedSupport = true
	binanceConfig.API.Credentials.Key = apiKey
	binanceConfig.API.Credentials.Secret = apiSecret
	b.SetDefaults()
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	if useTestNet {
		err = b.API.Endpoints.SetRunning(exchange.RestUSDTMargined.String(), testnetFutures)
		if err != nil {
			log.Fatal("Binance setup error", err)
		}
		err = b.API.Endpoints.SetRunning(exchange.RestCoinMargined.String(), testnetFutures)
		if err != nil {
			log.Fatal("Binance setup error", err)
		}
		err = b.API.Endpoints.SetRunning(exchange.RestSpot.String(), testnetSpotURL)
		if err != nil {
			log.Fatal("Binance setup error", err)
		}
	}
	err = b.Setup(binanceConfig)
	if err != nil {
		log.Fatal("Binance setup error", err)
	}
	b.setupOrderbookManager()
	request.MaxRequestJobs = 100
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	log.Printf(sharedtestvalues.LiveTesting, b.Name)
	err = b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		log.Fatal("Binance setup error", err)
	}
	os.Exit(m.Run())
}
