//+build mock_test_off

// This will build if build tag mock_test_off is parsed and will do live testing
// using all tests in (exchange)_test.go
package gemini

import (
	"os"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	log "github.com/thrasher-corp/gocryptotrader/logger"
)

var mockTests = false

func TestMain(m *testing.M) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		log.Error("Test Failed - Gemini Setup() init error", err)
		os.Exit(1)
	}
	geminiConfig.AuthenticatedAPISupport = true
	geminiConfig.APIKey = apiKey
	geminiConfig.APISecret = apiSecret
	g.SetDefaults()
	g.Setup(&geminiConfig)
	g.APIUrl = geminiSandboxAPIURL
	log.Debugf(sharedtestvalues.LiveTesting, g.GetName(), g.APIUrl)
	os.Exit(m.Run())
}
