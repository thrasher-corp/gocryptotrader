package risk

import (
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Handler interface {
	EvaluateOrder(order.OrderEvent, []holdings.Holding, compliance.Snapshot) (*order.Order, error)
}

type Risk struct {
	MaxLeverageRatio    map[string]map[asset.Item]map[currency.Pair]float64
	MaxLeverageRate     map[string]map[asset.Item]map[currency.Pair]float64
	MaximumHoldingRatio map[string]map[asset.Item]map[currency.Pair]float64 // I cant think of a term, but the ratio between the entire portfolio, eg BTC cannot be more than 50% of holdings
	CanUseLeverage      bool
	MaximumLeverage     float64
}

type Settings struct {
	MaxLeverageRatio             float64
	MaxLeverageRate              float64
	MaxDiversificationPercentage float64 // I cant think of a term, but the ratio between the entire portfolio, eg BTC cannot be more than 50% of holdings
	CanUseLeverage               bool
	MaximumLeverage              float64
}
