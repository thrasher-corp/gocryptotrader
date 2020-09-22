//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/binance/binance.json"

var mockTests = true

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
	b.SkipAuthCheck = true
	binanceConfig.API.AuthenticatedSupport = true
	binanceConfig.API.Credentials.Key = apiKey
	binanceConfig.API.Credentials.Secret = apiSecret
	b.SetDefaults()
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(binanceConfig)
	if err != nil {
		log.Fatal("Binance setup error", err)
	}

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}

	b.HTTPClient = newClient
	b.API.Endpoints.URL = serverDetails
	log.Printf(sharedtestvalues.MockTesting, b.Name, b.API.Endpoints.URL)
	os.Exit(m.Run())
}
