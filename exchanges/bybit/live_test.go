//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bybit

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

var tradablePairsOnce sync.Once

func TestMain(m *testing.M) {
	// Bybit refuses requests from the US, where CI runs
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Bybit")
		os.Exit(0)
	}
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bybit Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// testInstance returns a new live exchange with the test credentials, for tests that need their own instance
func testInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		ex.API.AuthenticatedSupport = true
		ex.API.AuthenticatedWebsocketSupport = true
		ex.SetCredentials(apiCredentials)
		ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	return ex
}

// ensureTradablePairs loads the live tradable pairs on first use, keeping API calls out of TestMain, and sets each
// asset's test pair to its first enabled pair
func ensureTradablePairs(tb testing.TB) {
	tb.Helper()
	testexch.UpdatePairsOnce(tb, e)
	tradablePairsOnce.Do(func() {
		for a, p := range map[asset.Item]*currency.Pair{
			asset.Spot:                &spotTradablePair,
			asset.USDTMarginedFutures: &usdtMarginedTradablePair,
			asset.USDCMarginedFutures: &usdcMarginedTradablePair,
			asset.CoinMarginedFutures: &inverseTradablePair,
			asset.Options:             &optionsTradablePair,
		} {
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(tb, err, "GetEnabledPairs must not error for %s", a)
			require.NotEmptyf(tb, pairs, "GetEnabledPairs must return pairs for %s", a)
			*p = pairs[0]
		}
	})
}
