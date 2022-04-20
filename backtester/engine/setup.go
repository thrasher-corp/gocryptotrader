package engine

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
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/api"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/csv"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/database"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies/base"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding/trackingcurrencies"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/convert"
	gctconfig "github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	gctdatabase "github.com/thrasher-corp/gocryptotrader/database"
	"github.com/thrasher-corp/gocryptotrader/engine"
	gctexchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(cfg *config.Config, templatePath, output string, verbose bool) (*BackTest, error) {
	log.Infoln(common.SubLoggers[common.Setup], "loading config...")
	if cfg == nil {
		return nil, errNilConfig
	}
	var err error
	bt := New()
	bt.exchangeManager = engine.SetupExchangeManager()
	bt.orderManager, err = engine.SetupOrderManager(bt.exchangeManager, &engine.CommunicationManager{}, &sync.WaitGroup{}, false, false)
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

	funds, err := funding.SetupFundingManager(
		bt.exchangeManager,
		cfg.FundingSettings.UseExchangeLevelFunding,
		cfg.StrategySettings.DisableUSDTracking,
	)
	if err != nil {
		return nil, err
	}

	if cfg.FundingSettings.UseExchangeLevelFunding {
		for i := range cfg.FundingSettings.ExchangeLevelFunding {
			var a asset.Item
			a, err = asset.New(cfg.FundingSettings.ExchangeLevelFunding[i].Asset)
			if err != nil {
				return nil, err
			}
			cq := currency.NewCode(cfg.FundingSettings.ExchangeLevelFunding[i].Currency)
			var item *funding.Item
			item, err = funding.CreateItem(cfg.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
				a,
				cq,
				cfg.FundingSettings.ExchangeLevelFunding[i].InitialFunds,
				cfg.FundingSettings.ExchangeLevelFunding[i].TransferFee)
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
		var conf *gctconfig.Exchange
		conf, err = exch.GetDefaultConfig()
		if err != nil {
			return nil, err
		}
		conf.Enabled = true
		conf.WebsocketTrafficTimeout = time.Second
		conf.Websocket = convert.BoolPtr(false)
		conf.WebsocketResponseCheckTimeout = time.Second
		conf.WebsocketResponseMaxLimit = time.Second
		conf.Verbose = verbose
		err = exch.Setup(conf)
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
		var avail, enabled currency.Pairs
		avail, err = exch.GetAvailablePairs(a)
		if err != nil {
			return nil, fmt.Errorf("could not format currency %v, %w", curr, err)
		}
		enabled, err = exch.GetEnabledPairs(a)
		if err != nil {
			return nil, fmt.Errorf("could not format currency %v, %w", curr, err)
		}

		avail = avail.Add(curr)
		enabled = enabled.Add(curr)
		err = exch.SetPairs(enabled, a, true)
		if err != nil {
			return nil, fmt.Errorf("could not format currency %v, %w", curr, err)
		}
		err = exch.SetPairs(avail, a, false)
		if err != nil {
			return nil, fmt.Errorf("could not format currency %v, %w", curr, err)
		}

		portSet := &risk.CurrencySettings{
			MaximumHoldingRatio: cfg.CurrencySettings[i].MaximumHoldingsRatio,
		}
		if cfg.CurrencySettings[i].FuturesDetails != nil {
			portSet.MaximumOrdersWithLeverageRatio = cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio
			portSet.MaxLeverageRate = cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate

		}
		portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][curr] = portSet
		if cfg.CurrencySettings[i].MakerFee != nil &&
			cfg.CurrencySettings[i].TakerFee != nil &&
			cfg.CurrencySettings[i].MakerFee.GreaterThan(*cfg.CurrencySettings[i].TakerFee) {
			log.Warnf(common.SubLoggers[common.Setup], "maker fee '%v' should not exceed taker fee '%v'. Please review config",
				cfg.CurrencySettings[i].MakerFee,
				cfg.CurrencySettings[i].TakerFee)
		}

		var baseItem, quoteItem, futureItem *funding.Item
		if cfg.FundingSettings.UseExchangeLevelFunding {
			switch {
			case a == asset.Spot:
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
			case a.IsFutures():
				// setup contract items
				c := funding.CreateFuturesCurrencyCode(b, q)
				futureItem, err = funding.CreateItem(cfg.CurrencySettings[i].ExchangeName,
					a,
					c,
					decimal.Zero,
					decimal.Zero)
				if err != nil {
					return nil, err
				}

				var collateralCurrency currency.Code
				collateralCurrency, _, err = exch.GetCollateralCurrencyForContract(a, currency.NewPair(b, q))
				if err != nil {
					return nil, err
				}

				err = funds.LinkCollateralCurrency(futureItem, collateralCurrency)
				if err != nil {
					return nil, err
				}
				err = funds.AddItem(futureItem)
				if err != nil {
					return nil, err
				}
			default:
				return nil, fmt.Errorf("%w: %v unsupported", errInvalidConfigAsset, a)
			}
		} else {
			var bFunds, qFunds decimal.Decimal
			if cfg.CurrencySettings[i].SpotDetails != nil {
				if cfg.CurrencySettings[i].SpotDetails.InitialBaseFunds != nil {
					bFunds = *cfg.CurrencySettings[i].SpotDetails.InitialBaseFunds
				}
				if cfg.CurrencySettings[i].SpotDetails.InitialQuoteFunds != nil {
					qFunds = *cfg.CurrencySettings[i].SpotDetails.InitialQuoteFunds
				}
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
			var pair *funding.SpotPair
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
		err = p.SetupCurrencySettingsMap(&e.CurrencySettings[i])
		if err != nil {
			return nil, err
		}
	}
	bt.Portfolio = p

	cfg.PrintSetting()

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	log.Infoln(common.SubLoggers[common.Setup], "setting exchange settings...")
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

		if cfg.CurrencySettings[i].USDTrackingPair {
			continue
		}

		bt.Datas.SetDataForCurrency(exchangeName, a, pair, klineData)

		var makerFee, takerFee decimal.Decimal
		if cfg.CurrencySettings[i].MakerFee != nil && cfg.CurrencySettings[i].MakerFee.GreaterThan(decimal.Zero) {
			makerFee = *cfg.CurrencySettings[i].MakerFee
		}
		if cfg.CurrencySettings[i].TakerFee != nil && cfg.CurrencySettings[i].TakerFee.GreaterThan(decimal.Zero) {
			takerFee = *cfg.CurrencySettings[i].TakerFee
		}
		if cfg.CurrencySettings[i].TakerFee == nil || cfg.CurrencySettings[i].MakerFee == nil {
			var apiMakerFee, apiTakerFee decimal.Decimal
			apiMakerFee, apiTakerFee = getFees(context.TODO(), exch, pair)
			if cfg.CurrencySettings[i].MakerFee == nil {
				makerFee = apiMakerFee
			}
			if cfg.CurrencySettings[i].TakerFee == nil {
				takerFee = apiTakerFee
			}
		}

		if cfg.CurrencySettings[i].MaximumSlippagePercent.LessThan(decimal.Zero) {
			log.Warnf(common.SubLoggers[common.Setup], "invalid maximum slippage percent '%v'. Slippage percent is defined as a number, eg '100.00', defaulting to '%v'",
				cfg.CurrencySettings[i].MaximumSlippagePercent,
				slippage.DefaultMaximumSlippagePercent)
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MaximumSlippagePercent.IsZero() {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MinimumSlippagePercent.LessThan(decimal.Zero) {
			log.Warnf(common.SubLoggers[common.Setup], "invalid minimum slippage percent '%v'. Slippage percent is defined as a number, eg '80.00', defaulting to '%v'",
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
				log.Warnf(common.SubLoggers[common.Setup], "exchange %s order execution limits supported but disabled for %s %s, live results may differ",
					cfg.CurrencySettings[i].ExchangeName,
					pair,
					a)
				cfg.CurrencySettings[i].ShowExchangeOrderLimitWarning = true
			}
		}
		var lev exchange.Leverage
		if cfg.CurrencySettings[i].FuturesDetails != nil {
			lev = exchange.Leverage{
				CanUseLeverage:                 cfg.CurrencySettings[i].FuturesDetails.Leverage.CanUseLeverage,
				MaximumLeverageRate:            cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate,
				MaximumOrdersWithLeverageRatio: cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio,
			}
		}
		resp.CurrencySettings = append(resp.CurrencySettings, exchange.Settings{
			Exchange:                  exch,
			MinimumSlippageRate:       cfg.CurrencySettings[i].MinimumSlippagePercent,
			MaximumSlippageRate:       cfg.CurrencySettings[i].MaximumSlippagePercent,
			Pair:                      pair,
			Asset:                     a,
			ExchangeFee:               takerFee,
			MakerFee:                  takerFee,
			TakerFee:                  makerFee,
			UseRealOrders:             realOrders,
			BuySide:                   buyRule,
			SellSide:                  sellRule,
			Leverage:                  lev,
			Limits:                    limits,
			SkipCandleVolumeFitting:   cfg.CurrencySettings[i].SkipCandleVolumeFitting,
			CanUseExchangeLimits:      cfg.CurrencySettings[i].CanUseExchangeLimits,
			UseExchangePNLCalculation: cfg.CurrencySettings[i].UseExchangePNLCalculation,
		})
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
		log.Warnf(common.SubLoggers[common.Setup], "no credentials set for %v, this is theoretical only", exchangeBase.Name)
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
		log.Errorf(common.SubLoggers[common.Setup], "Could not retrieve taker fee for %v. %v", exch.GetName(), err)
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
		log.Errorf(common.SubLoggers[common.Setup], "Could not retrieve maker fee for %v. %v", exch.GetName(), err)
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

	log.Infof(common.SubLoggers[common.Setup], "loading data for %v %v %v...\n", exch.GetName(), a, fPair)
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
			log.Warnf(common.SubLoggers[common.Setup], "%v", summary)
		}
	case cfg.DataSettings.DatabaseData != nil:
		if cfg.DataSettings.DatabaseData.InclusiveEndDate {
			cfg.DataSettings.DatabaseData.EndDate = cfg.DataSettings.DatabaseData.EndDate.Add(cfg.DataSettings.Interval)
		}
		if cfg.DataSettings.DatabaseData.Path == "" {
			cfg.DataSettings.DatabaseData.Path = filepath.Join(gctcommon.GetDefaultDataDir(runtime.GOOS), "database")
		}
		gctdatabase.DB.DataPath = cfg.DataSettings.DatabaseData.Path
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
				log.Error(common.SubLoggers[common.Setup], stopErr)
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
			log.Warnf(common.SubLoggers[common.Setup], "%v", summary)
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

	if a.IsFutures() {
		// returning the collateral currency along with using the
		// fPair base creates a pair that links the futures contract to
		// is underlying pair
		// eg BTC-PERP on FTX has a collateral currency of USD
		// taking the BTC base and USD as quote, allows linking
		// BTC-USD and BTC-PERP
		var curr currency.Code
		curr, _, err = exch.GetCollateralCurrencyForContract(a, fPair)
		if err != nil {
			return resp, err
		}
		resp.Item.UnderlyingPair = currency.NewPair(fPair.Base, curr)
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
		log.Warnf(common.SubLoggers[common.Setup], "%v", summary)
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
		log.Warn(common.SubLoggers[common.Setup], "invalid API credentials set, real orders set to false")
		cfg.DataSettings.LiveData.RealOrders = false
	}
	return nil
}
