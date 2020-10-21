package smacross

import (
	backtest "github.com/thrasher-corp/gocryptotrader/backtester"
)

type Strategy struct{}

func (s *Strategy) OnSignal(d backtest.DataHandler, _ backtest.PortfolioHandler) (backtest.SignalEvent, error) {
	signal := backtest.Signal{
		Event: backtest.Event{Time: d.Latest().GetTime(),
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
