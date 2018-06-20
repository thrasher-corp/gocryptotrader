package gateio

import (
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

func TestGetKline(t *testing.T) {
	t.Parallel()
	_, err := g.GetKline(GateioKlinesRequestParams{
		Symbol:   g.GetSymbol(),
		GroupSec: GateioIntervalFiveMinutes, //5分钟以内数据
		HourSize: 1,                         //1小时内数据
	})
	if err != nil {
		t.Errorf("Test failed - Gateio GetKline: %s", err)
	}
}
