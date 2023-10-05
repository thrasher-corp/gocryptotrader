//go:build !mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = true

func TestMain(m *testing.M) {
	b.SetDefaults()
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Bybit")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.Enabled = true
	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret
	b.Websocket = sharedtestvalues.NewTestWebsocket()
	request.MaxRequestJobs = 100
	err = b.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	// Turn on all pairs for testing
	supportedAssets := b.GetAssetTypes(false)
	for x := range supportedAssets {
		var avail currency.Pairs
		avail, err = b.GetAvailablePairs(supportedAssets[x])
		if err != nil {
			log.Fatal(err)
		}

		err = b.CurrencyPairs.StorePairs(supportedAssets[x], avail, true)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = instantiateTradablePairs()
	if err != nil {
		log.Fatalf("%s %v", b.Name, err)
	}
	b.checkAccountType(context.Background())
	os.Exit(m.Run())
}
