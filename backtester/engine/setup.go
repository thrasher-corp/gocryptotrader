package engine

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/api"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/csv"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/database"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/eventholder"
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

// NewBacktester returns a new BackTest instance
func NewBacktester() (*BackTest, error) {
	bt := &BackTest{
		shutdown:                 make(chan struct{}),
		DataHolder:               &data.HandlerPerCurrency{},
		EventQueue:               &eventholder.Holder{},
		hasProcessedDataAtOffset: make(map[int64]bool),
	}
	var err error
	bt.exchangeManager = engine.SetupExchangeManager()
	bt.orderManager, err = engine.SetupOrderManager(
		bt.exchangeManager,
		&engine.CommunicationManager{},
		&sync.WaitGroup{},
		false,
		false)
	if err != nil {
		return nil, err
	}
	err = bt.orderManager.Start()
	if err != nil {
		return nil, err
	}
	return bt, nil
}

// SetupFromConfig takes a strategy config and configures a backtester variable to run
func (bt *BackTest) SetupFromConfig(cfg *config.Config, templatePath, output string, verbose bool) error {
	log.Infoln(common.Setup, "loading config...")
	if cfg == nil {
		return errNilConfig
	}
	var err error
	if cfg.DataSettings.DatabaseData != nil {
		bt.databaseManager, err = engine.SetupDatabaseConnectionManager(&cfg.DataSettings.DatabaseData.Config)
		if err != nil {
			return err
		}
	}
	bt.verbose = verbose
	bt.DataHolder.Setup()
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
		return err
	}

	if cfg.FundingSettings.UseExchangeLevelFunding {
		for i := range cfg.FundingSettings.ExchangeLevelFunding {
			a := cfg.FundingSettings.ExchangeLevelFunding[i].Asset
			cq := cfg.FundingSettings.ExchangeLevelFunding[i].Currency
			var item *funding.Item
			item, err = funding.CreateItem(cfg.FundingSettings.ExchangeLevelFunding[i].ExchangeName,
				a,
				cq,
				cfg.FundingSettings.ExchangeLevelFunding[i].InitialFunds,
				cfg.FundingSettings.ExchangeLevelFunding[i].TransferFee)
			if err != nil {
				return err
			}
			err = funds.AddItem(item)
			if err != nil {
				return err
			}
		}
	}
	var wg sync.WaitGroup
	for i := range cfg.CurrencySettings {
		var exch gctexchange.IBotExchange
		exch, err = bt.exchangeManager.GetExchangeByName(cfg.CurrencySettings[i].ExchangeName)
		if err != nil {
			if errors.Is(err, engine.ErrExchangeNotFound) {
				exch, err = bt.exchangeManager.NewExchangeByName(cfg.CurrencySettings[i].ExchangeName)
				if err != nil {
					return err
				}
				exch.SetDefaults()
				exchBase := exch.GetBase()
				exchBase.Verbose = cfg.DataSettings.VerboseExchangeRequests
				exchBase.Config = &gctconfig.Exchange{
					Name:           exchBase.Name,
					HTTPTimeout:    gctexchange.DefaultHTTPTimeout,
					BaseCurrencies: exchBase.BaseCurrencies,
					CurrencyPairs:  &currency.PairsManager{},
				}
				err = exch.UpdateTradablePairs(context.TODO(), true)
				if err != nil {
					return err
				}
				assets := exchBase.CurrencyPairs.GetAssetTypes(false)
				for x := range assets {
					var pairs currency.Pairs
					pairs, err = exchBase.CurrencyPairs.GetPairs(assets[x], false)
					if err != nil {
						return err
					}
					exchBase.CurrencyPairs.StorePairs(assets[x], pairs, true)
				}
				err = exch.Start(&wg)
				if err != nil {
					return err
				}
				bt.exchangeManager.Add(exch)
			} else {
				return err
			}
		}

		wg.Wait()
		exchBase := exch.GetBase()
		exchangeAsset, ok := exchBase.CurrencyPairs.Pairs[cfg.CurrencySettings[i].Asset]
		if !ok {
			return fmt.Errorf("%v %v %w", cfg.CurrencySettings[i].ExchangeName, cfg.CurrencySettings[i].Asset, asset.ErrNotSupported)
		}
		exchangeAsset.AssetEnabled = convert.BoolPtr(true)
		cp := currency.NewPair(cfg.CurrencySettings[i].Base, cfg.CurrencySettings[i].Quote).Format(exchangeAsset.RequestFormat.Delimiter, exchangeAsset.RequestFormat.Uppercase)
		exchangeAsset.Available = exchangeAsset.Available.Add(cp)
		exchangeAsset.Enabled = exchangeAsset.Enabled.Add(cp)
		exchBase.Verbose = verbose
		exchBase.CurrencyPairs.Pairs[cfg.CurrencySettings[i].Asset] = exchangeAsset
	}

	portfolioRisk := &risk.Risk{
		CurrencySettings: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*risk.CurrencySettings),
	}

	for i := range cfg.CurrencySettings {
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName] = make(map[asset.Item]map[*currency.Item]map[*currency.Item]*risk.CurrencySettings)
		}
		a := cfg.CurrencySettings[i].Asset
		if !a.IsValid() {
			return fmt.Errorf(
				"%w for %v %v %v-%v. Err %v",
				asset.ErrNotSupported,
				cfg.CurrencySettings[i].ExchangeName,
				cfg.CurrencySettings[i].Asset,
				cfg.CurrencySettings[i].Base,
				cfg.CurrencySettings[i].Quote,
				err)
		}
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a] = make(map[*currency.Item]map[*currency.Item]*risk.CurrencySettings)
		}
		if portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][cfg.CurrencySettings[i].Base.Item] == nil {
			portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][cfg.CurrencySettings[i].Base.Item] = make(map[*currency.Item]*risk.CurrencySettings)
		}
		var curr currency.Pair
		var b, q currency.Code
		b = cfg.CurrencySettings[i].Base
		q = cfg.CurrencySettings[i].Quote
		curr = currency.NewPair(b, q).Format("", false)
		var exch gctexchange.IBotExchange
		exch, err = bt.exchangeManager.GetExchangeByName(cfg.CurrencySettings[i].ExchangeName)
		if err != nil {
			return err
		}
		portSet := &risk.CurrencySettings{
			MaximumHoldingRatio: cfg.CurrencySettings[i].MaximumHoldingsRatio,
		}
		if cfg.CurrencySettings[i].FuturesDetails != nil {
			portSet.MaximumOrdersWithLeverageRatio = cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrdersWithLeverageRatio
			portSet.MaxLeverageRate = cfg.CurrencySettings[i].FuturesDetails.Leverage.MaximumOrderLeverageRate
		}
		portfolioRisk.CurrencySettings[cfg.CurrencySettings[i].ExchangeName][a][curr.Base.Item][curr.Quote.Item] = portSet
		if cfg.CurrencySettings[i].MakerFee != nil &&
			cfg.CurrencySettings[i].TakerFee != nil &&
			cfg.CurrencySettings[i].MakerFee.GreaterThan(*cfg.CurrencySettings[i].TakerFee) {
			log.Warnf(common.Setup, "maker fee '%v' should not exceed taker fee '%v'. Please review config",
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
					return err
				}
				quoteItem, err = funding.CreateItem(cfg.CurrencySettings[i].ExchangeName,
					a,
					q,
					decimal.Zero,
					decimal.Zero)
				if err != nil {
					return err
				}
				err = funds.AddItem(baseItem)
				if err != nil && !errors.Is(err, funding.ErrAlreadyExists) {
					return err
				}
				err = funds.AddItem(quoteItem)
				if err != nil && !errors.Is(err, funding.ErrAlreadyExists) {
					return err
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
					return err
				}

				var collateralCurrency currency.Code
				collateralCurrency, _, err = exch.GetCollateralCurrencyForContract(a, currency.NewPair(b, q))
				if err != nil {
					return err
				}

				err = funds.LinkCollateralCurrency(futureItem, collateralCurrency)
				if err != nil {
					return err
				}
				err = funds.AddItem(futureItem)
				if err != nil {
					return err
				}
			default:
				return fmt.Errorf("%w: %v", asset.ErrNotSupported, a)
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
				return err
			}
			quoteItem, err = funding.CreateItem(
				cfg.CurrencySettings[i].ExchangeName,
				a,
				curr.Quote,
				qFunds,
				decimal.Zero)
			if err != nil {
				return err
			}
			var pair *funding.SpotPair
			pair, err = funding.CreatePair(baseItem, quoteItem)
			if err != nil {
				return err
			}
			err = funds.AddPair(pair)
			if err != nil {
				return err
			}
		}
	}

	bt.Funding = funds
	if cfg.DataSettings.LiveData != nil {
		err = bt.SetupLiveDataHandler(cfg.DataSettings.LiveData.NewEventTimeout, cfg.DataSettings.LiveData.DataCheckTimer, verbose)
		if err != nil {
			return err
		}
	}

	var p *portfolio.Portfolio
	p, err = portfolio.Setup(sizeManager, portfolioRisk, cfg.StatisticSettings.RiskFreeRate)
	if err != nil {
		return err
	}

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategySettings.Name, cfg.StrategySettings.SimultaneousSignalProcessing)
	if err != nil {
		return err
	}
	bt.Strategy.SetDefaults()
	if cfg.StrategySettings.CustomSettings != nil {
		err = bt.Strategy.SetCustomSettings(cfg.StrategySettings.CustomSettings)
		if err != nil && !errors.Is(err, base.ErrCustomSettingsUnsupported) {
			return err
		}
	}
	stats := &statistics.Statistic{
		StrategyName:                bt.Strategy.Name(),
		StrategyNickname:            cfg.Nickname,
		StrategyDescription:         bt.Strategy.Description(),
		StrategyGoal:                cfg.Goal,
		ExchangeAssetPairStatistics: make(map[string]map[asset.Item]map[*currency.Item]map[*currency.Item]*statistics.CurrencyPairStatistic),
		RiskFreeRate:                cfg.StatisticSettings.RiskFreeRate,
		CandleInterval:              cfg.DataSettings.Interval,
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
			return err
		}
	trackingPairCheck:
		for i := range trackingPairs {
			for j := range cfg.CurrencySettings {
				if cfg.CurrencySettings[j].ExchangeName == trackingPairs[i].Exchange &&
					cfg.CurrencySettings[j].Asset == trackingPairs[i].Asset &&
					cfg.CurrencySettings[j].Base.Equal(trackingPairs[i].Base) &&
					cfg.CurrencySettings[j].Quote.Equal(trackingPairs[i].Quote) {
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
		return err
	}

	bt.Exchange = &e
	for i := range e.CurrencySettings {
		err = p.SetupCurrencySettingsMap(&e.CurrencySettings[i])
		if err != nil {
			return err
		}
	}
	bt.Portfolio = p

	cfg.PrintSetting()
	if bt.LiveDataHandler != nil {
		if cfg.DataSettings.LiveData.RealOrders {
			// reset funding and use funds from account info
			exchanges, err := bt.exchangeManager.GetExchanges()
			if err != nil {
				return err
			}
			for x := range exchanges {
				assets := exchanges[x].GetAssetTypes(false)
				for y := range assets {
					acc, err := exchanges[x].FetchAccountInfo(context.TODO(), assets[y])
					if err != nil {
						return err
					}
					for z := range acc.Accounts {
						for i := range acc.Accounts[z].Currencies {
							err = bt.Funding.SetFunding(exchanges[x].GetName(), assets[y], acc.Accounts[z].Currencies[i].CurrencyName, decimal.NewFromFloat(acc.Accounts[z].Currencies[i].AvailableWithoutBorrow))
							if err != nil {
								return err
							}
						}
					}
				}
			}
		}
		return bt.LiveDataHandler.Start()
	}

	return nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	log.Infoln(common.Setup, "setting exchange settings...")
	resp := exchange.Exchange{}

	realOrders := false
	if cfg.DataSettings.LiveData != nil {
		realOrders = cfg.DataSettings.LiveData.RealOrders
	}

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
		klineData, err := bt.loadData(cfg, exch, pair, a, cfg.CurrencySettings[i].USDTrackingPair)
		if err != nil {
			return resp, err
		}
		if bt.LiveDataHandler == nil {
			err = bt.Funding.AddUSDTrackingData(klineData)
			if err != nil &&
				!errors.Is(err, trackingcurrencies.ErrCurrencyDoesNotContainsUSD) &&
				!errors.Is(err, funding.ErrUSDTrackingDisabled) {
				return resp, err
			}

			if cfg.CurrencySettings[i].USDTrackingPair {
				continue
			}

			bt.DataHolder.SetDataForCurrency(exchangeName, a, pair, klineData)
		}
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
				cfg.CurrencySettings[i].MakerFee = &makerFee
				cfg.CurrencySettings[i].UsingExchangeMakerFee = true
			}
			if cfg.CurrencySettings[i].TakerFee == nil {
				takerFee = apiTakerFee
				cfg.CurrencySettings[i].TakerFee = &takerFee
				cfg.CurrencySettings[i].UsingExchangeTakerFee = true
			}
		}

		if cfg.CurrencySettings[i].MaximumSlippagePercent.LessThan(decimal.Zero) {
			log.Warnf(common.Setup, "invalid maximum slippage percent '%v'. Slippage percent is defined as a number, eg '100.00', defaulting to '%v'",
				cfg.CurrencySettings[i].MaximumSlippagePercent,
				slippage.DefaultMaximumSlippagePercent)
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MaximumSlippagePercent.IsZero() {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MinimumSlippagePercent.LessThan(decimal.Zero) {
			log.Warnf(common.Setup, "invalid minimum slippage percent '%v'. Slippage percent is defined as a number, eg '80.00', defaulting to '%v'",
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

		if limits != (gctorder.MinMaxLevel{}) {
			if !cfg.CurrencySettings[i].CanUseExchangeLimits {
				log.Warnf(common.Setup, "exchange %s order execution limits supported but disabled for %s %s, live results may differ",
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
			MakerFee:                  makerFee,
			TakerFee:                  takerFee,
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

func (bt *BackTest) loadExchangePairAssetBase(exch string, base, quote currency.Code, ai asset.Item) (gctexchange.IBotExchange, currency.Pair, asset.Item, error) {
	e, err := bt.exchangeManager.GetExchangeByName(exch)
	if err != nil {
		return nil, currency.EMPTYPAIR, asset.Empty, err
	}

	var cp, fPair currency.Pair
	cp = currency.NewPair(base, quote)

	exchangeBase := e.GetBase()
	fPair, err = exchangeBase.FormatExchangeCurrency(cp, ai)
	if err != nil {
		return nil, currency.EMPTYPAIR, asset.Empty, err
	}
	return e, fPair, ai, nil
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
		log.Errorf(common.Setup, "Could not retrieve taker fee for %v. %v", exch.GetName(), err)
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
		log.Errorf(common.Setup, "Could not retrieve maker fee for %v. %v", exch.GetName(), err)
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

	log.Infof(common.Setup, "loading data for %v %v %v...\n", exch.GetName(), a, fPair)
	resp := &kline.DataFromKline{}
	underlyingPair := currency.EMPTYPAIR
	if a.IsFutures() {
		// returning the collateral currency along with using the
		// fPair base creates a pair that links the futures contract to
		// is underlyingPair pair
		// eg BTC-PERP on FTX has a collateral currency of USD
		// taking the BTC base and USD as quote, allows linking
		// BTC-USD and BTC-PERP
		var curr currency.Code
		curr, _, err = exch.GetCollateralCurrencyForContract(a, fPair)
		if err != nil {
			return resp, err
		}
		underlyingPair = currency.NewPair(fPair.Base, curr)
	}

	switch {
	case cfg.DataSettings.CSVData != nil:
		if cfg.DataSettings.Interval <= 0 {
			return nil, errIntervalUnset
		}
		resp, err = csv.LoadData(
			dataType,
			cfg.DataSettings.CSVData.FullPath,
			strings.ToLower(exch.GetName()),
			cfg.DataSettings.Interval.Duration(),
			fPair,
			a,
			isUSDTrackingPair)
		if err != nil {
			return nil, fmt.Errorf("%v. Please check your GoCryptoTrader configuration", err)
		}
		resp.Item.RemoveDuplicateCandlesByTime()
		resp.Item.SortCandlesByTimestamp(false)
		resp.RangeHolder, err = gctkline.CalculateCandleDateRanges(
			resp.Item.Candles[0].Time,
			resp.Item.Candles[len(resp.Item.Candles)-1].Time.Add(cfg.DataSettings.Interval.Duration()),
			cfg.DataSettings.Interval,
			0,
		)
		if err != nil {
			return nil, err
		}
		resp.RangeHolder.SetHasDataFromCandles(resp.Item.Candles)
		summary := resp.RangeHolder.DataSummary(false)
		if len(summary) > 0 {
			log.Warnf(common.Setup, "%v", summary)
		}
	case cfg.DataSettings.DatabaseData != nil:
		if cfg.DataSettings.DatabaseData.InclusiveEndDate {
			cfg.DataSettings.DatabaseData.EndDate = cfg.DataSettings.DatabaseData.EndDate.Add(cfg.DataSettings.Interval.Duration())
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
				log.Error(common.Setup, stopErr)
			}
		}()
		resp, err = loadDatabaseData(cfg, exch.GetName(), fPair, a, dataType, isUSDTrackingPair)
		if err != nil {
			return nil, fmt.Errorf("unable to retrieve data from GoCryptoTrader database. Error: %v. Please ensure the database is setup correctly and has data before use", err)
		}

		resp.Item.RemoveDuplicateCandlesByTime()
		resp.Item.SortCandlesByTimestamp(false)
		resp.RangeHolder, err = gctkline.CalculateCandleDateRanges(
			cfg.DataSettings.DatabaseData.StartDate,
			cfg.DataSettings.DatabaseData.EndDate,
			cfg.DataSettings.Interval,
			0,
		)
		if err != nil {
			return nil, err
		}
		resp.RangeHolder.SetHasDataFromCandles(resp.Item.Candles)
		summary := resp.RangeHolder.DataSummary(false)
		if len(summary) > 0 {
			log.Warnf(common.Setup, "%v", summary)
		}
	case cfg.DataSettings.APIData != nil:
		if cfg.DataSettings.APIData.InclusiveEndDate {
			cfg.DataSettings.APIData.EndDate = cfg.DataSettings.APIData.EndDate.Add(cfg.DataSettings.Interval.Duration())
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
		err = setExchangeCredentials(cfg, b)
		if err != nil {
			return nil, err
		}
		err = bt.LiveDataHandler.AppendDataSource(&liveDataSourceSetup{
			exchange:                  exch,
			interval:                  cfg.DataSettings.Interval,
			asset:                     a,
			pair:                      fPair,
			underlyingPair:            underlyingPair,
			dataType:                  dataType,
			dataRequestRetryTolerance: cfg.DataSettings.LiveData.DataRequestRetryTolerance,
			dataRequestRetryWaitTime:  cfg.DataSettings.LiveData.DataRequestRetryWaitTime,
			verboseExchangeRequest:    cfg.DataSettings.VerboseExchangeRequests,
		})
		return nil, err
	}
	if resp == nil {
		return nil, fmt.Errorf("processing error, response returned nil")
	}

	resp.Item.UnderlyingPair = underlyingPair
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
	err = bt.Reports.SetKlineData(&resp.Item)
	if err != nil {
		return nil, err
	}
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
		cfg.DataSettings.Interval.Duration(),
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
		cfg.DataSettings.Interval,
		resultLimit)
	if err != nil {
		return nil, err
	}
	candles, err := api.LoadData(context.TODO(),
		dataType,
		cfg.DataSettings.APIData.StartDate,
		cfg.DataSettings.APIData.EndDate,
		cfg.DataSettings.Interval.Duration(),
		exch,
		fPair,
		a)
	if err != nil {
		return nil, fmt.Errorf("%v. Please check your GoCryptoTrader configuration", err)
	}
	dates.SetHasDataFromCandles(candles.Candles)
	summary := dates.DataSummary(false)
	if len(summary) > 0 {
		log.Warnf(common.Setup, "%v", summary)
	}
	candles.FillMissingDataWithEmptyEntries(dates)
	candles.RemoveOutsideRange(cfg.DataSettings.APIData.StartDate, cfg.DataSettings.APIData.EndDate)
	return &kline.DataFromKline{
		Item:        *candles,
		RangeHolder: dates,
	}, nil
}

func setExchangeCredentials(cfg *config.Config, base *gctexchange.Base) error {
	if cfg == nil || base == nil || cfg.DataSettings.LiveData == nil {
		return common.ErrNilArguments
	}
	if cfg.DataSettings.Interval <= 0 {
		return errIntervalUnset
	}

	for i := range cfg.DataSettings.LiveData.ExchangeCredentials {
		if cfg.DataSettings.LiveData.ExchangeCredentials[i].Exchange != base.Name {
			continue
		}
		base.SetCredentials(
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.Key,
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.Secret,
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.ClientID,
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.SubAccount,
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.PEMKey,
			cfg.DataSettings.LiveData.ExchangeCredentials[i].Credentials.OneTimePassword,
		)
		validated := base.AreCredentialsValid(context.TODO())
		base.API.AuthenticatedSupport = validated
		if !validated && cfg.DataSettings.LiveData.RealOrders {
			log.Warn(common.Setup, "invalid API credentials set, real orders set to false")
			cfg.DataSettings.LiveData.RealOrders = false
		}

	}

	return nil
}
