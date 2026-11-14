//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binanceus

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

var tradablePairsOnce sync.Once

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Binanceus Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}
	e.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// ensureTradablePairs loads the live tradable pairs on first use, keeping API calls out of TestMain, and sets the test
// pair to the first enabled spot pair
func ensureTradablePairs(tb testing.TB) {
	tb.Helper()
	testexch.UpdatePairsOnce(tb, e)
	tradablePairsOnce.Do(func() {
		pairs, err := e.GetEnabledPairs(asset.Spot)
		require.NoError(tb, err, "GetEnabledPairs must not error")
		require.NotEmpty(tb, pairs, "GetEnabledPairs must return pairs")
		spotTradablePair = pairs[0]
	})
}
