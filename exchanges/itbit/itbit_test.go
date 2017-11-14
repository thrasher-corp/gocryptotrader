package itbit

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var i ItBit

// Please provide your own keys to do proper testing
const (
	apiKey    = ""
	apiSecret = ""
	clientID  = ""
)

func TestSetDefaults(t *testing.T) {
	i.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	itbitConfig, err := cfg.GetExchangeConfig("ITBIT")
	if err != nil {
		t.Error("Test Failed - Gemini Setup() init error")
	}

	itbitConfig.AuthenticatedAPISupport = true
	itbitConfig.APIKey = apiKey
	itbitConfig.APISecret = apiSecret
	itbitConfig.ClientID = clientID

	i.Setup(itbitConfig)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if i.GetFee(true) != -0.1 || i.GetFee(false) != 0.5 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := i.GetTicker("XBTUSD")
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := i.GetOrderbook("XBTSGD")
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := i.GetTradeHistory("XBTUSD", "0")
	if err != nil {
		t.Error("Test Failed - GetTradeHistory() error", err)
	}
}

func TestGetWallets(t *testing.T) {
	_, err := i.GetWallets(url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWallets() error", err)
	}
}

func TestCreateWallet(t *testing.T) {
	_, err := i.CreateWallet("test")
	if err == nil {
		t.Error("Test Failed - CreateWallet() error", err)
	}
}

func TestGetWallet(t *testing.T) {
	_, err := i.GetWallet("1337")
	if err == nil {
		t.Error("Test Failed - GetWallet() error", err)
	}
}

func TestGetWalletBalance(t *testing.T) {
	_, err := i.GetWalletBalance("1337", "XRT")
	if err == nil {
		t.Error("Test Failed - GetWalletBalance() error", err)
	}
}

func TestGetWalletTrades(t *testing.T) {
	_, err := i.GetWalletTrades("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetWalletTrades() error", err)
	}
}

func TestGetFundingHistory(t *testing.T) {
	_, err := i.GetFundingHistory("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetFundingHistory() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	_, err := i.PlaceOrder("1337", "buy", "limit", "USD", 1, 0.2, "banjo", "sauce")
	if err == nil {
		t.Error("Test Failed - PlaceOrder() error", err)
	}
}

func TestGetOrder(t *testing.T) {
	_, err := i.GetOrder("1337", url.Values{})
	if err == nil {
		t.Error("Test Failed - GetOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Skip()
	err := i.CancelOrder("1337", "1337order")
	if err == nil {
		t.Error("Test Failed - CancelOrder() error", err)
	}
}

func TestGetDepositAddress(t *testing.T) {
	_, err := i.GetDepositAddress("1337", "AUD")
	if err == nil {
		t.Error("Test Failed - GetDepositAddress() error", err)
	}
}

func TestWalletTransfer(t *testing.T) {
	_, err := i.WalletTransfer("1337", "mywallet", "anotherwallet", 200, "USD")
	if err == nil {
		t.Error("Test Failed - WalletTransfer() error", err)
	}
}
