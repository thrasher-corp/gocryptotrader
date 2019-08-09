//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Error("Test Failed - Binance Setup() init error", err)
		os.Exit(1)
	}
	binanceConfig.AuthenticatedAPISupport = true
	binanceConfig.APIKey = apiKey
	binanceConfig.APISecret = apiSecret
	b.SetDefaults()
	b.Setup(&binanceConfig)
	log.Debugf("Live testing framework in use for %s @ %s",
		b.GetName(),
		b.APIUrl)
	os.Exit(m.Run())
}
