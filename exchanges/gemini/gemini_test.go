package gemini

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var (
	g Gemini
)

// Please enter sandbox API keys & assigned roles for better testing procedures

const (
	apiKey1           = ""
	apiSecret1        = ""
	apiKeyRole1       = ""
	sessionHeartBeat1 = false

	apiKey2           = ""
	apiSecret2        = ""
	apiKeyRole2       = ""
	sessionHeartBeat2 = false
)

func TestAddSession(t *testing.T) {
	err := AddSession(&g, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
	err = AddSession(&g, 1, apiKey1, apiSecret1, apiKeyRole1, true, false)
	if err == nil {
		t.Error("Test failed - AddSession() error")
	}
	err = AddSession(&g, 2, apiKey2, apiSecret2, apiKeyRole2, false, true)
	if err != nil {
		t.Error("Test failed - AddSession() error")
	}
}

func TestSetDefaults(t *testing.T) {
	Session[1].SetDefaults()
	Session[2].SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	geminiConfig, err := cfg.GetExchangeConfig("Gemini")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	geminiConfig.AuthenticatedAPISupport = true

	Session[1].Setup(geminiConfig)
	Session[2].Setup(geminiConfig)
}

// func TestSandbox(t *testing.T) {
// 	t.Parallel()
// 	g.Sandbox(1)
// 	if Management[1].URL != geminiSandboxAPIURL {
// 		t.Error("Test Failed - Sandbox() error")
// 	}
// }

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetSymbols()
	if err != nil {
		t.Error("Test Failed - GetSymbols() error", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTicker("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = Session[1].GetTicker("bla")
	if err == nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrderbook("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTrades("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetTrades() error", err)
	}
}

func TestGetAuction(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetAuction("btcusd")
	if err != nil {
		t.Error("Test Failed - GetAuction() error", err)
	}
}

func TestGetAuctionHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetAuctionHistory("btcusd", url.Values{})
	if err != nil {
		t.Error("Test Failed - GetAuctionHistory() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
	_, err = Session[2].NewOrder("btcusd", 1, 4500, "buy", "exchange limit")
	if err == nil {
		t.Error("Test Failed - NewOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelOrder(1337)
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestCancelOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].CancelOrders(false)
	if err == nil {
		t.Error("Test Failed - CancelOrders() error", err)
	}
	_, err = Session[2].CancelOrders(true)
	if err == nil {
		t.Error("Test Failed - CancelOrders() error", err)
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOrderStatus() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetOrders()
	if err == nil {
		t.Error("Test Failed - GetOrders() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetTradeHistory("btcusd", 0)
	if err == nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	_, err := Session[2].GetTradeVolume()
	if err == nil {
		t.Error("Test Failed - GetTradeVolume() error", err)
	}
}

func TestGetBalances(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetBalances()
	if err == nil {
		t.Error("Test Failed - GetBalances() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	t.Parallel()
	_, err := Session[1].GetDepositAddress("LOL123", "btc")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := Session[1].WithdrawCrypto("LOL123", "btc", 1)
	if err == nil {
		t.Error("Test Failed - WithdrawCrypto() error", err)
	}
}

func TestPostHeartbeat(t *testing.T) {
	t.Parallel()
	_, err := Session[2].PostHeartbeat()
	if err == nil {
		t.Error("Test Failed - PostHeartbeat() error", err)
	}
}
