package portfolio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/settings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Portfolio struct {
	iteration                 float64
	riskFreeRate              float64
	sizeManager               SizeHandler
	riskManager               risk.Handler
	exchangeAssetPairSettings map[string]map[asset.Item]map[currency.Pair]*settings.Settings
}

type Handler interface {
	OnSignal(signal.Event, *exchange.Settings) (*order.Order, error)
	OnFill(fill.Event) (*fill.Fill, error)
	Update(common.DataEventHandler) error

	SetInitialFunds(string, asset.Item, currency.Pair, float64) error
	GetInitialFunds(string, asset.Item, currency.Pair) float64

	GetComplianceManager(string, asset.Item, currency.Pair) (*compliance.Manager, error)

	setHoldings(string, asset.Item, currency.Pair, holdings.Holding, bool) error
	ViewHoldingAtTimePeriod(string, asset.Item, currency.Pair, time.Time) (holdings.Holding, error)
	SetFee(string, asset.Item, currency.Pair, float64)
	GetFee(string, asset.Item, currency.Pair) float64
	Reset()
}

type SizeHandler interface {
	SizeOrder(order.Event, float64, *exchange.Settings) (*order.Order, error)
}
