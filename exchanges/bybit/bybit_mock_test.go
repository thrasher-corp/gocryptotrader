//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/bybit/bybit.json"

var mockTests = true

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	bybitConfig, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal("Bybit Setup() init error", err)
	}

	b.SkipAuthCheck = true
	bybitConfig.API.AuthenticatedSupport = true
	bybitConfig.API.Credentials.Key = apiKey
	bybitConfig.API.Credentials.Secret = apiSecret
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	request.MaxRequestJobs = 100
	err = b.Setup(bybitConfig)
	if err != nil {
		log.Fatal("Bybit setup error", err)
	}

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}
	err = b.SetHTTPClient(newClient)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}
	endpointMap := b.API.Endpoints.GetURLMap()
	for k := range endpointMap {
		err = b.API.Endpoints.SetRunning(k, serverDetails)
		if err != nil {
			log.Fatal(err)
		}
	}
	request.MaxRequestJobs = 100
	err = b.UpdateTradablePairs(context.Background(), true)
	if err != nil {
		log.Fatal("Bybit setup error", err)
	}

	spotTradablePair, err = b.ExtractCurrencyPair("BTCUSDT", asset.Spot, true)
	if err != nil {
		log.Fatal(err)
	}
	usdtMarginedTradablePair, err = b.ExtractCurrencyPair("10000LADYSUSDT", asset.USDTMarginedFutures, true)
	if err != nil {
		log.Fatal(err)
	}
	usdcMarginedTradablePair, err = b.ExtractCurrencyPair("ETHPERP", asset.USDCMarginedFutures, true)
	if err != nil {
		log.Fatal(err)
	}
	inverseTradablePair, err = b.ExtractCurrencyPair("ADAUSD", asset.CoinMarginedFutures, true)
	if err != nil {
		log.Fatal(err)
	}
	optionsTradablePair, err = b.ExtractCurrencyPair("BTC-29DEC23-80000-C", asset.Options, true)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
