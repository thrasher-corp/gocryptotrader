//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/mock"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

const mockfile = "../../testdata/http_mock/poloniex/poloniex.json"

var mockTests = true

func TestMain(m *testing.M) {
	p = new(Poloniex)
	if err := testexch.Setup(p); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		p.API.AuthenticatedSupport = true
		p.API.AuthenticatedWebsocketSupport = true
		p.SetCredentials(apiKey, apiSecret, "", "", "", "")
		p.Websocket.SetCanUseAuthenticatedEndpoints(true)
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
	spotTradablePair, err = p.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT", "_"), asset.Spot)
	if err != nil {
		log.Fatal(err)
	}
	futuresTradablePair, err = p.FormatExchangeCurrency(currency.NewPairWithDelimiter("BTC", "USDT_PERP", ""), asset.Futures)
	if err != nil {
		log.Fatal(err)
	}
	err = p.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, false)
	if err != nil {
		log.Fatal(err)
	}
	err = p.CurrencyPairs.StorePairs(asset.Spot, []currency.Pair{spotTradablePair}, true)
	if err != nil {
		log.Fatal(err)
	}
	err = p.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, false)
	if err != nil {
		log.Fatal(err)
	}
	err = p.CurrencyPairs.StorePairs(asset.Futures, []currency.Pair{futuresTradablePair}, true)
	if err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
