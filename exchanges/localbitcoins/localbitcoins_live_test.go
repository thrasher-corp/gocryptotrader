//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package localbitcoins

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var l LocalBitcoins

var isSetup bool
var mockTests = false
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
		if err != nil {
			t.Error("Test Failed - LocalBitcoins Setup() init error")
		}
		localbitcoinsConfig.AuthenticatedAPISupport = true
		localbitcoinsConfig.APIKey = apiKey
		localbitcoinsConfig.APISecret = apiSecret
		l.SetDefaults()
		l.Setup(&localbitcoinsConfig)
		log.Printf("Live testing framework in use for %s @ %s",
			l.GetName(),
			l.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
