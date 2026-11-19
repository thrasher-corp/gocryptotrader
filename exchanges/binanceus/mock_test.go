//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binanceus

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatalf("Binanceus mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from the recorded responses in testdata/http.json, signing with placeholder
// credentials so authenticated endpoints run against their recorded responses too
func setupMockExchange(ex *Exchange) error {
	if err := testexch.Setup(ex); err != nil {
		return err
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	if err := testexch.MockHTTPInstance(ex); err != nil {
		return err
	}
	// Mock responses are served locally, so throttling protects nothing
	if err := ex.DisableRateLimiter(); err != nil {
		return err
	}
	spotTradablePair = currency.NewBTCUSDT()
	if err := ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotTradablePair}, false); err != nil {
		return err
	}
	return ex.CurrencyPairs.StorePairs(asset.Spot, currency.Pairs{spotTradablePair}, true)
}

// ensureTradablePairs is a no-op for mock tests, which use the fixed pair their fixtures record
func ensureTradablePairs(testing.TB) {}
