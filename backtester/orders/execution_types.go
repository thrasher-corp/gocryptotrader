package orders

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
)

type ExecutionHandler interface {
	ExecuteOrder(OrderEvent, datahandler.DataHandler) (*fill.Fill, error)
}
