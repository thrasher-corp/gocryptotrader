package hitbtc

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var p HitBTC

// Please supply your own APIKEYS here for due diligence testing

const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	p.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	hitbtcConfig, err := cfg.GetExchangeConfig("HitBTC")
	if err != nil {
		t.Error("Test Failed - HitBTC Setup() init error")
	}

	hitbtcConfig.AuthenticatedAPISupport = true
	hitbtcConfig.APIKey = apiKey
	hitbtcConfig.APISecret = apiSecret

	p.Setup(hitbtcConfig)
}

func TestGetFee(t *testing.T) {
	if p.GetFee() != 0 {
		t.Error("Test faild - HitBTC GetFee() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := p.GetOrderbook("BTCUSD", 50)
	if err != nil {
		t.Error("Test faild - HitBTC GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := p.GetTrades("BTCUSD", "", "", "", "", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetTradeHistory() error", err)
	}
}

func TestGetChartCandles(t *testing.T) {
	_, err := p.GetCandles("BTCUSD", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := p.GetCurrencies("")
	if err != nil {
		t.Error("Test faild - HitBTC GetCurrencies() error", err)
	}
}
