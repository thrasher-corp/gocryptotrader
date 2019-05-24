//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var b Binance

var isSetup bool
var mockTests = false
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		binanceConfig, err := cfg.GetExchangeConfig("Binance")
		if err != nil {
			t.Error("Test Failed - Binance Setup() init error")
		}
		binanceConfig.AuthenticatedAPISupport = true
		binanceConfig.APIKey = apiKey
		binanceConfig.APISecret = apiSecret
		b.SetDefaults()
		b.Setup(&binanceConfig)
		log.Printf("Live testing framework in use for %s @ %s",
			b.GetName(),
			b.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
