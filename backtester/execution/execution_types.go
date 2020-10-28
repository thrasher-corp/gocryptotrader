package execution

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/datahandler"
	"github.com/thrasher-corp/gocryptotrader/backtester/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/orderbook"
)

type ExecutionHandler interface {
	ExecuteOrder(orderbook.OrderEvent, datahandler.DataHandler) (*fill.Fill, error)
}
