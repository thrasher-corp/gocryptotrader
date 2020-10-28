package main

import (
	"fmt"
	"os"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
)

func main() {
	bt := backtest.New()
	err := bt.Run()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
}
