package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var dcaConfigPath = filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat")

func TestExecuteStrategyFromFile(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromFile(context.Background(), nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expecting '%v'", err, common.ErrNilArguments)
	}

	_, err = s.ExecuteStrategyFromFile(context.Background(), &btrpc.ExecuteStrategyFromFileRequest{})
	if !errors.Is(err, config.ErrFileNotFound) {
		t.Errorf("received '%v' expecting '%v'", err, config.ErrFileNotFound)
	}

	_, err = s.ExecuteStrategyFromFile(context.Background(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath: dcaConfigPath,
	})
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expecting '%v'", err, common.ErrNilArguments)
	}

	s.BacktesterConfig = &config.BacktesterConfig{}
	_, err = s.ExecuteStrategyFromFile(context.Background(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath: dcaConfigPath,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expecting '%v'", err, nil)
	}
}

func TestExecuteStrategyFromConfig(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromConfig(context.Background(), nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expecting '%v'", err, common.ErrNilArguments)
	}

	s.BacktesterConfig = &config.BacktesterConfig{}
	_, err = s.ExecuteStrategyFromConfig(context.Background(), &btrpc.ExecuteStrategyFromConfigRequest{})
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expecting '%v'", err, common.ErrNilArguments)
	}

	defaultConfig, err := config.ReadStrategyConfigFromFile(dcaConfigPath)
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expecting '%v'", err, nil)
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
			fd.Leverage = &btrpc.Leverage{
				CanUseLeverage:                 defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
				MaximumOrdersWithLeverageRatio: defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio.String(),
				MaximumLeverageRate:            defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate.String(),
				MaximumCollateralLeverageRate:  defaultConfig.CurrencySettings[i].FuturesDetails.Leverage.MaximumCollateralLeverageRate.String(),
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
			MinSlippagePercent:         defaultConfig.CurrencySettings[i].MinimumSlippagePercent.String(),
			MaxSlippagePercent:         defaultConfig.CurrencySettings[i].MaximumSlippagePercent.String(),
			MakerFeeOverride:           makerFee,
			TakerFeeOverride:           takerFee,
			MaximumHoldingsRatio:       defaultConfig.CurrencySettings[i].MaximumHoldingsRatio.String(),
			SkipCandleVolumeFitting:    defaultConfig.CurrencySettings[i].SkipCandleVolumeFitting,
			UseExchangeOrderLimits:     defaultConfig.CurrencySettings[i].CanUseExchangeLimits,
			UseExchange_PNLCalculation: defaultConfig.CurrencySettings[i].UseExchangePNLCalculation,
			SpotDetails:                sd,
			FuturesDetails:             fd,
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
		Interval: uint64(defaultConfig.DataSettings.Interval.Duration().Nanoseconds()),
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
		dataSettings.LiveData = &btrpc.LiveData{
			ApiKeyOverride:        defaultConfig.DataSettings.LiveData.APIKeyOverride,
			ApiSecretOverride:     defaultConfig.DataSettings.LiveData.APISecretOverride,
			ApiClientIdOverride:   defaultConfig.DataSettings.LiveData.APIClientIDOverride,
			Api_2FaOverride:       defaultConfig.DataSettings.LiveData.API2FAOverride,
			ApiSubAccountOverride: defaultConfig.DataSettings.LiveData.APISubAccountOverride,
			UseRealOrders:         defaultConfig.DataSettings.LiveData.RealOrders,
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
			Port:     int64(defaultConfig.DataSettings.DatabaseData.Config.Port),
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
			Disable_USDTracking:             defaultConfig.StrategySettings.DisableUSDTracking,
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

	_, err = s.ExecuteStrategyFromConfig(context.Background(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	if !errors.Is(err, nil) {
		t.Errorf("received '%v' expecting '%v'", err, nil)
	}

	// coverage test to ensure the rest of the config can successfully be converted
	// this will not have a successful response
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
	cfg.DataSettings.LiveData = &btrpc.LiveData{}
	cfg.DataSettings.CsvData = &btrpc.CSVData{
		Path: "test",
	}
	_, err = s.ExecuteStrategyFromConfig(context.Background(), &btrpc.ExecuteStrategyFromConfigRequest{
		Config: cfg,
	})
	if !errors.Is(err, gctcommon.ErrStartEqualsEnd) {
		t.Errorf("received '%v' expecting '%v'", err, gctcommon.ErrStartEqualsEnd)
	}
}
