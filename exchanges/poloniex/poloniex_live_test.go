//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

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
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		log.Fatal("Test Failed - Poloniex Setup() init error", err)
	}
	poloniexConfig.AuthenticatedAPISupport = true
	poloniexConfig.APIKey = apiKey
	poloniexConfig.APISecret = apiSecret
	p.SetDefaults()
	p.Setup(&poloniexConfig)
	log.Printf(sharedtestvalues.LiveTesting, p.GetName(), p.APIUrl)
	os.Exit(m.Run())
}
