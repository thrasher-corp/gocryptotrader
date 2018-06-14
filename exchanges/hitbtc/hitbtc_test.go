package hitbtc

import (
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
)

var h HitBTC

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
	hitbtcConfig, err := cfg.GetExchangeConfig("HitBTC")
	if err != nil {
		t.Error("Test Failed - HitBTC Setup() init error")
	}

	hitbtcConfig.AuthenticatedAPISupport = true
	hitbtcConfig.APIKey = apiKey
	hitbtcConfig.APISecret = apiSecret

	h.Setup(hitbtcConfig)
}

func TestGetFee(t *testing.T) {
	if h.GetFee() != 0 {
		t.Error("Test faild - HitBTC GetFee() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := h.GetOrderbook("BTCUSD", 50)
	if err != nil {
		t.Error("Test faild - HitBTC GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	_, err := h.GetTrades("BTCUSD", "", "", "", "", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetTradeHistory() error", err)
	}
}

func TestGetChartCandles(t *testing.T) {
	_, err := h.GetCandles("BTCUSD", "", "")
	if err != nil {
		t.Error("Test faild - HitBTC GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := h.GetCurrencies("")
	if err != nil {
		t.Error("Test faild - HitBTC GetCurrencies() error", err)
	}
}

func TestGetExchangeHistory(t *testing.T) {
	p := pair.NewCurrencyPair("BTC", "USD")
	_, err := h.GetExchangeHistory(p, "SPOT", time.Time{}, 0)
	if err != nil {
		t.Error("Test faild - HitBTC GetExchangeHistory() error", err)
	}
}
