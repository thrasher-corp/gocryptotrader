//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

const mockfile = "../../testdata/http_mock/poloniex/poloniex.json"

var mockTests = true

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		log.Fatal(err)
	}
	p.SkipAuthCheck = true
	poloniexConfig.API.AuthenticatedSupport = true
	poloniexConfig.API.Credentials.Key = apiKey
	poloniexConfig.API.Credentials.Secret = apiSecret

	p.Verbose = true
	p.SetDefaults()
	p.Websocket = sharedtestvalues.NewTestWebsocket()
	err = p.Setup(poloniexConfig)
	if err != nil {
		log.Fatal(err)
	}

	serverDetails, newClient, err := mock.NewVCRServer(mockfile)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}

	err = p.SetHTTPClient(newClient)
	if err != nil {
		log.Fatalf("Mock server error %s", err)
	}
	endpoints := p.API.Endpoints.GetURLMap()
	for k := range endpoints {
		err = p.API.Endpoints.SetRunning(k, serverDetails)
		if err != nil {
			log.Fatal(err)
		}
	}
	spotTradablePair = currency.NewPairWithDelimiter("ETH", "USDT", "_")
	futuresTradablePair = currency.NewPairWithDelimiter("BTC", "USDTPERP", "")
	err = p.Websocket.Enable()
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
