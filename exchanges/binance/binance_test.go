package binance

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/config"
)

// Please supply your own keys here for due diligence testing
const (
	testAPIKey    = ""
	testAPISecret = ""
)

var b Binance

func TestSetDefaults(t *testing.T) {
	b.SetDefaults()
}

func TestSetup(t *testing.T) {
	cfg := config.GetConfig()
	cfg.LoadConfig("../../testdata/configtest.json")
	binanceConfig, err := cfg.GetExchangeConfig("Binance")
	if err != nil {
		t.Error("Test Failed - Binance Setup() init error")
	}

	binanceConfig.AuthenticatedAPISupport = true
	binanceConfig.APIKey = testAPIKey
	binanceConfig.APISecret = testAPISecret

	b.Setup(binanceConfig)
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

func TestGetCandleStickData(t *testing.T) {
	t.Parallel()
	_, err := b.GetCandleStickData("BTCUSDT", "1d", 5)
	if err != nil {
		t.Error("Test Failed - Binance GetCandleStickData() error", err)
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
	_, err := b.NewOrder(NewOrderRequest{})
	if err == nil {
		t.Error("Test Failed - Binance NewOrder() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := b.QueryOrder("", "", 1337)
	if err == nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	}
}
