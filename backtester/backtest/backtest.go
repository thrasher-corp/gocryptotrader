package backtest

import (
	"sync"

	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"

	kline2 "github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	exchange2 "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

// New returns a new BackTest instance
func New() *BackTest {
	return &BackTest{}
}

func NewFromConfig(configPath string) (*BackTest, error) {
	bt := New()
	cfg, err := config.ReadConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}

	bt.Portfolio = &portfolio.Portfolio{
		FundsPerCurrency:       nil,
		Holdings:               nil,
		Transactions:           nil,
		SizeManager:            nil,
		SizeManagerPerCurrency: nil,
		RiskManager:            nil,
	}

	engine.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return nil, err
	}

	err = engine.Bot.OrderManager.Start()
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	var errs common.Errors
	for i := range cfg.ExchangePairSettings {
		go func(exchName string) {
			err = engine.Bot.LoadExchange(exchName, true, &wg)
			if err != nil {
				errs = append(errs, err)
			}
		}(i)
	}
	wg.Wait()
	if len(errs) > 0 {
		return nil, errs
	}

	for exchangeName := range cfg.ExchangePairSettings {
		exch := engine.Bot.GetExchangeByName(exchangeName)
		if exch == nil {
			return nil, engine.ErrExchangeNotFound
		}
		exchangeCurrencies, ok := cfg.ExchangePairSettings[exchangeName]
		if !ok {
			return nil, engine.ErrExchangeNotFound
		}

		for i := range exchangeCurrencies {
			var cp, fPair currency.Pair
			cp, err = currency.NewPairFromStrings(exchangeCurrencies[i].Base, exchangeCurrencies[i].Quote)
			if err != nil {
				return nil, err
			}

			var a asset.Item
			a, err = asset.New(exchangeCurrencies[i].Asset)
			if err != nil {
				return nil, err
			}

			base := exch.GetBase()
			if !base.ValidateAPICredentials() {
				log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", base.Name)
			}

			fPair, err = base.FormatExchangeCurrency(cp, a)
			if err != nil {
				return nil, err
			}
			// load the data
			var candles kline.Item
			candles, err = exch.GetHistoricCandlesExtended(fPair, a, cfg.StartDate, cfg.EndDate, kline.Interval(cfg.CandleData.Interval))
			if err != nil {
				return nil, err
			}

			bt.Datas = append(bt.Datas, &kline2.DataFromKline{
				Item: candles,
			})
			err = bt.Datas[len(bt.Datas)-1].Load()
			if err != nil {
				return nil, err
			}

			var makerFee, takerFee float64
			takerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
				FeeType:       exchange2.CryptocurrencyTradeFee,
				Pair:          fPair,
				IsMaker:       false,
				PurchasePrice: 1,
				Amount:        1,
			})
			if err != nil {
				return nil, err
			}

			makerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
				FeeType:       exchange2.CryptocurrencyTradeFee,
				Pair:          fPair,
				IsMaker:       true,
				PurchasePrice: 1,
				Amount:        1,
			})
			if err != nil {
				return nil, err
			}

			if bt.Exchanges[exchangeName] == nil {
				bt.Exchanges[exchangeName] = &exchange.Exchange{
					Orders: orders.Orders{},
				}
			}
			bt.Exchanges[exchangeName] = append()
		}
	}

	statistic := statistics.Statistic{
		StrategyName: cfg.StrategyToLoad,
	}
	bt.Statistic = &statistic

	return bt, nil
}

/*
// NewFromSettings creates a new backtester from cmd or config settings
func NewFromSettings(s *settings.Settings) (*BackTest, error) {
	bt := New()
	var err error

	bt.Portfolio, err = portfolio.New()
	if err != nil {
		return nil, err
	}

	// load exchange
	engine.Bot, err = engine.NewFromSettings(&engine.Settings{
		EnableDryRun:   true,
		EnableAllPairs: true,
	}, nil)
	if err != nil {
		return nil, err
	}
	err = engine.Bot.LoadExchange(s.ExchangeName, false, nil)
	if err != nil {
		return nil, err
	}

	err = engine.Bot.OrderManager.Start()
	if err != nil {
		return nil, err
	}
	exch := engine.Bot.GetExchangeByName(s.ExchangeName)
	if exch == nil {
		return nil, engine.ErrExchangeNotFound
	}

	var cp, fPair currency.Pair
	cp, err = currency.NewPairFromString(s.CurrencyPair)
	if err != nil {
		return nil, err
	}

	var a asset.Item
	a, err = asset.New(s.AssetType)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	if !base.ValidateAPICredentials() {
		log.Warnf(log.BackTester, "no credentials set for %v, this is theoretical only", base.Name)
	}

	fPair, err = base.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, err
	}
	var tStart, tEnd time.Time
	tStart, err = time.Parse(common.SimpleTimeFormat, s.StartTime)
	if err != nil {
		return nil, err
	}

	tEnd, err = time.Parse(common.SimpleTimeFormat, s.EndTime)
	if err != nil {
		return nil, err
	}

	// load the data
	var candles kline.Item
	candles, err = exch.GetHistoricCandlesExtended(fPair, a, tStart, tEnd, kline.Interval(s.Interval))
	if err != nil {
		return nil, err
	}

	bt.Data = &kline2.DataFromKline{
		Item: candles,
	}
	err = bt.Data.Load()
	if err != nil {
		return nil, err
	}

	bt.Strategy, err = strategies.LoadStrategyByName(s.StrategyName)
	if err != nil {
		return nil, err
	}

	var makerFee, takerFee float64
	takerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
		FeeType:       exchange2.CryptocurrencyTradeFee,
		Pair:          fPair,
		IsMaker:       false,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return nil, err
	}

	makerFee, err = exch.GetFeeByType(&exchange2.FeeBuilder{
		FeeType:       exchange2.CryptocurrencyTradeFee,
		Pair:          fPair,
		IsMaker:       true,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return nil, err
	}

	bt.Exchange = &exchange.Exchange{
		CurrencyPair: fPair,
		AssetType:    a,
		ExchangeFee:  takerFee,
		MakerFee:     makerFee,
		TakerFee:     takerFee,
		Orders:       orders.Orders{},
	}

	statistic := statistics.Statistic{
		StrategyName: s.StrategyName,
	}
	bt.Statistic = &statistic

	return bt, nil
}
*/
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
		// verify strategy
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
