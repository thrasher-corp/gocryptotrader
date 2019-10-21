//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package anx

import (
	"log"
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatalf("ANX Setup() load config error: %s", err)
	}
	anxConfig, err := cfg.GetExchangeConfig("ANX")
	if err != nil {
		log.Fatalf("ANX Setup() init error: %s", err)
	}
	anxConfig.API.AuthenticatedSupport = true
	anxConfig.API.Credentials.Key = apiKey
	anxConfig.API.Credentials.Secret = apiSecret
	a.SetDefaults()
	err = a.Setup(anxConfig)
	if err != nil {
		log.Fatal("ANX setup error", err)
	}
	log.Printf(sharedtestvalues.LiveTesting, a.GetName(), a.API.Endpoints.URL)
	os.Exit(m.Run())
}
