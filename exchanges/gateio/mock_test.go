//go:build !mock_test_off

// This will build unless build tag mock_test_off is parsed and will do mock testing
// using all tests in (exchange)_test.go
package gateio

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := setupMockExchange(e); err != nil {
		log.Fatalf("GateIO mock setup error: %s", err)
	}
	os.Exit(m.Run())
}

// setupMockExchange serves an exchange from the recorded responses in testdata/http.json, signing with placeholder
// credentials so authenticated endpoints run against their recorded responses too
func setupMockExchange(ex *Exchange) error {
	if err := testexch.Setup(ex); err != nil {
		return err
	}
	ex.API.AuthenticatedSupport = true
	ex.API.AuthenticatedWebsocketSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret"})
	if err := testexch.MockHTTPInstance(ex, ""); err != nil {
		return err
	}
	// Mock responses are served locally, so throttling protects nothing and long windows such as the
	// 100-per-day CrossEx limits would otherwise stall any endpoint exercised more than once per run
	if err := ex.DisableRateLimiter(); err != nil {
		return err
	}
	enabledAssetPair = mockTestPairs()
	return storeTestPairs(ex)
}

// ensureTestPairs is a no-op for mock tests, which use the fixed pairs their fixtures record
func ensureTestPairs(testing.TB) {}
