package okex

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/idoall/gocryptotrader/config"
)

var o OKEX

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "okex",
		Enabled:                 true,
		Verbose:                 true,
		Websocket:               false,
		BaseAsset:               "eth",
		QuoteAsset:              "usdt",
		UseSandbox:              false,
		RESTPollingDelay:        10,
		HTTPTimeout:             15000000000,
		AuthenticatedAPISupport: true,
		APIKey:                  "",
		APISecret:               "",
		SupportsAutoPairUpdates: false,
		RequestCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "_",
		},
	}
}

func TestSetDefaults(t *testing.T) {
	o.SetDefaults()
	if o.GetName() != "OKEX" {
		t.Error("Test Failed - Bittrex - SetDefaults() error")
	}
}

func TestSetup(t *testing.T) {
	o.Setup(getDefaultConfig())
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
	list, err := o.GetSpotCandleStick("ltc_btc", "1min", 2, 0)
	if err != nil {
		t.Error("Test failed - okex GetSpotCandleStick() error", err)
	}
	for _, v := range list {
		b, _ := json.Marshal(v)
		fmt.Printf("%v \n", string(b))
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	_, err := o.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: o.GetSymbol(),
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestTypeBuy,
	})
	if err != nil {
		t.Error("Test failed - okex SpotNewOrder() error", err)
	}
}

func TestSpotCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := o.SpotCancelOrder(o.GetSymbol(), 519158961)
	if err != nil {
		t.Error("Test failed - okex SpotCancelOrder() error", err)
	}
}

func TestGetUserInfo(t *testing.T) {
	t.Parallel()
	userInfo, err := o.GetUserInfo()
	if err != nil {
		t.Error("Test failed - okex GetUserInfo() error", err)
	} else {

		t.Log("====账户余额")
		for k, v := range userInfo.Info["funds"]["free"] {
			t.Log(k, v)
		}

		t.Log("====账户冻结余额")
		for k, v := range userInfo.Info["funds"]["freezed"] {
			t.Log(k, v)
		}
	}
}
