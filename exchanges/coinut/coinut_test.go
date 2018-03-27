package coinut

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

var c COINUT

// Please supply your own keys here to do better tests
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	c.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("COINUT")
	if err != nil {
		t.Error("Test Failed - Coinut Setup() init error")
	}
	c.Setup(bConfig)

	if !c.IsEnabled() || c.AuthenticatedAPISupport || c.RESTPollingDelay != time.Duration(10) ||
		c.Verbose || c.Websocket || len(c.BaseCurrencies) < 1 ||
		len(c.AvailablePairs) < 1 || len(c.EnabledPairs) < 1 {
		t.Error("Test Failed - Coinut Setup values not set correctly")
	}
}

func TestGetInstruments(t *testing.T) {
	_, err := c.GetInstruments()
	if err != nil {
		t.Error("Test failed - GetInstruments() error", err)
	}
}
