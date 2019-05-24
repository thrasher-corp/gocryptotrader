//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/mock"
)

var b Binance

var isSetup bool
var mockTests = true
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		mock.NewVCRServer("../../testdata/http_mock/binance/binance.json", t)
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
		b.APIUrl = "http://localhost:3000"
		log.Printf("Mock testing framework in use for %s @ %s",
			b.GetName(),
			b.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
