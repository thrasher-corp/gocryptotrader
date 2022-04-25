package binanceus

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

// Please supply your own keys here to do authenticated endpoint testing
const (
	apiKey                  = ""
	apiSecret               = ""
	canManipulateRealOrders = false
)

var bi Binanceus

func TestMain(m *testing.M) {
	bi.SetDefaults()
	bi.validLimits = []int{5, 10, 20, 50, 100, 500, 1000}
	cfg := config.GetConfig()
	err := cfg.LoadConfig("../../testdata/configtest.json", true)
	if err != nil {
		log.Fatal(err)
	}

	exchCfg, err := cfg.GetExchangeConfig("Binanceus")
	if err != nil {
		log.Fatal(err)
	}

	exchCfg.API.AuthenticatedSupport = true
	exchCfg.API.AuthenticatedWebsocketSupport = true
	exchCfg.API.Credentials.Key = apiKey
	exchCfg.API.Credentials.Secret = apiSecret

	err = bi.Setup(exchCfg)
	if err != nil {
		log.Fatal(err)
	}

	os.Exit(m.Run())
}

// Ensures that this exchange package is compatible with IBotExchange
func TestInterface(t *testing.T) {
	var e exchange.IBotExchange
	if e = new(Binanceus); e == nil {
		t.Fatal("unable to allocate exchange")
	}
}

func areTestAPIKeysSet() bool {
	return bi.ValidateAPICredentials(bi.GetDefaultCredentials()) == nil
}

// Implement tests for API endpoints below

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := bi.GetExchangeInfo(context.Background())
	if err != nil {
		println("DERR: ", err.Error())
		t.Error(err)
	}
	// println(info)
	// if mockTests {
	// 	serverTime := time.Date(2022, 2, 25, 3, 50, 40, int(601*time.Millisecond), time.UTC)
	// 	if !info.Servertime.Equal(serverTime) {
	// 		t.Errorf("Expected %v, got %v", serverTime, info.Servertime)
	// 	}
	// }
}

// TestGetMostRecentTrades -- test most recent trades end-point
func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetMostRecentTrades(context.Background(), RecentTradeRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  15,
	})
	if err != nil {
		t.Error("Binanceus GetMostRecentTrades() error", err)
	}
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := bi.GetHistoricalTrades(context.Background(), HistoricalTradeParams{
		Symbol: "BTCUSDT",
		Limit:  5,
		FromID: 0,
	})
	if err != nil {
		t.Errorf("Binanceus GetHistoricalTrades() error: %v", err)
	}
}

func TestGetAggregateTrades(t *testing.T) {
	t.Parallel()
	// _, err := bi.GetAggregateTrades(context.Background(),
	// 	&AggregatedTradeRequestParams{
	// 		Symbol: currency.NewPair(currency.BTC, currency.USDT),
	// 		Limit:  1001,
	// 	})
	// if err != nil {
	// 	t.Error("Binanceus GetAggregateTrades() error", err)
	// }
	_, err := bi.GetAggregateTrades(context.Background(),
		&AggregatedTradeRequestParams{
			Symbol: currency.NewPair(currency.BTC, currency.USDT),
			Limit:  5,
		})
	if err != nil {
		t.Error("Binanceus GetAggregateTrades() error", err)
	}
	// _, err = bi.GetAggregateTrades(context.Background(),
	// 	&AggregatedTradeRequestParams{
	// 		Symbol:  currency.NewPair(currency.BTC, currency.USDT),
	// 		Limit:   5,
	// 		EndTime: uint64(time.Now().UnixMilli()),
	// 	})
	// if err != nil {
	// 	t.Error("Binanceus GetAggregateTrades() error", err)
	// }
}

func TestGetOrderBookDepth(t *testing.T) {
	t.Parallel()
	_, er := bi.GetOrderBookDepth(context.TODO(), &OrderBookDataRequestParams{
		Symbol: currency.NewPair(currency.BTC, currency.USDT),
		Limit:  1000,
	})
	if er != nil {
		t.Error("Binanceus GetOrderBook() error", er)
	}
}

func TestGetCandlestickData(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSpotKline(context.Background(), &KlinesRequestParams{
		Symbol:    currency.NewPair(currency.BTC, currency.USDT),
		Interval:  kline.FiveMin.Short(),
		Limit:     24,
		StartTime: time.Unix(1577836800, 0),
		EndTime:   time.Unix(1580515200, 0),
	})
	if er != nil {
		t.Error("Binanceus GetSpotKline() error", er)
	}
}

func TestGetPriceDatas(t *testing.T) {
	t.Parallel()
	_, er := bi.GetPriceDatas(context.TODO())
	if er != nil {
		t.Error("Binanceus GetPriceDatas() error", er)
	}
}

func TestGetSinglePriceData(t *testing.T) {
	t.Parallel()
	_, er := bi.GetSinglePriceData(context.Background(), currency.Pair{
		Base:  currency.BTC,
		Quote: currency.USDT,
	})
	if er != nil {
		t.Error("Binanceus GetSinglePriceData() error", er)
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()

	_, err := bi.GetAveragePrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetAveragePrice() error", err)
	}
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()

	_, err := bi.GetBestPrice(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binanceus GetBestPrice() error", err)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()

	_, err := bi.GetPriceChangeStats(context.Background(), currency.NewPair(currency.BTC, currency.USDT))
	if err != nil {
		t.Error("Binance GetPriceChangeStats() error", err)
	}
}

func TestGetTickers(t *testing.T) {
	t.Parallel()

	_, err := bi.GetTickers(context.Background())
	if err != nil {
		t.Error("Binance TestGetTickers error", err)
	}
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	_, er := bi.GetAccount(context.Background())
	if er != nil {
		t.Error("Binanceus GetAccount() error", er)
	}
}

func TestGetUserAccountStatus(t *testing.T) {
	t.Parallel()
	res, er := bi.GetUserAccountStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAccountStatus() error", er)
	}
	val, _ := json.Marshal(res)
	println("\n", string(val))
}

func TestGetUserAPITradingStatus(t *testing.T) {
	t.Parallel()
	_, er := bi.GetUserAPITradingStatus(context.Background(), 3000)
	if er != nil {
		t.Error("Binanceus GetUserAPITradingStatus() error", er)
	}
}
