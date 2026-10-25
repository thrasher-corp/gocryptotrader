//go:build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package bitstamp

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
)

var (
	mockTests = false
	// Please supply your own credentials here to do authenticated endpoint testing.
	apiCredentials = &accounts.Credentials{
		Key:      "",
		Secret:   "",
		ClientID: "", // customerID used to log in
	}
)

func TestMain(m *testing.M) {
	e = new(Exchange)
	if err := testexch.Setup(e); err != nil {
		log.Fatalf("Bitstamp Setup error: %s", err)
	}
	if apiCredentials.Key != "" && apiCredentials.Secret != "" {
		e.API.AuthenticatedSupport = true
		e.SetCredentials(apiCredentials)
	}
	log.Printf(sharedtestvalues.LiveTesting, e.Name)
	os.Exit(m.Run())
}
