//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package okx

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"golang.org/x/time/rate"
)

var mockTests = true

// mockRecordingTime is when the recorded responses were captured
var mockRecordingTime = time.Date(2026, time.November, 23, 6, 0, 0, 0, time.UTC)

func TestMain(m *testing.M) {
	// The recorded responses answer fixed windows, which the wrapper judges
	// against its clock, so the clock starts at their recording time instead
	// of the calendar's and runs from there
	started := time.Now()
	timeNow = func() time.Time { return mockRecordingTime.Add(time.Since(started)) }
	// Mock websockets dial local servers, which OKX's per IP connection pacing
	// would only queue behind each other across the package's parallel tests
	rateLimits[websocketConnectionEPL] = request.GetRateLimiterWithWeight(rate.NewLimiter(rate.Inf, 1), 1)
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Okx Setup error: %s", err)
	}
	if err := testexch.MockHTTPInstance(e, "api/v5/"); err != nil {
		log.Fatalf("Okx MockHTTPInstance error: %s", err)
	}
	if err := e.DisableRateLimiter(); err != nil {
		log.Fatalf("Okx DisableRateLimiter error: %s", err)
	}
	e.API.AuthenticatedSupport = true
	e.API.AuthenticatedWebsocketSupport = true
	e.SetCredentials(&accounts.Credentials{Key: "mock-api-key", Secret: "mock-api-secret", ClientID: "mock-passphrase"})
	os.Exit(m.Run())
}
