//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/mock"
)

var p Poloniex

var isSetup bool
var mockTests = true
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		mock.NewVCRServer("../../testdata/http_mock/poloniex/poloniex.json", t)
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
		p.APIUrl = "http://localhost:3000"
		log.Printf("Mock testing framework in use for %s @ %s",
			p.GetName(),
			p.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
