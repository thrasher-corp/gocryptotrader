//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitstamp

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
	bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		log.Fatal("Test Failed - Bitstamp Setup() init error", err)
	}
	bitstampConfig.AuthenticatedAPISupport = true
	bitstampConfig.APIKey = apiKey
	bitstampConfig.APISecret = apiSecret
	bitstampConfig.ClientID = customerID
	b.SetDefaults()
	b.Setup(&bitstampConfig)
	log.Printf(sharedtestvalues.LiveTesting, b.GetName(), b.APIUrl)
	os.Exit(m.Run())
}
