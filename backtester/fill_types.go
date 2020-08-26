package backtest

import (
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Fill struct {
	Event
	Exchange    string
	Direction   order.Side
	Amount      float64
	Price       float64
	Commission  float64
	ExchangeFee float64
	Cost        float64
}
