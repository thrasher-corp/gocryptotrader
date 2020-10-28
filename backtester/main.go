package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
	"github.com/thrasher-corp/gocryptotrader/backtester/settings"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func main() {
	var s settings.Settings
	flag.StringVar(&s.ExchangeName, "exchangename", "binance", "exchange to test against")
	flag.DurationVar(&s.Interval, "interval", kline.FifteenMin.Duration(), "candle size")
	flag.StringVar(&s.StartTime, "starttime", time.Now().Add(-time.Hour).Format(common.SimpleTimeFormat), "backtest start time")
	flag.StringVar(&s.EndTime, "endtime", time.Now().Format(common.SimpleTimeFormat), "backtest end time")
	flag.Float64Var(&s.InitialFunds, "initialfunds", 1337, "intial funds to trade with")

	bt := backtest.NewFromSettings(&s)
	err := bt.Run()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
