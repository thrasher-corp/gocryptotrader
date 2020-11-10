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
	flag.StringVar(&configPath, "configpath", filepath.Join("C:\\Users\\ScottGrant\\go\\src\\github.com\\thrasher-corp\\gocryptotrader\\backtester", "config", "examples", "dollar-cost-average.strat"), "the config containing strategy params")
	flag.Parse()

	bt, err := backtest.NewFromConfig(configPath)
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
