package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtest/fee"
	"github.com/thrasher-corp/gocryptotrader/currency"
)

type Handler interface{}

type Exchange struct {
	Pair currency.Pair

	Fee fee.Handler
}
