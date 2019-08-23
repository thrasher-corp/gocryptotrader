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
	cfg.LoadConfig("../../testdata/configtest.json")
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Fatal("Test Failed - Binance Setup() init error", err)
	}
	binanceConfig.AuthenticatedAPISupport = true
	binanceConfig.APIKey = apiKey
	binanceConfig.APISecret = apiSecret
	b.SetDefaults()
	b.Setup(&binanceConfig)

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Test Failed - Mock server error %s", err)
	}

	b.HTTPClient = newClient
	b.APIUrl = serverDetails

	log.Printf(sharedtestvalues.MockTesting, b.GetName(), b.APIUrl)
	os.Exit(m.Run())
}
