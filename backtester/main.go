package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
	"github.com/thrasher-corp/gocryptotrader/backtester/settings"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

func main() {
	var s settings.Settings
	defaultStartDate := time.Date(2017, 8, 17, 0, 0, 0, 0, time.UTC)
	defaultEndDate := defaultStartDate.AddDate(1, 0, 0)
	flag.StringVar(&s.StartTime, "starttime", defaultStartDate.Format(common.SimpleTimeFormat), "backtest start time")
	flag.StringVar(&s.EndTime, "endtime", defaultEndDate.Format(common.SimpleTimeFormat), "backtest end time")
	flag.DurationVar(&s.Interval, "interval", kline.OneDay.Duration(), "candle size")
	flag.Float64Var(&s.InitialFunds, "initialfunds", 133713371337, "intial funds to trade with")
	flag.StringVar(&s.ExchangeName, "exchangename", "Binance", "exchange to test against")
	flag.StringVar(&s.CurrencyPair, "currencypair", "btc-usdt", "currency pair to back test with")
	flag.StringVar(&s.AssetType, "assettype", asset.Spot.String(), "asset type to use eg spot")
	flag.StringVar(&s.RunName, "runname", "backtest"+time.Now().Format(common.SimpleTimeFormat), "a name reference for the resulting backtest run")
	flag.StringVar(&s.StrategyName, "strategy", "rsi420blazeit", "the strategy to use for the backtesting run")

	flag.Parse()

	bt, err := backtest.NewFromSettings(&s)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = bt.Run()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	ret := bt.Statistic.ReturnResults()
	for x := range ret.Transactions {
		if ret.Transactions[x].Amount > 0 {
			fmt.Println(ret.Transactions[x])
		}
	}
	fmt.Printf("Total Events: %v | Total Transactions: %v\n", ret.TotalEvents, ret.TotalTransactions)
}
