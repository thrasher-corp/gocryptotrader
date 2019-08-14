//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package localbitcoins

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
	localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
	if err != nil {
		log.Error("Test Failed - LocalBitcoins Setup() init error", err)
		os.Exit(1)
	}
	localbitcoinsConfig.AuthenticatedAPISupport = true
	localbitcoinsConfig.APIKey = apiKey
	localbitcoinsConfig.APISecret = apiSecret
	l.SetDefaults()
	l.Setup(&localbitcoinsConfig)
	log.Debugf(sharedtestvalues.LiveTesting, l.GetName(), l.APIUrl)
	os.Exit(m.Run())
}
