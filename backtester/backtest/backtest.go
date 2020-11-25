package backtest

import (
	"errors"
	"fmt"
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
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
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
func (b *BackTest) Reset() {
	b.EventQueue = nil
	b.Data.Reset()
	b.Portfolio.Reset()
	b.Statistic.Reset()
}

func (b *BackTest) PrintSettings(cfg *config.Config) {
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Backtester Settings------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Strategy Settings--------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Strategy: %s", b.Strategy.Name())
	if len(cfg.StrategySettings) > 0 {
		log.Info(log.BackTester, "Custom strategy variables:")
		for k, v := range cfg.StrategySettings {
			log.Infof(log.BackTester, "%s: %v", k, v)
		}
	} else {
		log.Info(log.BackTester, "Custom strategy variables: unset")
	}
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Info(log.BackTester, "------------------Exchange Settings--------------------------")
	log.Info(log.BackTester, "-------------------------------------------------------------")
	log.Infof(log.BackTester, "Exchange: %s", cfg.ExchangeSettings.Name)
	for i := range cfg.ExchangeSettings.CurrencySettings {
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "------------------%v %v-%v Settings--------------------------",
			cfg.ExchangeSettings.CurrencySettings[i].Asset,
			cfg.ExchangeSettings.CurrencySettings[i].Base,
			cfg.ExchangeSettings.CurrencySettings[i].Quote)
		log.Info(log.BackTester, "-------------------------------------------------------------")
		log.Infof(log.BackTester, "Initial funds: %v", cfg.ExchangeSettings.CurrencySettings[i].InitialFunds)
		log.Infof(log.BackTester, "Maker fee: %v", cfg.ExchangeSettings.CurrencySettings[i].TakerFee)
		log.Infof(log.BackTester, "Taker fee: %v", cfg.ExchangeSettings.CurrencySettings[i].MakerFee)
		log.Infof(log.BackTester, "Buy rules: %+v", cfg.ExchangeSettings.CurrencySettings[i].BuySide)
		log.Infof(log.BackTester, "Sell rules: %+v", cfg.ExchangeSettings.CurrencySettings[i].SellSide)
		log.Infof(log.BackTester, "Leverage rules: %+v", cfg.ExchangeSettings.CurrencySettings[i].Leverage)
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

	exchangeroo, err := bt.setupExchangeSettings(cfg)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &exchangeroo

	portfoliooo := &portfolio.Portfolio{
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

	for i := range exchangeroo.CurrencySettings {
		lookup := portfoliooo.SetupExchangeAssetPairMap(exchangeroo.Name, exchangeroo.CurrencySettings[i].AssetType, exchangeroo.CurrencySettings[i].CurrencyPair)
		lookup.Fee = exchangeroo.CurrencySettings[i].TakerFee
		lookup.Leverage = exchangeroo.CurrencySettings[i].Leverage
		lookup.BuySideSizing = exchangeroo.CurrencySettings[i].BuySide
		lookup.SellSideSizing = exchangeroo.CurrencySettings[i].SellSide
		lookup.SetInitialFunds(exchangeroo.CurrencySettings[i].InitialFunds)
		lookup.SetFunds(exchangeroo.CurrencySettings[i].InitialFunds)
	}
	bt.Portfolio = portfoliooo

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

	bt.Statistic = &statistics.Statistic{
		StrategyName: cfg.StrategyToLoad,
	}

	bt.PrintSettings(cfg)

	return bt, nil
}

func (bt *BackTest) setupExchangeSettings(cfg *config.Config) (exchange.Exchange, error) {
	exchangeroo := exchange.Exchange{
		UseRealOrders: cfg.LiveData.RealOrders,
	}

	exch, fPair, a, err := bt.loadExchangePairAssetBase(cfg)
	if err != nil {
		return exchangeroo, err
	}

	bt.Data, err = loadData(cfg, exch, fPair, a)
	if err != nil {
		return exchangeroo, err
	}

	for i := range cfg.ExchangeSettings.CurrencySettings {
		var makerFee, takerFee float64

		if cfg.ExchangeSettings.CurrencySettings[i].MakerFee > 0 {
			makerFee = cfg.ExchangeSettings.CurrencySettings[i].MakerFee
		}
		if cfg.ExchangeSettings.CurrencySettings[i].TakerFee > 0 {
			takerFee = cfg.ExchangeSettings.CurrencySettings[i].TakerFee
		}
		if makerFee == 0 || takerFee == 0 {
			var apiMakerFee, apiTakerFee float64
			apiMakerFee, apiTakerFee, err = getFees(exch, fPair)
			if err != nil {
				return exchangeroo, err
			}
			if makerFee == 0 {
				makerFee = apiMakerFee
			}
			if takerFee == 0 {
				takerFee = apiTakerFee
			}
		}

		if cfg.ExchangeSettings.CurrencySettings[i].MaximumSlippagePercent <= 0 {
			cfg.ExchangeSettings.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}
		if cfg.ExchangeSettings.CurrencySettings[i].MinimumSlippagePercent <= 0 {
			cfg.ExchangeSettings.CurrencySettings[i].MinimumSlippagePercent = slippage.DefaultMinimumSlippagePercent
		}
		if cfg.ExchangeSettings.CurrencySettings[i].MaximumSlippagePercent <= cfg.ExchangeSettings.CurrencySettings[i].MinimumSlippagePercent {
			cfg.ExchangeSettings.CurrencySettings[i].MaximumSlippagePercent = slippage.DefaultMaximumSlippagePercent
		}

		exchangeroo.CurrencySettings = append(exchangeroo.CurrencySettings, exchange.CurrencySettings{
			InitialFunds:        cfg.ExchangeSettings.CurrencySettings[i].InitialFunds,
			MinimumSlippageRate: cfg.ExchangeSettings.CurrencySettings[i].MinimumSlippagePercent,
			MaximumSlippageRate: cfg.ExchangeSettings.CurrencySettings[i].MaximumSlippagePercent,
			CurrencyPair:        fPair,
			AssetType:           a,
			ExchangeFee:         takerFee,
			MakerFee:            takerFee,
			TakerFee:            makerFee,
			BuySide: config.MinMax{
				MinimumSize:  cfg.ExchangeSettings.CurrencySettings[i].BuySide.MinimumSize,
				MaximumSize:  cfg.ExchangeSettings.CurrencySettings[i].BuySide.MaximumSize,
				MaximumTotal: cfg.ExchangeSettings.CurrencySettings[i].BuySide.MaximumTotal,
			},
			SellSide: config.MinMax{
				MinimumSize:  cfg.ExchangeSettings.CurrencySettings[i].SellSide.MinimumSize,
				MaximumSize:  cfg.ExchangeSettings.CurrencySettings[i].SellSide.MaximumSize,
				MaximumTotal: cfg.ExchangeSettings.CurrencySettings[i].SellSide.MaximumTotal,
			},
			Leverage: config.Leverage{
				CanUseLeverage:  cfg.ExchangeSettings.CurrencySettings[i].Leverage.CanUseLeverage,
				MaximumLeverage: cfg.ExchangeSettings.CurrencySettings[i].Leverage.MaximumLeverage,
			},
		})
	}

	return exchangeroo, nil
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
	bt.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return err
	}

	err = bt.Bot.LoadExchange(cfg.ExchangeSettings.Name, false, nil)
	if err != nil {
		return err
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
		resp, err = csv.LoadData(cfg.CSVData.FullPath, cfg.CSVData.DataType, cfg.ExchangeSettings.Name, cfg.CSVData.Interval, fPair, a)
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

		go constantlyLoadLiveDataKThanksGuy(resp, cfg, exch, fPair, a)
		return resp, nil
	} else if cfg.DatabaseData != nil {
		resp, err = database.LoadData(
			cfg.DatabaseData.ConfigOverride,
			cfg.DatabaseData.StartDate,
			cfg.DatabaseData.EndDate,
			cfg.DatabaseData.Interval,
			cfg.ExchangeSettings.Name,
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

func constantlyLoadLiveDataKThanksGuy(resp *kline.DataFromKline, cfg *config.Config, exch gctexchange.IBotExchange, fPair currency.Pair, a asset.Item) {
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

func (b *BackTest) Stop() {
	b.shutdown <- struct{}{}
}

// Run will iterate over loaded data events
// save them and then handle the event based on its type
func (b *BackTest) Run() error {
	for event, ok := b.nextEvent(); true; event, ok = b.nextEvent() {
		if !ok {
			data, ok := b.Data.Next()
			if !ok {
				break
			}
			b.EventQueue = append(b.EventQueue, data)
			continue
		}

		err := b.handleEvent(event)
		if err != nil {
			return err
		}
		b.Statistic.TrackEvent(event)
	}

	return nil
}

func (b *BackTest) RunLive() error {
	timerino := time.NewTimer(time.Minute * 5)
	tickerino := time.NewTicker(time.Second)
	doneARun := false
	for {
		select {
		case <-b.shutdown:
			return nil
		case <-timerino.C:
			return errors.New("no data returned in 5 minutes, shutting down")
		case <-tickerino.C:
			//
			// Go get latest candle of interval X, verify that it hasn't been run before, then append the event
			//
			for event, ok := b.nextEvent(); true; event, ok = b.nextEvent() {
				doneARun = true
				if !ok {
					data, ok := b.Data.Next()
					if !ok {
						break
					}
					b.EventQueue = append(b.EventQueue, data)
					continue
				}

				err := b.handleEvent(event)
				if err != nil {
					return err
				}
				b.Statistic.TrackEvent(event)
			}
			if doneARun {
				timerino = time.NewTimer(time.Minute * 5)
			}
		}
	}
}

func (b *BackTest) nextEvent() (e interfaces.EventHandler, ok bool) {
	if len(b.EventQueue) == 0 {
		return e, false
	}

	e = b.EventQueue[0]
	b.EventQueue = b.EventQueue[1:]

	return e, true
}

// handleEvent switches based on the eventHandler type
// it will then act on the event and if needed, will add more events to the queue to be handled
func (b *BackTest) handleEvent(e interfaces.EventHandler) error {
	switch event := e.(type) {
	case interfaces.DataEventHandler:
		b.Portfolio.Update(event)
		b.Statistic.Update(event, b.Portfolio)
		s, err := b.Strategy.OnSignal(b.Data, b.Portfolio)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(gctcommon.SimpleTimeFormat), err.Error())
			break
		}
		b.EventQueue = append(b.EventQueue, s)

	case signal.SignalEvent:
		cs := b.Exchange.GetCurrencySettings(event.GetExchange(), event.GetAssetType(), event.Pair())
		o, err := b.Portfolio.OnSignal(event, b.Data, &cs)
		if err != nil {
			if errors.Is(err, portfolio.NoHoldingsToSellErr) || errors.Is(err, portfolio.NotEnoughFundsErr) {
				log.Warnf(log.BackTester, "%s - %s", e.GetTime().Format(gctcommon.SimpleTimeFormat), err.Error())
			} else {
				log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(gctcommon.SimpleTimeFormat), err.Error())
			}
			break
		}
		b.EventQueue = append(b.EventQueue, o)

	case exchange.OrderEvent:
		fillEvent, err := b.Exchange.ExecuteOrder(event, b.Data)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(gctcommon.SimpleTimeFormat), err.Error())
			break
		}
		b.EventQueue = append(b.EventQueue, fillEvent)
	case fill.FillEvent:
		//b.Compliance.GetSnapshot(event.GetTime())
		t, err := b.Portfolio.OnFill(event, b.Data)
		if err != nil {
			log.Errorf(log.BackTester, "%s - %s", e.GetTime().Format(gctcommon.SimpleTimeFormat), err.Error())
			break
		}
		b.Statistic.TrackTransaction(t)
	}

	return nil
}
