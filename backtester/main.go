package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
	"github.com/thrasher-corp/gocryptotrader/backtester/settings"
)

func main() {
	var s settings.Settings
	/*
		defaultStartDate := time.Date(2017, 8, 17, 0, 0, 0, 0, time.UTC)
		defaultEndDate := defaultStartDate.AddDate(1, 0, 0)
		flag.StringVar(&s.StartTime, "starttime", defaultStartDate.Format(common.SimpleTimeFormat), "backtest start time")
		flag.StringVar(&s.EndTime, "endtime", defaultEndDate.Format(common.SimpleTimeFormat), "backtest end time")
		flag.DurationVar(&s.Interval, "interval", kline.OneDay.Duration(), "candle size")
		flag.Float64Var(&s.InitialFunds, "initialfunds", 2000, "intial funds to trade with")
		flag.Float64Var(&s.MaximumOrderSize, "ordersize", 1, "maximum order size")
		flag.StringVar(&s.ExchangeName, "exchangename", "Binance", "exchange to test against")
		flag.StringVar(&s.CurrencyPair, "currencypair", "btc-usdt", "currency pair to back test with")
		flag.StringVar(&s.AssetType, "assettype", asset.Spot.String(), "asset type to use eg spot")
		flag.StringVar(&s.RunName, "runname", "backtest"+time.Now().Format(common.SimpleTimeFormat), "a name reference for the resulting backtest run")
		flag.StringVar(&s.StrategyName, "strategy", "buyandhold", "the strategy to use for the backtesting run")
	*/
	flag.StringVar(&s.ConfigPath, "configpath", filepath.Join(".", "buy-and-hold.strat"), "the config containing strategy params")

	flag.Parse()

	bt, err := backtest.NewFromConfig(s.ConfigPath)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	err = bt.Run()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	bt.Statistic.PrintResult()
}
