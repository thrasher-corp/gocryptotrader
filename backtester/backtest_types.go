package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Backtest struct {
	data      DataHandler
	Portfolio PortfolioHandler
	algo      AlgoHandler
	execution ExecutionHandler

	config *Config
}

type Config struct {
	Item kline.Item
	Fee  float64
}

type Data struct{}

type AlgoHandler interface {
	Init() *Config
	OnData(d DataEvent, b *Backtest) (bool, error)
	OnEnd(b *Backtest)
}
