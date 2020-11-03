package orders

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
)

type ExecutionHandler interface {
	ExecuteOrder(OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}
