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
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal("Bitstamp load config error", err)
	}
	bitstampConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		log.Fatal("Bitstamp Setup() init error", err)
	}
	b.SkipAuthCheck = true
	bitstampConfig.API.AuthenticatedSupport = true
	bitstampConfig.API.Credentials.Key = apiKey
	bitstampConfig.API.Credentials.Secret = apiSecret
	bitstampConfig.API.Credentials.ClientID = customerID
	b.SetDefaults()
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	err = b.Setup(bitstampConfig)
	if err != nil {
		log.Fatal("Bitstamp setup error", err)
	}

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}

	b.HTTPClient = newClient
	endpointMap := b.API.Endpoints.GetURLMap()
	for k := range endpointMap {
		err = b.API.Endpoints.SetRunning(k, serverDetails+"/api")
		if err != nil {
			log.Fatal(err)
		}
	}
	log.Printf(sharedtestvalues.MockTesting, b.Name)
	os.Exit(m.Run())
}
