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
	log.Printf(sharedtestvalues.LiveTesting, b.GetName(), b.APIUrl)
	os.Exit(m.Run())
}
