package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/interfaces"
	"github.com/thrasher-corp/gocryptotrader/backtester/statistics/position"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type RiskHandler interface {
	EvaluateOrder(exchange.OrderEvent, interfaces.DataEventHandler, position.Position) (*order.Order, error)
}

type Risk struct {
	MaxLeverageRatio             map[string]map[asset.Item]map[currency.Pair]float64
	MaxLeverageRate              map[string]map[asset.Item]map[currency.Pair]float64
	MaxDiversificationPercentage map[string]map[asset.Item]map[currency.Pair]float64 // I cant think of a term, but the ratio between the entire portfolio, eg BTC cannot be more than 50% of holdings
	CanUseLeverage               bool
	MaximumLeverage              float64
}
