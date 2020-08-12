package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type Backtest struct {
	data      DataHandler
	portfolio PortfolioHandler
	algo      AlgoHandler

	config *Config
}

type Config struct {
	Item kline.Item
	Fee  float64
}

type Data struct{}

type AlgoHandler interface {
	Init() *Config
	OnData(t Data, b *Backtest) (bool, error)
	OnEnd(b *Backtest)
}
