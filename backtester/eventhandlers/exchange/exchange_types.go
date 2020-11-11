package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type ExecutionHandler interface {
	SetCurrency(Currency)
	ExecuteOrder(internalordermanager.OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}

type Exchange struct {
	Currency Currency
	Orders   internalordermanager.Orders
}

type Currency struct {
	CurrencyPair currency.Pair
	AssetType    asset.Item
	ExchangeFee  float64
	MakerFee     float64
	TakerFee     float64
}
