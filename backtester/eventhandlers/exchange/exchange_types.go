package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/internalordermanager"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type ExecutionHandler interface {
	SetCurrency(CurrencySettings)
	GetCurrency() CurrencySettings
	ExecuteOrder(internalordermanager.OrderEvent, interfaces.DataHandler) (*fill.Fill, error)
}

type Exchange struct {
	CurrencySettings CurrencySettings
	Orders           internalordermanager.Orders
}

type CurrencySettings struct {
	CurrencyPair currency.Pair
	AssetType    asset.Item

	ExchangeFee float64
	MakerFee    float64
	TakerFee    float64

	MinimumBuySize float64
	MaximumBuySize float64
	DefaultBuySize float64

	MinimumSellSize float64
	MaximumSellSize float64
	DefaultSellSize float64

	CanUseLeverage  bool
	MaximumLeverage float64
}
