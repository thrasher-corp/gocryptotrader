package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type ExecutionHandler interface {
	SetCurrency(Currency)
	ExecuteOrder(orders.OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}

type Exchange struct {
	Currency Currency
	Orders   orders.Orders
}

type Currency struct {
	CurrencyPair currency.Pair
	AssetType    asset.Item
	ExchangeFee  float64
	MakerFee     float64
	TakerFee     float64
}
