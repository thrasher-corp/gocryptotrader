package btcmarkets

import (
	"net/url"
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
	"github.com/thrasher-/gocryptotrader/currency/pair"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
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
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	bConfig, err := cfg.GetExchangeConfig("BTC Markets")
	if err != nil {
		t.Error("Test Failed - BTC Markets Setup() init error")
	}

	if apiKey != "" && apiSecret != "" {
		bConfig.APIKey = apiKey
		bConfig.APISecret = apiSecret
		bConfig.AuthenticatedAPISupport = true
	}

	bm.Setup(bConfig)
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
	_, err := bm.WithdrawAUD("BLA", "1337", "blawest", "1336", 10000000)
	if err == nil {
		t.Error("Test failed - WithdrawAUD() error", err)
	}
}

func TestGetExchangeAccountInfo(t *testing.T) {
	_, err := bm.GetExchangeAccountInfo()
	if err == nil {
		t.Error("Test failed - GetExchangeAccountInfo() error", err)
	}
}

func TestGetExchangeFundTransferHistory(t *testing.T) {
	_, err := bm.GetExchangeFundTransferHistory()
	if err == nil {
		t.Error("Test failed - GetExchangeAccountInfo() error", err)
	}
}

func TestSubmitExchangeOrder(t *testing.T) {
	p := pair.NewCurrencyPair("LTC", "AUD")
	_, err := bm.SubmitExchangeOrder(p, exchange.OrderSideSell(), exchange.OrderTypeMarket(), 0, 0.0, "testID001")
	if err == nil {
		t.Error("Test failed - SubmitExchangeOrder() error", err)
	}
}

func TestModifyExchangeOrder(t *testing.T) {
	_, err := bm.ModifyExchangeOrder(1337, exchange.ModifyOrder{})
	if err == nil {
		t.Error("Test failed - ModifyExchangeOrder() error", err)
	}
}

func TestCancelExchangeOrder(t *testing.T) {
	err := bm.CancelExchangeOrder(1337)
	if err == nil {
		t.Error("Test failed - CancelExchangeOrder() error", err)
	}
}

func TestCancelAllExchangeOrders(t *testing.T) {
	err := bm.CancelAllExchangeOrders()
	if err == nil {
		t.Error("Test failed - CancelAllExchangeOrders() error", err)
	}
}

func TestGetExchangeOrderInfo(t *testing.T) {
	_, err := bm.GetExchangeOrderInfo(1337)
	if err == nil {
		t.Error("Test failed - GetExchangeOrderInfo() error", err)
	}
}

func TestWithdrawCryptoExchangeFunds(t *testing.T) {
	_, err := bm.WithdrawCryptoExchangeFunds("someaddress", "ltc", 0)
	if err == nil {
		t.Error("Test failed - WithdrawExchangeFunds() error", err)
	}
}

func TestWithdrawFiatExchangeFunds(t *testing.T) {
	_, err := bm.WithdrawFiatExchangeFunds("AUD", 0)
	if err == nil {
		t.Error("Test failed - WithdrawFiatExchangeFunds() error", err)
	}
}
