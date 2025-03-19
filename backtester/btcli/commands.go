package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	doNotRunFlag = &cli.BoolFlag{
		Name:    "donotrunimmediately",
		Aliases: []string{"dnr"},
		Usage:   "if true, will load the strategy, but will not execute until another command is sent",
	}
	doNotStoreFlag = &cli.BoolFlag{
		Name:    "donotstore",
		Aliases: []string{"dns"},
		Usage:   "if true, will not store the run internally - cannot be run in conjunction with dnr",
	}
)

var executeStrategyFromFileCommand = &cli.Command{
	Name:      "executestrategyfromfile",
	Usage:     "runs the strategy from a config file",
	ArgsUsage: "<path>",
	Action:    executeStrategyFromFile,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "path",
			Aliases: []string{"p"},
			Usage:   "the filepath to a strategy to execute",
		},
		doNotRunFlag,
		doNotStoreFlag,
		&cli.StringFlag{
			Name:    "starttimeoverride",
			Aliases: []string{"s"},
			Usage:   fmt.Sprintf("override the strategy file's start time using your local time. eg '%v'", time.Now().Truncate(time.Hour).AddDate(0, -1, 0).Format(time.DateTime)),
		},
		&cli.StringFlag{
			Name:    "endtimeoverride",
			Aliases: []string{"e"},
			Usage:   fmt.Sprintf("override the strategy file's end time using your local time. eg '%v'", time.Now().Truncate(time.Hour).Format(time.DateTime)),
		},
		&cli.DurationFlag{
			Name:    "intervaloverride",
			Aliases: []string{"i"},
			Usage:   "override the strategy file's candle interval in the format of a time duration. eg '1m' for 1 minute",
		},
	},
}

func executeStrategyFromFile(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	var path string
	if c.IsSet("path") {
		path = c.String("path")
	} else {
		path = c.Args().First()
	}

	var dnr bool
	if c.IsSet("donotrunimmediately") {
		dnr = c.Bool("donotrunimmediately")
	}
	var dns bool
	if c.IsSet("donotstore") {
		dns = c.Bool("donotstore")
	}

	var startTimeOverride string
	if c.IsSet("starttimeoverride") {
		startTimeOverride = c.String("starttimeoverride")
	} else {
		startTimeOverride = c.Args().Get(3)
	}

	var endTimeOverride string
	if c.IsSet("endtimeoverride") {
		endTimeOverride = c.String("endtimeoverride")
	} else {
		endTimeOverride = c.Args().Get(4)
	}

	var s, e time.Time
	if startTimeOverride != "" {
		s, err = time.ParseInLocation(time.DateTime, startTimeOverride, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format for start: %v", err)
		}
	}
	if endTimeOverride != "" {
		e, err = time.ParseInLocation(time.DateTime, endTimeOverride, time.Local)
		if err != nil {
			return fmt.Errorf("invalid time format for end: %v", err)
		}
	}
	if !s.IsZero() && !e.IsZero() {
		err = common.StartEndTimeCheck(s, e)
		if err != nil {
			return err
		}
	}

	var intervalOverride time.Duration
	if c.IsSet("intervaloverride") {
		intervalOverride = c.Duration("intervaloverride")
	} else if c.Args().Get(5) != "" {
		intervalOverride, err = time.ParseDuration(c.Args().Get(5))
		if err != nil {
			return err
		}
	}
	if intervalOverride < 0 {
		return errors.New("interval override duration cannot be less than 0")
	}

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.ExecuteStrategyFromFile(
		c.Context,
		&btrpc.ExecuteStrategyFromFileRequest{
			StrategyFilePath:    path,
			DoNotRunImmediately: dnr,
			DoNotStore:          dns,
			StartTimeOverride:   timestamppb.New(s),
			EndTimeOverride:     timestamppb.New(e),
			IntervalOverride:    durationpb.New(intervalOverride),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var listAllTasksCommand = &cli.Command{
	Name:   "listalltasks",
	Usage:  "returns a list of all loaded strategy tasks",
	Action: listAllTasks,
}

func listAllTasks(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.ListAllTasks(
		c.Context,
		&btrpc.ListAllTasksRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var startTaskCommand = &cli.Command{
	Name:      "starttask",
	Usage:     "executes a strategy task loaded into the server",
	ArgsUsage: "<id>",
	Action:    startTask,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: "the id of the strategy task",
		},
	},
}

func startTask(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	} else {
		id = c.Args().First()
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.StartTask(
		c.Context,
		&btrpc.StartTaskRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var startAllTasksCommand = &cli.Command{
	Name:   "startalltasks",
	Usage:  "executes all strategies loaded into the server that have not been run",
	Action: startAllTasks,
}

func startAllTasks(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.StartAllTasks(
		c.Context,
		&btrpc.StartAllTasksRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var stopTaskCommand = &cli.Command{
	Name:      "stoptask",
	Usage:     "stops a strategy loaded into the server",
	ArgsUsage: "<id>",
	Action:    stopTask,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: "the id of the strategy task",
		},
	},
}

func stopTask(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	} else {
		id = c.Args().First()
	}

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.StopTask(
		c.Context,
		&btrpc.StopTaskRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var stopAllTasksCommand = &cli.Command{
	Name:   "stopalltasks",
	Usage:  "stops all strategies loaded into the server",
	Action: stopAllTasks,
}

func stopAllTasks(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.StopAllTasks(
		c.Context,
		&btrpc.StopAllTasksRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var clearTaskCommand = &cli.Command{
	Name:      "cleartask",
	Usage:     "clears/deletes a strategy loaded into the server - if it is not running",
	ArgsUsage: "<id>",
	Action:    clearTask,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "id",
			Usage: "the id of the strategy task",
		},
	},
}

func clearTask(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	var id string
	if c.IsSet("id") {
		id = c.String("id")
	} else {
		id = c.Args().First()
	}

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.ClearTask(
		c.Context,
		&btrpc.ClearTaskRequest{
			Id: id,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var clearAllTasksCommand = &cli.Command{
	Name:   "clearalltasks",
	Usage:  "clears all strategies loaded into the server. Only tasks not actively running will be cleared",
	Action: clearAllTasks,
}

func clearAllTasks(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.ClearAllTasks(
		c.Context,
		&btrpc.ClearAllTasksRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var executeStrategyFromConfigCommand = &cli.Command{
	Name:        "executestrategyfromconfig",
	Usage:       fmt.Sprintf("runs the default strategy config but via passing in as a struct instead of a filepath - this is a proof-of-concept implementation using %v", filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat")),
	Description: "the cli is not a good place to manage this type of command with n variables to pass in from a command line",
	Action:      executeStrategyFromConfig,
	Flags: []cli.Flag{
		doNotRunFlag,
		doNotStoreFlag,
	},
}

// executeStrategyFromConfig this is a proof of concept command
// it demonstrates that a user can send complex strategies via GRPC
// and have them execute. The ultimate goal is to allow a user to continuously
// tweak values and send them via GRPC and determine the best returns then test them across
// large ranges of data to avoid over fitting
func executeStrategyFromConfig(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	defaultPath := filepath.Join(
		"..",
		"config",
		"strategyexamples",
		"dca-api-candles.strat")
	defaultConfig, err := config.ReadStrategyConfigFromFile(defaultPath)
	if err != nil {
		return err
	}
	customSettings := make([]*btrpc.CustomSettings, len(defaultConfig.StrategySettings.CustomSettings))
	x := 0
	for k, v := range defaultConfig.StrategySettings.CustomSettings {
		customSettings[x] = &btrpc.CustomSettings{
			KeyField: k,
			KeyValue: fmt.Sprintf("%v", v),
		}
		x++
	}

	currencySettings := make([]*btrpc.CurrencySettings, len(defaultConfig.CurrencySettings))
	for i := range defaultConfig.CurrencySettings {
		var sd btrpc.SpotDetails
		if defaultConfig.CurrencySettings[i].SpotDetails != nil {
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
				sd.InitialBaseFunds = defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds.String()
			}
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
				sd.InitialQuoteFunds = defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds.String()
			}
		}
		var fd btrpc.FuturesDetails
		if defaultConfig.CurrencySettings[i].FuturesDetails != nil {
			fd.Leverage = &btrpc.Leverage{
				CanUseLeverage:                 defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio.String(),
				MaximumLeverageRate:            defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate.String(),
				MaximumCollateralLeverageRate:  defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumCollateralLeverageRate.String(),
			}
		}
		currencySettings[i] = &btrpc.CurrencySettings{
			ExchangeName: defaultConfig.CurrencySettings[i].ExchangeName,
			Asset:        defaultConfig.CurrencySettings[i].Asset.String(),
			Base:         defaultConfig.CurrencySettings[i].Base.String(),
			Quote:        defaultConfig.CurrencySettings[i].Quote.String(),
			BuySide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.CurrencySettings[i].BuySide.MinimumSize.String(),
				MaximumSize:  defaultConfig.CurrencySettings[i].BuySide.MaximumSize.String(),
				MaximumTotal: defaultConfig.CurrencySettings[i].BuySide.MaximumTotal.String(),
			},
			SellSide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.CurrencySettings[i].SellSide.MinimumSize.String(),
				MaximumSize:  defaultConfig.CurrencySettings[i].SellSide.MaximumSize.String(),
				MaximumTotal: defaultConfig.CurrencySettings[i].SellSide.MaximumTotal.String(),
			},
			MinSlippagePercent:        defaultConfig.CurrencySettings[i].MinimumSlippagePercent.String(),
			MaxSlippagePercent:        defaultConfig.CurrencySettings[i].MaximumSlippagePercent.String(),
			MakerFeeOverride:          defaultConfig.CurrencySettings[i].MakerFee.String(),
			TakerFeeOverride:          defaultConfig.CurrencySettings[i].TakerFee.String(),
			MaximumHoldingsRatio:      defaultConfig.CurrencySettings[i].MaximumHoldingsRatio.String(),
			SkipCandleVolumeFitting:   defaultConfig.CurrencySettings[i].SkipCandleVolumeFitting,
			UseExchangeOrderLimits:    defaultConfig.CurrencySettings[i].CanUseExchangeLimits,
			UseExchangePnlCalculation: defaultConfig.CurrencySettings[i].UseExchangePNLCalculation,
		}
		if sd.InitialQuoteFunds != "" || sd.InitialBaseFunds != "" {
			currencySettings[i].SpotDetails = &sd
		}
		if fd.Leverage != nil {
			currencySettings[i].FuturesDetails = &fd
		}
	}

	exchangeLevelFunding := make([]*btrpc.ExchangeLevelFunding, len(defaultConfig.FundingSettings.ExchangeLevelFunding))
	for i := range defaultConfig.FundingSettings.ExchangeLevelFunding {
		exchangeLevelFunding[i] = &btrpc.ExchangeLevelFunding{
			ExchangeName: defaultConfig.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
			Asset:        defaultConfig.FundingSettings.ExchangeLevelFunding[i].Asset.String(),
			Currency:     defaultConfig.FundingSettings.ExchangeLevelFunding[i].Currency.String(),
			InitialFunds: defaultConfig.FundingSettings.ExchangeLevelFunding[i].InitialFunds.String(),
			TransferFee:  defaultConfig.FundingSettings.ExchangeLevelFunding[i].TransferFee.String(),
		}
	}

	dataSettings := &btrpc.DataSettings{
		Interval: durationpb.New(defaultConfig.DataSettings.Interval.Duration()),
		Datatype: defaultConfig.DataSettings.DataType,
	}
	if defaultConfig.DataSettings.APIData != nil {
		dataSettings.ApiData = &btrpc.ApiData{
			StartDate:        timestamppb.New(defaultConfig.DataSettings.APIData.StartDate),
			EndDate:          timestamppb.New(defaultConfig.DataSettings.APIData.EndDate),
			InclusiveEndDate: defaultConfig.DataSettings.APIData.InclusiveEndDate,
		}
	}
	if defaultConfig.DataSettings.LiveData != nil {
		creds := make([]*btrpc.Credentials, len(defaultConfig.DataSettings.LiveData.ExchangeCredentials))
		for i := range defaultConfig.DataSettings.LiveData.ExchangeCredentials {
			creds[i] = &btrpc.Credentials{
				Exchange: defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Exchange,
				Keys: &btrpc.ExchangeCredentials{
					Key:             defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.Key,
					Secret:          defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.Secret,
					ClientId:        defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.ClientID,
					PemKey:          defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.PEMKey,
					SubAccount:      defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.SubAccount,
					OneTimePassword: defaultConfig.DataSettings.LiveData.ExchangeCredentials[i].Keys.OneTimePassword,
				},
			}
		}
		dataSettings.LiveData = &btrpc.LiveData{
			NewEventTimeout:           defaultConfig.DataSettings.LiveData.NewEventTimeout.Nanoseconds(),
			DataCheckTimer:            defaultConfig.DataSettings.LiveData.DataCheckTimer.Nanoseconds(),
			RealOrders:                defaultConfig.DataSettings.LiveData.RealOrders,
			ClosePositionsOnStop:      defaultConfig.DataSettings.LiveData.ClosePositionsOnStop,
			DataRequestRetryTolerance: defaultConfig.DataSettings.LiveData.DataRequestRetryTolerance,
			DataRequestRetryWaitTime:  defaultConfig.DataSettings.LiveData.DataRequestRetryWaitTime.Nanoseconds(),
			Credentials:               creds,
		}
	}
	if defaultConfig.DataSettings.CSVData != nil {
		dataSettings.CsvData = &btrpc.CSVData{
			Path: defaultConfig.DataSettings.CSVData.FullPath,
		}
	}
	if defaultConfig.DataSettings.DatabaseData != nil {
		dbConnectionDetails := &btrpc.DatabaseConnectionDetails{
			Host:     defaultConfig.DataSettings.DatabaseData.Config.Host,
			Port:     defaultConfig.DataSettings.DatabaseData.Config.Port,
			Password: defaultConfig.DataSettings.DatabaseData.Config.Password,
			Database: defaultConfig.DataSettings.DatabaseData.Config.Database,
			SslMode:  defaultConfig.DataSettings.DatabaseData.Config.SSLMode,
			UserName: defaultConfig.DataSettings.DatabaseData.Config.Username,
		}
		dbConfig := &btrpc.DatabaseConfig{
			Config: dbConnectionDetails,
		}
		dataSettings.DatabaseData = &btrpc.DatabaseData{
			StartDate:        timestamppb.New(defaultConfig.DataSettings.DatabaseData.StartDate),
			EndDate:          timestamppb.New(defaultConfig.DataSettings.DatabaseData.EndDate),
			Config:           dbConfig,
			Path:             defaultConfig.DataSettings.DatabaseData.Path,
			InclusiveEndDate: defaultConfig.DataSettings.DatabaseData.InclusiveEndDate,
		}
	}

	cfg := &btrpc.Config{
		Nickname: defaultConfig.Nickname,
		Goal:     defaultConfig.Goal,
		StrategySettings: &btrpc.StrategySettings{
			Name:                            defaultConfig.StrategySettings.Name,
			UseSimultaneousSignalProcessing: defaultConfig.StrategySettings.SimultaneousSignalProcessing,
			DisableUsdTracking:              defaultConfig.StrategySettings.DisableUSDTracking,
			CustomSettings:                  customSettings,
		},
		FundingSettings: &btrpc.FundingSettings{
			UseExchangeLevelFunding: defaultConfig.FundingSettings.UseExchangeLevelFunding,
			ExchangeLevelFunding:    exchangeLevelFunding,
		},
		CurrencySettings: currencySettings,
		DataSettings:     dataSettings,
		PortfolioSettings: &btrpc.PortfolioSettings{
			Leverage: &btrpc.Leverage{
				CanUseLeverage:                 defaultConfig.PortfolioSettings.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: defaultConfig.PortfolioSettings.Leverage.MaximumOrdersWithLeverageRatio.String(),
				MaximumLeverageRate:            defaultConfig.PortfolioSettings.Leverage.MaximumOrderLeverageRate.String(),
				MaximumCollateralLeverageRate:  defaultConfig.PortfolioSettings.Leverage.MaximumCollateralLeverageRate.String(),
			},
			BuySide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.PortfolioSettings.BuySide.MinimumSize.String(),
				MaximumSize:  defaultConfig.PortfolioSettings.BuySide.MaximumSize.String(),
				MaximumTotal: defaultConfig.PortfolioSettings.BuySide.MaximumTotal.String(),
			},
			SellSide: &btrpc.PurchaseSide{
				MinimumSize:  defaultConfig.PortfolioSettings.SellSide.MinimumSize.String(),
				MaximumSize:  defaultConfig.PortfolioSettings.SellSide.MaximumSize.String(),
				MaximumTotal: defaultConfig.PortfolioSettings.SellSide.MaximumTotal.String(),
			},
		},
		StatisticSettings: &btrpc.StatisticSettings{
			RiskFreeRate: defaultConfig.StatisticSettings.RiskFreeRate.String(),
		},
	}

	var dnr bool
	if c.IsSet("donotrunimmediately") {
		dnr = c.Bool("donotrunimmediately")
	}
	var dns bool
	if c.IsSet("donotstore") {
		dns = c.Bool("donotstore")
	}

	client := btrpc.NewBacktesterServiceClient(conn)
	result, err := client.ExecuteStrategyFromConfig(
		c.Context,
		&btrpc.ExecuteStrategyFromConfigRequest{
			Config:              cfg,
			DoNotRunImmediately: dnr,
			DoNotStore:          dns,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
