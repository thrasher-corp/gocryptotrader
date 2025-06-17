//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package binance

import (
	"context"
	"log"
	"os"
	"testing"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	b = new(Binance)
	if err := testexch.Setup(b); err != nil {
		log.Fatal(err)
	}

	if apiKey != "" && apiSecret != "" {
		b.API.AuthenticatedSupport = true
		b.API.CredentialsValidator.RequiresBase64DecodeSecret = false
		b.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}

	if useTestNet {
		for k, v := range map[exchange.URL]string{
			exchange.RestUSDTMargined: testnetFutures,
			exchange.RestCoinMargined: testnetFutures,
			exchange.RestSpot:         testnetSpotURL,
		} {
			if err := b.API.Endpoints.SetRunning(k.String(), v); err != nil {
				log.Fatalf("Testnet %q URL error with %q: %s", k, v, err)
			}
		}
	}

	ctx := context.Background()
	b.setupOrderbookManager(ctx)
	b.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	log.Printf(sharedtestvalues.LiveTesting, b.Name)
	if err := b.UpdateTradablePairs(ctx, true); err != nil {
		log.Fatal("Binance setup error", err)
	}

	os.Exit(m.Run())
}
