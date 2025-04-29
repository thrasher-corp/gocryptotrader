package engine

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/binancecashandcarry"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var dcaConfigPath = filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat")

func TestExecuteStrategyFromFile(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil config")

	s.config, err = config.GenerateDefaultConfig()
	require.NoError(t, err, "GenerateDefaultConfig must not error")

	s.config.Report.GenerateReport = false
	_, err = s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil task manager")

	s.manager = NewTaskManager()
	_, err = s.ExecuteStrategyFromFile(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromFile should error correctly with a nil request")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{})
	assert.ErrorIs(t, err, common.ErrFileNotFound)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath: dcaConfigPath,
	})
	assert.NoError(t, err, "ExecuteStrategyFromFile should not error")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now()),
		EndTimeOverride:   timestamppb.New(time.Now().Add(-time.Minute)),
	})
	assert.ErrorIs(t, err, gctcommon.ErrStartAfterEnd)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now().Add(-time.Minute)),
		EndTimeOverride:   timestamppb.New(time.Now()),
		IntervalOverride:  durationpb.New(time.Duration(1)),
	})
	assert.ErrorIs(t, err, gctkline.ErrInvalidInterval)

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:  dcaConfigPath,
		StartTimeOverride: timestamppb.New(time.Now().Add(-time.Hour * 6).Truncate(time.Hour)),
		EndTimeOverride:   timestamppb.New(time.Now().Add(-time.Hour * 2).Truncate(time.Hour)),
		IntervalOverride:  durationpb.New(time.Hour),
	})
	assert.NoError(t, err, "ExecuteStrategyFromFile should not error")

	_, err = s.ExecuteStrategyFromFile(t.Context(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath:    dcaConfigPath,
		DoNotRunImmediately: true,
		DoNotStore:          true,
	})
	assert.ErrorIs(t, err, errCannotHandleRequest)
}

func TestExecuteStrategyFromConfig(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil config")

	s.config, err = config.GenerateDefaultConfig()
	require.NoError(t, err, "GenerateDefaultConfig must not error")
	_, err = s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil task manager")

	s.config.Report.GenerateReport = false
	s.manager = NewTaskManager()
	_, err = s.ExecuteStrategyFromConfig(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer, "ExecuteStrategyFromConfig should error correctly with a nil request")

	defaultConfig, err := config.ReadStrategyConfigFromFile(dcaConfigPath)
	require.NoError(t, err, "ReadStrategyConfigFromFile must not error")
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
		var sd *btrpc.SpotDetails
		if defaultConfig.CurrencySettings[i].SpotDetails != nil {
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
				sd = &btrpc.SpotDetails{
					InitialBaseFunds: defaultConfig.CurrencySettings[i].SpotDetails.InitialBaseFunds.String(),
				}
			}
			if defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
				if sd == nil {
					sd = &btrpc.SpotDetails{}
				}
				sd.InitialQuoteFunds = defaultConfig.CurrencySettings[i].SpotDetails.InitialQuoteFunds.String()
			}
		}
		var fd *btrpc.FuturesDetails
		if defaultConfig.CurrencySettings[i].FuturesDetails != nil {
			fd = &btrpc.FuturesDetails{
				Leverage: &btrpc.Leverage{
					CanUseLeverage:                 defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
					MaximumOrdersWithLeverageRatio: defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio.String(),
					MaximumLeverageRate:            defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate.String(),
					MaximumCollateralLeverageRate:  defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumCollateralLeverageRate.String(),
				},
			}
		}
		var makerFee, takerFee string
		if defaultConfig.CurrencySettings[i].MakerFee != nil {
			makerFee = defaultConfig.CurrencySettings[i].MakerFee.String()
		}
		if defaultConfig.CurrencySettings[i].TakerFee != nil {
			takerFee = defaultConfig.CurrencySettings[i].TakerFee.String()
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
			MakerFeeOverride:          makerFee,
			TakerFeeOverride:          takerFee,
			MaximumHoldingsRatio:      defaultConfig.CurrencySettings[i].MaximumHoldingsRatio.String(),
			SkipCandleVolumeFitting:   defaultConfig.CurrencySettings[i].SkipCandleVolumeFitting,
			UseExchangeOrderLimits:    defaultConfig.CurrencySettings[i].CanUseExchangeLimits,
			UseExchangePnlCalculation: defaultConfig.CurrencySettings[i].UseExchangePNLCalculation,
			SpotDetails:               sd,
			FuturesDetails:            fd,
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
			RealOrders:  defaultConfig.DataSettings.LiveData.RealOrders,
			Credentials: creds,
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
			Enabled: false,
			Verbose: false,
			Driver:  "",
			Config:  dbConnectionDetails,
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

	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	require.NoError(t, err, "ExecuteStrategyFromConfig must not error")

	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		DoNotRunImmediately: true,
		DoNotStore:          true,
		Config:              cfg,
	})
	assert.ErrorIs(t, err, errCannotHandleRequest)

	// coverage test to ensure the rest of the config can successfully be converted
	// this will not have a successful response
	cfg.FundingSettings.UseExchangeLevelFunding = true
	cfg.StrategySettings.UseSimultaneousSignalProcessing = true
	cfg.FundingSettings.ExchangeLevelFunding = append(cfg.FundingSettings.ExchangeLevelFunding, &btrpc.ExchangeLevelFunding{
		ExchangeName: defaultConfig.CurrencySettings[0].ExchangeName,
		Asset:        defaultConfig.CurrencySettings[0].Asset.String(),
		Currency:     defaultConfig.CurrencySettings[0].Base.String(),
		InitialFunds: "1337",
		TransferFee:  "1337",
	})
	cfg.CurrencySettings[0].FuturesDetails = &btrpc.FuturesDetails{Leverage: &btrpc.Leverage{
		CanUseLeverage:                 false,
		MaximumOrdersWithLeverageRatio: "1337",
		MaximumLeverageRate:            "1337",
		MaximumCollateralLeverageRate:  "1337",
	}}
	cfg.DataSettings.DatabaseData = &btrpc.DatabaseData{
		StartDate: timestamppb.New(time.Now()),
		EndDate:   timestamppb.New(time.Now()),
		Config: &btrpc.DatabaseConfig{
			Enabled: false,
			Verbose: false,
			Driver:  "",
			Config:  &btrpc.DatabaseConnectionDetails{},
		},
		Path:             "test",
		InclusiveEndDate: false,
	}
	cfg.DataSettings.LiveData = &btrpc.LiveData{
		Credentials: []*btrpc.Credentials{
			{
				Exchange: "test",
				Keys: &btrpc.ExchangeCredentials{
					Key:             "1",
					Secret:          "2",
					ClientId:        "3",
					PemKey:          "4",
					SubAccount:      "5",
					OneTimePassword: "6",
				},
			},
		},
	}
	cfg.DataSettings.CsvData = &btrpc.CSVData{
		Path: "test",
	}
	for i := range cfg.CurrencySettings {
		cfg.CurrencySettings[i].SpotDetails.InitialQuoteFunds = ""
		cfg.CurrencySettings[i].SpotDetails.InitialBaseFunds = ""
	}
	_, err = s.ExecuteStrategyFromConfig(t.Context(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	assert.True(t, errors.Is(err, gctcommon.ErrStartEqualsEnd) || errors.Is(err, errAmbiguousDataSource),
		"ExecuteStrategyFromConfig should error")
}

func TestListAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ListAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ListAllTasks(t.Context(), nil)
	assert.NoError(t, err, "ListAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	resp, err := s.ListAllTasks(t.Context(), &btrpc.ListAllTasksRequest{})
	assert.NoError(t, err, "ListAllTasks should not error")
	assert.Len(t, resp.Tasks, 1, "ListAllTasks should return 1 task")
}

func TestGRPCStopTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StopTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StopTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	_, err = s.StopTask(t.Context(), &btrpc.StopTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	assert.ErrorIs(t, err, errTaskHasNotRan)
	require.Len(t, s.manager.tasks, 1, "StopTask must not remove task")

	s.manager.tasks[0].MetaData.DateStarted = time.Now()
	_, err = s.StopTask(t.Context(), &btrpc.StopTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	require.NoError(t, err, "StopTask must not error")
	assert.False(t, s.manager.tasks[0].MetaData.DateEnded.IsZero(), "DateEnded should not be zero")
}

func TestGRPCStopAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StopAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StopAllTasks(t.Context(), nil)
	assert.NoError(t, err, "StopAllTasks should not error")

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	resp, err := s.StopAllTasks(t.Context(), &btrpc.StopAllTasksRequest{})
	assert.NoError(t, err, "StopAllTasks should not error")
	require.Len(t, s.manager.tasks, 1, "StopAllTasks must return 1 task")
	assert.Empty(t, resp.TasksStopped, "TasksStopped should be empty")

	s.manager.tasks[0].MetaData.DateStarted = time.Now()
	resp, err = s.StopAllTasks(t.Context(), &btrpc.StopAllTasksRequest{})
	require.NoError(t, err, "StopAllTasks must not error")
	assert.False(t, s.manager.tasks[0].MetaData.DateEnded.IsZero(), "DateEnded should not be zero")
	assert.Len(t, resp.TasksStopped, 1, "TasksStopped should be 1 task")
}

func TestGRPCStartTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StartTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StartTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &fakeStrat{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &fakeStats{},
		Reports:    &fakeReport{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")
	_, err = s.StartTask(t.Context(), &btrpc.StartTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	require.NoError(t, err, "StartTask must not error")
	require.Len(t, s.manager.tasks, 1, "StartTask must return 1 task")
	assert.False(t, s.manager.tasks[0].MetaData.DateStarted.IsZero(), "DateStarted should not be zero")
}

func TestGRPCStartAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.StartAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.StartAllTasks(t.Context(), nil)
	assert.NoError(t, err, "StartAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.StartAllTasks(t.Context(), &btrpc.StartAllTasksRequest{})
	require.NoError(t, err, "StartAllTasks must not error")
	require.Len(t, s.manager.tasks, 1, "StartAllTasks must return 1 task")
	assert.False(t, s.manager.tasks[0].MetaData.DateStarted.IsZero(), "DateStarted should not be zero")
}

func TestGRPCClearTask(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ClearTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ClearTask(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.ClearTask(t.Context(), &btrpc.ClearTaskRequest{
		Id: bt.MetaData.ID.String(),
	})
	assert.NoError(t, err, "ClearTask should not error")
	assert.Empty(t, s.manager.tasks, "tasks should be empty")
}

func TestGRPCClearAllTasks(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ClearAllTasks(t.Context(), nil)
	assert.ErrorIs(t, err, gctcommon.ErrNilPointer)

	s.manager = NewTaskManager()
	_, err = s.ClearAllTasks(t.Context(), nil)
	assert.NoError(t, err, "ClearAllTasks should not error")

	bt := &BackTest{
		Strategy:   &binancecashandcarry.Strategy{},
		EventQueue: &eventholder.Holder{},
		DataHolder: &data.HandlerHolder{},
		Statistic:  &statistics.Statistic{},
		shutdown:   make(chan struct{}),
	}
	err = s.manager.AddTask(bt)
	assert.NoError(t, err, "AddTask should not error")

	_, err = s.ClearAllTasks(t.Context(), &btrpc.ClearAllTasksRequest{})
	assert.NoError(t, err, "ClearAllTasks should not error")
	assert.Empty(t, s.manager.tasks, "tasks should be empty")
}
