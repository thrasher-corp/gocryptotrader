package binance

import (
	"testing"

	"github.com/idoall/gocryptotrader/config"
)

// getDefaultConfig 获取默认配置
func getDefaultConfig() config.ExchangeConfig {
	return config.ExchangeConfig{
		Name:                    "binance",
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
		},
	}
}

var b Binance

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	b.Setup(getDefaultConfig())
}

func TestGetExchangeValidCurrencyPairs(t *testing.T) {
	t.Parallel()
	_, err := b.GetExchangeValidCurrencyPairs()
	if err != nil {
		t.Error("Test Failed - Binance GetExchangeValidCurrencyPairs() error", err)
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := b.GetOrderBook("BTCUSDT", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetOrderBook() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetRecentTrades("BTCUSDT", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetHistoricalTrades("BTCUSDT", 5, 1337)
	if err == nil {
		t.Error("Test Failed - Binance GetHistoricalTrades() error", err)
	}
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := b.GetAggregatedTrades("BTCUSDT", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetAggregatedTrades() error", err)
	}
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := b.GetSpotKline(KlinesRequestParams{
		Symbol:   b.GetSymbol(),
		Interval: TimeIntervalFiveMinutes,
		Limit:    24,
	})
	if err != nil {
		t.Error("Test Failed - Binance GetSpotKline() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := b.GetPriceChangeStats("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := b.GetTickers()
	if err != nil {
		t.Error("Test Failed - Binance TestGetTickers error", err)
	}
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetLatestSpotPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetLatestSpotPrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	_, err := b.GetBestPrice("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance GetBestPrice() error", err)
	}
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	_, err := b.NewOrderTest()
	if err != nil {
		t.Error("Test Failed - Binance NewOrderTest() error", err)
	}
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := b.NewOrder(NewOrderRequest{
		Symbol:      b.GetSymbol(),
		Side:        BinanceRequestParamsSideSell,
		TradeType:   BinanceRequestParamsOrderLimit,
		TimeInForce: BinanceRequestParamsTimeGTC,
		Quantity:    0.01,
		Price:       1536.1,
	})
	if err == nil {
		t.Error("Test Failed - Binance NewOrder() error", err)
	}
}

func TestCancelOrder(t *testing.T) {
	t.Parallel()
	_, err := b.CancelOrder(b.GetSymbol(), 82584683, "")
	if err == nil {
		t.Error("Test Failed - Binance CancelOrder() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := b.QueryOrder("", "", 1337)
	if err == nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	}
}
