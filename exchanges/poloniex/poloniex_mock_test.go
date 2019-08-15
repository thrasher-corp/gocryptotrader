//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

const mockfile = "../../testdata/http_mock/poloniex/poloniex.json"

var mockTests = true

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		log.Error("Test Failed - Poloniex Setup() init error")
		os.Exit(1)
	}
	poloniexConfig.AuthenticatedAPISupport = true
	poloniexConfig.APIKey = apiKey
	poloniexConfig.APISecret = apiSecret
	p.SetDefaults()
	p.Setup(&poloniexConfig)

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Errorf("Test Failed - Mock server error %s", err)
		os.Exit(1)
	}

	p.HTTPClient = newClient
	p.APIUrl = serverDetails

	log.Debugf(sharedtestvalues.MockTesting, p.GetName(), p.APIUrl)
	os.Exit(m.Run())
}
