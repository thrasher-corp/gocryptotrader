//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bybit

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	ex = new(Exchange)
	if err := testexch.Setup(ex); err != nil {
		log.Fatalf("Bybit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(apiKey, apiSecret, "", "", "", "")
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	if ex.API.AuthenticatedSupport {
		if _, err := ex.FetchAccountType(context.Background()); err != nil {
			log.Printf("%s unable to FetchAccountType: %v", ex.Name, err)
		}
	}

	instantiateTradablePairs()

	os.Exit(m.Run())
}

func instantiateTradablePairs() {
	handleError := func(msg string, err error) {
		if err != nil {
			log.Fatalf("Bybit %s: %v", msg, err)
		}
	}

	err := ex.UpdateTradablePairs(context.Background(), true)
	handleError("unable to UpdateTradablePairs", err)

	setTradablePair := func(assetType asset.Item, p *currency.Pair) {
		tradables, err := ex.GetEnabledPairs(assetType)
		handleError("unable to GetEnabledPairs", err)

		format, err := ex.GetPairFormat(assetType, true)
		handleError("unable to GetPairFormat", err)

		*p = tradables[0].Format(format)
	}

	setTradablePair(asset.Spot, &spotTradablePair)
	setTradablePair(asset.USDTMarginedFutures, &usdtMarginedTradablePair)
	setTradablePair(asset.USDCMarginedFutures, &usdcMarginedTradablePair)
	setTradablePair(asset.CoinMarginedFutures, &inverseTradablePair)
	setTradablePair(asset.Options, &optionsTradablePair)
}
