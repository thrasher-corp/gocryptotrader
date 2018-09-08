package binance

import (
	"encoding/json"
	"fmt"
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
	res, err := b.GetOrderBook(OrderBookDataRequestParams{
		Symbol: b.GetSymbol(),
		Limit:  10,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetOrderBook() error", err)
	} else {
		fmt.Println("----------Bids-------")
		for _, v := range res.Bids {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}
		fmt.Println("----------Asks-------")
		for _, v := range res.Asks {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}

	}
}

func TestGetRecentTrades(t *testing.T) {
	t.Parallel()

	list, err := b.GetRecentTrades(RecentTradeRequestParams{
		Symbol: b.GetSymbol(),
		Limit:  15,
	})

	if err != nil {
		t.Error("Test Failed - Binance GetRecentTrades() error", err)
	} else {
		for k, v := range list {
			b, _ := json.Marshal(v)
			fmt.Println(k, string(b))
		}

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
	res, err := b.QueryOrder(b.GetSymbol(), "", 1337)
	if err != nil {
		t.Error("Test Failed - Binance QueryOrder() error", err)
	} else {
		//{"code":0,"msg":"","symbol":"BTCUSDT","orderId":131046063,"clientOrderId":"2t38MQXdRe9HvctyRdUbIT","price":"100000","origQty":"0.01","executedQty":"0","status":"NEW","timeInForce":"GTC","type":"LIMIT","side":"SELL","stopPrice":"0","icebergQty":"0","time":1531384312008,"isWorking":true}
		b, _ := json.Marshal(res)
		fmt.Println(string(b))
	}
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	list, err := b.OpenOrders(b.GetSymbol())
	if err != nil {
		t.Error("Test Failed - Binance OpenOrders() error", err)
	} else {
		fmt.Println("----------OpenOrders-------")
		for _, v := range list {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}

	}
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	list, err := b.AllOrders(b.GetSymbol(), "", "")
	if err != nil {
		t.Error("Test Failed - Binance AllOrders() error", err)
	} else {
		fmt.Println("----------AllOrders-------")
		for _, v := range list {
			b, _ := json.Marshal(v)
			fmt.Println(string(b))
		}

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
