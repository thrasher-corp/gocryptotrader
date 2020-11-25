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
	SetCurrency(string, asset.Item, currency.Pair, CurrencySettings)
	GetCurrencySettings(string, asset.Item, currency.Pair) CurrencySettings
	ExecuteOrder(OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}

type Exchange struct {
	Name             string
	UseRealOrders    bool
	CurrencySettings []CurrencySettings
}

type CurrencySettings struct {
	InitialFunds float64

	CurrencyPair currency.Pair
	AssetType    asset.Item

	ExchangeFee float64
	MakerFee    float64
	TakerFee    float64

	BuySide  config.MinMax
	SellSide config.MinMax

	Leverage config.Leverage

	MinimumSlippageRate float64
	MaximumSlippageRate float64
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
