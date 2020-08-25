package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Backtest struct {
	data      DataHandler
	Portfolio PortfolioHandler
	Algo      AlgoHandler
	Execution ExecutionHandler
	Stats     StatisticHandler

	config *Config
}

type Config struct {
	Item         kline.Item
	Fee          float64
	InitialFunds float64
}

type Data struct{}

type AlgoHandler interface {
	Init() *Config
	OnData(d DataEvent, b *Backtest) (bool, error)
	OnEnd(b *Backtest)
}
