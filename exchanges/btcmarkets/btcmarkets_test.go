package btcmarkets

import (
	"net/url"
	"testing"
	"time"

	"github.com/thrasher-/gocryptotrader/config"
)

var bm BTCMarkets

// Please supply your own keys here to do better tests
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	bm.SetDefaults()
}

func TestSetup(t *testing.T) {
	t.Parallel()
	b := BTCMarkets{}
	b.Name = "BTC Markets"
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		t.Error("Test Failed - BTC Markets Setup() init error")
	}

	b.SetDefaults()
	b.Setup(bConfig)

	if !b.IsEnabled() || b.AuthenticatedAPISupport || b.RESTPollingDelay != time.Duration(10) ||
		b.Verbose || b.Websocket || len(b.BaseCurrencies) < 1 ||
		len(b.AvailablePairs) < 1 || len(b.EnabledPairs) < 1 {
		t.Error("Test Failed - BTC Markets Setup values not set correctly")
	}

	bConfig.Enabled = false
	b.Setup(bConfig)

	if b.IsEnabled() {
		t.Error("Test failed - BTC Markets TestSetup incorrect value")
	}
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if fee := bm.GetFee(); fee == 0 {
		t.Error("Test failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := bm.GetTicker("BTC", "AUD")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bm.GetOrderbook("BTC", "AUD")
	if err != nil {
		t.Error("Test failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := bm.GetTrades("BTC", "AUD", nil)
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}

	val := url.Values{}
	val.Set("since", "0")
	_, err = bm.GetTrades("BTC", "AUD", val)
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := bm.NewOrder("AUD", "BTC", 0, 0, "Bid", "limit", "testTest")
	if err == nil {
		t.Error("Test failed - NewOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := bm.CancelOrder([]int64{1337})
	if err == nil {
		t.Error("Test failed - CancelOrder() error", err)
	}
}

func TestGetOrders(t *testing.T) {
	t.Parallel()
	_, err := bm.GetOrders("AUD", "BTC", 10, 0, false)
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
	_, err = bm.GetOrders("AUD", "BTC", 10, 0, true)
	if err == nil {
		t.Error("Test failed - GetOrders() error", err)
	}
}

func TestGetOrderDetail(t *testing.T) {
	t.Parallel()
	_, err := bm.GetOrderDetail([]int64{1337})
	if err == nil {
		t.Error("Test failed - GetOrderDetail() error", err)
	}
}

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	_, err := bm.GetAccountBalance()
	if err == nil {
		t.Error("Test failed - GetAccountBalance() error", err)
	}
}

func TestWithdrawCrypto(t *testing.T) {
	t.Parallel()
	_, err := bm.WithdrawCrypto(0, "BTC", "LOLOLOL")
	if err == nil {
		t.Error("Test failed - WithdrawCrypto() error", err)
	}
}

func TestWithdrawAUD(t *testing.T) {
	t.Parallel()
	_, err := bm.WithdrawAUD("BLA", "1337", "blawest", "1336", "BTC", 10000000)
	if err == nil {
		t.Error("Test failed - WithdrawAUD() error", err)
	}
}
