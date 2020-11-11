package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
)

func main() {
	var configPath string
	var configSource string

	flag.StringVar(&configSource, "configsource", "file", "where to load a strategy configurations. 'file' or 'database'")
	flag.StringVar(&configPath, "configpath", filepath.Join("C:\\Users\\ScottGrant\\go\\src\\github.com\\thrasher-corp\\gocryptotrader\\backtester", "config", "examples", "dollar-cost-average.strat"), "the config containing strategy params")
	flag.Parse()

	var bt *backtest.BackTest
	var err error
	if configSource == "file" {
		bt, err = backtest.NewFromConfig(configPath)
		if err != nil {
			fmt.Print(err)
			os.Exit(1)
		}
	} else if configSource == "database" {
		// this is where one would check a 'config' database table which just contains
		// data like {{strategyName}} {{jsonContentsOfStrategy}}
	}
	err = bt.Run()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}

	bt.Statistic.PrintResult()
}
