package backtest

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/api"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/csv"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/database"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctdatabase "github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{
		shutdown: make(chan struct{}),
	}
}

// Reset BackTest values to default
func (bt *BackTest) Reset() {
	bt.EventQueue.Reset()
	bt.Datas.Reset()
	bt.Portfolio.Reset()
	bt.Statistic.Reset()
	bt.Exchange.Reset()
	bt.Bot = nil
}

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(cfg *config.Config, templatePath, output string) (*BackTest, error) {
	if cfg == nil {
		return nil, errors.New("unable to setup backtester with nil config")
	}
	bt := New()
	err := bt.engineBotSetup(cfg)
	if err != nil {
		return nil, err
	}

	var e exchange.Exchange
	bt.Datas = &data.HandlerPerCurrency{}
	bt.EventQueue = &eventholder.Holder{}
	reports := &report.Data{
		Config:       cfg,
		TemplatePath: templatePath,
		OutputPath:   output,
	}
	bt.Reports = reports

	e, err = bt.setupExchangeSettings(cfg)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &e

	buyRule := config.MinMax{
		MinimumSize:  cfg.PortfolioSettings.BuySide.MinimumSize,
		MaximumSize:  cfg.PortfolioSettings.BuySide.MaximumSize,
		MaximumTotal: cfg.PortfolioSettings.BuySide.MaximumTotal,
	}
	buyRule.Validate()
	sellRule := config.MinMax{
		MinimumSize:  cfg.PortfolioSettings.SellSide.MinimumSize,
		MaximumSize:  cfg.PortfolioSettings.SellSide.MaximumSize,
		MaximumTotal: cfg.PortfolioSettings.SellSide.MaximumTotal,
	}
	sellRule.Validate()
	sizeManager := &size.Size{
		BuySide:  buyRule,
		SellSide: sellRule,
	}

	portfolioRisk := &risk.Risk{
		CurrencySettings: make(map[string]map[asset.Item]map[currency.Pair]*risk.CurrencySettings),
	}
	for i := range cfg.CurrencySettings {
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName] = make(map[asset.Item]map[currency.Pair]*risk.CurrencySettings)
		}
		var a asset.Item
		a, err = asset.New(cfg.CurrencySettings[i].Asset)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid asset in config for %v %v %v. Err %v",
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base+cfg.CurrencySettings[i].Quote,
				err)
		}
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] = make(map[currency.Pair]*risk.CurrencySettings)
		}
		var curr currency.Pair
		curr, err = currency.NewPairFromString(cfg.CurrencySettings[i].Base + cfg.CurrencySettings[i].Quote)
		if err != nil {
			return nil, fmt.Errorf(
				"invalid currency in config for %v %v %v. Err %v",
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base+cfg.CurrencySettings[i].Quote,
				err)
		}
		portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][curr] = &risk.CurrencySettings{
			MaxLeverageRatio:    cfg.CurrencySettings[i].Leverage.MaximumLeverageRatio,
			MaxLeverageRate:     cfg.CurrencySettings[i].Leverage.MaximumLeverageRate,
			MaximumHoldingRatio: cfg.CurrencySettings[i].MaximumHoldingsRatio,
		}
	}
	var p *portfolio.Portfolio
	p, err = portfolio.Setup(sizeManager, portfolioRisk, cfg.StatisticSettings.RiskFreeRate)
	if err != nil {
		return nil, err
	}
	for i := range e.CurrencySettings {
		lookup, _ := p.SetupCurrencySettingsMap(e.CurrencySettings[i].ExchangeName, e.CurrencySettings[i].AssetType, e.CurrencySettings[i].CurrencyPair)
		lookup.Fee = e.CurrencySettings[i].TakerFee
		lookup.Leverage = e.CurrencySettings[i].Leverage
		lookup.BuySideSizing = e.CurrencySettings[i].BuySide
		lookup.SellSideSizing = e.CurrencySettings[i].SellSide
		lookup.InitialFunds = e.CurrencySettings[i].InitialFunds
		lookup.ComplianceManager = compliance.Manager{
			Snapshots: []compliance.Snapshot{},
		}
	}
	bt.Portfolio = p

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategySettings.Name, cfg.StrategySettings.IsMultiCurrency)
	if err != nil {
		return nil, err
	}
	if cfg.StrategySettings.CustomSettings != nil {
		err = bt.Strategy.SetCustomSettings(cfg.StrategySettings.CustomSettings)
		if err != nil && err.Error() != "unsupported" {
			return nil, err
		}
	} else {
		bt.Strategy.SetDefaults()
	}
	stats := &statistics.Statistic{
		StrategyName:                cfg.StrategySettings.Name,
		ExchangeAssetPairStatistics: make(map[string]map[asset.Item]map[currency.Pair]*currencystatstics.CurrencyStatistic),
		RiskFreeRate:                cfg.StatisticSettings.RiskFreeRate,
	}
	bt.Statistic = stats
	reports.Statistics = stats

	cfg.PrintSetting()

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	resp := exchange.Exchange{}

	for i := range cfg.CurrencySettings {
		exch, pair, a, err := bt.loadExchangePairAssetBase(
			cfg.CurrencySettings[i].ExchangeName,
			cfg.CurrencySettings[i].Base,
			cfg.CurrencySettings[i].Quote,
			cfg.CurrencySettings[i].Asset)
		if err != nil {
			return resp, err
		}
		if cfg.CurrencySettings[i].InitialFunds <= 0 {
			return resp, fmt.Errorf("initial funds unset for %s %s %s-%s",
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base,
				cfg.CurrencySettings[i].Quote)
		}

		exchangeName := strings.ToLower(exch.GetName())
		bt.Datas.Setup()
		klineData, err := bt.loadData(cfg, exch, pair, a)
		if err != nil {
			return resp, err
		}
		bt.Datas.SetDataForCurrency(exchangeName, a, pair, klineData)
		var makerFee, takerFee float64
		if cfg.CurrencySettings[i].MakerFee > 0 {
			makerFee = cfg.CurrencySettings[i].MakerFee
		}
		if cfg.CurrencySettings[i].TakerFee > 0 {
			takerFee = cfg.CurrencySettings[i].TakerFee
		}
		if makerFee == 0 || takerFee == 0 {
			var apiMakerFee, apiTakerFee float64
			apiMakerFee, apiTakerFee, err = getFees(exch, pair)
			if err != nil {
				return resp, err
			}
			if makerFee == 0 {
				makerFee = apiMakerFee
			}
			if takerFee == 0 {
				takerFee = apiTakerFee
			}
		}

		if cfg.CurrencySettings[i].MaximumSlippagePercent < 0 {
			log.Warnf(log.BackTester, "Invalid maximum slippage percent '%v'. Slippage percent is defined as a number, eg '100.00', defaulting to '%v'",
				cfg.CurrencySettings[i].MaximumSlippagePercent,
				slippage.DefaultMaximumSlippagePercent)
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MaximumSlippagePercent == 0 {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MinimumSlippagePercent < 0 {
			log.Warnf(log.BackTester, "Invalid minimum slippage percent '%v'. Slippage percent is defined as a number, eg '80.00', defaulting to '%v'",
				cfg.CurrencySettings[i].MinimumSlippagePercent,
				slippage.DefaultMinimumSlippagePercent)
			cfg.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
		}
		if cfg.CurrencySettings[i].MinimumSlippagePercent == 0 {
			cfg.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
		}
		if cfg.CurrencySettings[i].MaximumSlippagePercent < cfg.CurrencySettings[i].MinimumSlippagePercent {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}

		realOrders := false
		if cfg.DataSettings.LiveData != nil {
			realOrders = cfg.DataSettings.LiveData.RealOrders
		}

		buyRule := config.MinMax{
			MinimumSize:  cfg.CurrencySettings[i].BuySide.MinimumSize,
			MaximumSize:  cfg.CurrencySettings[i].BuySide.MaximumSize,
			MaximumTotal: cfg.CurrencySettings[i].BuySide.MaximumTotal,
		}
		buyRule.Validate()
		sellRule := config.MinMax{
			MinimumSize:  cfg.CurrencySettings[i].SellSide.MinimumSize,
			MaximumSize:  cfg.CurrencySettings[i].SellSide.MaximumSize,
			MaximumTotal: cfg.CurrencySettings[i].SellSide.MaximumTotal,
		}
		sellRule.Validate()
		resp.CurrencySettings = append(resp.CurrencySettings, exchange.Settings{
			ExchangeName:        cfg.CurrencySettings[i].ExchangeName,
			InitialFunds:        cfg.CurrencySettings[i].InitialFunds,
			MinimumSlippageRate: cfg.CurrencySettings[i].MinimumSlippagePercent,
			MaximumSlippageRate: cfg.CurrencySettings[i].MaximumSlippagePercent,
			CurrencyPair:        pair,
			AssetType:           a,
			ExchangeFee:         takerFee,
			MakerFee:            takerFee,
			TakerFee:            makerFee,
			UseRealOrders:       realOrders,
			BuySide:             buyRule,
			SellSide:            sellRule,
			Leverage: config.Leverage{
				CanUseLeverage:       cfg.CurrencySettings[i].Leverage.CanUseLeverage,
				MaximumLeverageRate:  cfg.CurrencySettings[i].Leverage.MaximumLeverageRate,
				MaximumLeverageRatio: cfg.CurrencySettings[i].Leverage.MaximumLeverageRatio,
			},
		})
	}

	return resp, nil
}

func (bt *BackTest) loadExchangePairAssetBase(exch, base, quote, ass string) (gctexchange.IBotExchange, currency.Pair, asset.Item, error) {
	var err error
	e := bt.Bot.GetExchangeByName(exch)
	if e == nil {
		return nil, currency.Pair{}, "", engine.ErrExchangeNotFound
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromStrings(base, quote)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}

	var a asset.Item
	a, err = asset.New(ass)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}

	exchangeBase := e.GetBase()
	if !exchangeBase.ValidateAPICredentials() {
		log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", exchangeBase.Name)
	}

	fPair, err = exchangeBase.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, currency.Pair{}, "", err
	}
	return e, fPair, a, nil
}

// engineBotSetup sets up a basic bot to retrieve exchange data
// as well as process orders
func (bt *BackTest) engineBotSetup(cfg *config.Config) error {
	var err error
	path := gctconfig.DefaultFilePath()
	if cfg.GoCryptoTraderConfigPath != "" {
		path = cfg.GoCryptoTraderConfigPath
	}
	bt.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
		ConfigFile:     path,
	}, nil)
	if err != nil {
		return err
	}

	if len(cfg.CurrencySettings) == 0 {
		return errors.New("expected at least one currency in the config")
	}

	for i := range cfg.CurrencySettings {
		err = bt.Bot.LoadExchange(cfg.CurrencySettings[i].ExchangeName, false, nil)
		if err != nil && err.Error() != "exchange already loaded" {
			return err
		}
	}

	err = bt.Bot.OrderManager.Start(bt.Bot)
	if err != nil {
		return err
	}

	return nil
}

// getFees will return an exchange's fee rate from GCT's wrapper function
func getFees(exch gctexchange.IBotExchange, fPair currency.Pair) (makerFee, takerFee float64, err error) {
	takerFee, err = exch.GetFeeByType(&gctexchange.FeeBuilder{
		FeeType:       gctexchange.OfflineTradeFee,
		Pair:          fPair,
		IsMaker:       false,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return makerFee, takerFee, err
	}

	makerFee, err = exch.GetFeeByType(&gctexchange.FeeBuilder{
		FeeType:       gctexchange.OfflineTradeFee,
		Pair:          fPair,
		IsMaker:       true,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return makerFee, takerFee, err
	}

	return makerFee, takerFee, err
}

// loadData will create kline data from the sources defined in start config files. It can exist from databases, csv or API endpoints
// it can also be generated from trade data which will be converted into kline data
func (bt *BackTest) loadData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) (*kline.DataFromKline, error) {
	if exch == nil {
		return nil, errors.New("nil exchange received")
	}
	base := exch.GetBase()
	if cfg.DataSettings.DatabaseData == nil &&
		cfg.DataSettings.LiveData == nil &&
		cfg.DataSettings.APIData == nil &&
		cfg.DataSettings.CSVData == nil {
		return nil, errors.New("no data settings set in config")
	}
	resp := &kline.DataFromKline{}
	var err error
	if (cfg.DataSettings.APIData != nil && cfg.DataSettings.DatabaseData != nil) ||
		(cfg.DataSettings.APIData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.APIData != nil && cfg.DataSettings.CSVData != nil) ||
		(cfg.DataSettings.DatabaseData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.CSVData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.CSVData != nil && cfg.DataSettings.DatabaseData != nil) {
		return nil, errors.New("ambiguous settings received. Only one data type can be set")
	}

	switch {
	case cfg.DataSettings.CSVData != nil:
		resp, err = csv.LoadData(
			cfg.DataSettings.CSVData.FullPath,
			cfg.DataSettings.DataType,
			strings.ToLower(exch.GetName()),
			cfg.DataSettings.Interval,
			fPair,
			a)
		if err != nil {
			return nil, err
		}
		resp.Item.RemoveDuplicates()
		resp.Item.SortCandlesByTimestamp(false)
		resp.Range = gctkline.CalculateCandleDateRanges(
			resp.Item.Candles[0].Time,
			resp.Item.Candles[len(resp.Item.Candles)-1].Time,
			gctkline.Interval(cfg.DataSettings.Interval),
			0,
		)
		err = resp.Range.Verify(resp.Item.Candles)
		if err != nil {
			if strings.Contains(err.Error(), "missing candles data between") {
				log.Warn(log.BackTester, err.Error())
			} else {
				return nil, err
			}
		}
	case cfg.DataSettings.DatabaseData != nil:
		if cfg.DataSettings.DatabaseData.ConfigOverride != nil {
			bt.Bot.Config.Database = *cfg.DataSettings.DatabaseData.ConfigOverride
			gctdatabase.DB.DataPath = filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "database")
			gctdatabase.DB.Config = cfg.DataSettings.DatabaseData.ConfigOverride
			err = bt.Bot.DatabaseManager.Start(bt.Bot)
			if err != nil {
				return nil, err
			}
			defer func() {
				err = bt.Bot.DatabaseManager.Stop()
				if err != nil {
					log.Error(log.BackTester, err)
				}
			}()
		}
		resp, err = loadDatabaseData(cfg, exch.GetName(), fPair, a)
		if err != nil {
			return resp, err
		}

		resp.Item.RemoveDuplicates()
		resp.Item.SortCandlesByTimestamp(false)
		resp.Range = gctkline.CalculateCandleDateRanges(
			cfg.DataSettings.DatabaseData.StartDate,
			cfg.DataSettings.DatabaseData.EndDate,
			gctkline.Interval(cfg.DataSettings.Interval),
			0,
		)
		err = resp.Range.Verify(resp.Item.Candles)
		if err != nil {
			if strings.Contains(err.Error(), "missing candles data between") {
				log.Warn(log.BackTester, err.Error())
			} else {
				return nil, err
			}
		}
	case cfg.DataSettings.APIData != nil:
		resp, err = loadAPIData(
			cfg,
			exch,
			fPair,
			a,
			base.Features.Enabled.Kline.ResultLimit)
		if err != nil {
			return resp, err
		}
	case cfg.DataSettings.LiveData != nil:
		if len(cfg.CurrencySettings) > 1 {
			return nil, errors.New("live data simulation only supports one currency")
		}
		err = loadLiveData(cfg, base)
		if err != nil {
			return nil, err
		}
		go bt.loadLiveDataLoop(
			resp,
			cfg,
			exch,
			fPair,
			a)
		return resp, nil
	}
	if resp == nil {
		return nil, fmt.Errorf("processing error, response returned nil")
	}

	err = base.ValidateKline(fPair, a, resp.Item.Interval)
	if err != nil {
		return nil, err
	}

	err = resp.Load()
	if err != nil {
		return nil, err
	}
	bt.Reports.AddKlineItem(&resp.Item)
	return resp, nil
}

func loadDatabaseData(cfg *config.Config, name string, fPair currency.Pair, a asset.Item) (*kline.DataFromKline, error) {
	if cfg == nil || cfg.DataSettings.DatabaseData == nil {
		return nil, errors.New("nil config data received")
	}
	if cfg.DataSettings.DatabaseData.StartDate.IsZero() || cfg.DataSettings.DatabaseData.EndDate.IsZero() ||
		cfg.DataSettings.DatabaseData.StartDate.After(cfg.DataSettings.DatabaseData.EndDate) {
		return nil, errors.New("database data start and end dates must be set")
	}
	resp, err := database.LoadData(
		cfg.DataSettings.DatabaseData.StartDate,
		cfg.DataSettings.DatabaseData.EndDate,
		cfg.DataSettings.Interval,
		strings.ToLower(name),
		cfg.DataSettings.DataType,
		fPair,
		a)
	if err != nil {
		return nil, err
	}
	return resp, err
}

func loadAPIData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, resultLimit uint32) (*kline.DataFromKline, error) {
	if cfg.DataSettings.APIData.StartDate.IsZero() || cfg.DataSettings.APIData.EndDate.IsZero() ||
		cfg.DataSettings.APIData.StartDate.After(cfg.DataSettings.APIData.EndDate) {
		return nil, errors.New("api data start and end dates must be set")
	}
	if cfg.DataSettings.Interval == 0 {
		return nil, errors.New("api data interval unset")
	}
	dates := gctkline.CalculateCandleDateRanges(
		cfg.DataSettings.APIData.StartDate,
		cfg.DataSettings.APIData.EndDate,
		gctkline.Interval(cfg.DataSettings.Interval),
		resultLimit)
	candles, err := api.LoadData(
		cfg.DataSettings.DataType,
		cfg.DataSettings.APIData.StartDate,
		cfg.DataSettings.APIData.EndDate,
		cfg.DataSettings.Interval,
		exch,
		fPair,
		a)
	if err != nil {
		return nil, err
	}
	err = dates.Verify(candles.Candles)
	if err != nil {
		if strings.Contains(err.Error(), "missing candles data between") {
			log.Warn(log.BackTester, err.Error())
		} else {
			return nil, err
		}
	}

	candles.FillMissingDataWithEmptyEntries(dates)
	return &kline.DataFromKline{
		Item:  *candles,
		Range: dates,
	}, nil
}

func loadLiveData(cfg *config.Config, base *gctexchange.Base) error {
	if cfg == nil || base == nil || cfg.DataSettings.LiveData == nil {
		return errors.New("received nil argument(s)")
	}

	if cfg.DataSettings.LiveData.APIKeyOverride != "" {
		base.API.Credentials.Key = cfg.DataSettings.LiveData.APIKeyOverride
	}
	if cfg.DataSettings.LiveData.APISecretOverride != "" {
		base.API.Credentials.Secret = cfg.DataSettings.LiveData.APISecretOverride
	}
	if cfg.DataSettings.LiveData.APIClientIDOverride != "" {
		base.API.Credentials.ClientID = cfg.DataSettings.LiveData.APIClientIDOverride
	}
	if cfg.DataSettings.LiveData.API2FAOverride != "" {
		base.API.Credentials.PEMKey = cfg.DataSettings.LiveData.API2FAOverride
	}
	validated := base.ValidateAPICredentials()
	base.API.AuthenticatedSupport = validated
	if !validated {
		log.Warn(log.BackTester, "bad credentials received, no live trading for you")
		cfg.DataSettings.LiveData.RealOrders = false
	}
	return nil
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (bt *BackTest) Run() error {
dataLoadingIssue:
	for ev, ok := bt.EventQueue.NextEvent(); true; ev, ok = bt.EventQueue.NextEvent() {
		if !ok {
			dataHandlerMap := bt.Datas.GetAllData()
			for exchangeName, exchangeMap := range dataHandlerMap {
				for assetItem, assetMap := range exchangeMap {
					var hasProcessedData bool
					for currencyPair, dataHandler := range assetMap {
						d, ok := dataHandler.Next()
						if !ok {
							if !bt.hasHandledEvent {
								log.Errorf(log.BackTester, "Unable to perform `Next` for %v %v %v", exchangeName, assetItem, currencyPair)
							}
							break dataLoadingIssue
						}
						if bt.Strategy.IsMultiCurrency() && hasProcessedData {
							continue
						}
						bt.EventQueue.AppendEvent(d)
						hasProcessedData = true
					}
				}
			}
		}

		err := bt.handleEvent(ev)
		if err != nil {
			return err
		}
		if !bt.hasHandledEvent {
			bt.hasHandledEvent = true
		}
	}

	return nil
}

// handleEvent is the main processor of data for the backtester
// after data has been loaded and Run has appended a data event to the queue,
// handle event will process events and add further events to the queue if they
// are required
func (bt *BackTest) handleEvent(e common.EventHandler) error {
	switch ev := e.(type) {
	case common.DataEventHandler:
		bt.processDataEvent(ev)
	case signal.Event:
		bt.processSignalEvent(ev)
	case order.Event:
		bt.processOrderEvent(ev)
	case fill.Event:
		bt.processFillEvent(ev)
	}

	return nil
}

// processDataEvent determines what signal events are generated and appended
// to the event queue based on whether it is running a multi-currency consideration strategy order not
//
// for multi-currency-consideration it will pass all currency datas to the strategy for it to determine what
// currencies to act upon
//
// for non-multi-currency-consideration strategies, it will simply process every currency individually
// against the strategy and generate signals
func (bt *BackTest) processDataEvent(e common.DataEventHandler) {
	if bt.Strategy.IsMultiCurrency() {
		var dataEvents []data.Handler
		dataHandlerMap := bt.Datas.GetAllData()
		for _, exchangeMap := range dataHandlerMap {
			for _, assetMap := range exchangeMap {
				for _, dataHandler := range assetMap {
					latestData := dataHandler.Latest()
					bt.updateStatsForDataEvent(latestData)
					dataEvents = append(dataEvents, dataHandler)
				}
			}
		}
		signals, err := bt.Strategy.OnSignals(dataEvents, bt.Portfolio)
		if err != nil {
			log.Error(log.BackTester, err)
		}
		for i := range signals {
			err = bt.Statistic.SetEventForTime(signals[i])
			if err != nil {
				log.Error(log.BackTester, err)
			}
		}
		for i := range signals {
			bt.EventQueue.AppendEvent(signals[i])
		}
	} else {
		bt.updateStatsForDataEvent(e)
		d := bt.Datas.GetDataForCurrency(e.GetExchange(), e.GetAssetType(), e.Pair())

		s, err := bt.Strategy.OnSignal(d, bt.Portfolio)
		if err != nil {
			log.Error(log.BackTester, err)
			return
		}
		err = bt.Statistic.SetEventForTime(s)
		if err != nil {
			log.Error(log.BackTester, err)
		}
		bt.EventQueue.AppendEvent(s)
	}
}

// updateStatsForDataEvent makes various systems aware of price movements from
// data events
func (bt *BackTest) updateStatsForDataEvent(e common.DataEventHandler) {
	// update portfolio with latest price
	err := bt.Portfolio.Update(e)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	// update statistics with latest price
	err = bt.Statistic.SetupEventForTime(e)
	if err != nil {
		log.Error(log.BackTester, err)
	}
}

func (bt *BackTest) processSignalEvent(ev signal.Event) {
	cs, _ := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	o, err := bt.Portfolio.OnSignal(ev, &cs)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}
	err = bt.Statistic.SetEventForTime(o)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	bt.EventQueue.AppendEvent(o)
}

func (bt *BackTest) processOrderEvent(ev order.Event) {
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	f, err := bt.Exchange.ExecuteOrder(ev, d, bt.Bot)
	if err != nil {
		if f == nil {
			log.Errorf(log.BackTester, "fill event should always be returned, please fix, %v", err)
			return
		}
		log.Errorf(log.BackTester, "%v %v %v %v", f.GetExchange(), f.GetAssetType(), f.Pair(), err)
	}
	err = bt.Statistic.SetEventForTime(f)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	bt.EventQueue.AppendEvent(f)
}

func (bt *BackTest) processFillEvent(ev fill.Event) {
	t, err := bt.Portfolio.OnFill(ev)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}

	err = bt.Statistic.SetEventForTime(t)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	var holding holdings.Holding
	holding, err = bt.Portfolio.ViewHoldingAtTimePeriod(ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetTime())
	if err != nil {
		log.Error(log.BackTester, err)
	}

	err = bt.Statistic.AddHoldingsForTime(&holding)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	var cp *compliance.Manager
	cp, err = bt.Portfolio.GetComplianceManager(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Error(log.BackTester, err)
	}

	snap := cp.GetLatestSnapshot()
	err = bt.Statistic.AddComplianceSnapshotForTime(snap, ev)
	if err != nil {
		log.Error(log.BackTester, err)
	}
}

// RunLive is a proof of concept function that does not yet support multi currency usage
// It runs by constantly checking for new live datas and running through the list of events
// once new data is processed. It will run until application close event has been received
func (bt *BackTest) RunLive() error {
	timerino := time.NewTimer(time.Minute * 5)
	tickerino := time.NewTicker(time.Second)
	doneARun := false
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timerino.C:
			return errors.New("no data returned in 5 minutes, shutting down")
		case <-tickerino.C:
			for e, ok := bt.EventQueue.NextEvent(); true; e, ok = bt.EventQueue.NextEvent() {
				if !ok {
					// as live only supports singular currency, just get the proper reference manually
					var d data.Handler
					dd := bt.Datas.GetAllData()
					for k1, v1 := range dd {
						for k2, v2 := range v1 {
							for k3 := range v2 {
								d = dd[k1][k2][k3]
							}
						}
					}
					de, ok := d.Next()
					if !ok {
						break
					}
					bt.EventQueue.AppendEvent(de)
					doneARun = true
					continue
				}
				err := bt.handleEvent(e)
				if err != nil {
					return err
				}
			}
			if doneARun {
				timerino = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) {
	startDate := time.Now()
	candles, err := live.LoadData(
		exch,
		cfg.DataSettings.DataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}
	resp.Item = *candles
	err = bt.loadLiveData(resp, cfg, exch, fPair, a, startDate)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}

	timerino := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-bt.shutdown:
			return
		case <-timerino.C:
			err = bt.loadLiveData(resp, cfg, exch, fPair, a, startDate)
			if err != nil {
				log.Error(log.BackTester, err)
				return
			}
		}
	}
}

func (bt *BackTest) loadLiveData(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, startDate time.Time) error {
	candles, err := live.LoadData(
		exch,
		cfg.DataSettings.DataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		return err
	}
	resp.Item.Candles = append(resp.Item.Candles, candles.Candles...)
	_, err = exch.FetchOrderbook(fPair, a)
	if err != nil {
		return err
	}
	resp.Item.RemoveDuplicates()
	resp.Item.SortCandlesByTimestamp(false)
	if len(candles.Candles) == 0 {
		return nil
	}
	endDate := candles.Candles[len(candles.Candles)-1].Time.Add(cfg.DataSettings.Interval)
	if resp.Range.Ranges == nil {
		dataRange := gctkline.CalculateCandleDateRanges(
			startDate,
			endDate,
			gctkline.Interval(cfg.DataSettings.Interval),
			0,
		)
		resp.Range = gctkline.IntervalRangeHolder{
			Start:  startDate,
			End:    endDate,
			Ranges: dataRange.Ranges,
		}
	}
	var intervalData []gctkline.IntervalData
	for i := range candles.Candles {
		intervalData = append(intervalData, gctkline.IntervalData{
			Start:   candles.Candles[i].Time,
			End:     candles.Candles[i].Time.Add(cfg.DataSettings.Interval),
			HasData: true,
		})
	}
	resp.Range.Ranges[0].Intervals = intervalData
	if len(intervalData) > 0 {
		resp.Range.Ranges[0].End = intervalData[len(intervalData)-1].End
	}

	resp.Append(candles)
	bt.Reports.AddKlineItem(&resp.Item)
	log.Info(log.BackTester, "sleeping for 30 seconds before checking for new candle data")
	return nil
}

// Stop shuts down the live data loop
func (bt *BackTest) Stop() {
	close(bt.shutdown)
}
