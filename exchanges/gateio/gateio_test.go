package gateio

import (
	"fmt"
	"testing"

	"github.com/idoall/gocryptotrader/config"
)

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "gateio",
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
		ClientID:                "",
		AvailablePairs:          "BTC-USDT,BCH-USDT",
		EnabledPairs:            "BTC-USDT",
		BaseCurrencies:          "USD",
		AssetTypes:              "SPOT",
		SupportsAutoPairUpdates: false,
		ConfigCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "-",
		},
		RequestCurrencyPairFormat: &config.CurrencyPairFormatConfig{
			Uppercase: true,
			Delimiter: "_",
		},
	}
}

var g Gateio

func TestSetDefaults(t *testing.T) {
	g.SetDefaults()
}

func TestSetup(t *testing.T) {
	g.Setup(getDefaultConfig())
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	_, err := g.GetSymbols()
	if err != nil {
		t.Errorf("Test failed - Gateio TestGetSymbols: %s", err)
	}
}

func TestGetMarketInfo(t *testing.T) {
	t.Parallel()
	_, err := g.GetMarketInfo()
	if err != nil {
		t.Errorf("Test failed - Gateio GetMarketInfo: %s", err)
	}
}

func TestSpotNewOrder(t *testing.T) {
	t.Parallel()
	_, err := g.SpotNewOrder(SpotNewOrderRequestParams{
		Symbol: g.GetSymbol(),
		Amount: 1.1,
		Price:  10.1,
		Type:   SpotNewOrderRequestParamsTypeSell,
	})
	if err != nil {
		t.Errorf("Test failed - Gateio SpotNewOrder: %s", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := g.CancelOrder(917591554, g.GetSymbol())
	if err != nil {
		t.Errorf("Test failed - Gateio CancelOrder: %s", err)
	}
}

func TestGetBalances(t *testing.T) {
	TestSetDefaults(t)
	TestSetup(t)
	t.Parallel()
	res, err := g.GetBalances()
	if err != nil {
		t.Errorf("Test failed - Gateio GetBalances: %s", err)
	}

	for k, v := range res.Available {
		fmt.Println(k, v)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := g.GetLatestSpotPrice(g.GetSymbol())
	if err != nil {
		t.Errorf("Test failed - Gateio GetLatestSpotPrice: %s", err)
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := g.GetTicker(g.GetSymbol())
	if err != nil {
		t.Errorf("Test failed - Gateio GetTicker: %s", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := g.GetSpotKline(KlinesRequestParams{
		Symbol:   g.GetSymbol(),
		GroupSec: TimeIntervalFiveMinutes, //5分钟以内数据
		HourSize: 1,                       //1小时内数据
	})
	if err != nil {
		t.Errorf("Test failed - Gateio GetSpotKline: %s", err)
	}
}
