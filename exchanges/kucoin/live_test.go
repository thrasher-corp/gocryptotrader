//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package kucoin

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
)

var mockTests = false

var tradablePairsOnce sync.Once

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Kucoin Setup error: %s", err)
	}
	setLiveCredentials(e)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// setLiveCredentials gives an exchange the test credentials, when they are set
func setLiveCredentials(ex *Exchange) {
	if apiCredentials.Key == "" || apiCredentials.Secret == "" || apiCredentials.ClientID == "" {
		return
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.API.CredentialsValidator.RequiresBase64DecodeSecret = false
	ex.SetCredentials(apiCredentials)
	ex.Websocket.SetCanUseAuthenticatedEndpoints(true)
}

// testInstance returns a new live exchange with the test credentials, for tests that need their own instance
func testInstance(tb testing.TB) *Exchange {
	tb.Helper()
	ex := new(Exchange)
	require.NoError(tb, testexch.Setup(ex), "Setup must not error")
	setLiveCredentials(ex)
	return ex
}

// ensureTradablePairs loads the live tradable pairs on first use, keeping API calls out of TestMain, and sets each
// asset's test pair to its first enabled pair
func ensureTradablePairs(tb testing.TB) {
	tb.Helper()
	testexch.UpdatePairsOnce(tb, e)
	tradablePairsOnce.Do(func() {
		for a, p := range map[asset.Item]*currency.Pair{
			asset.Spot:    &spotTradablePair,
			asset.Margin:  &marginTradablePair,
			asset.Futures: &futuresTradablePair,
		} {
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(tb, err, "GetEnabledPairs must not error for %s", a)
			require.NotEmptyf(tb, pairs, "GetEnabledPairs must return pairs for %s", a)
			*p = pairs[0]
		}
		futuresTradablePair.Delimiter = ""
		assertToTradablePairMap = map[asset.Item]currency.Pair{
			asset.Spot:    spotTradablePair,
			asset.Margin:  marginTradablePair,
			asset.Futures: futuresTradablePair,
		}
	})
}
