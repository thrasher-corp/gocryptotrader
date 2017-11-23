package poloniex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var p Poloniex

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
	poloniexConfig, err := cfg.GetExchangeConfig("Poloniex")
	if err != nil {
		t.Error("Test Failed - Poloniex Setup() init error")
	}

	poloniexConfig.AuthenticatedAPISupport = true
	poloniexConfig.APIKey = apiKey
	poloniexConfig.APISecret = apiSecret

	p.Setup(poloniexConfig)
}

func TestGetFee(t *testing.T) {
	if p.GetFee() != 0 {
		t.Error("Test faild - Poloniex GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	_, err := p.GetTicker()
	if err != nil {
		t.Error("Test faild - Poloniex GetTicker() error")
	}
}

func TestGetVolume(t *testing.T) {
	_, err := p.GetVolume()
	if err != nil {
		t.Error("Test faild - Poloniex GetVolume() error")
	}
}

func TestGetOrderbook(t *testing.T) {
	_, err := p.GetOrderbook("BTC_XMR", 50)
	if err != nil {
		t.Error("Test faild - Poloniex GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	_, err := p.GetTradeHistory("BTC_XMR", "", "")
	if err != nil {
		t.Error("Test faild - Poloniex GetTradeHistory() error", err)
	}
}

func TestGetChartData(t *testing.T) {
	_, err := p.GetChartData("BTC_XMR", "1405699200", "1405699400", "300")
	if err != nil {
		t.Error("Test faild - Poloniex GetChartData() error", err)
	}
}

func TestGetCurrencies(t *testing.T) {
	_, err := p.GetCurrencies()
	if err != nil {
		t.Error("Test faild - Poloniex GetCurrencies() error", err)
	}
}

func TestGetLoanOrders(t *testing.T) {
	_, err := p.GetLoanOrders("BTC")
	if err != nil {
		t.Error("Test faild - Poloniex GetLoanOrders() error", err)
	}
}
