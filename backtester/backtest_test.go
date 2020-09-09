package backtest

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/thrasher-corp/gct-ta/indicators"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type TestStrategy struct{}

func (s *TestStrategy) Name() string {
	return "TestStrategy"
}

func (s *TestStrategy) OnSignal(d DataHandler, p PortfolioHandler) (SignalEvent, error) {
	signal := Signal{
		Event: Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}

	smaFast := indicators.SMA(d.StreamClose(), 10)
	smaSlow := indicators.SMA(d.StreamClose(), 30)

	ret := indicators.Crossover(smaFast, smaSlow)
	if ret {
		signal.SetDirection(order.Buy)
	} else {
		signal.SetDirection(order.Sell)
	}

	rsiRet := indicators.RSI(d.StreamClose(), 14)[d.Offset()-1]
	if rsiRet < 30 {
		signal.Amount = 5
		signal.SetDirection(order.Buy)
	} else if rsiRet > 70 {
		signal.Amount = 5
		signal.SetDirection(order.Sell)
	}

	return &signal, nil
}

func TestBackTest(t *testing.T) {
	bt := New()

	data := DataFromKline{
		Item: genOHCLVData(),
	}
	err := data.Load()
	if err != nil {
		t.Fatal(err)
	}

	bt.data = &data
	bt.portfolio = &Portfolio{
		initialFunds: 1000,
		riskManager:  &Risk{},
		sizeManager: &Size{
			DefaultSize:  100,
			DefaultValue: 1000,
		},
	}

	bt.strategy = &TestStrategy{}
	bt.exchange = &Exchange{
		MakerFee: 0.00,
		TakerFee: 0.00,
	}

	statistic := Statistic{
		strategyName: "HelloWorld",
		pair: data.Item.Pair.String(),

	}
	bt.statistic = &statistic
	err = bt.Run()
	if err != nil {
		t.Fatal(err)
	}
	ret := statistic.ReturnResults()
	for x := range ret.Transactions {
		fmt.Println(ret.Transactions[x])
	}
	fmt.Printf("Total Events: %v | Total Transactions: %v\n", ret.TotalEvents, ret.TotalTransactions)
	// r, err := statistic.JSON(false)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	r := statistic.ReturnResults()
	err = GenerateOutput(r)
	if err != nil {
		t.Fatal(err)
	}
	// fmt.Println(string(r))
	// for x := range bt.orderbook.Orders() {
	// 	fmt.Println(bt.orderbook.Orders()[x])
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
