//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package poloniex

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Poloniex Setup error: %s", err)
	}
	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	e.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	e.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}
