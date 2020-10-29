package backtest

import (
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gct-ta/indicators"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	data2 "github.com/thrasher-corp/gocryptotrader/backtester/data"
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/exchange"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type TestStrategy struct{}

func (s *TestStrategy) Name() string {
	return "TestStrategy"
}

func (s *TestStrategy) OnSignal(d portfolio.DataHandler, p portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	signal := event.Signal{
		Event: event.Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}
	log.Printf("STREAM CLOSE at: %v", d.StreamClose())

	rsi := indicators.RSI(d.StreamClose(), 14)
	latestRSI := rsi[len(rsi)-1]
	log.Printf("RSI at: %v", latestRSI)
	if latestRSI <= 30 {
		// oversold, time to buy like a sweet pro
		signal.Direction = order.Buy
	} else if latestRSI >= 70 {
		// overbought, time to sell because granny is talking about BTC again
		signal.Direction = order.Sell
	} else {
		signal.Direction = common.DoNothing
	}

	return &signal, nil
}

func TestBackTest(t *testing.T) {
	bt := New()

	data := data2.DataFromKline{
		Item: genOHCLVData(),
	}
	err := data.Load()
	if err != nil {
		t.Fatal(err)
	}

	bt.Data = &data
	bt.Portfolio = &portfolio2.Portfolio{
		InitialFunds: 1000,
		RiskManager:  &risk.Risk{},
		SizeManager: &size.Size{
			DefaultSize:  100,
			DefaultValue: 1000,
		},
	}

	bt.Strategy = &TestStrategy{}
	bt.Exchange = &exchange.Exchange{
		MakerFee: 0.00,
		TakerFee: 0.00,
	}

	statistic := statistics.Statistic{
		StrategyName: "HelloWorld",
		Pair:         data.Item.Pair.String(),
	}
	bt.Statistic = &statistic
	err = bt.Run()
	if err != nil {
		t.Fatal(err)
	}
	ret := statistic.ReturnResults()
	for x := range ret.Transactions {
		fmt.Println(ret.Transactions[x])
	}
	fmt.Printf("Total Events: %v | Total Transactions: %v\n", ret.TotalEvents, ret.TotalTransactions)
	// r, err := Statistic.JSON(false)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// err = GenerateOutput(r)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Println(string(r))
	// for x := range bt.Orderbook.GetOrders() {
	// 	fmt.Println(bt.Orderbook.GetOrders()[x])
	// }
	bt.Reset()
}

func genOHCLVData() kline.Item {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

	var outItem kline.Item
	outItem.Interval = kline.OneDay
	outItem.Asset = asset.Spot
	outItem.Pair = currency.NewPair(currency.BTC, currency.USDT)
	outItem.Exchange = "test"

	outItem.Candles = make([]kline.Candle, 365)
	outItem.Candles[0] = kline.Candle{
		Time:   start,
		Open:   0,
		High:   10 + rand.Float64(),
		Low:    10 + rand.Float64(),
		Close:  10 + rand.Float64(),
		Volume: 10,
	}

	for x := 1; x < 365; x++ {
		outItem.Candles[x] = kline.Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   outItem.Candles[x-1].Close,
			High:   outItem.Candles[x-1].Open + rand.Float64(),
			Low:    outItem.Candles[x-1].Open - rand.Float64(),
			Close:  outItem.Candles[x-1].Open + rand.Float64(),
			Volume: float64(rand.Int63n(150)),
		}
	}

	return outItem
}
