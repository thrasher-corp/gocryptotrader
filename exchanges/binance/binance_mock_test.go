//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

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
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		log.Error("Test Failed - Binance Setup() init error", err)
		os.Exit(1)
	}
	binanceConfig.AuthenticatedAPISupport = true
	binanceConfig.APIKey = apiKey
	binanceConfig.APISecret = apiSecret
	b.SetDefaults()
	b.Setup(&binanceConfig)

	serverDetails, err := mock.NewVCRServer("../../testdata/http_mock/binance/binance.json")
	if err != nil {
		log.Warn("Test Failed - mock server error", err)
	} else {
		b.APIUrl = serverDetails
	}

	log.Debugf("Mock testing framework in use for %s @ %s",
		b.GetName(),
		b.APIUrl)
	os.Exit(m.Run())
}
