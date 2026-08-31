//go:build !mock_test_off

// This will build unless build tag mock_test_off is parsed and will do mock testing
// using all tests in (exchange)_test.go
package gateio

import (
	"log"
	"os"
	"testing"

	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatal(err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiCredentials)
	}
	if err := testexch.MockHTTPInstance(e, ""); err != nil {
		log.Fatalf("MockHTTPInstance error: %s", err)
	}
	// Mock responses are served locally, so throttling protects nothing and long windows such as the
	// 100-per-day CrossEx limits would otherwise stall any endpoint exercised more than once per run
	if err := e.DisableRateLimiter(); err != nil {
		log.Fatal(err)
	}
	if err := e.enablePairs(); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
