package okex

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

var o OKEX

// Please supply you own test keys here for due diligence testing.
const (
	apiKey    = ""
	apiSecret = ""
)

func TestSetDefaults(t *testing.T) {
	o.SetDefaults()
	if o.GetName() != "OKEX" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	okexConfig, err := cfg.GetExchangeConfig("OKEX")
	if err != nil {
		t.Error("Test Failed - Okex Setup() init error")
	}

	okexConfig.AuthenticatedAPISupport = true
	okexConfig.APIKey = apiKey
	okexConfig.APISecret = apiSecret

	o.Setup(okexConfig)
}

func TestGetContractPrice(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractPrice("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
	_, err = o.GetContractPrice("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
	_, err = o.GetContractPrice("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractPrice() error", err)
	}
}

func TestGetContractMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractMarketDepth("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
	_, err = o.GetContractMarketDepth("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
	_, err = o.GetContractMarketDepth("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractMarketDepth() error", err)
	}
}

func TestGetContractTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractTradeHistory("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
	_, err = o.GetContractTradeHistory("btc_bla", "123525")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
	_, err = o.GetContractTradeHistory("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetContractIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractIndexPrice("btc_usd")
	if err != nil {
		t.Error("Test failed - okex GetContractIndexPrice() error", err)
	}
	_, err = o.GetContractIndexPrice("lol123")
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetContractExchangeRate(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractExchangeRate()
	if err != nil {
		t.Error("Test failed - okex GetContractExchangeRate() error", err)
	}
}

func TestGetContractCandlestickData(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractCandlestickData("btc_usd", "1min", "this_week", 1, 2)
	if err != nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = o.GetContractCandlestickData("btc_bla", "1min", "this_week", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = o.GetContractCandlestickData("btc_usd", "min", "this_week", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
	_, err = o.GetContractCandlestickData("btc_usd", "1min", "this_wok", 1, 2)
	if err == nil {
		t.Error("Test failed - okex GetContractCandlestickData() error", err)
	}
}

func TestGetContractHoldingsNumber(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractHoldingsNumber("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
	_, err = o.GetContractHoldingsNumber("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
	_, err = o.GetContractHoldingsNumber("btc_usd", "this_bla")
	if err == nil {
		t.Error("Test failed - okex GetContractHoldingsNumber() error", err)
	}
}

func TestGetContractlimit(t *testing.T) {
	t.Parallel()
	_, err := o.GetContractlimit("btc_usd", "this_week")
	if err != nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
	_, err = o.GetContractlimit("btc_bla", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
	_, err = o.GetContractlimit("btc_usd", "this_bla")
	if err == nil {
		t.Error("Test failed - okex GetContractlimit() error", err)
	}
}

func TestGetContractUserInfo(t *testing.T) {
	t.Parallel()
	err := o.GetContractUserInfo()
	if err == nil {
		t.Error("Test failed - okex GetContractUserInfo() error", err)
	}
}

func TestGetContractPosition(t *testing.T) {
	t.Parallel()
	err := o.GetContractPosition("btc_usd", "this_week")
	if err == nil {
		t.Error("Test failed - okex GetContractPosition() error", err)
	}
}

func TestPlaceContractOrders(t *testing.T) {
	t.Parallel()
	_, err := o.PlaceContractOrders("btc_usd", "this_week", "1", 10, 1, 1, true)
	if err == nil {
		t.Error("Test failed - okex PlaceContractOrders() error", err)
	}
}

func TestGetContractFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	err := o.GetContractFuturesTradeHistory("btc_usd", "1972-01-01", 0)
	if err == nil {
		t.Error("Test failed - okex GetContractTradeHistory() error", err)
	}
}

func TestGetSpotTicker(t *testing.T) {
	t.Parallel()
	_, err := o.GetSpotTicker("ltc_btc")
	if err != nil {
		t.Error("Test failed - okex GetSpotTicker() error", err)
	}
}

func TestGetSpotMarketDepth(t *testing.T) {
	t.Parallel()
	_, err := o.GetSpotMarketDepth("eth_btc", "2")
	if err != nil {
		t.Error("Test failed - okex GetSpotMarketDepth() error", err)
	}
}

func TestGetSpotRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := o.GetSpotRecentTrades("ltc_btc", "0")
	if err != nil {
		t.Error("Test failed - okex GetSpotRecentTrades() error", err)
	}
}

func TestGetSpotCandleStick(t *testing.T) {
	t.Parallel()
	_, err := o.GetSpotCandleStick("ltc_btc", "1min", 2, 0)
	if err != nil {
		t.Error("Test failed - okex GetSpotCandleStick() error", err)
	}
}
