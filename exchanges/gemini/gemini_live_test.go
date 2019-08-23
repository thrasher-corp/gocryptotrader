//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gemini

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
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		log.Fatal("Test Failed - Gemini Setup() init error", err)
	}
	geminiConfig.AuthenticatedAPISupport = true
	geminiConfig.APIKey = apiKey
	geminiConfig.APISecret = apiSecret
	g.SetDefaults()
	g.Setup(&geminiConfig)
	g.APIUrl = geminiSandboxAPIURL
	log.Printf(sharedtestvalues.LiveTesting, g.GetName(), g.APIUrl)
	os.Exit(m.Run())
}
