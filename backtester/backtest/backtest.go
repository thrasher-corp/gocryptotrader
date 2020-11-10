package backtest

import (
	"errors"
	"fmt"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	kline2 "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/strategies"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange2 "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{}
}

// NewFromConfig takes a strategy config and configures a backtester variable to run
func NewFromConfig(configPath string) (*BackTest, error) {
	bt := New()
	cfg, err := config.ReadConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	err = botSetup(cfg)
	if err != nil {
		return nil, err
	}

	exch, fPair, a, base, err := loadExchangePairAssetBase(cfg)
	if err != nil {
		return nil, err
	}

	bt.Data, err = loadData(cfg, exch, fPair, a, base)
	if err != nil {
		return nil, err
	}
	err = bt.Data.Load()
	if err != nil {
		return nil, err
	}

	var makerFee, takerFee float64
	makerFee, takerFee, err = getFees(exch, fPair)
	if err != nil {
		return nil, err
	}

	bt.Exchange = &exchange.Exchange{
		Currency: exchange.Currency{
			CurrencyPair: fPair,
			AssetType:    a,
			ExchangeFee:  takerFee,
			MakerFee:     takerFee,
			TakerFee:     makerFee,
		},
		Orders: orders.Orders{},
	}

	bt.Portfolio = &portfolio.Portfolio{
		InitialFunds: cfg.ExchangeSettings.InitialFunds,
		SizeManager: &size.Size{
			DefaultSize: cfg.ExchangeSettings.DefaultOrderSize,
			MaxSize:     cfg.ExchangeSettings.MaximumOrderSize,
		},
		Funds:       cfg.ExchangeSettings.InitialFunds,
		RiskManager: &risk.Risk{},
	}

	// TODO: more nuanced maker/take fees
	bt.Portfolio.SetFee(cfg.ExchangeSettings.Name, a, fPair, takerFee)

	bt.Strategy, err = strategies.LoadStrategyByName(cfg.StrategyToLoad)
	if err != nil {
		return nil, err
	}

	bt.Statistic = &statistics.Statistic{
		StrategyName: cfg.StrategyToLoad,
	}

	return bt, nil
}

func loadExchangePairAssetBase(cfg *config.Config) (exchange2.IBotExchange, currency.Pair, asset.Item, *exchange2.Base, error) {
	var err error
	exch := engine.Bot.GetExchangeByName(cfg.ExchangeSettings.Name)
	if exch == nil {
		return nil, currency.Pair{}, "", nil, engine.ErrExchangeNotFound
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromStrings(cfg.ExchangeSettings.Base, cfg.ExchangeSettings.Quote)
	if err != nil {
		return nil, currency.Pair{}, "", nil, err
	}

	var a asset.Item
	a, err = asset.New(cfg.ExchangeSettings.Asset)
	if err != nil {
		return nil, currency.Pair{}, "", nil, err
	}

	base := exch.GetBase()
	if !base.ValidateAPICredentials() {
		log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", base.Name)
	}

	fPair, err = base.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, currency.Pair{}, "", nil, err
	}
	return exch, fPair, a, base, nil
}

func botSetup(cfg *config.Config) error {
	var err error
	engine.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return err
	}

	err = engine.Bot.LoadExchange(cfg.ExchangeSettings.Name, false, nil)
	if err != nil {
		return err
	}

	err = engine.Bot.OrderManager.Start()
	if err != nil {
		return err
	}

	return nil
}

func getFees(exch exchange2.IBotExchange, fPair currency.Pair) (makerFee float64, takerFee float64, err error) {
	takerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
		FeeType:       exchange2.CryptocurrencyTradeFee,
		Pair:          fPair,
		IsMaker:       false,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return makerFee, takerFee, err
	}

	makerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
		FeeType:       exchange2.CryptocurrencyTradeFee,
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

func loadData(cfg *config.Config, exch exchange2.IBotExchange, fPair currency.Pair, a asset.Item, base *exchange2.Base) (*kline2.DataFromKline, error) {
	if cfg.DatabaseData == nil && cfg.LiveData == nil && cfg.CandleData == nil {
		return nil, errors.New("no data settings set in config")
	}
	// load the data
	resp := &kline2.DataFromKline{}
	var err error
	if cfg.CandleData != nil && cfg.DatabaseData != nil ||
		cfg.CandleData != nil && cfg.LiveData != nil ||
		cfg.DatabaseData != nil && cfg.LiveData != nil {
		return nil, errors.New("ambiguous settings received. Only one data type can be set")
	}

	if cfg.CandleData != nil {
		var candles kline.Item
		candles, err = exch.GetHistoricCandlesExtended(fPair, a, cfg.CandleData.StartDate, cfg.CandleData.EndDate, kline.Interval(cfg.CandleData.Interval))
		if err != nil {
			return nil, err
		}

		resp.Item = candles
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
			log.Warn(log.BackTester, "bad credentials received, no live trading for run")
			cfg.LiveData.FakeOrders = true
		}
	} else if cfg.DatabaseData != nil {
		if cfg.DatabaseData.ConfigOverride != nil {
			engine.Bot.Config.Database = *cfg.DatabaseData.ConfigOverride
			err = engine.Bot.DatabaseManager.Start(engine.Bot)
			if err != nil {
				return nil, err
			}
		}
		if cfg.DatabaseData.DataType == "kline" {
			datarino, err := kline.LoadFromDatabase(
				cfg.ExchangeSettings.Name,
				fPair,
				a,
				kline.Interval(cfg.DatabaseData.Interval),
				cfg.DatabaseData.StartDate,
				cfg.DatabaseData.EndDate)
			if err != nil {
				return nil, err
			}
			resp.Item = datarino
		} else if cfg.DatabaseData.DataType == "trade" {
			trades, err := trade.GetTradesInRange(
				cfg.ExchangeSettings.Name,
				cfg.ExchangeSettings.Asset,
				cfg.ExchangeSettings.Base,
				cfg.ExchangeSettings.Quote,
				cfg.DatabaseData.StartDate,
				cfg.DatabaseData.EndDate)
			if err != nil {
				return nil, err
			}
			datarino, err := trade.ConvertTradesToCandles(
				kline.Interval(cfg.DatabaseData.Interval),
				trades...)
			if err != nil {
				return nil, err
			}
			resp.Item = datarino
		} else {
			return nil, fmt.Errorf("unexpected database datatype: %v", cfg.DatabaseData.DataType)
		}
	}

	return resp, nil
}

// Reset BackTest values to default
func (b *BackTest) Reset() {
	b.EventQueue = nil
	b.Data.Reset()
	b.Portfolio.Reset()
	b.Statistic.Reset()
}

// Run executes Backtest
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

		err := b.eventLoop(event)
		if err != nil {
			return err
		}
		b.Statistic.TrackEvent(event)
	}

	return nil
}

func (b *BackTest) nextEvent() (e interfaces.EventHandler, ok bool) {
	if len(b.EventQueue) == 0 {
		return e, false
	}

	e = b.EventQueue[0]
	b.EventQueue = b.EventQueue[1:]

	return e, true
}

func (b *BackTest) eventLoop(e interfaces.EventHandler) error {
	switch event := e.(type) {
	case interfaces.DataEventHandler:
		b.Portfolio.Update(event)
		b.Statistic.Update(event, b.Portfolio)
		s, err := b.Strategy.OnSignal(b.Data, b.Portfolio)
		if err != nil {
			log.Error(log.BackTester, err)
			break
		}
		b.EventQueue = append(b.EventQueue, s)

	case signal.SignalEvent:
		o, err := b.Portfolio.OnSignal(event, b.Data)
		if err != nil {
			log.Error(log.BackTester, err)
			break
		}
		b.EventQueue = append(b.EventQueue, o)

	case orders.OrderEvent:
		f, err := b.Exchange.ExecuteOrder(event, b.Data)
		if err != nil {
			log.Error(log.BackTester, err)
			break
		}
		b.EventQueue = append(b.EventQueue, f)
	case fill.FillEvent:
		t, err := b.Portfolio.OnFill(event, b.Data)
		if err != nil {
			log.Error(log.BackTester, err)
			break
		}
		b.Statistic.TrackTransaction(t)
	}

	return nil
}
