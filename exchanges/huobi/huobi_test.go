package huobi

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var h HUOBI

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	h.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	huobiConfig, err := cfg.GetExchangeConfig("Huobi")
	if err != nil {
		t.Error("Test Failed - Huobi Setup() init error")
	}

	huobiConfig.AuthenticatedAPISupport = true
	huobiConfig.APIKey = apiKey
	huobiConfig.APISecret = apiSecret

	h.Setup(huobiConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if h.GetFee() != 0 {
		t.Error("test failed - Huobi GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := h.GetTicker("btcusd")
	if err == nil {
		t.Error("test failed - Huobi GetTicker() error", err)
	}
}
