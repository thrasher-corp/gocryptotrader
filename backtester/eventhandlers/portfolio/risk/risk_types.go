package risk

import (
	"errors"

	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

var (
	errNoCurrencySettings       = errors.New("lacking currency settings, cannot evaluate order")
	errLeverageNotAllowed       = errors.New("order is using leverage when leverage is not enabled in config")
	errCannotPlaceLeverageOrder = errors.New("cannot place leveraged order")
)

// Handler defines what is expected to be able to assess risk of an order
type Handler interface {
	EvaluateOrder(order.Event, []holdings.Holding, compliance.Snapshot) (*order.Order, error)
}

// Risk contains all currency settings in order to evaluate potential orders
type Risk struct {
	CurrencySettings map[string]map[asset.Item]map[currency.Pair]*CurrencySettings
	CanUseLeverage   bool
	MaximumLeverage  float64
}

// CurrencySettings contains relevant limits to assess risk
type CurrencySettings struct {
	MaximumOrdersWithLeverageRatio float64
	MaxLeverageRate                float64
	MaximumHoldingRatio            float64
}
