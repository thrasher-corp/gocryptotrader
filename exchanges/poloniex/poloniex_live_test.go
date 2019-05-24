//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var p Poloniex

var isSetup bool
var mockTests = false
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
		if err != nil {
			t.Error("Test Failed - Poloniex Setup() init error")
		}
		poloniexConfig.AuthenticatedAPISupport = true
		poloniexConfig.APIKey = apiKey
		poloniexConfig.APISecret = apiSecret
		p.SetDefaults()
		p.Setup(&poloniexConfig)
		log.Printf("Live testing framework in use for %s @ %s",
			p.GetName(),
			p.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
