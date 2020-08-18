package backtest

import (
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type testBT struct{}

func (bt *testBT) Init() *Config {
	return &Config{}
}

func (bt *testBT) OnData(d DataEvent, b *Backtest) (bool, error) {
	return true, nil
}

func (bt *testBT) OnEnd(b *Backtest) {}

func TestBacktest_Run(t *testing.T) {
	bt := &testBT{}
	klineData := &DataFromKlineItem{
		Item: genOHCLVData(),
	}
	klineData.Load()
	err := Run(bt, klineData)
	if err != nil {
		t.Fatal(err)
	}
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
