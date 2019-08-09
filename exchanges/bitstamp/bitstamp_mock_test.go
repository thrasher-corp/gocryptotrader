//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitstamp

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = true

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		log.Error("Test Failed - Bitstamp Setup() init error", err)
		os.Exit(1)
	}
	bitstampConfig.AuthenticatedAPISupport = true
	bitstampConfig.APIKey = apiKey
	bitstampConfig.APISecret = apiSecret
	bitstampConfig.ClientID = customerID
	b.SetDefaults()
	b.Setup(&bitstampConfig)

	serverDetails, err := mock.NewVCRServer("../../testdata/http_mock/bitstamp/bitstamp.json")
	if err != nil {
		log.Warn("Test Failed - Mock server error", err)
	} else {
		b.APIUrl = serverDetails + "/api"
	}

	log.Debugf("Mock testing framework in use for %s @ %s",
		b.GetName(),
		b.APIUrl)
	os.Exit(m.Run())
}
