package backtest

import (
	"time"

	data2 "github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
	"github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/settings"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/size"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics"
	"github.com/thrasher-corp/gocryptotrader/backtester/strategies"
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

// NewFromSettings creates a new backtester from cmd or config settings
func NewFromSettings(s *settings.Settings) (*BackTest, error) {
	bt := New()
	bt.Portfolio = &portfolio.Portfolio{
		InitialFunds: s.InitialFunds,
		RiskManager:  &risk.Risk{},
		SizeManager: &size.Size{
			DefaultSize:  100,
			DefaultValue: 1000,
		},
	}

	// load exchange
	var err error
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

	exch := engine.Bot.GetExchangeByName(s.ExchangeName)
	if exch == nil {
		return nil, engine.ErrExchangeNotFound
	}

	cp, err := currency.NewPairFromString(s.CurrencyPair)
	if err != nil {
		return nil, err
	}

	a, err := asset.New(s.AssetType)
	if err != nil {
		return nil, err
	}

	base := exch.GetBase()
	fPair, err := base.FormatExchangeCurrency(cp, a)
	if err != nil {
		return nil, err
	}
	tStart, err := time.Parse(common.SimpleTimeFormat, s.StartTime)
	if err != nil {
		return nil, err
	}

	tEnd, err := time.Parse(common.SimpleTimeFormat, s.EndTime)
	if err != nil {
		return nil, err
	}

	// load the data
	candles, err := exch.GetHistoricCandlesExtended(fPair, a, tStart, tEnd, kline.Interval(s.Interval))
	if err != nil {
		return nil, err
	}
	candles.SortCandlesByTimestamp(true)
	bt.Data = &data2.DataFromKline{
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

	takerFee, err := exch.GetFeeByType(&exchange2.FeeBuilder{
		FeeType:       exchange2.CryptocurrencyTradeFee,
		Pair:          fPair,
		IsMaker:       false,
		PurchasePrice: 1,
		Amount:        1,
	})
	if err != nil {
		return nil, err
	}

	makerFee, err := exch.GetFeeByType(&exchange2.FeeBuilder{
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
		MakerFee:     makerFee,
		TakerFee:     takerFee,
		CurrencyPair: fPair,
	}

	statistic := statistics.Statistic{
		StrategyName: s.StrategyName,
		Pair:         fPair.String(),
	}
	bt.Statistic = &statistic

	return bt, nil
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
	b.Portfolio.SetFunds(b.Portfolio.GetInitialFunds())
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

func (b *BackTest) nextEvent() (e datahandler.EventHandler, ok bool) {
	if len(b.EventQueue) == 0 {
		return e, false
	}

	e = b.EventQueue[0]
	b.EventQueue = b.EventQueue[1:]

	return e, true
}

func (b *BackTest) eventLoop(e datahandler.EventHandler) error {
	switch event := e.(type) {
	case datahandler.DataEventHandler:
		b.Portfolio.Update(event)
		b.Statistic.Update(event, b.Portfolio)

		s, err := b.Strategy.OnSignal(b.Data, b.Portfolio)
		if err != nil {
			log.Error(log.Global, err)
			break
		}
		b.EventQueue = append(b.EventQueue, s)

	case signal.SignalEvent:
		o, err := b.Portfolio.OnSignal(event, b.Data)
		if err != nil {
			log.Error(log.Global, err)
			break
		}
		b.EventQueue = append(b.EventQueue, o)

	case orderbook.OrderEvent:
		f, err := b.Exchange.ExecuteOrder(event, b.Data)
		if err != nil {
			log.Error(log.Global, err)
			break
		}
		b.Orderbook.Add(event)
		b.EventQueue = append(b.EventQueue, f)
	case fill.FillEvent:
		t, err := b.Portfolio.OnFill(event, b.Data)
		if err != nil {
			log.Error(log.Global, err)
			break
		}
		b.Statistic.TrackTransaction(t)
	}

	return nil
}
