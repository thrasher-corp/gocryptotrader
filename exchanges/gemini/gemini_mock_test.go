//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package gemini

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
		log.Fatalf("Gemini Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("Gemini MockHTTPInstance error: %s", err)
	}

	os.Exit(m.Run())
}
