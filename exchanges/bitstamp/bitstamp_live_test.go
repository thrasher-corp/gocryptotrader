//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitstamp

import (
	"os"

	"github.com/thrasher-corp/gocryptotrader/config"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = false

func TestMain(m *testing.m) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		log.Error("Test Failed - Poloniex Setup() init error", err)
		os.Exit(1)
	}
	bitstampConfig.AuthenticatedAPISupport = true
	bitstampConfig.APIKey = apiKey
	bitstampConfig.APISecret = apiSecret
	bitstampConfig.ClientID = customerID
	b.SetDefaults()
	b.Setup(&bitstampConfig)
	log.Debugf("Live testing framework in use for %s @ %s",
		b.GetName(),
		b.APIUrl)
	os.Exit(m.Run())
}
