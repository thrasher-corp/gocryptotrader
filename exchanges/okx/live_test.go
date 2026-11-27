//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package okx

import (
	"log"
	"os"
	"testing"

	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/internal/testing/livetest"
)

var mockTests = false

func TestMain(m *testing.M) {
	if livetest.ShouldSkip() {
		log.Printf(livetest.LiveTestingSkipped, "Okx")
		os.Exit(0)
	}
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Okx Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" && apiCredentials.ClientID != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
		e.Websocket.SetCanUseAuthenticatedEndpoints(true)
	}
	os.Exit(m.Run())
}
