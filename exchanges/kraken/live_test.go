//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package kraken

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchange/stream"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Kraken")
		os.Exit(0)
	}
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Kraken Setup error: %s", err)
	}
	if err := configureTestExchange(e); err != nil {
		log.Fatalf("Kraken live setup error: %s", err)
	}
	e.Websocket.DataHandler = stream.NewRelay(sharedtestvalues.WebsocketRelayBufferCapacity)
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}
