package exchange

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/orders"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Exchange struct {
	CurrencyPair currency.Pair
	AssetType    asset.Item
	ExchangeFee  float64
	MakerFee     float64
	TakerFee     float64
	Orders       orders.Orders
}
