package backtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type testBT struct{}

func (bt *testBT) Init() *Config {
	fmt.Println("Init()")

	return &Config{
		Item: genOHCLVData(),
	}
}

func (bt *testBT) OnData(d DataEvent, b *Backtest) (bool, error) {
	fmt.Println(d.Candle())
	if d.Price() == 1000 {
		b.Portfolio.Order(900, 1)
		fmt.Println(b.Portfolio.Position())
	}
	return true, nil
}

func (bt *testBT) OnEnd(b *Backtest) {
	fmt.Println("OnEnd()")
	fmt.Printf("%+v\n", b.Stats.PrintResult())
}

func TestBacktest_Run(t *testing.T) {
	g := &testBT{}
	Run(g)
}

func genOHCLVData() (outItem kline.Item) {
	start := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)

	outItem.Interval = kline.OneDay
	outItem.Asset = asset.Spot
	outItem.Pair = currency.NewPair(currency.BTC, currency.USDT)
	outItem.Exchange = "test"

	for x := 0; x < 365; x++ {
		outItem.Candles = append(outItem.Candles, kline.Candle{
			Time:   start.Add(time.Hour * 24 * time.Duration(x)),
			Open:   1000,
			High:   1000,
			Low:    1000,
			Close:  1000,
			Volume: 1000,
		})
	}

	return outItem
}
