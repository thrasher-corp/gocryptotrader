//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"log"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

var tradablePairsOnce sync.Once

func TestMain(m *testing.M) {
	// Binance refuses requests from the US, where CI runs
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Binance")
		os.Exit(0)
	}
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Binance Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}
	if useTestNet {
		for k, v := range map[exchange.URL]string{
			exchange.RestUSDTMargined:      "https://testnet.binancefuture.com",
			exchange.RestCoinMargined:      "https://testnet.binancefuture.com",
			exchange.RestSpotSupplementary: "https://testnet.binance.vision",
		} {
			if err := e.API.Endpoints.SetRunningURL(k.String(), v); err != nil {
				log.Fatalf("Binance SetRunningURL error: %s", err)
			}
		}
	}
	e.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}

// ensureTradablePairs loads the live tradable pairs on first use, keeping API calls out of TestMain, and sets each
// asset's test pair to its first enabled pair
func ensureTradablePairs(tb testing.TB) {
	tb.Helper()
	testexch.UpdatePairsOnce(tb, e)
	tradablePairsOnce.Do(func() {
		assetToTradablePairMap = make(map[asset.Item]currency.Pair)
		for _, a := range e.GetAssetTypes(true) {
			pairs, err := e.GetEnabledPairs(a)
			require.NoErrorf(tb, err, "GetEnabledPairs must not error for %s", a)
			require.NotEmptyf(tb, pairs, "GetEnabledPairs must return pairs for %s", a)
			assetToTradablePairMap[a] = pairs[0]
		}
		spotTradablePair = assetToTradablePairMap[asset.Spot]
		marginTradablePair = assetToTradablePairMap[asset.Margin]
		usdtmTradablePair = assetToTradablePairMap[asset.USDTMarginedFutures]
		coinmTradablePair = assetToTradablePairMap[asset.CoinMarginedFutures]
		optionsTradablePair = assetToTradablePairMap[asset.Options]
	})
}
