//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package binance

import (
	"context"
	"log"
	"os"
	"testing"

	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var mockTests = true

func TestMain(m *testing.M) {
	if useTestNet {
		log.Fatal("cannot use testnet with mock tests")
	}

	b = new(Binance)
	if err := testexch.Setup(b); err != nil {
		log.Fatalf("Binance Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(b); err != nil {
		log.Fatalf("Binance MockHTTPInstance error: %s", err)
	}

	if err := b.UpdateTradablePairs(context.Background(), true); err != nil {
		log.Fatalf("Binance UpdateTradablePairs error: %s", err)
	}

	os.Exit(m.Run())
}
