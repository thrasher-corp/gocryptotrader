//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitstamp

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

const sandboxAPIURL = "https://sandbox.bitstamp.net/api"

var (
	mockTests = false
	// Please supply your own credentials here to do authenticated endpoint testing.
	// Sandbox credentials are required when useTestNet is enabled.
	apiCredentials = &accounts.Credentials{
		Key:    "",
		Secret: "",
	}
)

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Bitstamp")
		os.Exit(0)
	}

	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bitstamp Setup error: %s", err)
	}

	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}

	if useTestNet {
		// The sandbox only serves the REST API, so websocket tests continue to use the production server
		if err := e.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), sandboxAPIURL); err != nil {
			log.Fatalf("Bitstamp SetRunningURL error: %s", err)
		}
	}
	e.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}
