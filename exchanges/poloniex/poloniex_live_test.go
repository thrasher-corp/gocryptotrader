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
	p = new(Poloniex)
	if err := testexch.Setup(p); err != nil {
		log.Fatalf("Poloniex Setup error: %s", err)
	}
	if apiKey != "" && apiSecret != "" {
		p.API.AuthenticatedSupport = true
		p.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	log.Printf(sharedtestvalues.LiveTesting, p.Name)
	p.Websocket.DataHandler = sharedtestvalues.GetWebsocketInterfaceChannelOverride()
	p.Websocket.TrafficAlert = sharedtestvalues.GetWebsocketStructChannelOverride()
	os.Exit(m.Run())
}
