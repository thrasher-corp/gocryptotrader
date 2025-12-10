package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	backtest "github.com/thrasher-corp/gocryptotrader/backtester/engine"
	"github.com/thrasher-corp/gocryptotrader/backtester/plugins/strategies"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

var (
	singleTaskStrategyPath, templatePath, outputPath, btConfigDir, strategyPluginPath, pprofURL string
	printLogo, generateReport, darkReport, colourOutput, logSubHeader, enablePProf              bool
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Could not get working directory. Error: %v\n", err)
		os.Exit(1)
	}

	flags := parseFlags(wd)
	var btCfg *config.BacktesterConfig
	if btConfigDir == "" {
		btConfigDir = config.DefaultBTConfigDir
		log.Infof(log.Global, "Blank config received, using default path '%v'", btConfigDir)
	}

	if enablePProf {
		go func() {
			server := &http.Server{
				Addr:              pprofURL,
				ReadHeaderTimeout: time.Minute,
				ReadTimeout:       time.Minute,
			}

			fmt.Println(server.ListenAndServe())
		}()
	}

	fe := file.Exists(btConfigDir)
	switch {
	case fe:
		btCfg, err = config.ReadBacktesterConfigFromPath(btConfigDir)
		if err != nil {
			fmt.Printf("Could not read config. Error: %v\n", err)
			os.Exit(1)
		}
	case !fe && btConfigDir == config.DefaultBTConfigDir:
		btCfg, err = config.GenerateDefaultConfig()
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v\n", err)
			os.Exit(1)
		}
		var btCfgJSON []byte
		btCfgJSON, err = json.MarshalIndent(btCfg, "", " ")
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v\n", err)
			os.Exit(1)
		}
		err = os.MkdirAll(config.DefaultBTDir, file.DefaultPermissionOctal)
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v\n", err)
			os.Exit(1)
		}
		err = os.WriteFile(btConfigDir, btCfgJSON, file.DefaultPermissionOctal)
		if err != nil {
			fmt.Printf("Could not generate config. Error: %v\n", err)
			os.Exit(1)
		}
	default:
		log.Errorf(log.Global, "Non-standard config '%v' does not exist. Exiting...", btConfigDir)
		return
	}

	flagSet := engine.FlagSet(flags)
	flagSet.WithBool("printlogo", &printLogo, btCfg.PrintLogo)
	flagSet.WithBool("darkreport", &darkReport, btCfg.Report.DarkMode)
	flagSet.WithBool("generatereport", &generateReport, btCfg.Report.GenerateReport)
	flagSet.WithBool("logsubheaders", &logSubHeader, btCfg.LogSubheaders)
	flagSet.WithBool("colouroutput", &colourOutput, btCfg.UseCMDColours)

	if singleTaskStrategyPath != "" && !file.Exists(singleTaskStrategyPath) {
		fmt.Printf("Strategy config path not found '%v'", singleTaskStrategyPath)
		os.Exit(1)
	}

	defaultTemplate := filepath.Join(
		wd,
		"report",
		"tpl.gohtml")
	defaultReportOutput := filepath.Join(
		wd,
		"results")

	if templatePath != defaultTemplate {
		btCfg.Report.TemplatePath = templatePath
	}
	if !file.Exists(btCfg.Report.TemplatePath) {
		fmt.Printf("Report template path not found '%v'", btCfg.Report.TemplatePath)
		os.Exit(1)
	}

	if outputPath != defaultReportOutput {
		btCfg.Report.OutputPath = outputPath
	}
	if !file.Exists(btCfg.Report.OutputPath) {
		fmt.Printf("Report output path not found '%v'", btCfg.Report.OutputPath)
		os.Exit(1)
	}

	if colourOutput {
		common.SetColours(&btCfg.Colours)
	} else {
		common.PurgeColours()
	}

	defaultLogSettings := log.GenDefaultSettings()
	defaultLogSettings.AdvancedSettings.ShowLogSystemName = convert.BoolPtr(logSubHeader)
	defaultLogSettings.AdvancedSettings.Headers.Info = common.CMDColours.Info + "[INFO]" + common.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Warn = common.CMDColours.Warn + "[WARN]" + common.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Debug = common.CMDColours.Debug + "[DEBUG]" + common.CMDColours.Default
	defaultLogSettings.AdvancedSettings.Headers.Error = common.CMDColours.Error + "[ERROR]" + common.CMDColours.Default
	err = log.SetGlobalLogConfig(defaultLogSettings)
	if err != nil {
		fmt.Printf("Could not setup global logger. Error: %v\n", err)
		os.Exit(1)
	}

	err = log.SetupGlobalLogger("gct/backtester", false)
	if err != nil {
		fmt.Printf("Could not setup global logger. Error: %v\n", err)
		os.Exit(1)
	}

	err = common.RegisterBacktesterSubLoggers()
	if err != nil {
		fmt.Printf("Could not register subloggers. Error: %v\n", err)
		os.Exit(1)
	}

	if printLogo {
		fmt.Println(common.Logo())
	}

	if strategyPluginPath == "" && btCfg.PluginPath != "" {
		strategyPluginPath = btCfg.PluginPath
	}
	if strategyPluginPath != "" {
		err = strategies.LoadCustomStrategies(strategyPluginPath)
		if err != nil {
			fmt.Printf("Could not load custom strategies. Error: %v\n", err)
			os.Exit(1)
		}
		log.Infof(common.Backtester, "Loaded plugin %v\n", strategyPluginPath)
	}

	if singleTaskStrategyPath != "" {
		dir := singleTaskStrategyPath
		var cfg *config.Config
		cfg, err = config.ReadStrategyConfigFromFile(dir)
		if err != nil {
			fmt.Printf("Could not read strategy config. Error: %v\n", err)
			os.Exit(1)
		}
		var bt *backtest.BackTest
		bt, err = backtest.NewBacktesterFromConfigs(cfg, &config.BacktesterConfig{
			Report: config.Report{
				GenerateReport: generateReport,
				TemplatePath:   btCfg.Report.TemplatePath,
				OutputPath:     btCfg.Report.OutputPath,
				DarkMode:       darkReport,
			},
		})
		if err != nil {
			fmt.Printf("Could not execute strategy. Error: %v\n", err)
			os.Exit(1)
		}
		if bt.MetaData.LiveTesting {
			err = bt.ExecuteStrategy(false)
			if err != nil {
				fmt.Printf("Could not stop task %v %v. Error: %v\n", bt.MetaData.ID, bt.MetaData.Strategy, err)
				os.Exit(1)
			}
			interrupt := <-signaler.WaitForInterrupt()
			log.Infof(log.Global, "Captured %v, shutdown requested\n", interrupt)
			log.Infoln(log.Global, "Exiting.")
			err = bt.Stop()
			if err != nil {
				fmt.Printf("Could not stop task %v %v. Error: %v\n", bt.MetaData.ID, bt.MetaData.Strategy, err)
				os.Exit(1)
			}
		} else {
			err = bt.ExecuteStrategy(true)
			if err != nil {
				fmt.Printf("Could not stop task %v %v. Error: %v\n", bt.MetaData.ID, bt.MetaData.Strategy, err)
				os.Exit(1)
			}
		}
		return
	}

	// grpc server mode
	btCfg.Report.DarkMode = darkReport
	btCfg.Report.GenerateReport = generateReport

	runManager := backtest.NewTaskManager()

	go func(c *config.BacktesterConfig) {
		log.Infoln(log.GRPCSys, "Starting RPC server")
		var s *backtest.GRPCServer
		s, err = backtest.SetupRPCServer(c, runManager)
		err = backtest.StartRPCServer(s)
		if err != nil {
			fmt.Printf("Could not start RPC server. Error: %v\n", err)
			os.Exit(1)
		}
		log.Infoln(log.GRPCSys, "Ready to receive commands")
	}(btCfg)
	interrupt := <-signaler.WaitForInterrupt()
	log.Infof(log.Global, "Captured %v, shutdown requested\n", interrupt)
	if btCfg.StopAllTasksOnClose {
		log.Infoln(log.Global, "Stopping all running tasks on close")
		var stopped []*backtest.TaskSummary
		stopped, err = runManager.StopAllTasks()
		if err != nil {
			log.Errorln(common.Backtester, err)
		}
		for i := range stopped {
			log.Infof(common.Backtester, "Task %v %v was stopped", stopped[i].MetaData.ID, stopped[i].MetaData.Strategy)
		}
	} else {
		var tasks []*backtest.TaskSummary
		tasks, err = runManager.List()
		if err != nil {
			log.Errorln(common.Backtester, err)
		}
		for i := range tasks {
			if tasks[i].MetaData.ClosePositionsOnStop && !tasks[i].MetaData.Closed {
				err = runManager.StopTask(tasks[i].MetaData.ID)
				if err != nil {
					log.Errorln(common.Backtester, err)
					continue
				}
				log.Infof(common.Backtester, "Task %v %v was stopped", tasks[i].MetaData.ID, tasks[i].MetaData.Strategy)
			}
		}
	}
	log.Infoln(log.Global, "Exiting. Have a nice day")
	err = log.CloseLogger()
	if err != nil {
		fmt.Println(err)
	}
}

func parseFlags(wd string) map[string]bool {
	defaultStrategy := filepath.Join(
		wd,
		"config",
		"strategyexamples",
		"dca-api-candles.strat")
	defaultTemplate := filepath.Join(
		wd,
		"report",
		"tpl.gohtml")

	defaultReportOutput := filepath.Join(
		wd,
		"results")
	flag.StringVar(
		&singleTaskStrategyPath,
		"singlerunstrategypath",
		"",
		fmt.Sprintf("path to a strategy file. Will execute strategy and exit, instead of creating a GRPC server. Example %v", defaultStrategy))
	flag.StringVar(
		&btConfigDir,
		"backtesterconfigpath",
		config.DefaultBTConfigDir,
		"the location of the backtester config")
	flag.StringVar(
		&templatePath,
		"templatepath",
		defaultTemplate,
		"the report template to use")
	flag.BoolVar(
		&generateReport,
		"generatereport",
		true,
		"whether to generate the report file")
	flag.StringVar(
		&outputPath,
		"outputpath",
		defaultReportOutput,
		"the path where to output results")
	flag.BoolVar(
		&darkReport,
		"darkreport",
		false,
		"sets the output report to use a dark theme by default")
	flag.BoolVar(
		&colourOutput,
		"colouroutput",
		true,
		"if enabled, will print in colours, if your terminal supports \033[38;5;99m[colours like this]\u001b[0m")
	flag.BoolVar(
		&logSubHeader,
		"logsubheader",
		true,
		"displays logging subheader to track where activity originates")
	flag.BoolVar(
		&printLogo,
		"printlogo",
		true,
		"shows the stunning, profit inducing logo on startup")
	flag.StringVar(
		&strategyPluginPath,
		"strategypluginpath",
		"",
		"example path: "+filepath.Join(wd, "plugins", "strategies", "example", "example.so"))
	flag.BoolVar(
		&enablePProf,
		"enablepprof",
		false,
		"if enabled, runs a pprof server for debugging")
	flag.StringVar(
		&pprofURL,
		"pprofurl",
		"http://localhost:6060",
		"")
	flag.Parse()
	// collect flags
	flags := make(map[string]bool)
	// Stores the set flags
	flag.Visit(func(f *flag.Flag) { flags[f.Name] = true })
	return flags
}
