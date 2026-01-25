//go:build !mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
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
	if apiKey != "" && apiSecret != "" {
		e.API.AuthenticatedSupport = true
		e.API.AuthenticatedWebsocketSupport = true
		e.SetCredentials(apiKey, apiSecret, "", "", "", "")
	}
	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("Poloniex MockHTTPInstance error: %s", err)
	}
	os.Exit(m.Run())
}
