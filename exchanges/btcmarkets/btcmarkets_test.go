package btcmarkets

import (
	"net/url"
	"testing"

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
	conf := config.ExchangeConfig{}
	bm.Setup(conf)

	conf = config.ExchangeConfig{
		APIKey:                  apiKey,
		APISecret:               apiSecret,
		Enabled:                 true,
		AuthenticatedAPISupport: true,
	}
	bm.Setup(conf)
}

func TestGetFee(t *testing.T) {
	t.Parallel()
	if fee := bm.GetFee(); fee == 0 {
		t.Error("Test failed - GetFee() error")
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := bm.GetTicker("BTC")
	if err != nil {
		t.Error("Test failed - GetTicker() error", err)
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := bm.GetOrderbook("BTC")
	if err != nil {
		t.Error("Test failed - GetOrderbook() error", err)
	}
}

func TestGetTrades(t *testing.T) {
	t.Parallel()
	_, err := bm.GetTrades("BTC", nil)
	if err != nil {
		t.Error("Test failed - GetTrades() error", err)
	}

	val := url.Values{}
	val.Set("since", "0")
	_, err = bm.GetTrades("BTC", val)
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
