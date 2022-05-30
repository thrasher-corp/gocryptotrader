package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"plugin"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	backtest "github.com/thrasher-corp/gocryptotrader/backtester/engine"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

var configPath, templatePath, reportOutput, strategyPluginPath string
var printLogo, generateReport, darkReport, verbose, colourOutput, logSubHeader bool

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Could not get working directory. Error: %v.\n", err)
		os.Exit(1)
	}
	parseFlags(wd)
	if !colourOutput {
		common.PurgeColours()
	}
	var bt *backtest.BackTest
	var cfg *config.Config
	log.GlobalLogConfig = log.GenDefaultSettings()
	log.GlobalLogConfig.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(logSubHeader)
	log.GlobalLogConfig.AdvancedSettings.Headers.Info = common.ColourInfo + "[INFO]" + common.ColourDefault
	log.GlobalLogConfig.AdvancedSettings.Headers.Warn = common.ColourWarn + "[WARN]" + common.ColourDefault
	log.GlobalLogConfig.AdvancedSettings.Headers.Debug = common.ColourDebug + "[DEBUG]" + common.ColourDefault
	log.GlobalLogConfig.AdvancedSettings.Headers.Error = common.ColourError + "[ERROR]" + common.ColourDefault
	err = log.SetupGlobalLogger()
	if err != nil {
		fmt.Printf("Could not setup global logger. Error: %v.\n", err)
		os.Exit(1)
	}

	err = common.RegisterBacktesterSubLoggers()
	if err != nil {
		fmt.Printf("Could not register subloggers. Error: %v.\n", err)
		os.Exit(1)
	}

	if strategyPluginPath != "" {
		err = loadCustomStrategy(strategyPluginPath)
	}

	cfg, err = config.ReadConfigFromFile(configPath)
	if err != nil {
		fmt.Printf("Could not read config. Error: %v.\n", err)
		os.Exit(1)
	}
	if printLogo {
		fmt.Println(common.Logo())
	}

	err = cfg.Validate()
	if err != nil {
		fmt.Printf("Could not read config. Error: %v.\n", err)
		os.Exit(1)
	}

	bt, err = backtest.NewFromConfig(cfg, templatePath, reportOutput, verbose)
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
		log.Infof(log.Global, "Captured %v, shutdown requested.\n", interrupt)
		bt.Stop()
	} else {
		bt.Run()
	}

	err = bt.Statistic.CalculateAllResults()
	if err != nil {
		log.Error(log.Global, err)
		os.Exit(1)
	}

	if generateReport {
		bt.Reports.UseDarkMode(darkReport)
		err = bt.Reports.GenerateReport()
		if err != nil {
			log.Error(log.Global, err)
		}
	}
}

func loadCustomStrategy(strategyPluginPath string) error {
	p, err := plugin.Open(strategyPluginPath)
	if err != nil {
		return fmt.Errorf("could not open plugin: %w", err)
	}
	v, err := p.Lookup("GetStrategy")
	if err != nil {
		return fmt.Errorf("could not lookup plugin. Plugin must have function `GetStrategy`. Error: %w", err)
	}
	customStrategy, ok := v.(func() strategies.Handler)
	if !ok {
		return fmt.Errorf("could not cast plugin to strategies.Handler")
	}
	strategies.AddStrategy(customStrategy())
	return nil
}

func parseFlags(wd string) {
	flag.StringVar(
		&configPath,
		"configpath",
		filepath.Join(
			wd,
			"config",
			"examples",
			"ftx-cash-carry.strat"),
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
	flag.BoolVar(
		&darkReport,
		"darkreport",
		false,
		"sets the output report to use a dark theme by default")
	flag.BoolVar(
		&verbose,
		"verbose",
		false,
		"if enabled, will set exchange requests to verbose for debugging purposes")
	flag.BoolVar(
		&colourOutput,
		"colouroutput",
		false,
		"if enabled, will print in colours, if your terminal supports \033[38;5;99m[colours like this]\u001b[0m")
	flag.BoolVar(
		&logSubHeader,
		"logsubheader",
		true,
		"displays logging subheader to track where activity originates")
	flag.StringVar(
		&strategyPluginPath,
		"strategypluginpath",
		"",
		"example path: "+filepath.Join(wd, "plugins", "strategies", "example", "example.so"))
	flag.Parse()
}
