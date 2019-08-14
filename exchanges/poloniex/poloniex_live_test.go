//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		log.Error("Test Failed - Poloniex Setup() init error", err)
		os.Exit(1)
	}
	poloniexConfig.AuthenticatedAPISupport = true
	poloniexConfig.APIKey = apiKey
	poloniexConfig.APISecret = apiSecret
	p.SetDefaults()
	p.Setup(&poloniexConfig)
	log.Debugf(sharedtestvalues.LiveTesting, p.GetName(), p.APIUrl)
	os.Exit(m.Run())
}
