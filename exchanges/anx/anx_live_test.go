//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package anx

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
	anxConfig, err := cfg.GetExchangeConfig("ANX")
	if err != nil {
		log.Error("Test Failed - ANX Setup() init error", err)
		os.Exit(1)
	}
	anxConfig.AuthenticatedAPISupport = true
	anxConfig.APIKey = apiKey
	anxConfig.APISecret = apiSecret
	a.SetDefaults()
	a.Setup(&anxConfig)
	log.Debugf(sharedtestvalues.LiveTesting, a.GetName(), a.APIUrl)
	os.Exit(m.Run())
}
