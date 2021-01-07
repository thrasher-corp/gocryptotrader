package portfolio

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/risk"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

type Portfolio struct {
	iteration                 float64
	RiskFreeRate              float64
	SizeManager               SizeHandler
	RiskManager               risk.RiskHandler
	ExchangeAssetPairSettings map[string]map[asset.Item]map[currency.Pair]*ExchangeAssetPairSettings
}

type ExchangeAssetPairSettings struct {
	InitialFunds      float64
	Fee               float64
	BuySideSizing     config.MinMax
	SellSideSizing    config.MinMax
	Leverage          config.Leverage
	HoldingsSnapshots holdings.Snapshots
	ComplianceManager compliance.Manager
}

type Handler interface {
	OnSignal(signal.SignalEvent, data.Handler, *exchange.CurrencySettings) (*order.Order, error)
	OnFill(fill.FillEvent, data.Handler) (*fill.Fill, error)
	Update(common.DataEventHandler)

	SetInitialFunds(string, asset.Item, currency.Pair, float64)
	GetInitialFunds(string, asset.Item, currency.Pair) float64

	GetComplianceManager(string, asset.Item, currency.Pair) (*compliance.Manager, error)

	SetHoldings(string, asset.Item, currency.Pair, time.Time, holdings.Holding, bool) error
	ViewHoldingAtTimePeriod(string, asset.Item, currency.Pair, time.Time) holdings.Holding
	SetFee(string, asset.Item, currency.Pair, float64)
	GetFee(string, asset.Item, currency.Pair) float64
	Reset()
}

type SizeHandler interface {
	SizeOrder(order.OrderEvent, float64, *exchange.CurrencySettings) (*order.Order, error)
}
