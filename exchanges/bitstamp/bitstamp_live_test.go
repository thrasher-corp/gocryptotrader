//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitstamp

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var b Bitstamp

var isSetup bool
var mockTests = false
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
		if err != nil {
			t.Error("Test Failed - Poloniex Setup() init error")
		}
		bitstampConfig.AuthenticatedAPISupport = true
		bitstampConfig.APIKey = apiKey
		bitstampConfig.APISecret = apiSecret
		bitstampConfig.ClientID = customerID
		b.SetDefaults()
		b.Setup(&bitstampConfig)
		log.Printf("Live testing framework in use for %s @ %s",
			b.GetName(),
			b.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
