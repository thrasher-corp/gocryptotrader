package backtest

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/api"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/csv"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/database"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline/live"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange/slippage"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	gctcommon "github.com/thrasher-corp/gocryptotrader/common"
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
	bt.EventQueue = nil
	bt.Data.Reset()
	bt.Portfolio.Reset()
	bt.Statistic.Reset()
}

func (bt *BackTest) PrintSettings(cfg *config.Config) {
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Backtester Settings------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Strategy Settings--------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Strategy: %s", bt.Strategy.Name())
	if len(cfg.StrategySettings) > 0 {
		log.Info(log.BackTester, "Custom strategy variables:")
		for k, v := range cfg.StrategySettings {
			log.Infof(log.BackTester, "%s: %v", k, v)
		}
	} else {
		log.Info(log.BackTester, "Custom strategy variables: unset")
	}
	for i := range cfg.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		currStr := fmt.Sprintf("------------------%v %v-%v Settings--------------------------",
			cfg.CurrencySettings[i].Asset,
			cfg.CurrencySettings[i].Base,
			cfg.CurrencySettings[i].Quote)
		log.Infof(log.BackTester, currStr[:61])
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Exchange: %v", cfg.CurrencySettings[i].ExchangeName)
		log.Infof(log.BackTester, "Initial funds: %v", cfg.CurrencySettings[i].InitialFunds)
		log.Infof(log.BackTester, "Maker fee: %v", cfg.CurrencySettings[i].TakerFee)
		log.Infof(log.BackTester, "Taker fee: %v", cfg.CurrencySettings[i].MakerFee)
		log.Infof(log.BackTester, "Buy rules: %+v", cfg.CurrencySettings[i].BuySide)
		log.Infof(log.BackTester, "Sell rules: %+v", cfg.CurrencySettings[i].SellSide)
		log.Infof(log.BackTester, "Leverage rules: %+v", cfg.CurrencySettings[i].Leverage)
	}
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Portfolio Settings-------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Buy rules: %+v", cfg.PortfolioSettings.BuySide)
	log.Infof(log.BackTester, "Sell rules: %+v", cfg.PortfolioSettings.SellSide)
	log.Infof(log.BackTester, "Leverage rules: %+v", cfg.PortfolioSettings.Leverage)
	if cfg.LiveData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Live Settings------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.LiveData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.LiveData.Interval)
		log.Infof(log.BackTester, "REAL ORDERS: %v", cfg.LiveData.RealOrders)
		log.Infof(log.BackTester, "Overriding GCT API settings: %v", cfg.LiveData.APIClientIDOverride != "")
	}
	if cfg.APIData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------API Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.APIData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.APIData.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.APIData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.APIData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	if cfg.CSVData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------CSV Settings-------------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.CSVData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.CSVData.Interval)
	}
	if cfg.DatabaseData != nil {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Info(log.BackTester, "------------------Database Settings--------------------------")
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Data type: %v", cfg.DatabaseData.DataType)
		log.Infof(log.BackTester, "Interval: %v", cfg.DatabaseData.Interval)
		log.Infof(log.BackTester, "Start date: %v", cfg.DatabaseData.StartDate.Format(gctcommon.SimpleTimeFormat))
		log.Infof(log.BackTester, "End date: %v", cfg.DatabaseData.EndDate.Format(gctcommon.SimpleTimeFormat))
	}
	log.Info(log.BackTester, "-------------------------------------------------------------")
}

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(cfg *config.Config) (*BackTest, error) {
	bt := New()
	err := bt.engineBotSetup(cfg)
	if err != nil {
		return nil, err
	}

	var e exchange.Exchange
	e, err = bt.setupExchangeSettings(cfg)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &e

	p := &portfolio.Portfolio{
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

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategyToLoad)
	if err != nil {
		return nil, err
	}
	if cfg.StrategySettings != nil {
		err = bt.Strategy.SetCustomSettings(cfg.StrategySettings)
		if err != nil {
			return nil, err
		}
	} else {
		bt.Strategy.SetDefaults()
	}

	stats := &statistics.Statistic{
		StrategyName: cfg.StrategyToLoad,
		EventsByTime: make(map[string]map[asset.Item]map[currency.Pair][]statistics.EventStore),
	}
	bt.Statistic = stats
	bt.PrintSettings(cfg)

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	e := exchange.Exchange{}

	for i := range cfg.CurrencySettings {
		exch, fPair, a, err := bt.loadExchangePairAssetBase(cfg.CurrencySettings[i].ExchangeName, cfg.CurrencySettings[i].Base, cfg.CurrencySettings[i].Quote, cfg.CurrencySettings[i].Asset)
		if err != nil {
			return e, err
		}

		// despite going to all this effort, we only support one data for now
		bt.Data, err = loadData(cfg, exch, fPair, a)
		if err != nil {
			return e, err
		}
		var makerFee, takerFee float64

		if cfg.CurrencySettings[i].MakerFee > 0 {
			makerFee = cfg.CurrencySettings[i].MakerFee
		}
		if cfg.CurrencySettings[i].TakerFee > 0 {
			takerFee = cfg.CurrencySettings[i].TakerFee
		}
		if makerFee == 0 || takerFee == 0 {
			var apiMakerFee, apiTakerFee float64
			apiMakerFee, apiTakerFee, err = getFees(exch, fPair)
			if err != nil {
				return e, err
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

		e.CurrencySettings = append(e.CurrencySettings, exchange.CurrencySettings{
			ExchangeName:        cfg.CurrencySettings[i].ExchangeName,
			InitialFunds:        cfg.CurrencySettings[i].InitialFunds,
			MinimumSlippageRate: cfg.CurrencySettings[i].MinimumSlippagePercent,
			MaximumSlippageRate: cfg.CurrencySettings[i].MaximumSlippagePercent,
			CurrencyPair:        fPair,
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

	return e, nil
}

func (bt *BackTest) loadExchangePairAssetBase(exch, baaa, quote, ass string) (gctexchange.IBotExchange, currency.Pair, asset.Item, error) {
	var err error
	e := bt.Bot.GetExchangeByName(exch)
	if e == nil {
		return nil, currency.Pair{}, "", engine.ErrExchangeNotFound
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromStrings(baaa, quote)
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
		if err != nil {
			return err
		}
	}

	err = bt.Bot.OrderManager.Start()
	if err != nil {
		return err
	}

	return nil
}

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

func loadData(cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) (*kline.DataFromKline, error) {
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

	return resp, nil
}

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
	for event, ok := bt.nextEvent(); true; event, ok = bt.nextEvent() {
		if !ok {
			data, ok := bt.Data.Next()
			if !ok {
				break
			}
			bt.EventQueue = append(bt.EventQueue, data)
			continue
		}

		err := bt.handleEvent(event)
		if err != nil {
			return err
		}
		bt.Statistic.TrackEvent(event)
	}

	return nil
}

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
			//
			// Go get latest candle of interval X, verify that it hasn't been run before, then append the event
			//
			for event, ok := bt.nextEvent(); true; event, ok = bt.nextEvent() {
				doneARun = true
				if !ok {
					data, ok := bt.Data.Next()
					if !ok {
						break
					}
					bt.EventQueue = append(bt.EventQueue, data)
					continue
				}

				err := bt.handleEvent(event)
				if err != nil {
					return err
				}
				bt.Statistic.TrackEvent(event)
			}
			if doneARun {
				timerino = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

func (bt *BackTest) nextEvent() (e interfaces.EventHandler, ok bool) {
	if len(bt.EventQueue) == 0 {
		return e, false
	}

	e = bt.EventQueue[0]
	bt.EventQueue = bt.EventQueue[1:]

	return e, true
}

// handleEvent switches based on the eventHandler type
// it will then act on the event and if needed, will add more events to the queue to be handled
func (bt *BackTest) handleEvent(e interfaces.EventHandler) error {
	switch event := e.(type) {
	case interfaces.DataEventHandler:
		// update portfolio with latest price
		bt.Portfolio.Update(event)
		// update statistics with latest price
		bt.Statistic.Update(event, bt.Portfolio)
		bt.Statistic.AddDataEventForTime(event)

		s, err := bt.Strategy.OnSignal(bt.Data, bt.Portfolio)
		if err != nil {
			bt.Statistic.AddSignalEventForTime(s)
			break
		}

		bt.EventQueue = append(bt.EventQueue, s)
	case signal.SignalEvent:
		cs := bt.Exchange.GetCurrencySettings(event.GetExchange(), event.GetAssetType(), event.Pair())
		o, err := bt.Portfolio.OnSignal(event, bt.Data, &cs)
		if err != nil {
			bt.Statistic.AddExchangeEventForTime(o)
			break
		}

		bt.EventQueue = append(bt.EventQueue, o)
	case order.OrderEvent:
		f, err := bt.Exchange.ExecuteOrder(event, bt.Data)
		if err != nil {
			bt.Statistic.AddFillEventForTime(f)
			break
		}
		bt.EventQueue = append(bt.EventQueue, f)
	case fill.FillEvent:
		t, err := bt.Portfolio.OnFill(event, bt.Data)
		if err != nil {
			bt.Statistic.AddFillEventForTime(t)
			break
		}
		holding := bt.Portfolio.ViewHoldingAtTimePeriod(event.GetExchange(), event.GetAssetType(), event.Pair(), event.GetTime())
		bt.Statistic.AddHoldingsForTime(holding)
		cp, err := bt.Portfolio.GetComplianceManager(event.GetExchange(), event.GetAssetType(), event.Pair())
		if err != nil {
			log.Error(log.BackTester, err)
		}
		snap, err := cp.GetSnapshot(event.GetTime())
		if err != nil {
			log.Error(log.BackTester, err)
		}
		bt.Statistic.AddComplianceSnapshotForTime(snap, event)
		bt.Statistic.AddFillEventForTime(t)
		bt.Statistic.TrackTransaction(t)
	}

	return nil
}
