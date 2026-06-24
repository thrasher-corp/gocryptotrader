//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gateio

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
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Gateio Setup error: %s", err)
	}

	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	if err := e.populateTradablePairs(); err != nil {
		log.Fatal(err)
	}
	e.HTTPRecording = true
	os.Exit(m.Run())
}

func (e *Exchange) populateTradablePairs() error {
	if err := e.UpdateTradablePairs(context.Background()); err != nil {
		return err
	}
	enabledAssetPair = make(map[asset.Item]currency.Pair, 7)
	for _, a := range e.GetAssetTypes(true) {
		tradablePairs, err := e.GetEnabledPairs(asset.Spot)
		if err != nil {
			return err
		} else if len(tradablePairs) == 0 {
			return currency.ErrCurrencyPairsEmpty
		}
		enabledAssetPair[a], err = e.FormatExchangeCurrency(tradablePairs[0], a)
		if err != nil {
			return err
		}
	}
	return nil
}
