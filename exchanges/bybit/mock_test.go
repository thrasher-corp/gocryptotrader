//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package bybit

import (
	"log"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	// The fixtures record these pairs. Tests read them in parallel, so they are set once, before any test runs
	spotTradablePair = currency.NewBTCUSDT()
	usdtMarginedTradablePair = currency.NewPair(currency.NewCode("10000LADYS"), currency.USDT)
	usdcMarginedTradablePair = currency.NewPair(currency.ETH, currency.PERP)
	inverseTradablePair = currency.NewPair(currency.ADA, currency.USD)
	optionsTradablePair = currency.NewPairWithDelimiter(currency.BTC.String(), "26NOV24-92000-C", currency.DashDelimiter)
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatalf("Bybit mock setup error: %s", err)
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
	for a, p := range map[asset.Item]currency.Pair{
		asset.Spot:                spotTradablePair,
		asset.USDTMarginedFutures: usdtMarginedTradablePair,
		asset.USDCMarginedFutures: usdcMarginedTradablePair,
		asset.CoinMarginedFutures: inverseTradablePair,
		asset.Options:             optionsTradablePair,
	} {
		if enabled, err := ex.CurrencyPairs.IsPairEnabled(p, a); err == nil && enabled {
			continue
		}
		available, err := ex.GetAvailablePairs(a)
		if err != nil {
			return err
		}
		if !available.Contains(p, true) {
			if err := ex.CurrencyPairs.StorePairs(a, append(available, p), false); err != nil {
				return err
			}
		}
		if err := ex.CurrencyPairs.EnablePair(a, p); err != nil {
			return err
		}
	}
	return nil
}

// testInstance returns a new exchange served from the recorded responses, for tests that need their own instance
func testInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, setupMockExchange(ex), "setupMockExchange must not error")
	return ex
}

// ensureTradablePairs is a no-op for mock tests, which use the fixed pairs their fixtures record
func ensureTradablePairs(testing.TB) {}
