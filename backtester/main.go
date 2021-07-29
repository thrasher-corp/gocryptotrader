package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/backtest"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	var configPath, templatePath, reportOutput string
	var printLogo, generateReport bool
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Could not get working directory. Error: %v.\n", err)
		os.Exit(1)
	}
	flag.StringVar(
		&configPath,
		"configpath",
		filepath.Join(
			wd,
			"config",
			"examples",
			"dca-api-candles.strat"),
		"the config containing strategy params")
	flag.StringVar(
		&templatePath,
		"templatepath",
		filepath.Join(
			wd,
			"report",
			"tpl.gohtml"),
		"the report template to use")
	flag.BoolVar(
		&generateReport,
		"generatereport",
		true,
		"whether to generate the report file")
	flag.StringVar(
		&reportOutput,
		"outputpath",
		filepath.Join(
			wd,
			"results"),
		"the path where to output results")
	flag.BoolVar(
		&printLogo,
		"printlogo",
		true,
		"print out the logo to the command line, projected profits likely won't be affected if disabled")

	flag.Parse()

	var bt *backtest.BackTest
	var cfg *config.Config
	fmt.Println("reading config...")
	cfg, err = config.ReadConfigFromFile(configPath)
	if err != nil {
		fmt.Printf("Could not read config. Error: %v.\n", err)
		os.Exit(1)
	}
	if printLogo {
		fmt.Print(common.ASCIILogo)
	}

	path := gctconfig.DefaultFilePath()
	if cfg.GoCryptoTraderConfigPath != "" {
		path = cfg.GoCryptoTraderConfigPath
	}
	var bot *engine.Engine
	flags := map[string]bool{
		"tickersync":    false,
		"orderbooksync": false,
		"tradesync":     false,
		"ratelimiter":   true,
		"ordermanager":  false,
	}
	bot, err = engine.NewFromSettings(&engine.Settings{
		ConfigFile:                    path,
		EnableDryRun:                  true,
		EnableAllPairs:                true,
		EnableExchangeHTTPRateLimiter: true,
	}, flags)
	if err != nil {
		fmt.Printf("Could not load backtester. Error: %v.\n", err)
		os.Exit(-1)
	}
	bt, err = backtest.NewFromConfig(cfg, templatePath, reportOutput, bot)
	if err != nil {
		fmt.Printf("Could not setup backtester from config. Error: %v.\n", err)
		os.Exit(1)
	}
	if cfg.DataSettings.LiveData != nil {
		go func() {
			err = bt.RunLive()
			if err != nil {
				fmt.Printf("Could not complete live run. Error: %v.\n", err)
				os.Exit(-1)
			}
		}()
		interrupt := signaler.WaitForInterrupt()
		gctlog.Infof(gctlog.Global, "Captured %v, shutdown requested.\n", interrupt)
		bt.Stop()
	} else {
		err = bt.Run()
		if err != nil {
			fmt.Printf("Could not complete run. Error: %v.\n", err)
			os.Exit(1)
		}
	}

	err = bt.Statistic.CalculateAllResults()
	if err != nil {
		gctlog.Error(gctlog.BackTester, err)
		os.Exit(1)
	}

	if generateReport {
		err = bt.Reports.GenerateReport()
		if err != nil {
			gctlog.Error(gctlog.BackTester, err)
		}
	}
}
