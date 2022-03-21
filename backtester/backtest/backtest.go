package backtest

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shopspring/decimal"
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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctdatabase "github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{
		shutdown:   make(chan struct{}),
		Datas:      &data.HandlerPerCurrency{},
		EventQueue: &eventholder.Holder{},
	}
}

// Reset BackTest values to default
func (bt *BackTest) Reset() {
	bt.EventQueue.Reset()
	bt.Datas.Reset()
	bt.Portfolio.Reset()
	bt.Statistic.Reset()
	bt.Exchange.Reset()
	bt.Funding.Reset()
	bt.exchangeManager = nil
	bt.orderManager = nil
	bt.databaseManager = nil
}

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(cfg *config.Config, templatePath, output string) (*BackTest, error) {
	log.Infoln(log.BackTester, "loading config...")
	if cfg == nil {
		return nil, errNilConfig
	}
	var err error
	bt := New()
	bt.exchangeManager = engine.SetupExchangeManager()
	bt.orderManager, err = engine.SetupOrderManager(bt.exchangeManager, &engine.CommunicationManager{}, &sync.WaitGroup{}, false)
	if err != nil {
		return nil, err
	}
	err = bt.orderManager.Start()
	if err != nil {
		return nil, err
	}
	if cfg.DataSettings.DatabaseData != nil {
		bt.databaseManager, err = engine.SetupDatabaseConnectionManager(&cfg.DataSettings.DatabaseData.Config)
		if err != nil {
			return nil, err
		}
	}

	reports := &report.Data{
		Config:       cfg,
		TemplatePath: templatePath,
		OutputPath:   output,
	}
	bt.Reports = reports

	buyRule := exchange.MinMax{
		MinimumSize:  cfg.PortfolioSettings.BuySide.MinimumSize,
		MaximumSize:  cfg.PortfolioSettings.BuySide.MaximumSize,
		MaximumTotal: cfg.PortfolioSettings.BuySide.MaximumTotal,
	}
	sellRule := exchange.MinMax{
		MinimumSize:  cfg.PortfolioSettings.SellSide.MinimumSize,
		MaximumSize:  cfg.PortfolioSettings.SellSide.MaximumSize,
		MaximumTotal: cfg.PortfolioSettings.SellSide.MaximumTotal,
	}
	sizeManager := &size.Size{
		BuySide:  buyRule,
		SellSide: sellRule,
	}

	funds := funding.SetupFundingManager(
		cfg.StrategySettings.UseExchangeLevelFunding,
		cfg.StrategySettings.DisableUSDTracking,
	)
	if cfg.StrategySettings.UseExchangeLevelFunding {
		for i := range cfg.StrategySettings.ExchangeLevelFunding {
			var a asset.Item
			a, err = asset.New(cfg.StrategySettings.ExchangeLevelFunding[i].Asset)
			if err != nil {
				return nil, err
			}
			cq := currency.NewCode(cfg.StrategySettings.ExchangeLevelFunding[i].Currency)
			var item *funding.Item
			item, err = funding.CreateItem(cfg.StrategySettings.ExchangeLevelFunding[i].ExchangeName,
				a,
				cq,
				cfg.StrategySettings.ExchangeLevelFunding[i].InitialFunds,
				cfg.StrategySettings.ExchangeLevelFunding[i].TransferFee)
			if err != nil {
				return nil, err
			}
			err = funds.AddItem(item)
			if err != nil {
				return nil, err
			}
		}
	}

	var emm = make(map[string]gctexchange.IBotExchange)
	for i := range cfg.CurrencySettings {
		_, ok := emm[cfg.CurrencySettings[i].ExchangeName]
		if ok {
			continue
		}
		var exch gctexchange.IBotExchange
		exch, err = bt.exchangeManager.NewExchangeByName(cfg.CurrencySettings[i].ExchangeName)
		if err != nil {
			return nil, err
		}
		_, err = exch.GetDefaultConfig()
		if err != nil {
			return nil, err
		}
		exchBase := exch.GetBase()
		err = exch.UpdateTradablePairs(context.Background(), true)
		if err != nil {
			return nil, err
		}
		assets := exchBase.CurrencyPairs.GetAssetTypes(false)
		for i := range assets {
			exchBase.CurrencyPairs.Pairs[assets[i]].AssetEnabled = convert.BoolPtr(true)
			err = exch.SetPairs(exchBase.CurrencyPairs.Pairs[assets[i]].Available, assets[i], true)
			if err != nil {
				return nil, err
			}
		}

		bt.exchangeManager.Add(exch)
		emm[cfg.CurrencySettings[i].ExchangeName] = exch
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
				"%w for %v %v %v. Err %v",
				errInvalidConfigAsset,
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base+cfg.CurrencySettings[i].Quote,
				err)
		}
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] = make(map[currency.Pair]*risk.CurrencySettings)
		}
		var curr currency.Pair
		var b, q currency.Code
		b = currency.NewCode(cfg.CurrencySettings[i].Base)
		q = currency.NewCode(cfg.CurrencySettings[i].Quote)
		curr = currency.NewPair(b, q)
		var exch gctexchange.IBotExchange
		exch, err = bt.exchangeManager.GetExchangeByName(cfg.CurrencySettings[i].ExchangeName)
		if err != nil {
			return nil, err
		}
		exchBase := exch.GetBase()
		var requestFormat currency.PairFormat
		requestFormat, err = exchBase.GetPairFormat(a, true)
		if err != nil {
			return nil, fmt.Errorf("could not format currency %v, %w", curr, err)
		}
		curr = curr.Format(requestFormat.Delimiter, requestFormat.Uppercase)
		err = exchBase.CurrencyPairs.EnablePair(a, curr)
		if err != nil && !errors.Is(err, currency.ErrPairAlreadyEnabled) {
			return nil, fmt.Errorf(
				"could not enable currency %v %v %v. Err %w",
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base+cfg.CurrencySettings[i].Quote,
				err)
		}
		portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][curr] = &risk.CurrencySettings{
			MaximumOrdersWithLeverageRatio: cfg.CurrencySettings[i].Leverage.MaximumOrdersWithLeverageRatio,
			MaxLeverageRate:                cfg.CurrencySettings[i].Leverage.MaximumLeverageRate,
			MaximumHoldingRatio:            cfg.CurrencySettings[i].MaximumHoldingsRatio,
		}
		if cfg.CurrencySettings[i].MakerFee.GreaterThan(cfg.CurrencySettings[i].TakerFee) {
			log.Warnf(log.BackTester, "maker fee '%v' should not exceed taker fee '%v'. Please review config",
				cfg.CurrencySettings[i].MakerFee,
				cfg.CurrencySettings[i].TakerFee)
		}

		var baseItem, quoteItem *funding.Item
		if cfg.StrategySettings.UseExchangeLevelFunding {
			// add any remaining currency items that have no funding data in the strategy config
			baseItem, err = funding.CreateItem(cfg.CurrencySettings[i].ExchangeName,
				a,
				b,
				decimal.Zero,
				decimal.Zero)
			if err != nil {
				return nil, err
			}
			quoteItem, err = funding.CreateItem(cfg.CurrencySettings[i].ExchangeName,
				a,
				q,
				decimal.Zero,
				decimal.Zero)
			if err != nil {
				return nil, err
			}
			err = funds.AddItem(baseItem)
			if err != nil && !errors.Is(err, funding.ErrAlreadyExists) {
				return nil, err
			}
			err = funds.AddItem(quoteItem)
			if err != nil && !errors.Is(err, funding.ErrAlreadyExists) {
				return nil, err
			}
		} else {
			var bFunds, qFunds decimal.Decimal
			if cfg.CurrencySettings[i].InitialBaseFunds != nil {
				bFunds = *cfg.CurrencySettings[i].InitialBaseFunds
			}
			if cfg.CurrencySettings[i].InitialQuoteFunds != nil {
				qFunds = *cfg.CurrencySettings[i].InitialQuoteFunds
			}
			baseItem, err = funding.CreateItem(
				cfg.CurrencySettings[i].ExchangeName,
				a,
				curr.Base,
				bFunds,
				decimal.Zero)
			if err != nil {
				return nil, err
			}
			quoteItem, err = funding.CreateItem(
				cfg.CurrencySettings[i].ExchangeName,
				a,
				curr.Quote,
				qFunds,
				decimal.Zero)
			if err != nil {
				return nil, err
			}
			var pair *funding.Pair
			pair, err = funding.CreatePair(baseItem, quoteItem)
			if err != nil {
				return nil, err
			}
			err = funds.AddPair(pair)
			if err != nil {
				return nil, err
			}
		}
	}

	bt.Funding = funds
	var p *portfolio.Portfolio
	p, err = portfolio.Setup(sizeManager, portfolioRisk, cfg.StatisticSettings.RiskFreeRate)
	if err != nil {
		return nil, err
	}

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategySettings.Name, cfg.StrategySettings.SimultaneousSignalProcessing)
	if err != nil {
		return nil, err
	}
	bt.Strategy.SetDefaults()
	if cfg.StrategySettings.CustomSettings != nil {
		err = bt.Strategy.SetCustomSettings(cfg.StrategySettings.CustomSettings)
		if err != nil && !errors.Is(err, base.ErrCustomSettingsUnsupported) {
			return nil, err
		}
	}
	stats := &statistics.Statistic{
		StrategyName:                bt.Strategy.Name(),
		StrategyNickname:            cfg.Nickname,
		StrategyDescription:         bt.Strategy.Description(),
		StrategyGoal:                cfg.Goal,
		ExchangeAssetPairStatistics: make(map[string]map[asset.Item]map[currency.Pair]*statistics.CurrencyPairStatistic),
		RiskFreeRate:                cfg.StatisticSettings.RiskFreeRate,
		CandleInterval:              gctkline.Interval(cfg.DataSettings.Interval),
		FundManager:                 bt.Funding,
	}
	bt.Statistic = stats
	reports.Statistics = stats

	if !cfg.StrategySettings.DisableUSDTracking {
		var trackingPairs []trackingcurrencies.TrackingPair
		for i := range cfg.CurrencySettings {
			trackingPairs = append(trackingPairs, trackingcurrencies.TrackingPair{
				Exchange: cfg.CurrencySettings[i].ExchangeName,
				Asset:    cfg.CurrencySettings[i].Asset,
				Base:     cfg.CurrencySettings[i].Base,
				Quote:    cfg.CurrencySettings[i].Quote,
			})
		}
		trackingPairs, err = trackingcurrencies.CreateUSDTrackingPairs(trackingPairs, bt.exchangeManager)
		if err != nil {
			return nil, err
		}
	trackingPairCheck:
		for i := range trackingPairs {
			for j := range cfg.CurrencySettings {
				if cfg.CurrencySettings[j].ExchangeName == trackingPairs[i].Exchange &&
					cfg.CurrencySettings[j].Asset == trackingPairs[i].Asset &&
					cfg.CurrencySettings[j].Base == trackingPairs[i].Base &&
					cfg.CurrencySettings[j].Quote == trackingPairs[i].Quote {
					continue trackingPairCheck
				}
			}
			cfg.CurrencySettings = append(cfg.CurrencySettings, config.CurrencySettings{
				ExchangeName:    trackingPairs[i].Exchange,
				Asset:           trackingPairs[i].Asset,
				Base:            trackingPairs[i].Base,
				Quote:           trackingPairs[i].Quote,
				USDTrackingPair: true,
			})
		}
	}

	e, err := bt.setupExchangeSettings(cfg)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &e
	for i := range e.CurrencySettings {
		var lookup *portfolio.Settings

		lookup, err = p.SetupCurrencySettingsMap(&e.CurrencySettings[i])
		if err != nil {
			return nil, err
		}
		lookup.Fee = e.CurrencySettings[i].TakerFee
		lookup.Leverage = e.CurrencySettings[i].Leverage
		lookup.BuySideSizing = e.CurrencySettings[i].BuySide
		lookup.SellSideSizing = e.CurrencySettings[i].SellSide
		lookup.ComplianceManager = compliance.Manager{
			Snapshots: []compliance.Snapshot{},
		}
	}
	bt.Portfolio = p

	cfg.PrintSetting()

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	log.Infoln(log.BackTester, "setting exchange settings...")
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

		exchangeName := strings.ToLower(exch.GetName())
		bt.Datas.Setup()
		klineData, err := bt.loadData(cfg, exch, pair, a, cfg.CurrencySettings[i].USDTrackingPair)
		if err != nil {
			return resp, err
		}

		err = bt.Funding.AddUSDTrackingData(klineData)
		if err != nil &&
			!errors.Is(err, trackingcurrencies.ErrCurrencyDoesNotContainsUSD) &&
			!errors.Is(err, funding.ErrUSDTrackingDisabled) {
			return resp, err
		}

		if !cfg.CurrencySettings[i].USDTrackingPair {
			bt.Datas.SetDataForCurrency(exchangeName, a, pair, klineData)
			var makerFee, takerFee decimal.Decimal
			if cfg.CurrencySettings[i].MakerFee.GreaterThan(decimal.Zero) {
				makerFee = cfg.CurrencySettings[i].MakerFee
			}
			if cfg.CurrencySettings[i].TakerFee.GreaterThan(decimal.Zero) {
				takerFee = cfg.CurrencySettings[i].TakerFee
			}
			if makerFee.IsZero() || takerFee.IsZero() {
				var apiMakerFee, apiTakerFee decimal.Decimal
				apiMakerFee, apiTakerFee = getFees(context.TODO(), exch, pair)
				if makerFee.IsZero() {
					makerFee = apiMakerFee
				}
				if takerFee.IsZero() {
					takerFee = apiTakerFee
				}
			}

			if cfg.CurrencySettings[i].MaximumSlippagePercent.LessThan(decimal.Zero) {
				log.Warnf(log.BackTester, "invalid maximum slippage percent '%v'. Slippage percent is defined as a number, eg '100.00', defaulting to '%v'",
					cfg.CurrencySettings[i].MaximumSlippagePercent,
					slippage.DefaultMaximumSlippagePercent)
				cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
			}
			if cfg.CurrencySettings[i].MaximumSlippagePercent.IsZero() {
				cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
			}
			if cfg.CurrencySettings[i].MinimumSlippagePercent.LessThan(decimal.Zero) {
				log.Warnf(log.BackTester, "invalid minimum slippage percent '%v'. Slippage percent is defined as a number, eg '80.00', defaulting to '%v'",
					cfg.CurrencySettings[i].MinimumSlippagePercent,
					slippage.DefaultMinimumSlippagePercent)
				cfg.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
			}
			if cfg.CurrencySettings[i].MinimumSlippagePercent.IsZero() {
				cfg.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
			}
			if cfg.CurrencySettings[i].MaximumSlippagePercent.LessThan(cfg.CurrencySettings[i].MinimumSlippagePercent) {
				cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
			}

			realOrders := false
			if cfg.DataSettings.LiveData != nil {
				realOrders = cfg.DataSettings.LiveData.RealOrders
			}

			buyRule := exchange.MinMax{
				MinimumSize:  cfg.CurrencySettings[i].BuySide.MinimumSize,
				MaximumSize:  cfg.CurrencySettings[i].BuySide.MaximumSize,
				MaximumTotal: cfg.CurrencySettings[i].BuySide.MaximumTotal,
			}
			sellRule := exchange.MinMax{
				MinimumSize:  cfg.CurrencySettings[i].SellSide.MinimumSize,
				MaximumSize:  cfg.CurrencySettings[i].SellSide.MaximumSize,
				MaximumTotal: cfg.CurrencySettings[i].SellSide.MaximumTotal,
			}

			limits, err := exch.GetOrderExecutionLimits(a, pair)
			if err != nil && !errors.Is(err, gctorder.ErrExchangeLimitNotLoaded) {
				return resp, err
			}

			if limits != nil {
				if !cfg.CurrencySettings[i].CanUseExchangeLimits {
					log.Warnf(log.BackTester, "exchange %s order execution limits supported but disabled for %s %s, live results may differ",
						cfg.CurrencySettings[i].ExchangeName,
						pair,
						a)
					cfg.CurrencySettings[i].ShowExchangeOrderLimitWarning = true
				}
			}

			resp.CurrencySettings = append(resp.CurrencySettings, exchange.Settings{
				Exchange:            cfg.CurrencySettings[i].ExchangeName,
				MinimumSlippageRate: cfg.CurrencySettings[i].MinimumSlippagePercent,
				MaximumSlippageRate: cfg.CurrencySettings[i].MaximumSlippagePercent,
				Pair:                pair,
				Asset:               a,
				ExchangeFee:         takerFee,
				MakerFee:            takerFee,
				TakerFee:            makerFee,
				UseRealOrders:       realOrders,
				BuySide:             buyRule,
				SellSide:            sellRule,
				Leverage: exchange.Leverage{
					CanUseLeverage:                 cfg.CurrencySettings[i].Leverage.CanUseLeverage,
					MaximumLeverageRate:            cfg.CurrencySettings[i].Leverage.MaximumLeverageRate,
					MaximumOrdersWithLeverageRatio: cfg.CurrencySettings[i].Leverage.MaximumOrdersWithLeverageRatio,
				},
				Limits:                  limits,
				SkipCandleVolumeFitting: cfg.CurrencySettings[i].SkipCandleVolumeFitting,
				CanUseExchangeLimits:    cfg.CurrencySettings[i].CanUseExchangeLimits,
			})
		}
	}

	return resp, nil
}

func (bt *BackTest) loadExchangePairAssetBase(exch, base, quote, ass string) (gctexchange.IBotExchange, currency.Pair, asset.Item, error) {
	e, err := bt.exchangeManager.GetExchangeByName(exch)
	if err != nil {
		return nil, currency.EMPTYPAIR, "", err
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromStrings(base, quote)
	if err != nil {
		return nil, currency.EMPTYPAIR, "", err
	}

	var a asset.Item
	a, err = asset.New(ass)
	if err != nil {
		return nil, currency.EMPTYPAIR, "", err
	}

	exchangeBase := e.GetBase()
	if exchangeBase.ValidateAPICredentials(exchangeBase.GetDefaultCredentials()) != nil {
		log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", exchangeBase.Name)
	}

	fPair, err = exchangeBase.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, currency.EMPTYPAIR, "", err
	}
	return e, fPair, a, nil
}

// getFees will return an exchange's fee rate from GCT's wrapper function
func getFees(ctx context.Context, exch gctexchange.IBotExchange, fPair currency.Pair) (makerFee, takerFee decimal.Decimal) {
	fTakerFee, err := exch.GetFeeByType(ctx,
		&gctexchange.FeeBuilder{FeeType: gctexchange.OfflineTradeFee,
			Pair:          fPair,
			IsMaker:       false,
			PurchasePrice: 1,
			Amount:        1,
		})
	if err != nil {
		log.Errorf(log.BackTester, "Could not retrieve taker fee for %v. %v", exch.GetName(), err)
	}

	fMakerFee, err := exch.GetFeeByType(ctx,
		&gctexchange.FeeBuilder{
			FeeType:       gctexchange.OfflineTradeFee,
			Pair:          fPair,
			IsMaker:       true,
			PurchasePrice: 1,
			Amount:        1,
		})
	if err != nil {
		log.Errorf(log.BackTester, "Could not retrieve maker fee for %v. %v", exch.GetName(), err)
	}

	return decimal.NewFromFloat(fMakerFee), decimal.NewFromFloat(fTakerFee)
}

// loadData will create kline data from the sources defined in start config files. It can exist from databases, csv or API endpoints
// it can also be generated from trade data which will be converted into kline data
func (bt *BackTest) loadData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, isUSDTrackingPair bool) (*kline.DataFromKline, error) {
	if exch == nil {
		return nil, engine.ErrExchangeNotFound
	}
	b := exch.GetBase()
	if cfg.DataSettings.DatabaseData == nil &&
		cfg.DataSettings.LiveData == nil &&
		cfg.DataSettings.APIData == nil &&
		cfg.DataSettings.CSVData == nil {
		return nil, errNoDataSource
	}
	if (cfg.DataSettings.APIData != nil && cfg.DataSettings.DatabaseData != nil) ||
		(cfg.DataSettings.APIData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.APIData != nil && cfg.DataSettings.CSVData != nil) ||
		(cfg.DataSettings.DatabaseData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.CSVData != nil && cfg.DataSettings.LiveData != nil) ||
		(cfg.DataSettings.CSVData != nil && cfg.DataSettings.DatabaseData != nil) {
		return nil, errAmbiguousDataSource
	}

	dataType, err := common.DataTypeToInt(cfg.DataSettings.DataType)
	if err != nil {
		return nil, err
	}

	log.Infof(log.BackTester, "loading data for %v %v %v...\n", exch.GetName(), a, fPair)
	resp := &kline.DataFromKline{}
	switch {
	case cfg.DataSettings.CSVData != nil:
		if cfg.DataSettings.Interval <= 0 {
			return nil, errIntervalUnset
		}
		resp, err = csv.LoadData(
			dataType,
			cfg.DataSettings.CSVData.FullPath,
			strings.ToLower(exch.GetName()),
			cfg.DataSettings.Interval,
			fPair,
			a,
			isUSDTrackingPair)
		if err != nil {
			return nil, fmt.Errorf("%v. Please check your GoCryptoTrader configuration", err)
		}
		resp.Item.RemoveDuplicates()
		resp.Item.SortCandlesByTimestamp(false)
		resp.RangeHolder, err = gctkline.CalculateCandleDateRanges(
			resp.Item.Candles[0].Time,
			resp.Item.Candles[len(resp.Item.Candles)-1].Time.Add(cfg.DataSettings.Interval),
			gctkline.Interval(cfg.DataSettings.Interval),
			0,
		)
		if err != nil {
			return nil, err
		}
		resp.RangeHolder.SetHasDataFromCandles(resp.Item.Candles)
		summary := resp.RangeHolder.DataSummary(false)
		if len(summary) > 0 {
			log.Warnf(log.BackTester, "%v", summary)
		}
	case cfg.DataSettings.DatabaseData != nil:
		if cfg.DataSettings.DatabaseData.InclusiveEndDate {
			cfg.DataSettings.DatabaseData.EndDate = cfg.DataSettings.DatabaseData.EndDate.Add(cfg.DataSettings.Interval)
		}
		if cfg.DataSettings.DatabaseData.Path == "" {
			cfg.DataSettings.DatabaseData.Path = filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "database")
		}
		gctdatabase.DB.DataPath = filepath.Join(cfg.DataSettings.DatabaseData.Path)
		err = gctdatabase.DB.SetConfig(&cfg.DataSettings.DatabaseData.Config)
		if err != nil {
			return nil, err
		}
		err = bt.databaseManager.Start(&sync.WaitGroup{})
		if err != nil {
			return nil, err
		}
		defer func() {
			stopErr := bt.databaseManager.Stop()
			if stopErr != nil {
				log.Error(log.BackTester, stopErr)
			}
		}()
		resp, err = loadDatabaseData(cfg, exch.GetName(), fPair, a, dataType, isUSDTrackingPair)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve data from GoCryptoTrader database. Error: %v. Please ensure the database is setup correctly and has data before use", err)
		}

		resp.Item.RemoveDuplicates()
		resp.Item.SortCandlesByTimestamp(false)
		resp.RangeHolder, err = gctkline.CalculateCandleDateRanges(
			cfg.DataSettings.DatabaseData.StartDate,
			cfg.DataSettings.DatabaseData.EndDate,
			gctkline.Interval(cfg.DataSettings.Interval),
			0,
		)
		if err != nil {
			return nil, err
		}
		resp.RangeHolder.SetHasDataFromCandles(resp.Item.Candles)
		summary := resp.RangeHolder.DataSummary(false)
		if len(summary) > 0 {
			log.Warnf(log.BackTester, "%v", summary)
		}
	case cfg.DataSettings.APIData != nil:
		if cfg.DataSettings.APIData.InclusiveEndDate {
			cfg.DataSettings.APIData.EndDate = cfg.DataSettings.APIData.EndDate.Add(cfg.DataSettings.Interval)
		}
		resp, err = loadAPIData(
			cfg,
			exch,
			fPair,
			a,
			b.Features.Enabled.Kline.ResultLimit,
			dataType)
		if err != nil {
			return resp, err
		}
	case cfg.DataSettings.LiveData != nil:
		if isUSDTrackingPair {
			return nil, errLiveUSDTrackingNotSupported
		}
		if len(cfg.CurrencySettings) > 1 {
			return nil, errors.New("live data simulation only supports one currency")
		}
		err = loadLiveData(cfg, b)
		if err != nil {
			return nil, err
		}
		go bt.loadLiveDataLoop(
			resp,
			cfg,
			exch,
			fPair,
			a,
			dataType)
		return resp, nil
	}
	if resp == nil {
		return nil, fmt.Errorf("processing error, response returned nil")
	}

	err = b.ValidateKline(fPair, a, resp.Item.Interval)
	if err != nil {
		if dataType != common.DataTrade || !strings.EqualFold(err.Error(), "interval not supported") {
			return nil, err
		}
	}

	err = resp.Load()
	if err != nil {
		return nil, err
	}
	bt.Reports.AddKlineItem(&resp.Item)
	return resp, nil
}

func loadDatabaseData(cfg *config.Config, name string, fPair currency.Pair, a asset.Item, dataType int64, isUSDTrackingPair bool) (*kline.DataFromKline, error) {
	if cfg == nil || cfg.DataSettings.DatabaseData == nil {
		return nil, errors.New("nil config data received")
	}
	if cfg.DataSettings.Interval <= 0 {
		return nil, errIntervalUnset
	}

	return database.LoadData(
		cfg.DataSettings.DatabaseData.StartDate,
		cfg.DataSettings.DatabaseData.EndDate,
		cfg.DataSettings.Interval,
		strings.ToLower(name),
		dataType,
		fPair,
		a,
		isUSDTrackingPair)
}

func loadAPIData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, resultLimit uint32, dataType int64) (*kline.DataFromKline, error) {
	if cfg.DataSettings.Interval <= 0 {
		return nil, errIntervalUnset
	}
	dates, err := gctkline.CalculateCandleDateRanges(
		cfg.DataSettings.APIData.StartDate,
		cfg.DataSettings.APIData.EndDate,
		gctkline.Interval(cfg.DataSettings.Interval),
		resultLimit)
	if err != nil {
		return nil, err
	}
	candles, err := api.LoadData(context.TODO(),
		dataType,
		cfg.DataSettings.APIData.StartDate,
		cfg.DataSettings.APIData.EndDate,
		cfg.DataSettings.Interval,
		exch,
		fPair,
		a)
	if err != nil {
		return nil, fmt.Errorf("%v. Please check your GoCryptoTrader configuration", err)
	}
	dates.SetHasDataFromCandles(candles.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(log.BackTester, "%v", summary)
	}
	candles.FillMissingDataWithEmptyEntries(dates)
	candles.RemoveOutsideRange(cfg.DataSettings.APIData.StartDate, cfg.DataSettings.APIData.EndDate)
	return &kline.DataFromKline{
		Item:        *candles,
		RangeHolder: dates,
	}, nil
}

func loadLiveData(cfg *config.Config, base *gctexchange.Base) error {
	if cfg == nil || base == nil || cfg.DataSettings.LiveData == nil {
		return common.ErrNilArguments
	}
	if cfg.DataSettings.Interval <= 0 {
		return errIntervalUnset
	}

	if cfg.DataSettings.LiveData.APIKeyOverride != "" {
		base.API.SetKey(cfg.DataSettings.LiveData.APIKeyOverride)
	}
	if cfg.DataSettings.LiveData.APISecretOverride != "" {
		base.API.SetSecret(cfg.DataSettings.LiveData.APISecretOverride)
	}
	if cfg.DataSettings.LiveData.APIClientIDOverride != "" {
		base.API.SetClientID(cfg.DataSettings.LiveData.APIClientIDOverride)
	}
	if cfg.DataSettings.LiveData.API2FAOverride != "" {
		base.API.SetPEMKey(cfg.DataSettings.LiveData.API2FAOverride)
	}
	if cfg.DataSettings.LiveData.APISubAccountOverride != "" {
		base.API.SetSubAccount(cfg.DataSettings.LiveData.APISubAccountOverride)
	}

	validated := base.AreCredentialsValid(context.TODO())
	base.API.AuthenticatedSupport = validated
	if !validated && cfg.DataSettings.LiveData.RealOrders {
		log.Warn(log.BackTester, "invalid API credentials set, real orders set to false")
		cfg.DataSettings.LiveData.RealOrders = false
	}
	return nil
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (bt *BackTest) Run() error {
	log.Info(log.BackTester, "running backtester against pre-defined data")
dataLoadingIssue:
	for ev := bt.EventQueue.NextEvent(); ; ev = bt.EventQueue.NextEvent() {
		if ev == nil {
			dataHandlerMap := bt.Datas.GetAllData()
			for exchangeName, exchangeMap := range dataHandlerMap {
				for assetItem, assetMap := range exchangeMap {
					var hasProcessedData bool
					for currencyPair, dataHandler := range assetMap {
						d := dataHandler.Next()
						if d == nil {
							if !bt.hasHandledEvent {
								log.Errorf(log.BackTester, "Unable to perform `Next` for %v %v %v", exchangeName, assetItem, currencyPair)
							}
							break dataLoadingIssue
						}
						if bt.Strategy.UsingSimultaneousProcessing() && hasProcessedData {
							continue
						}
						bt.EventQueue.AppendEvent(d)
						hasProcessedData = true
					}
				}
			}
		}
		if ev != nil {
			err := bt.handleEvent(ev)
			if err != nil {
				return err
			}
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
func (bt *BackTest) handleEvent(ev common.EventHandler) error {
	funds, err := bt.Funding.GetFundingForEvent(ev)
	if err != nil {
		return err
	}
	switch eType := ev.(type) {
	case common.DataEventHandler:
		if bt.Strategy.UsingSimultaneousProcessing() {
			err = bt.processSimultaneousDataEvents()
			if err != nil {
				return err
			}
			bt.Funding.CreateSnapshot(ev.GetTime())
			return nil
		}
		err = bt.processSingleDataEvent(eType, funds)
		if err != nil {
			return err
		}
		bt.Funding.CreateSnapshot(ev.GetTime())
		return nil
	case signal.Event:
		bt.processSignalEvent(eType, funds)
	case order.Event:
		bt.processOrderEvent(eType, funds)
	case fill.Event:
		bt.processFillEvent(eType, funds)
	default:
		return fmt.Errorf("%w %v received, could not process",
			errUnhandledDatatype,
			ev)
	}

	return nil
}

func (bt *BackTest) processSingleDataEvent(ev common.DataEventHandler, funds funding.IPairReader) error {
	err := bt.updateStatsForDataEvent(ev, funds)
	if err != nil {
		return err
	}
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	s, err := bt.Strategy.OnSignal(d, bt.Funding, bt.Portfolio)
	if err != nil {
		if errors.Is(err, base.ErrTooMuchBadData) {
			// too much bad data is a severe error and backtesting must cease
			return err
		}
		log.Error(log.BackTester, err)
		return nil
	}
	err = bt.Statistic.SetEventForOffset(s)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	bt.EventQueue.AppendEvent(s)

	return nil
}

// processSimultaneousDataEvents determines what signal events are generated and appended
// to the event queue based on whether it is running a multi-currency consideration strategy order not
//
// for multi-currency-consideration it will pass all currency datas to the strategy for it to determine what
// currencies to act upon
//
// for non-multi-currency-consideration strategies, it will simply process every currency individually
// against the strategy and generate signals
func (bt *BackTest) processSimultaneousDataEvents() error {
	var dataEvents []data.Handler
	dataHandlerMap := bt.Datas.GetAllData()
	for _, exchangeMap := range dataHandlerMap {
		for _, assetMap := range exchangeMap {
			for _, dataHandler := range assetMap {
				latestData := dataHandler.Latest()
				funds, err := bt.Funding.GetFundingForEAP(latestData.GetExchange(), latestData.GetAssetType(), latestData.Pair())
				if err != nil {
					return err
				}
				err = bt.updateStatsForDataEvent(latestData, funds)
				if err != nil && err == statistics.ErrAlreadyProcessed {
					continue
				}
				dataEvents = append(dataEvents, dataHandler)
			}
		}
	}
	signals, err := bt.Strategy.OnSimultaneousSignals(dataEvents, bt.Funding, bt.Portfolio)
	if err != nil {
		if errors.Is(err, base.ErrTooMuchBadData) {
			// too much bad data is a severe error and backtesting must cease
			return err
		}
		log.Error(log.BackTester, err)
		return nil
	}
	for i := range signals {
		err = bt.Statistic.SetEventForOffset(signals[i])
		if err != nil {
			log.Error(log.BackTester, err)
		}
		bt.EventQueue.AppendEvent(signals[i])
	}
	return nil
}

// updateStatsForDataEvent makes various systems aware of price movements from
// data events
func (bt *BackTest) updateStatsForDataEvent(ev common.DataEventHandler, funds funding.IPairReader) error {
	// update statistics with the latest price
	err := bt.Statistic.SetupEventForTime(ev)
	if err != nil {
		if err == statistics.ErrAlreadyProcessed {
			return err
		}
		log.Error(log.BackTester, err)
	}
	// update portfolio manager with the latest price
	err = bt.Portfolio.UpdateHoldings(ev, funds)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	return nil
}

// processSignalEvent receives an event from the strategy for processing under the portfolio
func (bt *BackTest) processSignalEvent(ev signal.Event, funds funding.IPairReserver) {
	cs, err := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}
	var o *order.Order
	o, err = bt.Portfolio.OnSignal(ev, &cs, funds)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}
	err = bt.Statistic.SetEventForOffset(o)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	bt.EventQueue.AppendEvent(o)
}

func (bt *BackTest) processOrderEvent(ev order.Event, funds funding.IPairReleaser) {
	d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
	f, err := bt.Exchange.ExecuteOrder(ev, d, bt.orderManager, funds)
	if err != nil {
		if f == nil {
			log.Errorf(log.BackTester, "fill event should always be returned, please fix, %v", err)
			return
		}
		log.Errorf(log.BackTester, "%v %v %v %v", f.GetExchange(), f.GetAssetType(), f.Pair(), err)
	}
	err = bt.Statistic.SetEventForOffset(f)
	if err != nil {
		log.Error(log.BackTester, err)
	}
	bt.EventQueue.AppendEvent(f)
}

func (bt *BackTest) processFillEvent(ev fill.Event, funds funding.IPairReader) {
	t, err := bt.Portfolio.OnFill(ev, funds)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}

	err = bt.Statistic.SetEventForOffset(t)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	var holding *holdings.Holding
	holding, err = bt.Portfolio.ViewHoldingAtTimePeriod(ev)
	if err != nil {
		log.Error(log.BackTester, err)
	}

	err = bt.Statistic.AddHoldingsForTime(holding)
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
	log.Info(log.BackTester, "running backtester against live data")
	timeoutTimer := time.NewTimer(time.Minute * 5)
	// a frequent timer so that when a new candle is released by an exchange
	// that it can be processed quickly
	processEventTicker := time.NewTicker(time.Second)
	doneARun := false
	for {
		select {
		case <-bt.shutdown:
			return nil
		case <-timeoutTimer.C:
			return errLiveDataTimeout
		case <-processEventTicker.C:
			for e := bt.EventQueue.NextEvent(); ; e = bt.EventQueue.NextEvent() {
				if e == nil {
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
					de := d.Next()
					if de == nil {
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
				timeoutTimer = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func (bt *BackTest) loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) {
	startDate := time.Now().Add(-cfg.DataSettings.Interval * 2)
	dates, err := gctkline.CalculateCandleDateRanges(
		startDate,
		startDate.AddDate(1, 0, 0),
		gctkline.Interval(cfg.DataSettings.Interval),
		0)
	if err != nil {
		log.Errorf(log.BackTester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		log.Errorf(log.BackTester, "%v. Please check your GoCryptoTrader configuration", err)
		return
	}
	dates.SetHasDataFromCandles(candles.Candles)
	resp.RangeHolder = dates
	resp.Item = *candles

	loadNewDataTimer := time.NewTimer(time.Second * 5)
	for {
		select {
		case <-bt.shutdown:
			return
		case <-loadNewDataTimer.C:
			log.Infof(log.BackTester, "fetching data for %v %v %v %v", exch.GetName(), a, fPair, cfg.DataSettings.Interval)
			loadNewDataTimer.Reset(time.Second * 15)
			err = bt.loadLiveData(resp, cfg, exch, fPair, a, dataType)
			if err != nil {
				log.Error(log.BackTester, err)
				return
			}
		}
	}
}

func (bt *BackTest) loadLiveData(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, dataType int64) error {
	if resp == nil {
		return errNilData
	}
	if cfg == nil {
		return errNilConfig
	}
	if exch == nil {
		return errNilExchange
	}
	candles, err := live.LoadData(context.TODO(),
		exch,
		dataType,
		cfg.DataSettings.Interval,
		fPair,
		a)
	if err != nil {
		return err
	}
	if len(candles.Candles) == 0 {
		return nil
	}
	resp.AppendResults(candles)
	bt.Reports.UpdateItem(&resp.Item)
	log.Info(log.BackTester, "sleeping for 30 seconds before checking for new candle data")
	return nil
}

// Stop shuts down the live data loop
func (bt *BackTest) Stop() {
	close(bt.shutdown)
}
