//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitstamp

import (
	"log"
	"sync"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/exchanges/mock"
)

var b Bitstamp

var isSetup bool
var mockTests = true
var mtx sync.Mutex

func TestSetup(t *testing.T) {
	t.Parallel()

	mtx.Lock()
	if !isSetup {
		serverDetails, err := mock.NewVCRServer("../../testdata/http_mock/bitstamp/bitstamp.json", t)
		if err != nil {
			log.Fatal("Test Failed - Mock server error", err)
		}
		cfg := config.GetConfig()
		cfg.LoadConfig("../../testdata/configtest.json")
		bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
		if err != nil {
			t.Error("Test Failed - Bitstamp Setup() init error")
		}
		bitstampConfig.AuthenticatedAPISupport = true
		bitstampConfig.APIKey = apiKey
		bitstampConfig.APISecret = apiSecret
		bitstampConfig.ClientID = customerID
		b.SetDefaults()
		b.Setup(&bitstampConfig)
		b.APIUrl = serverDetails + "/api"
		log.Printf("Mock testing framework in use for %s @ %s",
			b.GetName(),
			b.APIUrl)
		isSetup = true
	}
	mtx.Unlock()
}
