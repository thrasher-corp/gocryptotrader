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
	b = new(Bybit)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Bybit Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		b.API.AuthenticatedSupport = true
		b.API.AuthenticatedWebsocketSupport = true
		b.SetCredentials(apiKey, apiSecret, "", "", "", "")
		b.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	if b.API.AuthenticatedSupport {
		if _, err := b.FetchAccountType(context.Background()); err != nil {
			log.Printf("%s unable to FetchAccountType: %v", b.Name, err)
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

	err := b.UpdateTradablePairs(context.Background(), true)
	handleError("unable to UpdateTradablePairs", err)

	setTradablePair := func(assetType asset.Item, p *currency.Pair) {
		tradables, err := b.GetEnabledPairs(assetType)
		handleError("unable to GetEnabledPairs", err)

		format, err := b.GetPairFormat(assetType, true)
		handleError("unable to GetPairFormat", err)

		*p = tradables[0].Format(format)
	}

	setTradablePair(asset.Spot, &spotTradablePair)
	setTradablePair(asset.USDTMarginedFutures, &usdtMarginedTradablePair)
	setTradablePair(asset.USDCMarginedFutures, &usdcMarginedTradablePair)
	setTradablePair(asset.CoinMarginedFutures, &inverseTradablePair)
	setTradablePair(asset.Options, &optionsTradablePair)
}
