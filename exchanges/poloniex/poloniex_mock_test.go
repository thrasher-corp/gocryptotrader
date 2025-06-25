//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package poloniex

import (
	"log"
	"os"
	"testing"

	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	p = new(Poloniex)
	if err := testexch.Setup(p); err != nil {
		log.Fatalf("Poloniex Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(p); err != nil {
		log.Fatalf("Poloniex MockHTTPInstance error: %s", err)
	}

	os.Exit(m.Run())
}
