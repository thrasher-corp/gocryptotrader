//go:build !mock_test_off

// This will build if build tag mock_test_off is not parsed and will try to mock
// all tests in _test.go
package mexc

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/currency"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("MEXC Setup error: %s", err)
	}

	if err := testexch.MockHTTPInstance(e); err != nil {
		log.Fatalf("MEXC MockHTTPInstance error: %s", err)
	}
	// if err := populateTradablePairs(); err != nil {
	// 	log.Fatal(err)
	// }
	var err error
	spotTradablePair, err = currency.NewPairFromString("BTCUSDT")
	if err != nil {
		log.Fatal(err)
	}
	futuresTradablePair, err = currency.NewPairFromString("BTC_USDT")
	if err != nil {
		log.Fatal(err)
	}
	if err := e.setEnabledPairs(spotTradablePair, futuresTradablePair); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}
