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
	flag.StringVar(&configPath, "configpath", filepath.Join("C:\\Users\\ScottGrant\\go\\src\\github.com\\thrasher-corp\\gocryptotrader\\backtester", "config", "examples", "dollar-cost-average.strat"), "the config containing strategy params")
	flag.Parse()

	var bt *backtest.BackTest
	var err error
	var cfg *config.Config
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
	} else {
		err := bt.Run()
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
	}

	err = bt.Statistic.CalculateTheResults()
	if err != nil {
		gctlog.Error(gctlog.BackTester, err)
	}
}
