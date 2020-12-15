package backtest

import (
	"errors"
	"fmt"
	"strings"
	"time"

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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics/currencystatstics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/report"
	"github.com/thrasher-corp/gocryptotrader/currency"
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
func NewFromConfig(cfg *config.Config) (*BackTest, error) {
	bt := New()
	err := bt.engineBotSetup(cfg)
	if err != nil {
		return nil, err
	}

	var e exchange.Exchange
	bt.Datas = &data.DataHolder{}
	bt.EventQueue = &eventholder.Holder{}
	reports := &report.Data{}

	e, err = bt.setupExchangeSettings(cfg, reports)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &e

	p := &portfolio.Portfolio{
		RiskFreeRate: cfg.StatisticSettings.RiskFreeRate,
		SizeManager: &size.Size{
			BuySide: config.MinMax{
				MinimumSize:  cfg.PortfolioSettings.BuySide.MinimumSize,
				MaximumSize:  cfg.PortfolioSettings.BuySide.MaximumSize,
				MaximumTotal: cfg.PortfolioSettings.BuySide.MaximumTotal,
			},
			SellSide: config.MinMax{
				MinimumSize:  cfg.PortfolioSettings.SellSide.MinimumSize,
				MaximumSize:  cfg.PortfolioSettings.SellSide.MaximumSize,
				MaximumTotal: cfg.PortfolioSettings.SellSide.MaximumTotal,
			},
			Leverage: config.Leverage{
				CanUseLeverage:  cfg.PortfolioSettings.Leverage.CanUseLeverage,
				MaximumLeverage: cfg.PortfolioSettings.Leverage.MaximumLeverage,
			},
		},
		RiskManager: &risk.Risk{
			MaxLeverageRatio:             nil,
			MaxLeverageRate:              nil,
			MaxDiversificationPercentage: nil,
		},
	}
	for i := range e.CurrencySettings {
		lookup := p.SetupExchangeAssetPairMap(e.CurrencySettings[i].ExchangeName, e.CurrencySettings[i].AssetType, e.CurrencySettings[i].CurrencyPair)
		lookup.Fee = e.CurrencySettings[i].TakerFee
		lookup.Leverage = e.CurrencySettings[i].Leverage
		lookup.BuySideSizing = e.CurrencySettings[i].BuySide
		lookup.SellSideSizing = e.CurrencySettings[i].SellSide
		lookup.SetInitialFunds(e.CurrencySettings[i].InitialFunds)
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
		if err != nil {
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

	bt.Reports = reports

	cfg.PrintSetting()

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config, reports *report.Data) (exchange.Exchange, error) {
	resp := exchange.Exchange{}

	for i := range cfg.CurrencySettings {
		exch, p, a, err := bt.loadExchangePairAssetBase(
			cfg.CurrencySettings[i].ExchangeName,
			cfg.CurrencySettings[i].Base,
			cfg.CurrencySettings[i].Quote,
			cfg.CurrencySettings[i].Asset)
		if err != nil {
			return resp, err
		}

		z := strings.ToLower(exch.GetName())
		bt.Datas.Setup()
		e, err := loadData(cfg, exch, p, a, reports)
		if err != nil {
			return resp, err
		}
		bt.Datas.AddDataForCurrency(z, a, p, e)
		var makerFee, takerFee float64

		if cfg.CurrencySettings[i].MakerFee > 0 {
			makerFee = cfg.CurrencySettings[i].MakerFee
		}
		if cfg.CurrencySettings[i].TakerFee > 0 {
			takerFee = cfg.CurrencySettings[i].TakerFee
		}
		if makerFee == 0 || takerFee == 0 {
			var apiMakerFee, apiTakerFee float64
			apiMakerFee, apiTakerFee, err = getFees(exch, p)
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

		if cfg.CurrencySettings[i].MaximumSlippagePercent <= 0 {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.CurrencySettings[i].MinimumSlippagePercent <= 0 {
			cfg.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
		}
		if cfg.CurrencySettings[i].MaximumSlippagePercent <= cfg.CurrencySettings[i].MinimumSlippagePercent {
			cfg.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}

		resp.CurrencySettings = append(resp.CurrencySettings, exchange.CurrencySettings{
			ExchangeName:        cfg.CurrencySettings[i].ExchangeName,
			InitialFunds:        cfg.CurrencySettings[i].InitialFunds,
			MinimumSlippageRate: cfg.CurrencySettings[i].MinimumSlippagePercent,
			MaximumSlippageRate: cfg.CurrencySettings[i].MaximumSlippagePercent,
			CurrencyPair:        p,
			AssetType:           a,
			ExchangeFee:         takerFee,
			MakerFee:            takerFee,
			TakerFee:            makerFee,
			BuySide: config.MinMax{
				MinimumSize:  cfg.CurrencySettings[i].BuySide.MinimumSize,
				MaximumSize:  cfg.CurrencySettings[i].BuySide.MaximumSize,
				MaximumTotal: cfg.CurrencySettings[i].BuySide.MaximumTotal,
			},
			SellSide: config.MinMax{
				MinimumSize:  cfg.CurrencySettings[i].SellSide.MinimumSize,
				MaximumSize:  cfg.CurrencySettings[i].SellSide.MaximumSize,
				MaximumTotal: cfg.CurrencySettings[i].SellSide.MaximumTotal,
			},
			Leverage: config.Leverage{
				CanUseLeverage:  cfg.CurrencySettings[i].Leverage.CanUseLeverage,
				MaximumLeverage: cfg.CurrencySettings[i].Leverage.MaximumLeverage,
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
	engine.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return err
	}

	bt.Bot = engine.Bot

	for i := range cfg.CurrencySettings {
		err = bt.Bot.LoadExchange(cfg.CurrencySettings[i].ExchangeName, false, nil)
		if err != nil && err.Error() != "exchange already loaded" {
			return err
		}
	}

	err = bt.Bot.OrderManager.Start()
	if err != nil {
		return err
	}

	return nil
}

// getFees will return an exchange's fee rate from GCT's wrapper function
func getFees(exch gctexchange.IBotExchange, fPair currency.Pair) (makerFee float64, takerFee float64, err error) {
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

// loadData will create kline data from the sources defined in strat config files. It can exist from databases, csv or API endpoints
// it can also be generated from trade data which will be converted into kline data
func loadData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item, reports *report.Data) (*kline.DataFromKline, error) {
	base := exch.GetBase()
	if cfg.DatabaseData == nil && cfg.LiveData == nil && cfg.APIData == nil && cfg.CSVData == nil {
		return nil, errors.New("no data settings set in config")
	}
	resp := &kline.DataFromKline{}
	var candles *gctkline.Item
	var err error
	if (cfg.APIData != nil && cfg.DatabaseData != nil) ||
		(cfg.APIData != nil && cfg.LiveData != nil) ||
		(cfg.APIData != nil && cfg.CSVData != nil) ||
		(cfg.DatabaseData != nil && cfg.LiveData != nil) ||
		(cfg.CSVData != nil && cfg.LiveData != nil) ||
		(cfg.CSVData != nil && cfg.DatabaseData != nil) {
		return nil, errors.New("ambiguous settings received. Only one data type can be set")
	}

	if cfg.CSVData != nil {
		resp, err = csv.LoadData(cfg.CSVData.FullPath, cfg.CSVData.DataType, strings.ToLower(exch.GetName()), cfg.CSVData.Interval, fPair, a)
		if err != nil {
			return nil, err
		}
	} else if cfg.APIData != nil {
		candles, err = api.LoadData(cfg.APIData.DataType, cfg.APIData.StartDate, cfg.APIData.EndDate, cfg.APIData.Interval, exch, fPair, a)
		if err != nil {
			return nil, err
		}

		resp = &kline.DataFromKline{
			Item: *candles,
		}
	} else if cfg.LiveData != nil {
		if cfg.LiveData.APIKeyOverride != "" {
			base.API.Credentials.Key = cfg.LiveData.APIKeyOverride
		}
		if cfg.LiveData.APISecretOverride != "" {
			base.API.Credentials.Secret = cfg.LiveData.APISecretOverride
		}
		if cfg.LiveData.APIClientIDOverride != "" {
			base.API.Credentials.ClientID = cfg.LiveData.APIClientIDOverride
		}
		if cfg.LiveData.API2FAOverride != "" {
			base.API.Credentials.PEMKey = cfg.LiveData.API2FAOverride
		}
		validated := base.ValidateAPICredentials()
		base.API.AuthenticatedSupport = validated
		if !validated {
			log.Warn(log.BackTester, "bad credentials received, no live trading for you")
			cfg.LiveData.RealOrders = false
		}

		go loadLiveDataLoop(resp, cfg, exch, fPair, a)
		return resp, nil
	} else if cfg.DatabaseData != nil {
		resp, err = database.LoadData(
			cfg.DatabaseData.ConfigOverride,
			cfg.DatabaseData.StartDate,
			cfg.DatabaseData.EndDate,
			cfg.DatabaseData.Interval,
			strings.ToLower(exch.GetName()),
			cfg.DatabaseData.DataType,
			fPair,
			a)
		if err != nil {
			return nil, err
		}
	}
	if resp == nil {
		return nil, fmt.Errorf("SOMEHOW ENDED UP IN THIS HOLE: %+v", cfg)
	}
	err = resp.Load()
	if err != nil {
		return nil, err
	}
	reports.OriginalCandles = append(reports.OriginalCandles, candles)
	return resp, nil
}

// loadLiveDataLoop is an incomplete function to continuously retrieve exchange data on a loop
// from live. Its purpose is to be able to perform strategy analysis against current data
func loadLiveDataLoop(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) {
	candles, err := live.LoadData(exch, cfg.LiveData.DataType, cfg.LiveData.Interval, fPair, a)
	if err != nil {
		log.Error(log.BackTester, err)
		return
	}
	resp.Append(*candles)
	timerino := time.NewTicker(time.Minute)
	for {
		select {
		case <-timerino.C:
			candles, err := live.LoadData(exch, cfg.LiveData.DataType, cfg.LiveData.Interval, fPair, a)
			if err != nil {
				log.Error(log.BackTester, err)
				return
			}
			resp.Append(*candles)
		}
	}
}

func (bt *BackTest) Stop() {
	bt.shutdown <- struct{}{}
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (bt *BackTest) Run() error {
dataLoadingIssue:
	for ev, ok := bt.EventQueue.NextEvent(); true; ev, ok = bt.EventQueue.NextEvent() {
		if !ok {
			d := bt.Datas.GetAllData()
			for i, e := range d {
				for j, a := range e {
					var z int64
					for k, p := range a {
						d, ok := p.Next()
						if !ok {
							if !hasHandledAnEvent {
								log.Errorf(log.BackTester, "Unable to perform `Next` for %v %v %v", i, j, k)
							}
							break dataLoadingIssue
						}
						if bt.Strategy.IsMultiCurrency() && z != 0 {
							continue
						}
						bt.EventQueue.AppendEvent(d)
						z++
					}
				}
			}
		}

		err := bt.handleEvent(ev)
		if err != nil {
			return err
		}
		if !hasHandledAnEvent {
			hasHandledAnEvent = true
		}
		//bt.Statistic.TrackEvent(ev)
	}

	return nil
}

// handleEvent switches based on the eventHandler type
// it will then act on the event and if needed, will add more events to the queue to be handled
func (bt *BackTest) handleEvent(e interfaces.EventHandler) error {
	switch ev := e.(type) {
	case interfaces.DataEventHandler:
		bt.appendSignalEventsFromDataEvents(ev)
	case signal.SignalEvent:
		cs := bt.Exchange.GetCurrencySettings(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
		d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
		o, err := bt.Portfolio.OnSignal(ev, d, &cs)
		if err != nil {
			bt.Statistic.AddExchangeEventForTime(o)
			break
		}

		bt.EventQueue.AppendEvent(o)
	case order.OrderEvent:
		d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
		f, err := bt.Exchange.ExecuteOrder(ev, d)
		if err != nil {
			bt.Statistic.AddFillEventForTime(f)
			break
		}
		bt.EventQueue.AppendEvent(f)
	case fill.FillEvent:
		d := bt.Datas.GetDataForCurrency(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
		t, err := bt.Portfolio.OnFill(ev, d)
		if err != nil {
			bt.Statistic.AddFillEventForTime(t)
			break
		}
		holding := bt.Portfolio.ViewHoldingAtTimePeriod(ev.GetExchange(), ev.GetAssetType(), ev.Pair(), ev.GetTime())
		bt.Statistic.AddHoldingsForTime(holding)
		cp, err := bt.Portfolio.GetComplianceManager(ev.GetExchange(), ev.GetAssetType(), ev.Pair())
		if err != nil {
			log.Error(log.BackTester, err)
		}
		snap, err := cp.GetSnapshot(ev.GetTime())
		if err != nil {
			log.Error(log.BackTester, err)
		}
		bt.Statistic.AddComplianceSnapshotForTime(snap, ev)
		bt.Statistic.AddFillEventForTime(t)
	}

	return nil
}

// appendSignalEventsFromDataEvents determines what signal events are generated and appended
// to the event queue based on whether it is running a multi-currency consideration strategy order not
//
// for multi-currency-consideration it will pass all currency datas to the strategy for it to determine what
// currencies to act upon
//
// for non-multi-currency-consideration strategies, it will simply process every currency individually
// against the strategy and generate signals
func (bt *BackTest) appendSignalEventsFromDataEvents(e interfaces.DataEventHandler) {
	if bt.Strategy.IsMultiCurrency() {
		var dataEvents []data.Handler
		ad := bt.Datas.GetAllData()
		for _, e := range ad {
			for _, a := range e {
				for _, p := range a {
					latestData := p.Latest()
					bt.updateStatsForDataEvent(latestData)
					dataEvents = append(dataEvents, p)
				}
			}
		}
		signals, err := bt.Strategy.OnSignals(dataEvents, bt.Portfolio)
		if err != nil {
			for i := range signals {
				bt.Statistic.AddSignalEventForTime(signals[i])
			}
			return
		}
		for i := range signals {
			bt.EventQueue.AppendEvent(signals[i])
		}
	} else {
		bt.updateStatsForDataEvent(e)
		d := bt.Datas.GetDataForCurrency(e.GetExchange(), e.GetAssetType(), e.Pair())
		s, err := bt.Strategy.OnSignal(d, bt.Portfolio)
		if err != nil {
			bt.Statistic.AddSignalEventForTime(s)
			return
		}
		bt.EventQueue.AppendEvent(s)
	}
}

// updateStatsForDataEvent makes various systems aware of price movements from
// data events
func (bt *BackTest) updateStatsForDataEvent(e interfaces.DataEventHandler) {
	// update portfolio with latest price
	bt.Portfolio.Update(e)
	// update statistics with latest price
	bt.Statistic.AddDataEventForTime(e)
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
				doneARun = true
				if !ok {
					d := bt.Datas.GetDataForCurrency(e.GetExchange(), e.GetAssetType(), e.Pair())
					de, ok := d.Next()
					if !ok {
						break
					}
					bt.EventQueue.AppendEvent(de)
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
