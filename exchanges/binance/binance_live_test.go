//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Test Failed - Binance load config error", err)
	}
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Fatal("Test Failed - Binance Setup() init error", err)
	}
	binanceConfig.API.AuthenticatedSupport = true
	binanceConfig.API.Credentials.Key = apiKey
	binanceConfig.API.Credentials.Secret = apiSecret
	b.SetDefaults()
	err = b.Setup(binanceConfig)
	if err != nil {
		log.Fatal("Test Failed - Binance setup error", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, b.GetName(), b.API.Endpoints.URL)
	os.Exit(m.Run())
}
