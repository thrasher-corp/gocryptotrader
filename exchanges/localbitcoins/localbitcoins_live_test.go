//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package localbitcoins

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	localbitcoinsConfig, err := cfg.GetExchangeConfig("LocalBitcoins")
	if err != nil {
		log.Fatal("Test Failed - LocalBitcoins Setup() init error", err)
	}
	localbitcoinsConfig.API.AuthenticatedSupport = true
	localbitcoinsConfig.API.Credentials.Key = apiKey
	localbitcoinsConfig.API.Credentials.Secret = apiSecret
	l.SetDefaults()
	l.Setup(localbitcoinsConfig)
	log.Printf(sharedtestvalues.LiveTesting, l.GetName(), l.API.Endpoints.URL)
	os.Exit(m.Run())
}
