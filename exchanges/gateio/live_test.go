//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gateio

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

var testPairsOnce sync.Once

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "GateIO")
		os.Exit(0)
	}
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("GateIO Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// ensureTestPairs loads the live tradable pairs on first use, keeping API calls out of TestMain, and sets each
// asset's test pair to its first enabled pair
func ensureTestPairs(tb testing.TB) {
	tb.Helper()
	testexch.UpdatePairsOnce(tb, e)
	testPairsOnce.Do(func() {
		pairs := make(map[asset.Item]currency.Pair, len(e.GetAssetTypes(true)))
		for _, a := range e.GetAssetTypes(true) {
			enabled, err := e.GetEnabledPairs(a)
			require.NoErrorf(tb, err, "GetEnabledPairs must not error for %s", a)
			require.NotEmptyf(tb, enabled, "GetEnabledPairs must return pairs for %s", a)
			pairs[a], err = e.FormatExchangeCurrency(enabled[0], a)
			require.NoErrorf(tb, err, "FormatExchangeCurrency must not error for %s", a)
		}
		enabledAssetPair = pairs
	})
}
