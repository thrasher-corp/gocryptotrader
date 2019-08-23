//+build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bitstamp

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/bitstamp/bitstamp.json"

var mockTests = true

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

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Test Failed - Mock server error %s", err)
	}

	b.HTTPClient = newClient
	b.APIUrl = serverDetails + "/api"

	log.Printf(sharedtestvalues.MockTesting, b.GetName(), b.APIUrl)
	os.Exit(m.Run())
}
