package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	var configPath string
	var configSource string

	flag.StringVar(&configSource, "configsource", "file", "where to load a strategy configurations. 'file' or 'database'")
	flag.StringVar(&configPath, "configpath", filepath.Join("C:\\Users\\ScottGrant\\go\\src\\github.com\\thrasher-corp\\gocryptotrader\\backtester", "config", "examples", "dollar-cost-average.strat"), "the config containing strategy params")
	flag.Parse()

	var bt *backtest.BackTest
	var err error
	var cfg *config.Config
	if configSource == "file" {
		cfg, err = config.ReadConfigFromFile(configPath)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
		bt, err = backtest.NewFromConfig(cfg)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
	} else if configSource == "database" {
		// this is where one would check a 'config' database table which just contains
		// data like {{strategyName}} {{jsonContentsOfStrategy}}
	}

	if cfg.LiveData != nil {
		go func() {
			err = bt.RunLive()
			if err != nil {
				fmt.Print(err)
				os.Exit(-1)
			}
		}()
		interrupt := signaler.WaitForInterrupt()
		gctlog.Infof(gctlog.Global, "Captured %v, shutdown requested.\n", interrupt)
		bt.Stop()
		bt.Statistic.PrintResult()
	} else {
		err := bt.Run()
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
	}

	bt.Statistic.PrintResult()
}
