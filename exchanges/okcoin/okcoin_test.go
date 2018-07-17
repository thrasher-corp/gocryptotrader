package okcoin

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var o OKCoin

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	o.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	okcoinConfig, err := cfg.GetExchangeConfig("OKCOIN International")
	if err != nil {
		t.Error("Test Failed - OKCoin Setup() init error")
	}

	okcoinConfig.AuthenticatedAPISupport = true
	okcoinConfig.APIKey = apiKey
	okcoinConfig.APISecret = apiSecret

	o.Setup(okcoinConfig)
}
