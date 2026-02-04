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
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Bybit")
		os.Exit(0)
	}

	e = testInstance()

	if e.API.AuthenticatedSupport {
		if _, err := e.FetchAccountType(context.Background()); err != nil {
			log.Printf("%s unable to FetchAccountType: %v", e.Name, err)
		}
	}

	instantiateTradablePairs()

	os.Exit(m.Run())
}

func testInstance() *Exchange {
	e := new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bybit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	return e
}

func instantiateTradablePairs() {
	handleError := func(msg string, err error) {
		if err != nil {
			log.Fatalf("Bybit %s: %v", msg, err)
		}
	}

	err := e.UpdateTradablePairs(context.Background())
	handleError("unable to UpdateTradablePairs", err)

	setTradablePair := func(assetType asset.Item, p *currency.Pair) {
		tradables, err := e.GetEnabledPairs(assetType)
		handleError("unable to GetEnabledPairs", err)

		format, err := e.GetPairFormat(assetType, true)
		handleError("unable to GetPairFormat", err)

		*p = tradables[0].Format(format)
	}

	setTradablePair(asset.Spot, &spotTradablePair)
	setTradablePair(asset.USDTMarginedFutures, &usdtMarginedTradablePair)
	setTradablePair(asset.USDCMarginedFutures, &usdcMarginedTradablePair)
	setTradablePair(asset.CoinMarginedFutures, &inverseTradablePair)
	setTradablePair(asset.Options, &optionsTradablePair)
}
