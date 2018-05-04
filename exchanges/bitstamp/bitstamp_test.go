package bitstamp

import (
	"net/url"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please add your private keys and customerID for better tests
const (
	apiKey     = ""
	apiSecret  = ""
	customerID = ""
)

var b Bitstamp

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()

	if b.Name != "Bitstamp" {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Enabled != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Verbose != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.Websocket != false {
		t.Error("Test Failed - SetDefaults() error")
	}
	if b.RESTPollingDelay != 10 {
		t.Error("Test Failed - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("Bitstamp")
	if err != nil {
		t.Error("Test Failed - Bitstamp Setup() init error")
	}
	b.Setup(bConfig)

	if !b.IsEnabled() || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - Bitstamp Setup values not set correctly")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if resp := b.GetFee("BTCUSD"); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
	if resp := b.GetFee("bla"); resp != 0 {
		t.Error("Test Failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := b.GetTicker("BTCUSD", false)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
	_, err = b.GetTicker("BTCUSD", true)
	if err != nil {
		t.Error("Test Failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderbook("BTCUSD")
	if err != nil {
		t.Error("Test Failed - GetOrderbook() error", err)
	}
}

func TestGetTradingPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetTradingPairs()
	if err != nil {
		t.Error("Test Failed - GetTradingPairs() error", err)
	}
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	value := url.Values{}
	value.Set("time", "hour")

	_, err := b.GetTransactions("BTCUSD", value)
	if err != nil {
		t.Error("Test Failed - GetTransactions() error", err)
	}
	_, err = b.GetTransactions("wigwham", value)
	if err == nil {
		t.Error("Test Failed - GetTransactions() error")
	}
}

func TestGetEURUSDConversionRate(t *testing.T) {
	t.Parallel()
	_, err := b.GetEURUSDConversionRate()
	if err != nil {
		t.Error("Test Failed - GetEURUSDConversionRate() error", err)
	}
}

func TestGetBalance(t *testing.T) {
	t.Parallel()
	_, err := b.GetBalance()
	if err != nil {
		t.Error("Test Failed - GetBalance() error", err)
	}
}

func TestGetUserTransactions(t *testing.T) {
	t.Parallel()
	_, err := b.GetUserTransactions("")
	if err == nil {
		t.Error("Test Failed - GetUserTransactions() error", err)
	}

	_, err = b.GetUserTransactions("btcusd")
	if err == nil {
		t.Error("Test Failed - GetUserTransactions() error", err)
	}
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()

	_, err := b.GetOpenOrders("btcusd")
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error", err)
	}
	_, err = b.GetOpenOrders("wigwham")
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error")
	}
}

func TestGetOrderStatus(t *testing.T) {
	t.Parallel()

	_, err := b.GetOrderStatus(1337)
	if err == nil {
		t.Error("Test Failed - GetOpenOrders() error")
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()

	resp, err := b.CancelOrder(1337)
	if err == nil || resp != false {
		t.Error("Test Failed - CancelOrder() error")
	}
}

func TestCancelAllOrders(t *testing.T) {
	t.Parallel()

	_, err := b.CancelAllOrders()
	if err == nil {
		t.Error("Test Failed - CancelAllOrders() error", err)
	}
}

func TestPlaceOrder(t *testing.T) {
	t.Parallel()

	_, err := b.PlaceOrder("btcusd", 0.01, 1, true, true)
	if err == nil {
		t.Error("Test Failed - PlaceOrder() error")
	}
}

func TestGetWithdrawalRequests(t *testing.T) {
	t.Parallel()

	_, err := b.GetWithdrawalRequests(0)
	if err == nil {
		t.Error("Test Failed - GetWithdrawalRequests() error", err)
	}
	_, err = b.GetWithdrawalRequests(-1)
	if err == nil {
		t.Error("Test Failed - GetWithdrawalRequests() error")
	}
}

func TestCryptoWithdrawal(t *testing.T) {
	t.Parallel()

	_, err := b.CryptoWithdrawal(0, "bla", "btc", "", true)
	if err == nil {
		t.Error("Test Failed - CryptoWithdrawal() error", err)
	}
}

func TestGetBitcoinDepositAddress(t *testing.T) {
	t.Parallel()

	_, err := b.GetCryptoDepositAddress("btc")
	if err == nil {
		t.Error("Test Failed - GetCryptoDepositAddress() error", err)
	}
}

func TestGetUnconfirmedBitcoinDeposits(t *testing.T) {
	t.Parallel()

	_, err := b.GetUnconfirmedBitcoinDeposits()
	if err == nil {
		t.Error("Test Failed - GetUnconfirmedBitcoinDeposits() error", err)
	}
}

func TestTransferAccountBalance(t *testing.T) {
	t.Parallel()

	_, err := b.TransferAccountBalance(1, "", "", true)
	if err == nil {
		t.Error("Test Failed - TransferAccountBalance() error", err)
	}
	_, err = b.TransferAccountBalance(1, "btc", "", false)
	if err == nil {
		t.Error("Test Failed - TransferAccountBalance() error", err)
	}
}
