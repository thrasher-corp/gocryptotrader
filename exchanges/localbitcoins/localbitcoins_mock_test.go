//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package localbitcoins

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = true

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
	if err != nil {
		log.Error("Test Failed - Localbitcoins Setup() init error", err)
		os.Exit(1)
	}
	localbitcoinsConfig.AuthenticatedAPISupport = true
	localbitcoinsConfig.APIKey = apiKey
	localbitcoinsConfig.APISecret = apiSecret
	l.SetDefaults()
	l.Setup(&localbitcoinsConfig)

	serverDetails, err := mock.NewVCRServer("../../testdata/http_mock/localbitcoins/localbitcoins.json")
	if err != nil {
		log.Warn("Test Failed - Mock server error", err)
	} else {
		l.APIUrl = serverDetails
	}

	log.Printf(sharedtestvalues.MockTesting, l.GetName(), l.APIUrl)
	os.Exit(m.Run())
}
