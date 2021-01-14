package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/engine"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type ExecutionHandler interface {
	SetCurrency(string, asset.Item, currency.Pair, *Settings)
	GetCurrencySettings(string, asset.Item, currency.Pair) (Settings, error)
	ExecuteOrder(order.Event, data.Handler, *engine.Engine) (*fill.Fill, error)
	Reset()
}

type Exchange struct {
	CurrencySettings []Settings
}

type Settings struct {
	ExchangeName  string
	UseRealOrders bool

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
