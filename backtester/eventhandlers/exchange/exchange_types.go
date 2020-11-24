package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type ExecutionHandler interface {
	SetCurrency(CurrencySettings)
	GetCurrency() CurrencySettings
	ExecuteOrder(OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}

type Exchange struct {
	UseRealOrders       bool
	MinimumSlippageRate float64
	MaximumSlippageRate float64
	CurrencySettings    CurrencySettings
}

type CurrencySettings struct {
	CurrencyPair currency.Pair
	AssetType    asset.Item

	ExchangeFee float64
	MakerFee    float64
	TakerFee    float64

	BuySide  config.MinMax
	SellSide config.MinMax

	Leverage config.Leverage
}

// OrderEvent
type OrderEvent interface {
	interfaces.EventHandler
	interfaces.Directioner

	SetAmount(float64)
	GetAmount() float64
	IsOrder() bool
	GetWhy() string
	GetStatus() order.Status
	SetID(id string)
	GetID() string
	GetLimit() float64
	IsLeveraged() bool
}
