//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gemini

import (
	"log"
	"os"
	"testing"

	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = false

func TestMain(m *testing.M) {
	g = new(Gemini)
	if err := testexch.Setup(g); err != nil {
		log.Fatalf("Gemini Setup error: %s", err)
	}
	if apiKey != "" && apiSecret != "" {
		g.API.AuthenticatedSupport = true
		g.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	if err := g.API.Endpoints.SetRunningURL(exchange.RestSpot.String(), geminiAPIURL); err != nil {
		log.Fatalf("Gemini SetRunningURL error: %s", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, g.Name)
	os.Exit(m.Run())
}
