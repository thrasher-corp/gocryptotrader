package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	backtest "github.com/thrasher-corp/gocryptotrader/backtester/engine"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	var strategyConfigPath, templatePath, reportOutput, btConfigDir string
	var generateReport, darkReport, verbose, colourOutput, logSubHeader, singleRun bool
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Could not get working directory. Error: %v.\n", err)
		os.Exit(1)
	}
	flag.BoolVar(
		&singleRun,
		"singlerun",
		false,
		"will execute the strategyconfigpath strategy and exit")
	flag.StringVar(
		&strategyConfigPath,
		"strategyconfigpath",
		filepath.Join(
			wd,
			"config",
			"strategyexamples",
			"dca-api-candles.strat"),
		"the config containing strategy params, only used if --singlerun=true")
	flag.StringVar(
		&btConfigDir,
		"backtesterconfigpath",
		config.DefaultBTConfigDir,
		"the location of the backtester config")
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
	flag.Parse()

	if btConfigDir == "" {
		btConfigDir = config.DefaultBTConfigDir
		log.Infof(log.Global, "blank config received, using default path '%v'", btConfigDir)
	}
	fe := file.Exists(btConfigDir)
	var btCfg *config.BacktesterConfig
	switch {
	case fe:
		btCfg, err = config.ReadBacktesterConfigFromPath(btConfigDir)
		if err != nil {
			fmt.Printf("Could not read config. Error: %v.\n", err)
			os.Exit(1)
		}
	case !fe && btConfigDir == config.DefaultBTConfigDir:
		btCfg, err = config.GenerateDefaultConfig()
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v.\n", err)
			os.Exit(1)
		}
		var btCfgJSON []byte
		btCfgJSON, err = json.MarshalIndent(btCfg, "", " ")
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v.\n", err)
			os.Exit(1)
		}
		err = os.MkdirAll(config.DefaultBTDir, file.DefaultPermissionOctal)
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v.\n", err)
			os.Exit(1)
		}
		err = os.WriteFile(btConfigDir, btCfgJSON, file.DefaultPermissionOctal)
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v.\n", err)
			os.Exit(1)
		}
	default:
		log.Errorf(log.Global, "non-standard config '%v' does not exist. Exiting...", btConfigDir)
		return
	}

	if !btCfg.UseCMDColours && colourOutput {
		btCfg.UseCMDColours = colourOutput
	}
	if !btCfg.Report.GenerateReport && generateReport {
		btCfg.Report.GenerateReport = generateReport
	}
	if btCfg.Report.TemplatePath != templatePath && templatePath != filepath.Join(wd, "report", "tpl.gohtml") {
		btCfg.Report.TemplatePath = templatePath
	}
	if btCfg.Report.OutputPath != reportOutput && reportOutput != filepath.Join(wd, "results") {
		btCfg.Report.OutputPath = reportOutput
	}

	if btCfg.UseCMDColours {
		common.SetColours(&btCfg.Colours)
	} else {
		common.PurgeColours()
	}

	log.GlobalLogConfig = log.GenDefaultSettings()
	log.GlobalLogConfig.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(logSubHeader)
	log.GlobalLogConfig.AdvancedSettings.Headers.Info = common.CMDColours.Info + "[INFO]" + common.CMDColours.Default
	log.GlobalLogConfig.AdvancedSettings.Headers.Warn = common.CMDColours.Warn + "[WARN]" + common.CMDColours.Default
	log.GlobalLogConfig.AdvancedSettings.Headers.Debug = common.CMDColours.Debug + "[DEBUG]" + common.CMDColours.Default
	log.GlobalLogConfig.AdvancedSettings.Headers.Error = common.CMDColours.Error + "[ERROR]" + common.CMDColours.Default
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

	if btCfg.PrintLogo {
		fmt.Println(common.Logo())
	}

	if singleRun {
		dir := strategyConfigPath
		var cfg *config.Config
		cfg, err = config.ReadStrategyConfigFromFile(dir)
		if err != nil {
			fmt.Printf("Could not read strategy config. Error: %v.\n", err)
			os.Exit(1)
		}
		err = backtest.ExecuteStrategy(cfg, &config.BacktesterConfig{
			Report: config.Report{
				GenerateReport: generateReport,
				TemplatePath:   templatePath,
				OutputPath:     reportOutput,
				DarkMode:       darkReport,
			},
		})
		if err != nil {
			fmt.Printf("Could not execute strategy. Error: %v.\n", err)
			os.Exit(1)
		}
		return
	}

	go func(c *config.BacktesterConfig) {
		log.Info(log.GRPCSys, "starting GRPC server")
		s := backtest.SetupRPCServer(c)
		err = backtest.StartRPCServer(s)
		if err != nil {
			fmt.Printf("Could not read config. Error: %v.\n", err)
			os.Exit(1)
		}
		log.Info(log.GRPCSys, "ready to receive commands")
	}(btCfg)
	interrupt := signaler.WaitForInterrupt()
	log.Infof(log.Global, "Captured %v, shutdown requested.\n", interrupt)
	log.Infoln(log.Global, "Exiting.")
}
