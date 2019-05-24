//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package localbitcoins

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/mock"
)

var l LocalBitcoins

var isSetup bool
var mockTests = true
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		mock.NewVCRServer("../../testdata/http_mock/localbitcoins/localbitcoins.json", t)
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
		if err != nil {
			t.Error("Test Failed - Localbitcoins Setup() init error")
		}
		localbitcoinsConfig.AuthenticatedAPISupport = true
		localbitcoinsConfig.APIKey = apiKey
		localbitcoinsConfig.APISecret = apiSecret
		l.SetDefaults()
		l.Setup(&localbitcoinsConfig)
		l.APIUrl = "http://localhost:3000"
		log.Printf("Mock testing framework in use for %s @ %s",
			l.GetName(),
			l.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
