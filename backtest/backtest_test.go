package backtest

import (
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtest/data"
	"github.com/thrasher-corp/gocryptotrader/backtest/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtest/strategy"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

func TestBacktest_Run(t *testing.T) {
	pair := currency.NewPair(currency.BTC, currency.USDT)
	pt := portfolio.Portfolio{}
	dt := data.Data{}
	st := strategy.Strategy{}
	bt := New(pair, dt, pt, st)
	err := bt.Run()
	if err != nil {
		t.Fatal(err)
	}
}
