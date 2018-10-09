package binance

import (
	"testing"

	"github.com/thrasher-/gocryptotrader/currency/symbol"

	"github.com/thrasher-/gocryptotrader/config"
	exchange "github.com/thrasher-/gocryptotrader/exchanges"
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
	_, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: "BTCUSDT",
		Limit:  10,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetOrderBook() error", err)
	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	_, err := b.GetRecentTrades(RecentTradeRequestParams{
		Symbol: "BTCUSDT",
		Limit:  15,
	})

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
		Symbol:   "BTCUSDT",
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

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	_, err := b.NewOrder(NewOrderRequest{
		Symbol:      "BTCUSDT",
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

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.CancelOrder("BTCUSDT", 82584683, "")
	if err == nil {
		t.Error("Test Failed - Binance CancelOrder() error", err)
	}
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.QueryOrder("BTCUSDT", "", 1337)
	if err != nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.OpenOrders("BTCUSDT")
	if err != nil {
		t.Error("Test Failed - Binance OpenOrders() error", err)
	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()

	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}

	_, err := b.AllOrders("BTCUSDT", "", "")
	if err != nil {
		t.Error("Test Failed - Binance AllOrders() error", err)
	}
}

func TestGetAccount(t *testing.T) {
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)
	account, err := b.GetAccount()
	if err != nil {
		t.Fatal("Test Failed - Binance GetAccount() error", err)
	}
	if account.MakerCommission <= 0 {
		t.Fatalf("Test Failed. Expected > 0, Recieved %d", account.MakerCommission)
	}
	if account.TakerCommission <= 0 {
		t.Fatalf("Test Failed. Expected > 0, Recieved %d", account.TakerCommission)
	}

	t.Logf("Current makerFee: %d", account.MakerCommission)
	t.Logf("Current takerFee: %d", account.TakerCommission)
}

func TestGetFee(t *testing.T) {
	if testAPIKey == "" || testAPISecret == "" {
		t.Skip()
	}
	t.Parallel()
	b.SetDefaults()
	TestSetup(t)

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC, 100, 100, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, true, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, true, false); resp != float64(0.100000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.100000), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 1, 1, false, true); resp != float64(0.100000) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0.100000), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyTradeFee, symbol.BTC+symbol.LTC, 10000000000, -1000000000, false, true); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CryptocurrencyWithdrawalFee, symbol.BTC, 1, 5, false, false); resp != float64(0.000500) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(000500), resp)
	}

	if resp, err := b.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 0.001, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.CyptocurrencyDepositFee, symbol.BTC, 1, 555, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankDepositFee, symbol.BTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

	if resp, err := b.GetFee(exchange.InternationalBankWithdrawalFee, symbol.BTC, 1, 1, false, false); resp != float64(0) || err != nil {
		t.Errorf("Test Failed - GetFee() error. Expected: %f, Recieved: %f", float64(0), resp)
	}

}
