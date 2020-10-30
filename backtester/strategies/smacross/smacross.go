package smacross

import (
	portfolio "github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/event"
	portfolio2 "github.com/thrasher-corp/gocryptotrader/backtester/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/signal"
)

type Strategy struct{}

const name = "SMACross"

func (s *Strategy) Name() string {
	return name
}

func (s *Strategy) OnSignal(d portfolio.DataHandler, _ portfolio2.PortfolioHandler) (signal.SignalEvent, error) {
	signal := signal.Signal{
		Event: event.Event{Time: d.Latest().GetTime(),
			CurrencyPair: d.Latest().Pair()},
	}

	//smaFast := indicators.SMA(d.StreamClose(), 10)
	//smaSlow := indicators.SMA(d.StreamClose(), 30)

	//ret := indicators.Crossover(smaFast, smaSlow)
	//if ret {
	//	signal.SetDirection(order.Buy)
	//} else {
	//	signal.SetDirection(order.Sell)
	//}

	return &signal, nil
}
